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

	// 2.5 买家(openid/buyer)风控（对齐 epay functions.php checkBlockUser）：
	//     渠道回调拿到 buyer 后，命中 type=0 黑名单 → 拒付；pay_userlimit>0 时买家当日已付单数达上限 → 拒付。
	//     仅对首次翻转生效（重复回调走下面的 BackfillCallbackFields 分支，不再校验）。
	if buyer := raw["buyer"]; buyer != "" {
		if err := s.checkBlockBuyer(buyer); err != nil {
			return nil, err
		}
	}

	// 3. 金额二次校验防篡改：回调金额需 == 实际待付额 realmoney（对齐 epay round(total)==round(realmoney)）。
	//    realmoney 含随机微调/加费；为空时退回比 money（mock/内部单）。
	expectMoney := order.Money
	if order.RealMoney != nil && order.RealMoney.GreaterThan(decimal.Zero) {
		expectMoney = *order.RealMoney
	}
	if !nr.Money.Equal(decimal.Zero) && !nr.Money.Equal(expectMoney) {
		return nil, payErr("回调金额与订单金额不一致")
	}

	// 4. 幂等改单：条件 UPDATE，仅未终态→已付翻转一次
	flipped, err := s.orders.MarkPaid(tradeNo, nr.ChannelNo, raw["buyer"], time.Now())
	if err != nil {
		return nil, err
	}
	if !flipped {
		// 重复回调 / 并发：订单已终态，不重复入账。但补写缺失的 api_trade_no/buyer/bill_trade_no
		// （A-10，对齐 epay processOrder 的 elseif 补填分支）。
		_ = s.orders.BackfillCallbackFields(tradeNo, nr.ChannelNo, raw["buyer"], raw["bill_trade_no"])
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

	// 回调入账方向按通道模式分派（对齐 epay functions.php:612-618）：
	//   - mode==1 商户直清：钱已直接到商户账户，平台只【扣】订单服务费 reducemoney（reducemoney>0 才扣）。
	//   - mode!=1 平台代收：钱进平台，平台【加】商户实得 getmoney（阶段A无费率则退回按订单金额）。
	// 之前无 mode 分支、恒 add getmoney，直清通道下方向反了（G-1 资金方向 P0）。
	if s.isDirectChannel(order.Channel) {
		reduce := s.calcReduceMoneyOnSettle(order)
		if reduce.GreaterThan(decimal.Zero) {
			if err := s.accounts.ChangeUserMoney(order.UID, reduce, false, "订单服务费", order.TradeNo); err != nil {
				return err
			}
		}
	} else {
		addMoney := order.GetMoney
		if addMoney.LessThanOrEqual(decimal.Zero) {
			addMoney = order.Money
		}
		if err := s.accounts.ChangeUserMoney(order.UID, addMoney, true, "订单收入", order.TradeNo); err != nil {
			return err
		}
	}

	// 计算并落库平台利润 profitmoney（对齐 epay processOrder：reducemoney=realmoney-getmoney，
	// profitmoney=reducemoney - realmoney*通道成本费率costrate/100）。供日报利润 + 邀请返现 type=2 口径用。
	profitMoney := s.calcProfitMoney(order)
	if profitMoney.GreaterThan(decimal.Zero) {
		_ = s.orders.SetProfitMoney(order.TradeNo, profitMoney)
	}

	// 邀请返现：下单商户若有上级(upid)，按比例实时返现到上级余额（对齐 epay functions.php 结算钩子）。
	// 失败不回滚入账（返现是附属激励），仅静默跳过。
	if s.invite != nil {
		getMoney := order.GetMoney
		s.invite.SettleOnPaid(order.UID, order.Money, getMoney, order.TradeNo)
	}

	// 对外通知（K-1 order 场景）：支付成功后按商户 msgconfig 发微信/邮件/短信到账提醒
	// （对齐 epay functions.php:629 MsgNotice::send('order')）。异步触发，不阻塞入账。
	if s.notice != nil && order.UID > 0 {
		payTime := ""
		if order.EndTime != nil {
			payTime = order.EndTime.Format("2006-01-02 15:04:05")
		} else {
			payTime = time.Now().Format("2006-01-02 15:04:05")
		}
		go s.notice.Send("order", order.UID, map[string]string{
			"trade_no":     order.TradeNo,
			"out_trade_no": order.OutTradeNo,
			"name":         order.Name,
			"money":        order.Money.StringFixed(2),
			"type":         order.TypeShow,
			"time":         payTime,
		})
	}

	// daytop 单日限额累计（对齐 epay functions.php:654-663）：通道设了 daytop>0 时，
	// 累计该通道今日已付 realmoney，达到 daytop → 置 daystatus=1 暂停该通道（次日 cron 重置）。
	s.accrueChannelDaytop(order.Channel)

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

// accrueChannelDaytop 累计通道当日已付 realmoney，达到 daytop 阈值则置 daystatus=1 暂停该通道。
// 对齐 epay functions.php 的 daytop 累计逻辑（我方直接按订单表实时聚合，免缓存漂移）。
// channelID<=0 或通道未设 daytop 时跳过；任何查询失败静默返回（不影响入账主流程）。
func (s *PayService) accrueChannelDaytop(channelID int) {
	if channelID <= 0 || s.channels == nil {
		return
	}
	ch, err := s.channels.FindByID(uint(channelID))
	if err != nil || ch == nil || ch.DayTop <= 0 || ch.DayStatus != 0 {
		return
	}
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	sum, err := s.orders.SumTodayPaidRealMoneyByChannel(channelID, dayStart, dayEnd)
	if err != nil {
		return
	}
	if sum.GreaterThanOrEqual(decimal.NewFromInt(int64(ch.DayTop))) {
		_ = s.channels.SetDayStatus(uint(channelID), 1)
	}
}

// checkBlockBuyer 买家维度回调风控（对齐 epay checkBlockUser）：
//  1. 命中 type=0(账号/openid) 黑名单 → 拒付；
//  2. 全局 pay_userlimit>0 时，买家「今日」已付订单数(status>0) 达上限 → 拒付。
// 依赖 blacklist / cfg 注入；未注入则跳过对应校验（向后兼容）。
func (s *PayService) checkBlockBuyer(buyer string) error {
	if s.blacklist != nil && s.blacklist.IsBlocked(0, buyer) {
		return payErr("系统异常无法完成付款")
	}
	if s.cfg != nil {
		limit := s.cfg.Int("pay_userlimit", 0)
		if limit > 0 {
			now := time.Now()
			dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			dayEnd := dayStart.AddDate(0, 0, 1)
			cnt, err := s.orders.CountPaidByBuyerRange(buyer, dayStart, dayEnd)
			if err == nil && cnt >= int64(limit) {
				return payErr("你今天已无法再发起支付，请明天再试")
			}
		}
	}
	return nil
}

// ResendNotify 商户主动触发重新通知（补单/重发回调）。复用 fireMerchantNotify 的通知逻辑。
func (s *PayService) ResendNotify(tradeNo string) {
	s.fireMerchantNotify(context.Background(), tradeNo)
}

// SettleOnFill 后台手动补单入账：订单已被置为已付后，执行入账 + 分账 + 商户通知。
// 复用支付成功链路 settle（对齐 epay fillorder → processOrder）。
func (s *PayService) SettleOnFill(tradeNo string) error {
	return s.settle(context.Background(), tradeNo)
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
	params := s.buildCallbackParams(order, m)
	if doNotify(ctx, appendQuery(order.NotifyURL, params)) {
		_ = s.orders.SetNotifySuccess(tradeNo)
		return
	}
	// 首次失败 → notify=1，下次重试 1 分钟后（对齐 epay 首档）。
	_ = s.orders.SetNotifyRetry(tradeNo, 1, time.Now().Add(notifyBackoff(1)))
}
