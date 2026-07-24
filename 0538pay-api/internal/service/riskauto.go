package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// RiskAutoService 风控自动关停（对齐 epay cron do=check）：
//   - 连续通知失败关停：某商户最近 N 个 status>0 订单全部 notify>0 → 关停支付 + 写 pay_risk type=2。
//   - 商户成功率关停：窗口内订单数达样本下限且成功率低于阈值 → 关停支付 + 写 pay_risk type=1。
//   - 通道/子通道关停：窗口内最近 N 个订单全部未支付 → 关停该(子)通道 status=0（不写商户风控记录）。
//
// 由 scheduler 定时调用。阈值来自 config group=risk（保存即热更新）。
type RiskAutoService struct {
	repo *repository.RiskAutoRepo
	cfg  *ConfigService
}

func NewRiskAutoService(repo *repository.RiskAutoRepo, cfg *ConfigService) *RiskAutoService {
	return &RiskAutoService{repo: repo, cfg: cfg}
}

// Run 执行一轮风控自动检查，返回本轮关停数（商户+通道）。
func (s *RiskAutoService) Run() (int, error) {
	closed := 0
	closed += s.checkNotifyFail()
	closed += s.checkSuccessRate()
	closed += s.checkChannelRate()
	return closed, nil
}

// checkChannelRate 通道/子通道自动关停（F-5/F-7，1:1 对齐 epay cron do=check 的 auto_check_channel 段）。
// 口径修正：epay 不是「成功率百分比」，而是「窗口内最近 failcount 个订单【全部未支付】→ 关停」。
// 两级：主通道 config 含方括号占位且有启用子通道时，逐子通道按窗口判定并只关停子通道；否则整通道判定关停。
// F-7 修正：epay 通道关停仅 echo+邮件，不写 pre_risk(pre_risk 只记商户维度)，故这里不再写 pay_risk。
// 邮件通知依赖 SMTP 凭证，待凭证；核心关停闭环已落库。
func (s *RiskAutoService) checkChannelRate() int {
	if s.cfg.Int("auto_check_channel", 0) != 1 {
		return 0
	}
	windowSec := s.cfg.Int("check_channel_second", 600)
	failCount := s.cfg.Int("check_channel_failcount", 0)
	if windowSec <= 0 || failCount <= 0 {
		return 0 // 对齐 epay：second==0||failcount==0 视为未开启
	}
	channelIDs := parseIDList(s.cfg.Str("check_channel_ids")) // 留空=全部通道
	start := time.Now().Add(-time.Duration(windowSec) * time.Second)
	channels, err := s.repo.EnabledChannelsForCheck(channelIDs)
	if err != nil {
		return 0
	}
	n := 0
	for i := range channels {
		ch := &channels[i]
		cid := int(ch.ID)
		// 判定是否走子通道级：config 含 [ 与 ] 占位 且 有启用子通道（对齐 epay 判定条件）。
		hasSubPlaceholder := strings.Contains(ch.Config, "[") && strings.Contains(ch.Config, "]")
		if hasSubPlaceholder {
			if subs, e := s.repo.EnabledSubchannelsOfChannel(cid); e == nil && len(subs) > 0 {
				for j := range subs {
					sub := &subs[j]
					allUnpaid, e := s.repo.LastNOrdersAllUnpaidBySubchannel(cid, int(sub.ID), start, failCount)
					if e != nil || !allUnpaid {
						continue
					}
					if ok, _ := s.repo.CloseSubchannel(sub.ID); ok {
						n++
					}
				}
				continue // 已走子通道级，不再整通道判定
			}
		}
		// 通道级：最近 failCount 单全未支付 → 关停整通道。
		allUnpaid, e := s.repo.LastNOrdersAllUnpaidByChannel(cid, start, failCount)
		if e != nil || !allUnpaid {
			continue
		}
		if ok, _ := s.repo.CloseChannel(cid); ok {
			n++
		}
	}
	return n
}

// parseIDList 解析逗号分隔的通道ID列表（对齐 epay check_channel_ids，留空返回 nil=全部）。
func parseIDList(s string) []int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var ids []int
	for _, part := range strings.Split(s, ",") {
		if v, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && v > 0 {
			ids = append(ids, v)
		}
	}
	return ids
}

// checkNotifyFail 连续通知失败关停（type=2）。
func (s *RiskAutoService) checkNotifyFail() int {
	if s.cfg.Int("auto_check_notify", 0) != 1 {
		return 0
	}
	count := s.cfg.Int("check_notify_count", 10)
	if count <= 0 {
		return 0
	}
	// F-6 口径对齐 epay cron.php:183-203：候选=近窗口内有 notify>0(重试中或放弃)订单的 pay=1 商户，
	// 再逐个判「最近 count 个 status>0 订单是否全部 notify>0」，全失败才关停（不再只数 notify=-1）。
	start := time.Now().Add(-24 * time.Hour)
	uids, err := s.repo.MerchantsWithRecentNotifyFail(start)
	if err != nil {
		return 0
	}
	n := 0
	for _, uid := range uids {
		allFail, err := s.repo.LastNOrdersAllNotifyFail(uid, count)
		if err != nil || !allFail {
			continue
		}
		ok, err := s.repo.ClosePay(uid)
		if err != nil || !ok {
			continue
		}
		_ = s.repo.WriteRisk(&model.RiskRecord{
			UID: uid, Type: 2, Content: fmt.Sprintf("连续%d个订单通知失败，自动关停支付", count), Date: time.Now(),
		})
		n++
	}
	return n
}

// checkSuccessRate 商户成功率关停（type=1）。
func (s *RiskAutoService) checkSuccessRate() int {
	if s.cfg.Int("auto_check_sucrate", 0) != 1 {
		return 0
	}
	windowSec := s.cfg.Int("check_sucrate_second", 600)
	minCount := s.cfg.Int("check_sucrate_count", 20)
	minRate := s.cfg.Int("check_sucrate_value", 30)
	if windowSec <= 0 || minCount <= 0 {
		return 0
	}
	start := time.Now().Add(-time.Duration(windowSec) * time.Second)
	uids, err := s.repo.ActivePayMerchants(start)
	if err != nil {
		return 0
	}
	n := 0
	for _, uid := range uids {
		total, paid, err := s.repo.MerchantOrderRate(uid, start)
		if err != nil || total < int64(minCount) {
			continue // 样本不足不判定，避免误伤
		}
		rate := paid * 100 / total
		if rate >= int64(minRate) {
			continue
		}
		ok, err := s.repo.ClosePay(uid)
		if err != nil || !ok {
			continue
		}
		_ = s.repo.WriteRisk(&model.RiskRecord{
			UID: uid, Type: 1,
			Content: fmt.Sprintf("成功率%d%%(低于%d%%,%d单)，自动关停支付", rate, minRate, total),
			Date:    time.Now(),
		})
		n++
	}
	return n
}
