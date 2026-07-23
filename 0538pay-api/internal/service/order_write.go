package service

import (
	"context"
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/shopspring/decimal"
)

// OrderError 携带业务错误码与提示，handler 据此返回 code+msg。
type OrderError struct {
	Code int
	Msg  string
}

func (e *OrderError) Error() string { return e.Msg }

func odErr(msg string) *OrderError { return &OrderError{Code: 1002, Msg: msg} }

// 退款手续费处理策略（对齐 epay 全局 refund_fee_type）：
// false=平台承担手续费(退回商户,扣 getmoney，refund_fee_type=0)；
// true=商户承担(全额退时扣 realmoney，refund_fee_type=1)。config 域加载后刷新。
var refundFeeType = false

// reloadOrderConfig 从 config 域刷新订单退款相关常量。
func reloadOrderConfig(cfg *ConfigService) {
	refundFeeType = cfg.Str("refund_fee_type") == "1"
}

// SetStatus 裸改订单状态（改未完成 0 / 改已完成 1，对齐 epay setStatus）。
// 仅改字段，不触发入账/资金（与"手动补单"区分）。
func (s *OrderService) SetStatus(tradeNo string, status int8) error {
	if status != 0 && status != 1 {
		return odErr("状态值不合法")
	}
	o, err := s.repo.FindByTradeNo(tradeNo)
	if err != nil {
		return err
	}
	if o == nil {
		return odErr("订单不存在")
	}
	return s.repo.SetStatus(tradeNo, status)
}

// Delete 物理删除订单（对齐 epay setStatus=5，无级联，不退款不改余额）。
func (s *OrderService) Delete(tradeNo string) error {
	o, err := s.repo.FindByTradeNo(tradeNo)
	if err != nil {
		return err
	}
	if o == nil {
		return odErr("订单不存在")
	}
	return s.repo.Delete(tradeNo)
}

// Freeze 冻结订单（对齐 epay Order::freeze）：status 1→3，从商户余额扣 getmoney。
// 前置：status==1 且 通道非商户直清(mode!=1)。
func (s *OrderService) Freeze(tradeNo string) error {
	o, err := s.repo.FindByTradeNo(tradeNo)
	if err != nil {
		return err
	}
	if o == nil {
		return odErr("订单不存在")
	}
	if o.Status != 1 {
		return odErr("仅已支付订单可冻结")
	}
	if s.channelIsDirect(o.Channel) {
		return odErr("商户直清通道不支持冻结")
	}
	ok, err := s.repo.SetStatusFrom(tradeNo, 1, 3)
	if err != nil {
		return err
	}
	if !ok {
		return odErr("订单状态已变更，冻结未执行")
	}
	if o.GetMoney.GreaterThan(decimal.Zero) {
		if err := s.accounts.ChangeUserMoney(o.UID, o.GetMoney, false, "订单冻结", tradeNo); err != nil {
			return err
		}
	}
	return nil
}

// Unfreeze 解冻订单（对齐 epay Order::unfreeze）：status 3→1，把 getmoney 加回商户余额。
func (s *OrderService) Unfreeze(tradeNo string) error {
	o, err := s.repo.FindByTradeNo(tradeNo)
	if err != nil {
		return err
	}
	if o == nil {
		return odErr("订单不存在")
	}
	if o.Status != 3 {
		return odErr("仅已冻结订单可解冻")
	}
	if s.channelIsDirect(o.Channel) {
		return odErr("商户直清通道不支持解冻")
	}
	ok, err := s.repo.SetStatusFrom(tradeNo, 3, 1)
	if err != nil {
		return err
	}
	if !ok {
		return odErr("订单状态已变更，解冻未执行")
	}
	if o.GetMoney.GreaterThan(decimal.Zero) {
		if err := s.accounts.ChangeUserMoney(o.UID, o.GetMoney, true, "订单解冻", tradeNo); err != nil {
			return err
		}
	}
	return nil
}

// RefundInfo 退款前查询可退金额（对齐 epay refund_info）。
func (s *OrderService) RefundInfo(tradeNo string) (*dto.OrderRefundInfo, error) {
	o, err := s.repo.FindByTradeNo(tradeNo)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, odErr("订单不存在")
	}
	if o.Status != 1 && o.Status != 2 && o.Status != 3 {
		return nil, odErr("当前订单状态不可退款")
	}
	real := decimal.Zero
	if o.RealMoney != nil {
		real = *o.RealMoney
	}
	if real.LessThanOrEqual(decimal.Zero) {
		real = o.Money
	}
	if o.Status == 2 && o.RefundMoney.LessThanOrEqual(decimal.Zero) {
		return nil, odErr("该订单已退款")
	}
	if o.RefundMoney.GreaterThanOrEqual(real) && o.RefundMoney.GreaterThan(decimal.Zero) {
		return nil, odErr("该订单已全额退款")
	}
	refundable := real
	if o.RefundMoney.GreaterThan(decimal.Zero) {
		refundable = real.Sub(o.RefundMoney).Round(2)
	}
	return &dto.OrderRefundInfo{
		TradeNo:    tradeNo,
		RealMoney:  real.InexactFloat64(),
		Refunded:   o.RefundMoney.InexactFloat64(),
		Refundable: refundable.InexactFloat64(),
		CanAPI:     o.APITradeNo != "" && !s.channelIsDirect(o.Channel),
	}, nil
}

// Refund 订单退款（对齐 epay Order::refund）。
// 校验：status∈[1,2,3]、money≤realmoney、未超剩余可退。
// 扣商户余额 reducemoney 四分支：冻结单/直清=0，(手续费不退且全额)=realmoney，(手续费退且全额或≥分成)=getmoney，部分=退款额。
// API 退款需管理员密码，且真实渠道退款待凭证——余额层退款照走，渠道侧原路退款如实返回待凭证提示。
func (s *OrderService) Refund(adminID uint, req dto.OrderRefundReq) error {
	o, err := s.repo.FindByTradeNo(req.TradeNo)
	if err != nil {
		return err
	}
	if o == nil {
		return odErr("订单不存在")
	}
	if o.Status != 1 && o.Status != 2 && o.Status != 3 {
		return odErr("当前订单状态不可退款")
	}
	money, err := decimal.NewFromString(strings.TrimSpace(req.Money))
	if err != nil || money.LessThanOrEqual(decimal.Zero) {
		return odErr("请输入有效的退款金额")
	}
	real := o.Money
	if o.RealMoney != nil && o.RealMoney.GreaterThan(decimal.Zero) {
		real = *o.RealMoney
	}
	if money.GreaterThan(real) {
		return odErr("退款金额不能超过订单实付金额")
	}
	if o.RefundMoney.GreaterThan(decimal.Zero) {
		if o.RefundMoney.GreaterThanOrEqual(real) {
			return odErr("该订单已全额退款")
		}
		if money.GreaterThan(real.Sub(o.RefundMoney).Round(2)) {
			return odErr("退款金额超过剩余可退")
		}
	}

	// API 退款：校验管理员密码；真实渠道原路退款。
	if req.API {
		if err := s.verifyAdminPwd(adminID, req.Password); err != nil {
			return err
		}
		if o.APITradeNo == "" {
			return odErr("该订单无接口订单号，无法原路退款")
		}
		if s.channelIsDirect(o.Channel) {
			return odErr("商户直清通道不支持 API 退款")
		}
		// 真实渠道原路退款：渠道实现 Refunder 则调真实退款接口（需真实凭证，无凭证会报错）。
		// mock 等无 Refunder 的渠道跳过，仅走余额层。
		if s.pay != nil {
			outRefundNo := "RF" + req.TradeNo
			handled, rerr := s.pay.RefundViaChannel(context.Background(), o, money, outRefundNo)
			if rerr != nil {
				return odErr("渠道原路退款失败(可能待真实凭证): " + rerr.Error())
			}
			_ = handled // 渠道已受理；余额层退款继续（对齐 epay 渠道退款成功后扣商户余额）
		}
	}

	// reducemoney 四分支（对齐 epay Order::refund）
	reduce := s.calcReduceMoney(o, money, real)

	// 改单：status→2 + refundmoney 累加（对齐 epay API 退款；手动退款 epay 不累加，
	// 但我们统一累加以支持部分退款可视化，语义更完整）。
	ok, err := s.repo.MarkRefundedPartial(req.TradeNo, money)
	if err != nil {
		return err
	}
	if !ok {
		return odErr("订单状态已变更，退款未执行")
	}
	// 扣商户余额。双重防重：①前置 guard(已全额退款/超过剩余可退)+MarkRefundedPartial 的条件 UPDATE；
	// ②ChangeUserMoney 内对 type=订单退款+trade_no 记录级幂等(B1-23,对齐 epay changeUserMoney:678-681)。
	if reduce.GreaterThan(decimal.Zero) {
		if err := s.accounts.ChangeUserMoney(o.UID, reduce, false, "订单退款", req.TradeNo); err != nil {
			return err
		}
	}
	return nil
}

// calcReduceMoney 计算退款时从商户余额扣多少（epay Order::refund 四分支）。
func (s *OrderService) calcReduceMoney(o *model.Order, money, real decimal.Decimal) decimal.Decimal {
	// 已冻结(余额已在冻结时扣过) 或 商户直清 → 不再扣
	if o.Status == 3 || s.channelIsDirect(o.Channel) {
		return decimal.Zero
	}
	if refundFeeType && money.Equal(real) {
		return real // 手续费不退：全额退时扣实付全额
	}
	if !refundFeeType && (money.Equal(real) || money.GreaterThanOrEqual(o.GetMoney)) {
		return o.GetMoney // 手续费退回商户：扣商户分成
	}
	return money // 部分退：按退款额扣
}

// FillOrder 手动补单（对齐 epay fillorder）：仅 status==0 → 置已付 + 入账 + 分账 + 通知。
func (s *OrderService) FillOrder(tradeNo string) error {
	o, err := s.repo.FindByTradeNo(tradeNo)
	if err != nil {
		return err
	}
	if o == nil {
		return odErr("订单不存在")
	}
	if o.Status != 0 {
		return odErr("当前订单不是未完成状态")
	}
	ok, err := s.repo.FillOrder(tradeNo, time.Now())
	if err != nil {
		return err
	}
	if !ok {
		return odErr("订单状态已变更，补单未执行")
	}
	// 复用支付成功链路：入账 + 分账 + 商户通知
	if s.pay != nil {
		if err := s.pay.SettleOnFill(tradeNo); err != nil {
			return err
		}
	}
	return nil
}

// Renotify 重新通知商户（对齐 epay notify）：仅已支付订单，复用支付服务的通知重发。
func (s *OrderService) Renotify(tradeNo string) error {
	o, err := s.repo.FindByTradeNo(tradeNo)
	if err != nil {
		return err
	}
	if o == nil {
		return odErr("订单不存在")
	}
	if o.Status != 1 {
		return odErr("仅已支付订单可重新通知")
	}
	if o.NotifyURL == "" {
		return odErr("该订单未设置异步通知地址")
	}
	if s.pay != nil {
		s.pay.ResendNotify(tradeNo)
	}
	return nil
}

// Batch 批量操作（对齐 epay operation）：0改未完成 1改已完成 2冻结 3解冻 4删除。
// 逐条处理，返回成功条数；单条失败跳过不中断（与 epay 循环处理一致）。
func (s *OrderService) Batch(adminID uint, req dto.OrderBatchReq) (int, error) {
	n := 0
	for _, tradeNo := range req.TradeNos {
		var err error
		switch req.Action {
		case 0:
			err = s.SetStatus(tradeNo, 0)
		case 1:
			err = s.SetStatus(tradeNo, 1)
		case 2:
			err = s.Freeze(tradeNo)
		case 3:
			err = s.Unfreeze(tradeNo)
		case 4:
			err = s.Delete(tradeNo)
		default:
			return n, odErr("未知的批量操作")
		}
		if err == nil {
			n++
		}
	}
	return n, nil
}

// channelIsDirect 判断通道是否商户直清模式(mode==1)。查不到通道按非直清处理。
func (s *OrderService) channelIsDirect(channelID int) bool {
	if s.channels == nil || channelID == 0 {
		return false
	}
	c, err := s.channels.FindByID(uint(channelID))
	if err != nil || c == nil {
		return false
	}
	return c.Mode == 1
}

// verifyAdminPwd 校验管理员支付密码（对齐 epay admin_paypwd，独立于登录密码）。
// adminID 保留以兼容调用签名，实际校验走全局支付密码。
func (s *OrderService) verifyAdminPwd(_ uint, pwd string) error {
	if pwd == "" {
		return odErr("请输入支付密码")
	}
	if err := verifyPayPwd(pwd); err != nil {
		return odErr("支付密码不正确")
	}
	return nil
}
