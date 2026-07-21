package service

import (
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// MerchantCenterService 商户中心业务：工作台聚合、结算记录、提现、退款、重新通知。
type MerchantCenterService struct {
	merchants *repository.MerchantRepo
	orders    *repository.OrderRepo
	records   *repository.RecordRepo
	settles   *repository.SettleRepo
	accounts  *repository.AccountRepo
	channels  *repository.ChannelRepo
	pay       *PayService // 复用商户通知重发
}

func NewMerchantCenterService(
	merchants *repository.MerchantRepo,
	orders *repository.OrderRepo,
	records *repository.RecordRepo,
	settles *repository.SettleRepo,
	accounts *repository.AccountRepo,
	channels *repository.ChannelRepo,
	pay *PayService,
) *MerchantCenterService {
	return &MerchantCenterService{
		merchants: merchants, orders: orders, records: records,
		settles: settles, accounts: accounts, channels: channels, pay: pay,
	}
}

// dayStart 返回某时刻当天 00:00:00（本地时区）。
func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// merchantStatusText 把 Merchant 的数值状态综合映射为工作台字符串枚举（对齐前端 mock）。
// 优先级：封禁 > 未审核 > 未实名 > 支付关 > 结算关 > 正常。
func merchantStatusText(m *model.Merchant) string {
	switch {
	case m.Status == 0:
		return "banned"
	case m.Status == 2:
		return "auditing"
	case m.Cert == 0:
		return "uncert"
	case m.Pay == 0:
		return "payoff"
	case m.Settle == 0:
		return "settleoff"
	default:
		return "normal"
	}
}

// Dashboard 组装工作台聚合数据。
func (s *MerchantCenterService) Dashboard(uid uint) (*dto.MerchantDashboard, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}

	now := time.Now()
	today := dayStart(now)
	yesterday := today.AddDate(0, 0, -1)

	ordersTotal, err := s.orders.CountPaidByMerchant(uid, time.Time{})
	if err != nil {
		return nil, err
	}
	ordersToday, err := s.orders.CountPaidByMerchant(uid, today)
	if err != nil {
		return nil, err
	}
	todayIncome, err := s.orders.SumPaidMoneyByMerchant(uid, today, now.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	yesterdayIncome, err := s.orders.SumPaidMoneyByMerchant(uid, yesterday, today)
	if err != nil {
		return nil, err
	}
	settled, err := s.settles.SumSettledByMerchant(uid)
	if err != nil {
		return nil, err
	}

	info := dto.MerchantDashInfo{
		UID:             m.UID,
		Name:            merchantName(m.UID),
		QQ:              m.QQ,
		Status:          merchantStatusText(m),
		GroupName:       groupName(m.GID),
		Money:           m.Money.InexactFloat64(),
		SettleMoney:     settled.InexactFloat64(),
		TodayIncome:     todayIncome.InexactFloat64(),
		YesterdayIncome: yesterdayIncome.InexactFloat64(),
		Orders:          ordersTotal,
		OrdersToday:     ordersToday,
	}

	alerts := dto.MerchantAlerts{
		NeedCert:   m.Cert == 0,
		NoSecurity: m.Phone == "" && m.Email == "",
		NoLoginPwd: m.Password == "",
	}

	// 通道费率表：列出各支付方式的费率（收入统计暂给 0，接订单按通道聚合后补）。
	channels := s.channelStats()

	// 趋势图：最近 9 条已完成结算的实际到账（升序）。
	trend := s.settleTrend(uid, 9)

	return &dto.MerchantDashboard{
		Info:      info,
		Alerts:    alerts,
		Channels:  channels,
		Announces: []dto.AnnounceView{}, // 公告域未建，先空（前端已容错）
		Trend:     trend,
	}, nil
}

// channelStats 汇总已开启通道的费率（按支付方式展示，收入统计留待订单聚合）。
func (s *MerchantCenterService) channelStats() []dto.MerchantChannel {
	q := dto.ChannelQuery{Page: 1, PageSize: 100}
	one := 1
	q.Status = &one
	list, _, err := s.channels.List(q)
	if err != nil {
		return []dto.MerchantChannel{}
	}
	out := make([]dto.MerchantChannel, 0, len(list))
	for i := range list {
		c := &list[i]
		out = append(out, dto.MerchantChannel{
			TypeName:    c.TypeName,
			ShowName:    c.TypeShow,
			Today:       0,
			Yesterday:   0,
			SuccessRate: 0,
			Rate:        c.Rate.StringFixed(2),
		})
	}
	return out
}

// settleTrend 组装最近 n 条结算趋势（时间升序）。
func (s *MerchantCenterService) settleTrend(uid uint, n int) dto.SettleTrendView {
	recent, err := s.settles.RecentSettledByMerchant(uid, n)
	if err != nil || len(recent) == 0 {
		return dto.SettleTrendView{Labels: []string{}, Data: []float64{}}
	}
	// repo 按 end_time 倒序返回，反转为升序
	labels := make([]string, 0, len(recent))
	data := make([]float64, 0, len(recent))
	for i := len(recent) - 1; i >= 0; i-- {
		r := recent[i]
		label := r.AddTime.Format("01-02")
		if r.EndTime != nil {
			label = r.EndTime.Format("01-02")
		}
		labels = append(labels, label)
		data = append(data, r.RealMoney.InexactFloat64())
	}
	return dto.SettleTrendView{Labels: labels, Data: data}
}


// ===== 结算记录（商户端）=====

// Settles 商户端结算记录分页查询（返回商户端专用 View，对齐 mock/merchant/settle.ts）。
func (s *MerchantCenterService) Settles(uid uint, status *int, page, pageSize int) ([]dto.MerchantSettleView, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := s.settles.ListByMerchant(uid, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.MerchantSettleView, 0, len(list))
	for i := range list {
		r := &list[i]
		views = append(views, dto.MerchantSettleView{
			ID:         r.ID,
			Type:       r.Type,
			Auto:       r.Auto == 1,
			Account:    r.Account,
			Money:      r.Money.InexactFloat64(),
			RealMoney:  r.RealMoney.InexactFloat64(),
			AddTime:    r.AddTime.Format(timeLayout),
			Status:     r.Status,
			FailReason: r.Result,
		})
	}
	return views, total, nil
}

// ===== 申请提现 =====

// enableMoney 计算可提现余额（对齐 epay apply.php）：
// D+1(settleType=2) 扣掉当日已成功订单收入；D+0(settleType=1) 全部余额。
func (s *MerchantCenterService) enableMoney(uid uint, balance decimal.Decimal) (decimal.Decimal, error) {
	if settleTypeDPlus1 {
		now := time.Now()
		today := dayStart(now)
		todayIncome, err := s.orders.SumPaidMoneyByMerchant(uid, today, now.AddDate(0, 0, 1))
		if err != nil {
			return decimal.Zero, err
		}
		en := balance.Sub(todayIncome)
		if en.IsNegative() {
			return decimal.Zero, nil
		}
		return en, nil
	}
	return balance, nil
}

// ApplyInfo 返回申请提现页所需信息。
func (s *MerchantCenterService) ApplyInfo(uid uint) (*dto.ApplyInfo, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}
	en, err := s.enableMoney(uid, m.Money)
	if err != nil {
		return nil, err
	}
	todayCount, err := s.settles.CountByMerchantSince(uid, dayStart(time.Now()))
	if err != nil {
		return nil, err
	}
	settleType := 1
	if settleTypeDPlus1 {
		settleType = 2
	}
	return &dto.ApplyInfo{
		SettleName:     settleTypeName(int8(m.SettleID)),
		Account:        m.Account,
		Username:       m.Username,
		Money:          m.Money.InexactFloat64(),
		EnableMoney:    en.InexactFloat64(),
		SettleMin:      settleMoney.InexactFloat64(),
		SettleMaxLimit: settleMaxLimit,
		SettleRate:     settleRate.InexactFloat64(),
		SettleFeeMin:   settleFeeMin.InexactFloat64(),
		SettleFeeMax:   settleFeeMax.InexactFloat64(),
		SettleType:     settleType,
		TodayCount:     int(todayCount),
	}, nil
}

// Apply 商户申请提现：校验(金额/门槛/可提余额/结算权限/次数) → 生成结算单并扣款(复用 CreateWithDebit)。
func (s *MerchantCenterService) Apply(uid uint, req dto.ApplyReq) error {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	if m.Settle != 1 {
		return maErr("结算功能未开启，无法提现")
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return maErr("请输入有效的提现金额")
	}
	if amount.LessThan(settleMoney) {
		return maErr("最低提现额为 ¥" + settleMoney.StringFixed(2))
	}
	en, err := s.enableMoney(uid, m.Money)
	if err != nil {
		return err
	}
	if amount.GreaterThan(en) {
		return maErr("超过可提现余额")
	}
	todayCount, err := s.settles.CountByMerchantSince(uid, dayStart(time.Now()))
	if err != nil {
		return err
	}
	if settleMaxLimit > 0 && int(todayCount) >= settleMaxLimit {
		return maErr("今日提现已达上限")
	}

	_, realmoney := calcFee(amount)
	rec := &model.SettleRecord{
		UID:       uid,
		Auto:      0, // 手动申请
		Type:      int8(m.SettleID),
		Account:   m.Account,
		Username:  m.Username,
		Money:     amount,
		RealMoney: realmoney,
		AddTime:   time.Now(),
		Status:    0, // 待结算
	}
	if err := s.settles.CreateWithDebit(rec, "手动提现"); err != nil {
		if err == repository.ErrInsufficientBalance {
			return maErr("余额不足")
		}
		return err
	}
	return nil
}

// ===== 订单退款（余额层逆向）=====

// Refund 商户端订单退款（全额）：校验归属+状态 → 幂等改单(status→2) → 退回已入账分成 + 退款流水。
// 渠道侧真实退款(调第三方)待真实凭证，此处为余额层逆向，对齐 epay 退款的资金语义。
func (s *MerchantCenterService) Refund(uid uint, tradeNo string) error {
	o, err := s.orders.FindByTradeNoAndUID(tradeNo, uid)
	if err != nil {
		return err
	}
	if o == nil {
		return maErr("订单不存在或不属于当前商户")
	}
	if o.Status != 1 {
		return maErr("仅已支付订单可退款")
	}
	// 幂等改单：status 1→2，写全额退款金额
	ok, err := s.orders.MarkRefunded(tradeNo, o.Money)
	if err != nil {
		return err
	}
	if !ok {
		return maErr("订单状态已变更，退款未执行")
	}
	// 逆向退回入账金额（入账时入的是 getmoney，无费率时为 money，与 pay_notify.settle 对称）
	refundAmount := o.GetMoney
	if refundAmount.LessThanOrEqual(decimal.Zero) {
		refundAmount = o.Money
	}
	if err := s.accounts.ChangeUserMoney(uid, refundAmount, false, "订单退款", tradeNo); err != nil {
		return err
	}
	return nil
}

// Renotify 重新通知商户（补单/重发回调）：仅对已支付订单，复用支付服务的通知重发。
func (s *MerchantCenterService) Renotify(uid uint, tradeNo string) error {
	o, err := s.orders.FindByTradeNoAndUID(tradeNo, uid)
	if err != nil {
		return err
	}
	if o == nil {
		return maErr("订单不存在或不属于当前商户")
	}
	if o.Status != 1 {
		return maErr("仅已支付订单可重新通知")
	}
	if o.NotifyURL == "" {
		return maErr("该订单未设置异步通知地址")
	}
	s.pay.ResendNotify(tradeNo)
	return nil
}

