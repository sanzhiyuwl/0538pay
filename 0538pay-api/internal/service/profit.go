package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// ProfitService 分账业务：下单匹配规则、支付成功建分账单、列表/统计、状态流转（提交/查询/回退/取消/改额）。
// 真实渠道分账 API 待凭证，提交/查询走本地状态流转（对齐 epay 状态机，资金语义完整）。
type ProfitService struct {
	repo     *repository.ProfitRepo
	channels *repository.ChannelRepo
}

func NewProfitService(repo *repository.ProfitRepo, channels *repository.ChannelRepo) *ProfitService {
	return &ProfitService{repo: repo, channels: channels}
}

// ProfitError 携带业务错误码与提示。
type ProfitError struct {
	Code int
	Msg  string
}

func (e *ProfitError) Error() string { return e.Msg }

func psErr(msg string) *ProfitError { return &ProfitError{Code: 1107, Msg: msg} }

// MatchRuleForOrder 下单时匹配分账规则，命中返回规则 id（写入 order.profits），未命中返回 0。
// realMoney 为订单实收金额（用于 minmoney 门槛判断）。对齐 epay updateOrderProfits。
func (s *ProfitService) MatchRuleForOrder(channelID, subChannel int, uid uint, realMoney decimal.Decimal) uint {
	if channelID <= 0 {
		return 0
	}
	rule, err := s.repo.MatchRule(channelID, subChannel, uid)
	if err != nil || rule == nil {
		return 0
	}
	// minmoney 门槛：realmoney >= minmoney 才分账
	if rule.MinMoney.GreaterThan(decimal.Zero) && realMoney.LessThan(rule.MinMoney) {
		return 0
	}
	return rule.ID
}

// CreateOrderOnPaid 支付成功回调触发：按规则比例创建分账订单（对齐 epay functions.php 664-672）。
// psmoney = round(floor(realmoney*rate)/100, 2)。规则绑定商户+通道 mode=0 时记 PsUID（成功时扣其余额）。
// 幂等：同 trade_no 已有分账单则跳过。ruleID=0 或规则不存在则不建。
func (s *ProfitService) CreateOrderOnPaid(ruleID uint, tradeNo, apiTradeNo string, realMoney decimal.Decimal) error {
	if ruleID == 0 {
		return nil
	}
	exist, err := s.repo.ExistOrderByTradeNo(tradeNo)
	if err != nil || exist {
		return err
	}
	rule, err := s.repo.FindRule(ruleID)
	if err != nil || rule == nil {
		return err
	}
	// 比例算分账金额：floor(realmoney*rate)/100 再 round 2（对齐 epay）
	psmoney := realMoney.Mul(rule.Rate).Div(hundred).Round(2)
	if psmoney.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	// 是否从商户余额扣款：规则绑定商户 + 通道 mode=0（进件/代付模式）
	var psUID *uint
	if rule.UID != nil && *rule.UID > 0 {
		if ch, _ := s.channels.FindByID(uint(rule.Channel)); ch != nil && ch.Mode == 0 {
			psUID = rule.UID
		}
	}
	o := &model.ProfitOrder{
		RID:        rule.ID,
		TradeNo:    tradeNo,
		APITradeNo: apiTradeNo,
		PsUID:      psUID,
		Money:      psmoney,
		Status:     0, // 待分账
		AddTime:    time.Now(),
	}
	return s.repo.CreateOrder(o)
}

// List 分账订单列表（分页 + 筛选，装配规则/通道派生字段）。
func (s *ProfitService) List(q dto.PsOrderQuery) ([]dto.PsOrderView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.PsOrderView, 0, len(list))
	for i := range list {
		views = append(views, s.toPsOrderView(&list[i]))
	}
	return views, total, nil
}

// Stats 分账统计概况。
func (s *ProfitService) Stats(q dto.PsOrderQuery) (dto.PsStats, error) {
	q.Normalize()
	tm, sm, fm, tc, sc, fc, err := s.repo.Stats(q)
	if err != nil {
		return dto.PsStats{}, err
	}
	rate := 0.0
	if tc > 0 {
		rate = float64(int(float64(sc)/float64(tc)*10000+0.5)) / 100 // 两位小数成功率
	}
	return dto.PsStats{
		TotalMoney:   tm.InexactFloat64(),
		SuccessMoney: sm.InexactFloat64(),
		FailMoney:    fm.InexactFloat64(),
		TotalCount:   tc,
		SuccessCount: sc,
		FailCount:    fc,
		SuccessRate:  rate,
	}, nil
}

// Operate 单条分账订单状态操作（对齐 epay ps_order 操作列）：
//   - submit：提交分账（status 0/3 → 成功2 并按需扣款）。真实渠道 API 待凭证，此处直接判成功。
//   - query：查询结果（status 1 → 成功2）。
//   - return：分账回退（status 2 → 取消4，已扣则解冻退回）。
//   - cancel：取消/解冻（status 0/3 → 取消4，已扣则解冻退回）。
//   - editmoney：改分账金额（仅 status 0）。
//   - delete：删除记录（不涉及资金；已扣款的不允许直接删，需先回退）。
func (s *ProfitService) Operate(id uint, req dto.PsStatusReq) error {
	o, err := s.repo.FindOrder(id)
	if err != nil {
		return err
	}
	if o == nil {
		return psErr("分账记录不存在")
	}

	switch req.Action {
	case "submit":
		if o.Status != 0 && o.Status != 3 {
			return psErr("仅待分账/失败的记录可提交分账")
		}
		// 真实渠道分账 API 待凭证：本地直接置成功（成功时按规则扣商户余额）。
		settleNo := "PS" + time.Now().Format("20060102150405")
		flipped, err := s.repo.MarkSuccessWithDebit(id, settleNo)
		if err != nil {
			if err == repository.ErrInsufficientBalance {
				return psErr("商户余额不足，无法扣除分账金额")
			}
			return err
		}
		if !flipped {
			return psErr("分账状态已变更，提交未执行")
		}
		return nil
	case "query":
		if o.Status != 1 {
			return psErr("仅已提交的记录可查询结果")
		}
		settleNo := o.SettleNo
		if settleNo == "" {
			settleNo = "PS" + time.Now().Format("20060102150405")
		}
		_, err := s.repo.MarkSuccessWithDebit(id, settleNo)
		return err
	case "return":
		if o.Status != 2 {
			return psErr("仅分账成功的记录可回退")
		}
		flipped, err := s.repo.CancelOrRefund(id, []int{2}, "分账回退")
		if err != nil {
			return err
		}
		if !flipped {
			return psErr("分账状态已变更，回退未执行")
		}
		return nil
	case "cancel":
		if o.Status != 0 && o.Status != 3 {
			return psErr("仅待分账/失败的记录可取消")
		}
		flipped, err := s.repo.CancelOrRefund(id, []int{0, 3}, "分账取消退回")
		if err != nil {
			return err
		}
		if !flipped {
			return psErr("分账状态已变更，取消未执行")
		}
		return nil
	case "editmoney":
		if o.Status != 0 {
			return psErr("仅待分账的记录可改金额")
		}
		amount, err := decimal.NewFromString(strings.TrimSpace(req.Money))
		if err != nil || amount.LessThanOrEqual(decimal.Zero) {
			return psErr("请输入有效的分账金额")
		}
		if !amount.Equal(amount.Round(2)) {
			return psErr("金额最多两位小数")
		}
		return s.repo.UpdateOrder(id, map[string]interface{}{"money": amount})
	case "delete":
		if o.Debited == 1 {
			return psErr("该分账已扣款，请先回退再删除")
		}
		return s.repo.DeleteOrder(id)
	default:
		return psErr("不支持的操作")
	}
}

// AutoExecute 自动执行待分账(status=0)订单：逐条提交分账（扣款置成功）。
// 对齐 epay cron do=profitsharing。余额不足/状态已变的单跳过（下轮再试），返回成功执行数。
// 由 scheduler 定时调用；真实渠道分账 API 待凭证，本地按 Operate submit 同一逻辑判成功。
func (s *ProfitService) AutoExecute(limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	ids, err := s.repo.ListPendingIDs(limit)
	if err != nil {
		return 0, err
	}
	done := 0
	for _, id := range ids {
		settleNo := "PS" + time.Now().Format("20060102150405")
		flipped, err := s.repo.MarkSuccessWithDebit(id, settleNo)
		if err != nil {
			// 余额不足等：跳过该单，下轮再试，不中断整批。
			continue
		}
		if flipped {
			done++
		}
	}
	return done, nil
}

func (s *ProfitService) toPsOrderView(o *model.ProfitOrder) dto.PsOrderView {
	v := dto.PsOrderView{
		ID:         o.ID,
		TradeNo:    o.TradeNo,
		APITradeNo: o.APITradeNo,
		RID:        o.RID,
		Money:      o.Money.StringFixed(2),
		AddTime:    o.AddTime.Format(timeLayout),
		Status:     o.Status,
		Result:     o.Result,
	}
	// 派生规则名/接收方/通道名
	if rule, _ := s.repo.FindRule(o.RID); rule != nil {
		v.RuleName = fmt.Sprintf("分账 %s%%", rule.Rate.StringFixed(0))
		v.Receiver = rule.Account
		v.ChannelID = rule.Channel
		if ch, _ := s.channels.FindByID(uint(rule.Channel)); ch != nil {
			v.ChannelName = ch.Name
		}
	}
	return v
}
