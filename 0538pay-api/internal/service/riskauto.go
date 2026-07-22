package service

import (
	"fmt"
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
)

// RiskAutoService 风控自动关停（对齐 epay cron do=check）：
//   - 连续通知失败关停：窗口内某商户通知放弃(notify=-1)单数达阈值 → 关停支付 + 写 pay_risk type=2。
//   - 商户成功率关停：窗口内订单数达样本下限且成功率低于阈值 → 关停支付 + 写 pay_risk type=1。
//
// 由 scheduler 定时调用。阈值来自 config group=risk（保存即热更新）。
type RiskAutoService struct {
	repo *repository.RiskAutoRepo
	cfg  *ConfigService
}

func NewRiskAutoService(repo *repository.RiskAutoRepo, cfg *ConfigService) *RiskAutoService {
	return &RiskAutoService{repo: repo, cfg: cfg}
}

// Run 执行一轮风控自动检查，返回本轮关停商户数。
func (s *RiskAutoService) Run() (int, error) {
	closed := 0
	closed += s.checkNotifyFail()
	closed += s.checkSuccessRate()
	return closed, nil
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
	// 窗口取近 24 小时（通知放弃是较慢的信号）。
	start := time.Now().Add(-24 * time.Hour)
	uids, err := s.repo.MerchantsWithGivenUpNotify(count, start)
	if err != nil {
		return 0
	}
	n := 0
	for _, uid := range uids {
		ok, err := s.repo.ClosePay(uid)
		if err != nil || !ok {
			continue
		}
		_ = s.repo.WriteRisk(&model.RiskRecord{
			UID: uid, Type: 2, Content: fmt.Sprintf("连续通知失败达%d单，自动关停支付", count), Date: time.Now(),
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
