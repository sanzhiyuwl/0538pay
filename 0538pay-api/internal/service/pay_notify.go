package service

import (
	"context"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/shopspring/decimal"
)

// NotifyResult 回调处理结果，供 handler 决定回给渠道的应答。
type NotifyResult struct {
	AckContent string // 需回给渠道的字符串（如 "success"）
	Handled    bool   // 是否本次真正完成了订单翻转（重复回调=false 但也算成功受理）
}

// Notify 处理第三方渠道的支付回调（对齐 epay pay/notify → Payment::processOrder）。
// 流程：定位订单 → 取渠道解析回调(验签在渠道内) → 金额二次校验 → 幂等改单(条件UPDATE)
//        → 入账(A6) → 触发商户异步通知(A5)。
// tradeNo 来自路由；raw 为渠道回调的原始参数。
func (s *PayService) Notify(ctx context.Context, tradeNo string, raw map[string]string) (*NotifyResult, error) {
	// 1. 定位订单
	order, err := s.orders.FindByTradeNo(tradeNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, payErr("订单不存在")
	}

	// 2. 渠道解析回调（真实渠道在此验签；mock 直接读参数）
	ch, ok := channel.Get(order.Plugin)
	if !ok {
		return nil, payErr("支付渠道不可用：" + order.Plugin)
	}
	cfg := s.loadChannelConfig(order.Channel)
	nr, err := ch.Notify(ctx, cfg, raw)
	if err != nil {
		return nil, payErr("回调解析失败：" + err.Error())
	}
	if !nr.Success {
		// 渠道明确非成功，不改单，回默认应答
		return &NotifyResult{AckContent: nr.AckContent, Handled: false}, nil
	}

	// 3. 金额二次校验防篡改：回调金额需 == 订单金额（decimal 精确比较）
	if !nr.Money.Equal(decimal.Zero) && !nr.Money.Equal(order.Money) {
		return nil, payErr("回调金额与订单金额不一致")
	}

	// 4. 幂等改单：条件 UPDATE，仅未终态→已付翻转一次
	flipped, err := s.orders.MarkPaid(tradeNo, nr.ChannelNo, raw["buyer"], time.Now())
	if err != nil {
		return nil, err
	}
	if !flipped {
		// 重复回调 / 并发：订单已是终态，直接受理成功（幂等），不重复入账
		return &NotifyResult{AckContent: nr.AckContent, Handled: false}, nil
	}

	// 5. 入账 + 6. 商户通知。入账失败视为严重错误（订单已翻转），返回错误让渠道重试。
	if err := s.settle(ctx, tradeNo); err != nil {
		return nil, err
	}
	return &NotifyResult{AckContent: nr.AckContent, Handled: true}, nil
}

// settle 订单入账 + 触发商户异步通知。订单此时已置为已付。
// 阶段 A：mock 渠道无费率，getmoney=0，按订单金额全额入账（对齐 epay 非直清模式 changeUserMoney(addmoney)）。
func (s *PayService) settle(ctx context.Context, tradeNo string) error {
	order, err := s.orders.FindByTradeNo(tradeNo)
	if err != nil {
		return err
	}
	if order == nil {
		return payErr("入账时订单丢失")
	}

	// 内部业务订单（tid≠0）按类型分派入账（对齐 epay processOrder 的 tid 分支）。
	if order.Tid == 2 {
		// 充值余额：全额入商户余额（对齐 epay tid=2，加费模式下 getmoney=充值全额）。
		if err := s.accounts.ChangeUserMoney(order.UID, order.Money, true, "余额充值", order.TradeNo); err != nil {
			return err
		}
		return nil // 内部充值无需商户异步通知
	}

	// 入账金额：epay 非直清模式入 getmoney（商户实得）。阶段A无费率，getmoney=0 时退回按订单金额入账。
	addMoney := order.GetMoney
	if addMoney.LessThanOrEqual(decimal.Zero) {
		addMoney = order.Money
	}
	if err := s.accounts.ChangeUserMoney(order.UID, addMoney, true, "订单收入", order.TradeNo); err != nil {
		return err
	}

	// 分账：命中规则的订单支付成功后按比例创建分账订单（待分账），对齐 epay。
	// 失败不回滚入账（分账可后台补建/重试），仅记日志级别忽略。
	if s.profit != nil && order.Profits > 0 {
		realMoney := order.Money
		if order.RealMoney != nil && order.RealMoney.GreaterThan(decimal.Zero) {
			realMoney = *order.RealMoney
		}
		_ = s.profit.CreateOrderOnPaid(order.Profits, order.TradeNo, order.APITradeNo, realMoney)
	}

	// 触发商户异步通知（A5）。失败不回滚入账，仅置重试标志，交由 cron 重试(阶段E)。
	s.fireMerchantNotify(ctx, order.TradeNo)
	return nil
}

// ResendNotify 商户主动触发重新通知（补单/重发回调）。复用 fireMerchantNotify 的通知逻辑。
func (s *PayService) ResendNotify(tradeNo string) {
	s.fireMerchantNotify(context.Background(), tradeNo)
}

// fireMerchantNotify 拼带签名的回调 URL，GET 商户 notify_url，据结果置通知状态/重试计数。
// 首次通知：成功 notify=0；失败 notify=1 + 下次重试时间（首档 1 分钟），后续由 cron RetryNotify 接管。
func (s *PayService) fireMerchantNotify(ctx context.Context, tradeNo string) {
	order, err := s.orders.FindByTradeNo(tradeNo)
	if err != nil || order == nil || order.NotifyURL == "" {
		return
	}
	m, err := s.merchants.FindByUID(order.UID)
	if err != nil {
		return
	}
	params := buildCallbackParams(order, m)
	if doNotify(ctx, appendQuery(order.NotifyURL, params)) {
		_ = s.orders.SetNotifySuccess(tradeNo)
		return
	}
	// 首次失败 → notify=1，下次重试 1 分钟后（对齐 epay 首档）。
	_ = s.orders.SetNotifyRetry(tradeNo, 1, time.Now().Add(notifyBackoff(1)))
}
