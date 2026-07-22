package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
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
}

func NewSettleService(repo *repository.SettleRepo, merchantRepo *repository.MerchantRepo) *SettleService {
	return &SettleService{repo: repo, merchantRepo: merchantRepo}
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
	return s.repo.CompleteBatch(batch, time.Now())
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
	case 4: // 删除并退回余额
		_, err := s.repo.DeleteWithRefund(id, "结算失败退回")
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return stErr("结算记录不存在")
			}
			return err
		}
		return nil
	case 1: // 已完成
		return s.repo.SetStatus(id, map[string]interface{}{
			"status": 1, "end_time": time.Now(), "result": "",
		})
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
	created := 0
	now := time.Now()
	for i := range merchants {
		m := merchants[i]
		money := m.Money // 结算全部可用余额（预留余额/D+1 待接商户端申请域细化）
		if money.LessThan(settleMoney) {
			continue
		}
		_, realmoney := calcFee(money)
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
