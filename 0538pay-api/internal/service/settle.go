package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 结算计费配置（对齐 epay pre_config）。初始为默认值，config 域加载后由 reloadSettleConfig 覆盖，
// 且系统设置保存时经 ConfigService.OnChange 回调实时刷新。键名对齐 epay set.php。
var (
	settleRate   = decimal.RequireFromString("0.5")  // settle_rate 结算手续费率（%）
	settleMoney  = decimal.RequireFromString("30")   // settle_money 最低结算金额（也是自动结算门槛）
	settleFeeMin = decimal.RequireFromString("0.1")  // settle_fee_min 手续费封底
	settleFeeMax = decimal.RequireFromString("20")   // settle_fee_max 手续费封顶
	hundred      = decimal.RequireFromString("100")
)

// 结算周期与提现次数配置（对齐 epay settle_type / settle_maxlimit）。
var (
	settleTypeDPlus1 = true // settle_type=1 时 D+1（可提现余额扣当日已收）；0=D+0 全部余额
	settleMaxLimit   = 5    // settle_maxlimit 每日手动提现次数上限（0=不限）
)

// ApplyConfig 用 config 域当前值刷新所有业务服务的缓存常量（结算/代付/退款/保证金实名/支付屏蔽词）。
// 启动时调一次，并注册到 ConfigService.OnChange，设置保存后自动刷新。
func ApplyConfig(cfg *ConfigService) {
	reloadSettleConfig(cfg)
	reloadOrderConfig(cfg)
	reloadMerchantCenterConfig(cfg)
	reloadPayConfig(cfg)
	payPwdVerifier = cfg // 支付密码校验器（转账/结算/退款二次校验统一走 admin_paypwd）
}

// payPwdVerifier 持有配置服务，供各资金操作校验管理员支付密码（对齐 epay admin_paypwd）。
// 由 ApplyConfig 在启动时挂上；未挂时 verifyPayPwd 返回不可用错误（防止误放行）。
var payPwdVerifier interface {
	VerifyPayPwd(pwd string) error
}

// verifyPayPwd 统一的管理员支付密码校验入口（transfer/settle/order_write 共用）。
func verifyPayPwd(pwd string) error {
	if payPwdVerifier == nil {
		return stErr("支付密码校验不可用")
	}
	return payPwdVerifier.VerifyPayPwd(pwd)
}

// reloadSettleConfig 从 config 域刷新结算/代付相关常量（启动 + 设置保存后调用）。
// 放这里因结算常量在本文件；代付 transfer_* 也一并刷新（transfer.go 用到）。
func reloadSettleConfig(cfg *ConfigService) {
	settleRate = cfg.Dec("settle_rate", settleRate)
	settleMoney = cfg.Dec("settle_money", settleMoney)
	settleFeeMin = cfg.Dec("settle_fee_min", settleFeeMin)
	settleFeeMax = cfg.Dec("settle_fee_max", settleFeeMax)
	settleTypeDPlus1 = cfg.Str("settle_type") != "0" // 非 "0" 即 D+1
	settleMaxLimit = cfg.Int("settle_maxlimit", settleMaxLimit)
	reloadTransferConfig(cfg)
}

// settleTypeName 结算方式 ID → 名称（对齐前端 mock settleTypeMeta）。
func settleTypeName(t int8) string {
	switch t {
	case 1:
		return "支付宝"
	case 2:
		return "微信"
	case 3:
		return "QQ钱包"
	case 4:
		return "银行卡"
	case 5:
		return "支付机构"
	default:
		return "支付宝"
	}
}

// SettleService 结算业务逻辑：明细/批次查询、状态流转、余额扣减退回、自动结算。
type SettleService struct {
	repo         *repository.SettleRepo
	merchantRepo *repository.MerchantRepo
	admins       *repository.AdminRepo // 支付密码二次校验（可空；SetAdminRepo 注入）
	cfg          *ConfigService        // 打款备注 transfer_desc（可空；SetConfigService 注入）
	notice       *NoticeService        // 结算完成通知（可空；SetNoticeService 注入）
	groups       *repository.GroupRepo // 组级 settle_open/settle_rate 覆盖（可空；SetGroupRepo 注入）
}

func NewSettleService(repo *repository.SettleRepo, merchantRepo *repository.MerchantRepo) *SettleService {
	return &SettleService{repo: repo, merchantRepo: merchantRepo}
}

// SetNoticeService 注入对外通知中枢（K-1）。结算记录置为已完成时发 settle 场景通知。
func (s *SettleService) SetNoticeService(n *NoticeService) { s.notice = n }

// notifySettleDone 结算完成后发 settle 通知（对齐 epay ajax_settle.php MsgNotice::send('settle')）。
// 异步触发，通道未配置静默降级，不阻塞结算状态流转。
func (s *SettleService) notifySettleDone(rec *model.SettleRecord) {
	if s.notice == nil || rec == nil || rec.UID == 0 {
		return
	}
	go s.notice.Send("settle", rec.UID, map[string]string{
		"money":     rec.Money.StringFixed(2),
		"realmoney": rec.RealMoney.StringFixed(2),
		"account":   rec.Account,
		"time":      time.Now().Format("2006-01-02 15:04:05"),
	})
}

// SetConfigService 注入配置域（银行结算导出取 transfer_desc 打款备注）。
func (s *SettleService) SetConfigService(c *ConfigService) { s.cfg = c }

// SetGroupRepo 注入用户组仓储，使自动结算按商户所在组的 settle_open/settle_rate 覆盖全局
// （对齐 epay cron.php:32-36 getGroupConfig）。
func (s *SettleService) SetGroupRepo(g *repository.GroupRepo) { s.groups = g }

// resolveSettleConf 计算某商户所在组(gid)生效的自动结算配置，1:1 对齐 epay cron.php:31-36：
//   - settle_rate：默认取全局 settleRate；组级 settle_rate 非空则覆盖
//   - skip：是否跳过该商户不结算。逻辑对齐 epay——
//     若组级 settle_open 已设置且 >0：仅当组级 ==2（组显式关停）跳过；
//     否则回落全局：全局 settle_open 不在 {1,3} 则跳过。
func (s *SettleService) resolveSettleConf(gid int) (rate decimal.Decimal, skip bool) {
	rate = settleRate
	// 全局 settle_open：1=开启且商户可提现 / 3=开启且强制自动结算；其余(0/2)视为全局关闭自动结算。
	globalOpen := 1
	if s.cfg != nil {
		globalOpen = s.cfg.Int("settle_open", 1)
	}
	globalAllow := globalOpen == 1 || globalOpen == 3

	if s.groups == nil {
		return rate, !globalAllow
	}
	g, err := s.groups.FindByID(gid)
	if err != nil || g == nil || strings.TrimSpace(g.Config) == "" {
		return rate, !globalAllow
	}
	var gc map[string]interface{}
	if json.Unmarshal([]byte(g.Config), &gc) != nil {
		return rate, !globalAllow
	}
	// 组级 settle_open：>0 时以组为准（==2 关停），否则回落全局判定（对齐 epay 的 if/elseif）。
	if v, ok := groupConfStr(gc, "settle_open"); ok {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			if n == 2 {
				return rate, true // 组显式关停自动结算
			}
			// 组开启(1/3)：不回落全局，直接放行；下面继续取组级费率
		} else if !globalAllow {
			return rate, true
		}
	} else if !globalAllow {
		return rate, true
	}
	// 组级 settle_rate 覆盖全局费率（非空才覆盖，对齐 epay isset($group['settle_rate'])）。
	if v, ok := groupConfStr(gc, "settle_rate"); ok {
		if d, e := decimal.NewFromString(v); e == nil {
			rate = d
		}
	}
	return rate, false
}

// calcFeeWithRate 用指定费率计算结算手续费（组级费率覆盖时用），其余封顶/兜底逻辑同 calcFee。
func calcFeeWithRate(money, rate decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if rate.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, money
	}
	fee := money.Mul(rate).Div(hundred).Round(2)
	if fee.LessThan(settleFeeMin) {
		fee = settleFeeMin
	}
	if fee.GreaterThan(settleFeeMax) {
		fee = settleFeeMax
	}
	if fee.GreaterThan(money) {
		fee = money
	}
	return fee, money.Sub(fee)
}

// ExportBatch 生成某批次的银行专用打款文件（C-4，对齐 epay download.php?act=settle）。
// tmpl: mybank(网商:支付宝+银行卡) / alipay(支付宝批量转账) / wxpay(微信付款到零钱) / common(通用明细)。
// 返回 (CSV 内容, 文件名)。UTF-8 BOM，Excel 不乱码（不复刻 epay 的 GBK，现代 Excel 兼容 BOM+UTF-8）。
func (s *SettleService) ExportBatch(batch, tmpl string) (string, string, error) {
	if strings.TrimSpace(batch) == "" {
		return "", "", stErr("批次号不能为空")
	}
	remark := "货款结算"
	if s.cfg != nil {
		if d := strings.TrimSpace(s.cfg.Str("transfer_desc")); d != "" {
			remark = d
		}
	}
	var rows []model.SettleRecord
	var err error
	var b strings.Builder
	b.WriteString("\xEF\xBB\xBF") // UTF-8 BOM

	switch tmpl {
	case "mybank": // 网商银行批量：收款方(支付宝+银行卡)
		rows, err = s.repo.ListByBatch(batch, []int8{1, 4})
		if err != nil {
			return "", "", err
		}
		b.WriteString("收款人姓名,收款账号,收款方开户行,收款方联行号,金额,用途/备注\r\n")
		for _, r := range rows {
			bankType := ""
			if r.Type == 1 {
				bankType = "支付宝"
			}
			b.WriteString(csvJoin(r.Username, r.Account, bankType, "", r.RealMoney.StringFixed(2), remark))
		}
	case "alipay": // 支付宝批量转账模板：仅结算方式=支付宝
		rows, err = s.repo.ListByBatch(batch, []int8{1})
		if err != nil {
			return "", "", err
		}
		b.WriteString("支付宝批量付款文件模板\r\n")
		b.WriteString("序号,收款支付宝账号,收款姓名,金额（单位元）,备注（可选）\r\n")
		for i, r := range rows {
			b.WriteString(csvJoin(fmt.Sprintf("%d", i+1), r.Account, r.Username, r.RealMoney.StringFixed(2), remark))
		}
	case "wxpay": // 微信付款到零钱模板：仅结算方式=微信
		rows, err = s.repo.ListByBatch(batch, []int8{2})
		if err != nil {
			return "", "", err
		}
		allMoney := decimal.Zero
		var table strings.Builder
		table.WriteString("商家明细单号,收款用户openid,收款用户姓名（选填）,收款用户身份证（选填）,转账金额（单位元）,转账备注\r\n")
		for i, r := range rows {
			table.WriteString(csvJoin(fmt.Sprintf("%s%d", batch, i+1), r.Account, r.Username, "", r.RealMoney.StringFixed(2), remark))
			allMoney = allMoney.Add(r.RealMoney)
		}
		b.WriteString("微信支付批量转账到零钱模板（请勿删除）\r\n")
		b.WriteString(csvJoin("商家批次单号", batch))
		b.WriteString(csvJoin("批次名称", "批量转账"+batch))
		b.WriteString(csvJoin("转账总金额（单位元）", allMoney.StringFixed(2)))
		b.WriteString(csvJoin("转账总笔数", fmt.Sprintf("%d", len(rows))))
		b.WriteString(csvJoin("本次备注", "批量转账"+batch))
		b.WriteString(",\r\n")
		b.WriteString("转账明细（请勿删除）\r\n")
		b.WriteString(table.String())
	default: // common 通用明细
		tmpl = "common"
		rows, err = s.repo.ListByBatch(batch, nil)
		if err != nil {
			return "", "", err
		}
		b.WriteString("序号,收款方式,收款人账号,收款人姓名,金额（元）,交易描述\r\n")
		for i, r := range rows {
			b.WriteString(csvJoin(fmt.Sprintf("%d", i+1), settleTypeName(r.Type), r.Account, r.Username, r.RealMoney.StringFixed(2), remark))
		}
	}
	return b.String(), fmt.Sprintf("pay_%s_%s.csv", tmpl, batch), nil
}

// parseRemainMoney 解析商户预留余额字符串（epay remain_money 为 varchar）。空/非法返回 0。
func parseRemainMoney(s string) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	v, err := decimal.NewFromString(s)
	if err != nil || v.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return v
}

// csvJoin 把字段用逗号拼成一行 CSV（含 \r\n）。含逗号/引号/换行的字段加引号转义。
func csvJoin(fields ...string) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		if strings.ContainsAny(f, ",\"\r\n") {
			f = "\"" + strings.ReplaceAll(f, "\"", "\"\"") + "\""
		}
		parts[i] = f
	}
	return strings.Join(parts, ",") + "\r\n"
}

// SetAdminRepo 注入管理员仓储（删除退回需支付密码二次校验，对齐 epay admin_paypwd）。
func (s *SettleService) SetAdminRepo(a *repository.AdminRepo) { s.admins = a }

// verifyAdminPwd 校验管理员支付密码（对齐 epay admin_paypwd，独立于登录密码）。
// adminID 保留以兼容调用签名，实际校验走全局支付密码。
func (s *SettleService) verifyAdminPwd(_ uint, pwd string) error {
	if pwd == "" {
		return stErr("请输入支付密码")
	}
	if err := verifyPayPwd(pwd); err != nil {
		return stErr("支付密码不正确")
	}
	return nil
}

// SettleError 携带业务错误码与提示，handler 据此返回 code+msg。
type SettleError struct {
	Code int
	Msg  string
}

func (e *SettleError) Error() string { return e.Msg }

func stErr(msg string) *SettleError { return &SettleError{Code: 1105, Msg: msg} }

// calcFee 计算结算手续费与实际到账。fee = round(money*rate/100,2)，clamp[min,max]；rate=0 则 fee=0。
// 对齐 epay cron/apply 的手续费算法。返回 (fee, realmoney)。
func calcFee(money decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if settleRate.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, money
	}
	fee := money.Mul(settleRate).Div(hundred).Round(2)
	if fee.LessThan(settleFeeMin) {
		fee = settleFeeMin
	}
	if fee.GreaterThan(settleFeeMax) {
		fee = settleFeeMax
	}
	// 手续费不应超过结算金额本身（极端小额保护）
	if fee.GreaterThan(money) {
		fee = money
	}
	return fee, money.Sub(fee)
}

// List 返回分页结算明细（转对外 View）。
func (s *SettleService) List(q dto.SettleQuery) ([]dto.SettleView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.SettleView, 0, len(list))
	for i := range list {
		views = append(views, toSettleView(&list[i]))
	}
	return views, total, nil
}

// Stats 结算明细概况（全量聚合，按与列表相同筛选，供概况卡使用）。
func (s *SettleService) Stats(q dto.SettleQuery) (dto.SettleStats, error) {
	q.Normalize()
	return s.repo.Stats(q)
}

// ExportRows 按与列表相同筛选返回全部匹配明细（转 View），供服务端 CSV 导出。
func (s *SettleService) ExportRows(q dto.SettleQuery) ([]dto.SettleView, error) {
	list, err := s.repo.ExportRows(q)
	if err != nil {
		return nil, err
	}
	views := make([]dto.SettleView, 0, len(list))
	for i := range list {
		views = append(views, toSettleView(&list[i]))
	}
	return views, nil
}

// ListBatches 返回分页结算批次（转对外 View）。
func (s *SettleService) ListBatches(page, pageSize int) ([]dto.SettleBatchView, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := s.repo.ListBatches(page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.SettleBatchView, 0, len(list))
	for i := range list {
		views = append(views, dto.SettleBatchView{
			Batch:    list[i].Batch,
			AllMoney: list[i].AllMoney.StringFixed(2),
			Count:    list[i].Count,
			Time:     list[i].Time.Format(timeLayout),
			Status:   list[i].Status,
		})
	}
	return views, total, nil
}

// CreateBatch 生成结算批次：把当前所有待结算记录收批并置为"正在结算"。
// 批次号格式 B + YmdHis（自研，避免 epay Ymd+rand 的碰撞风险）。返回批次号与收批条数。
func (s *SettleService) CreateBatch(now time.Time) (string, int, error) {
	batchNo := "B" + now.Format("20060102150405")
	count, _, err := s.repo.CreateBatchFromPending(batchNo, now)
	if err != nil {
		if errors.Is(err, repository.ErrNoPending) {
			return "", 0, stErr("当前不存在待结算的记录")
		}
		return "", 0, err
	}
	return batchNo, count, nil
}

// CompleteBatch 把批次内正在结算的记录一次性置为已完成（手动打款后确认）。
func (s *SettleService) CompleteBatch(batch string) (int64, error) {
	b, err := s.repo.FindBatch(batch)
	if err != nil {
		return 0, err
	}
	if b == nil {
		return 0, stErr("批次不存在")
	}
	// 完成前取批内正在结算的记录，用于结算完成通知（对齐 epay ajax_settle.php 批量收批后逐条 MsgNotice）。
	var pending []model.SettleRecord
	if s.notice != nil {
		if recs, e := s.repo.ListByBatch(batch, []int8{2}); e == nil {
			pending = recs
		}
	}
	n, err := s.repo.CompleteBatch(batch, time.Now())
	if err != nil {
		return n, err
	}
	for i := range pending {
		s.notifySettleDone(&pending[i])
	}
	return n, nil
}

// SetStatus 变更单条结算记录状态。
//   - status=4：删除记录并把结算金额退回商户余额（对齐 epay setStatusDo）。
//   - status=1：置已完成 + 写完成时间 + 清失败原因。
//   - status=0/2：改状态并清完成时间。
//   - status=3：置结算失败 + 写失败原因（result）。
func (s *SettleService) SetStatus(id uint, req dto.SettleStatusReq) error {
	exist, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if exist == nil {
		return stErr("结算记录不存在")
	}

	switch req.Status {
	case 4: // 删除并退回余额（资金操作，需管理员支付密码二次校验）
		if err := s.verifyAdminPwd(req.AdminID, req.Password); err != nil {
			return err
		}
		_, err := s.repo.DeleteWithRefund(id, "结算失败退回")
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return stErr("结算记录不存在")
			}
			return err
		}
		return nil
	case 1: // 已完成
		if err := s.repo.SetStatus(id, map[string]interface{}{
			"status": 1, "end_time": time.Now(), "result": "",
		}); err != nil {
			return err
		}
		s.notifySettleDone(exist) // 结算完成通知（K-1 settle 场景）
		return nil
	case 0, 2: // 待结算 / 正在结算
		return s.repo.SetStatus(id, map[string]interface{}{
			"status": req.Status, "end_time": nil,
		})
	case 3: // 结算失败
		return s.repo.SetStatus(id, map[string]interface{}{
			"status": 3, "end_time": nil, "result": req.Result,
		})
	default:
		return stErr("状态值不合法")
	}
}

// RunAutoSettle 自动结算：每日一次，选出满足条件的商户按余额生成结算单并即时扣款。
// 对齐 epay cron do=settle 的核心逻辑（每日幂等锁 + 门槛 + 手续费 + 生单扣款）。
// limit 为单次处理商户数上限。返回本次生成的结算单数。
func (s *SettleService) RunAutoSettle(ctx context.Context, limit int) (int, error) {
	// 每日幂等锁：当天已生成过自动结算单则跳过（对齐 epay settle_time 每日只跑一次）。
	todayStart := time.Now().Truncate(24 * time.Hour)
	n, err := s.repo.CountAutoSince(todayStart)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, nil // 今日已结算
	}

	merchants, err := s.merchantRepo.FindSettleable(settleMoney, limit)
	if err != nil {
		return 0, err
	}
	// cert_force：开启强制实名时，未实名商户不参与自动结算（B-8，对齐 epay cron settle 分支）。
	certForce := s.cfg != nil && s.cfg.Bool("cert_force")
	created := 0
	now := time.Now()
	for i := range merchants {
		m := merchants[i]
		if certForce && m.Cert == 0 {
			continue // 强制实名下未实名商户跳过结算
		}
		// 组级 settle_open 关停 / 全局未开启自动结算 → 跳过；组级 settle_rate 覆盖费率
		// （对齐 epay cron.php:31-36 getGroupConfig）。
		rate, skip := s.resolveSettleConf(m.GID)
		if skip {
			continue
		}
		// remain_money 预留余额：自动结算不结算预留部分（B-8，对齐 epay remain_money）。
		money := m.Money
		if rm := parseRemainMoney(m.RemainMoney); rm.GreaterThan(decimal.Zero) {
			money = money.Sub(rm)
		}
		if money.LessThan(settleMoney) {
			continue
		}
		_, realmoney := calcFeeWithRate(money, rate)
		rec := &model.SettleRecord{
			UID:       m.UID,
			Auto:      1,
			Type:      int8(m.SettleID),
			Account:   m.Account,
			Username:  m.Username,
			Money:     money,
			RealMoney: realmoney,
			AddTime:   now,
			Status:    0,
		}
		if err := s.repo.CreateWithDebit(rec, "自动结算"); err != nil {
			if errors.Is(err, repository.ErrInsufficientBalance) {
				continue // 余额被其它操作扣走，跳过
			}
			log.Printf("[settle] 商户 %d 自动结算生单失败: %v", m.UID, err)
			continue
		}
		created++
	}
	return created, nil
}

func toSettleView(s *model.SettleRecord) dto.SettleView {
	var endTime *string
	if s.EndTime != nil {
		t := s.EndTime.Format(timeLayout)
		endTime = &t
	}
	return dto.SettleView{
		ID:        s.ID,
		Batch:     s.Batch,
		UID:       s.UID,
		Merchant:  fmt.Sprintf("商户%d", s.UID), // 商户名派生（接商户名字段后补）
		Type:      s.Type,
		Auto:      s.Auto,
		Account:   s.Account,
		Username:  s.Username,
		Money:     s.Money.StringFixed(2),
		RealMoney: s.RealMoney.StringFixed(2),
		AddTime:   s.AddTime.Format(timeLayout),
		EndTime:   endTime,
		Status:    s.Status,
		Result:    s.Result,
	}
}
