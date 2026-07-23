package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/sign"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// MerchantCenterService 商户中心业务：工作台聚合、结算记录、提现、退款、重新通知、保证金、购买会员。
type MerchantCenterService struct {
	merchants *repository.MerchantRepo
	orders    *repository.OrderRepo
	records   *repository.RecordRepo
	settles   *repository.SettleRepo
	accounts  *repository.AccountRepo
	channels  *repository.ChannelRepo
	groups    *repository.GroupRepo
	pay       *PayService        // 复用商户通知重发
	certVerify *CertVerifyService // 实名第三方核验（可空；SetCertVerify 注入）
}

// SetCertVerify 注入实名第三方核验服务。
func (s *MerchantCenterService) SetCertVerify(c *CertVerifyService) { s.certVerify = c }

func NewMerchantCenterService(
	merchants *repository.MerchantRepo,
	orders *repository.OrderRepo,
	records *repository.RecordRepo,
	settles *repository.SettleRepo,
	accounts *repository.AccountRepo,
	channels *repository.ChannelRepo,
	groups *repository.GroupRepo,
	pay *PayService,
) *MerchantCenterService {
	return &MerchantCenterService{
		merchants: merchants, orders: orders, records: records,
		settles: settles, accounts: accounts, channels: channels, groups: groups, pay: pay,
	}
}

// 保证金门槛（对齐 epay user_deposit_min）。config 域加载后刷新。
var depositMin = decimal.RequireFromString("1000")

// DepositInfo 返回保证金页信息。
func (s *MerchantCenterService) DepositInfo(uid uint) (*dto.DepositInfo, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}
	return &dto.DepositInfo{
		Deposit:    m.Deposit.InexactFloat64(),
		DepositMin: depositMin.InexactFloat64(),
		Money:      m.Money.InexactFloat64(),
	}, nil
}

// DepositRecharge 保证金充值。余额支付路径即时划转(money→deposit)；渠道支付待凭证(返回错误提示)。
func (s *MerchantCenterService) DepositRecharge(uid uint, req dto.DepositReq) error {
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return maErr("请输入有效的充值金额")
	}
	if req.PayType != "" && req.PayType != "balance" {
		return maErr("渠道充值保证金待支付渠道凭证接入，请先用余额支付")
	}
	if err := s.accounts.DepositFromBalance(uid, amount); err != nil {
		if err == repository.ErrInsufficientBalance {
			return maErr("可用余额不足")
		}
		return err
	}
	return nil
}

// DepositWithdraw 保证金提取回余额。
func (s *MerchantCenterService) DepositWithdraw(uid uint, req dto.DepositReq) error {
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return maErr("请输入有效的提取金额")
	}
	// F-2 保证金提取冻结期风控（对齐 epay deposit_withdraw user_deposit_day）：
	// 全局 user_deposit_day>0 时，最近 N 天内有成功订单则拒绝提取。
	//（epay 还查最近 N 天投诉记录，我方 pre_complain 属 complain 域待凭证，暂只做成功订单口径。）
	if userDepositDay > 0 {
		since := time.Now().AddDate(0, 0, -userDepositDay)
		if cnt, e := s.orders.CountPaidByMerchant(uid, since); e == nil && cnt > 0 {
			return maErr("你在最近" + strconv.Itoa(userDepositDay) + "天内有订单，无法提取保证金")
		}
	}
	if err := s.accounts.DepositWithdraw(uid, amount); err != nil {
		if err == repository.ErrInsufficientDeposit {
			return maErr("保证金余额不足")
		}
		return err
	}
	return nil
}

// ===== 实名认证（第三方认证待凭证）=====

// 实名工本费（对齐 epay cert_money）。config 域加载后刷新。
var certMoney = decimal.RequireFromString("0")

// 保证金提取冻结天数（对齐 epay user_deposit_day）：>0 时最近 N 天有成功订单禁提取。0=不限。
var userDepositDay = 0

// reloadMerchantCenterConfig 从 config 域刷新保证金门槛/实名工本费/提取冻结天数。
func reloadMerchantCenterConfig(cfg *ConfigService) {
	depositMin = cfg.Dec("user_deposit_min", depositMin)
	certMoney = cfg.Dec("cert_money", certMoney)
	userDepositDay = cfg.Int("user_deposit_day", 0)
}

// CertInfo 返回实名认证页信息（脱敏）。
func (s *MerchantCenterService) CertInfo(uid uint) (*dto.CertInfo, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}
	info := &dto.CertInfo{
		Cert:      m.Cert,
		CertType:  m.CertType,
		CertName:  maskName(m.CertName),
		CertNo:    maskIDNo(m.CertNo),
		CertCorp:  m.CertCorp,
		CertMoney: certMoney.InexactFloat64(),
		Method:    "支付宝身份验证", // 认证方式由平台配置，第三方认证待凭证
		CorpOpen:  true,
	}
	if m.CertTime != nil {
		info.CertTime = m.CertTime.Format(timeLayout)
	}
	return info, nil
}

// CertSubmit 提交实名认证。
// 第三方认证（支付宝/微信/阿里云人脸+三要素）依赖外部凭证，当前无法真实核验：
// 存入认证信息为"审核中"(cert=0)，实际通过需第三方回调置 cert=1（待凭证接入）。
// 工本费认证成功才扣（对齐 epay），故此处不扣费。
func (s *MerchantCenterService) CertSubmit(uid uint, req dto.CertSubmitReq) error {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	if m.Cert == 1 && m.CertType >= req.CertType {
		return maErr("您已完成实名认证")
	}
	name := strings.TrimSpace(req.CertName)
	no := strings.TrimSpace(req.CertNo)
	if len([]rune(name)) < 2 {
		return maErr("请输入真实姓名")
	}
	if !validIDNo(no) {
		return maErr("身份证号格式不正确")
	}
	if req.CertType == 1 && strings.TrimSpace(req.CertCorp) == "" {
		return maErr("请填写企业名称")
	}
	// 工本费余额校验（认证成功才扣，此处仅预检）
	if certMoney.GreaterThan(decimal.Zero) && m.Money.LessThan(certMoney) {
		return maErr("余额不足以支付实名认证工本费")
	}
	// 先存认证信息（cert 保持 0 审核中）。
	fields := map[string]interface{}{
		"cert_type": req.CertType,
		"cert_name": name,
		"cert_no":   no,
		"cert_corp": strings.TrimSpace(req.CertCorp),
	}
	if err := s.merchants.UpdateFields(uid, fields); err != nil {
		return err
	}

	// 第三方核验（对齐 epay cert_open 分派）。同步方式(手机三要素)凭证到位即可真核；
	// 异步/人脸方式返回待凭证提示。核验通过才 cert=1 + 扣工本费（对齐 epay 成功才扣）。
	if s.certVerify == nil {
		return maErr("实名核验服务未启用")
	}
	passed, verr := s.certVerify.Verify(context.Background(), name, no, m.Phone)
	if verr != nil {
		return verr // 待凭证/未通过，如实上抛（cert 仍为 0，信息已暂存）
	}
	if !passed {
		return maErr("实名核验未通过，请核对姓名与证件信息")
	}
	// 核验通过：cert=1 + certtime，扣工本费。
	now := time.Now()
	if err := s.merchants.UpdateFields(uid, map[string]interface{}{"cert": 1, "cert_time": now}); err != nil {
		return err
	}
	if certMoney.GreaterThan(decimal.Zero) {
		_ = s.accounts.ChangeUserMoney(uid, certMoney, false, "实名认证", "")
	}
	return nil
}

// maskName 姓名脱敏（保留首字，其余打星）。
func maskName(name string) string {
	r := []rune(name)
	if len(r) <= 1 {
		return name
	}
	return string(r[0]) + strings.Repeat("*", len(r)-1)
}

// maskIDNo 证件号脱敏（保留前3后4）。
func maskIDNo(no string) string {
	if len(no) <= 7 {
		return no
	}
	return no[:3] + strings.Repeat("*", len(no)-7) + no[len(no)-4:]
}

// validIDNo 简单身份证号校验（18位数字，末位可 X）。
func validIDNo(no string) bool {
	if len(no) != 18 {
		return false
	}
	for i, c := range no {
		if c >= '0' && c <= '9' {
			continue
		}
		if i == 17 && (c == 'X' || c == 'x') {
			continue
		}
		return false
	}
	return true
}

// ===== 余额充值 =====

// Recharge 余额充值：下内部订单(tid=2)走渠道支付，回调入账（对齐 epay recharge）。
// mock 渠道可端到端真跑；真实渠道待凭证。返回收银台信息（trade_no + pay_url/qrcode）。
func (s *MerchantCenterService) Recharge(uid uint, req dto.RechargeReq) (*dto.SubmitResp, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return nil, maErr("请输入有效的充值金额")
	}
	plugin := req.Plugin
	if plugin == "" {
		plugin = "mock"
	}
	resp, err := s.pay.CreateInternalOrder(context.Background(), uid, 2, "余额充值", amount, plugin)
	if err != nil {
		return nil, maErr("充值下单失败: " + err.Error())
	}
	return resp, nil
}

// ===== 购买会员 =====

// GroupPlans 返回可购买会员套餐列表。
func (s *MerchantCenterService) GroupPlans() ([]dto.GroupPlanView, error) {
	list, err := s.groups.ListBuyable()
	if err != nil {
		return nil, err
	}
	views := make([]dto.GroupPlanView, 0, len(list))
	for i := range list {
		g := &list[i]
		views = append(views, dto.GroupPlanView{
			ID: g.GID, Name: g.Name,
			Price: g.Price.InexactFloat64(), Expire: g.Expire,
			Rates: parseGroupRates(g.Info),
		})
	}
	return views, nil
}

// CurrentGroup 返回商户当前会员状态。
func (s *MerchantCenterService) CurrentGroup(uid uint) (*dto.GroupCurrentView, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}
	expire := "—"
	if m.GroupEnd != nil {
		expire = m.GroupEnd.Format("2006-01-02")
	}
	return &dto.GroupCurrentView{GID: m.GID, Name: groupName(m.GID), Expire: expire}, nil
}

// BuyGroup 购买会员。余额支付即时扣款升组；渠道支付待凭证。
// 时长：续期(购买当前组)从原到期往后加；新购从当前时间往后加；expire=0 永久(endtime=nil)。
func (s *MerchantCenterService) BuyGroup(uid uint, req dto.GroupBuyReq) error {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	g, err := s.groups.FindByID(req.GID)
	if err != nil {
		return err
	}
	if g == nil || g.IsBuy != 1 {
		return maErr("该会员套餐不可购买")
	}
	// 防重复：已是该永久组
	if m.GID == g.GID && m.GroupEnd == nil && g.Expire == 0 {
		return maErr("您已是该会员且为永久有效")
	}

	num := req.Num
	if num < 1 {
		num = 1
	}
	price := g.Price
	var endTime *time.Time
	if g.Expire > 0 {
		price = g.Price.Mul(decimal.NewFromInt(int64(num)))
		months := num * g.Expire
		base := time.Now()
		// 续期：购买的是当前组，从现有到期时间往后加
		if m.GID == g.GID && m.GroupEnd != nil && m.GroupEnd.After(base) {
			base = *m.GroupEnd
		}
		t := base.AddDate(0, months, 0)
		endTime = &t
	}
	// expire=0 永久组：endTime=nil，price=单价

	if req.PayType != "" && req.PayType != "balance" {
		return maErr("渠道购买会员待支付渠道凭证接入，请先用余额支付")
	}
	if err := s.accounts.BuyGroupWithBalance(uid, price, g.GID, endTime); err != nil {
		if err == repository.ErrInsufficientBalance {
			return maErr("可用余额不足，需 ¥" + price.StringFixed(2))
		}
		return err
	}
	return nil
}

// parseGroupRates 解析用户组 Info JSON 的费率说明。格式宽松：[{label,rate}]；解析失败返回空。
func parseGroupRates(info string) []dto.GroupRateItem {
	if strings.TrimSpace(info) == "" {
		return []dto.GroupRateItem{}
	}
	var items []dto.GroupRateItem
	if err := json.Unmarshal([]byte(info), &items); err != nil {
		return []dto.GroupRateItem{}
	}
	return items
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

// ===== API 信息 / 资料 / 密码 =====

// ApiInfo 返回商户 API 信息（V1 MD5 密钥）。
func (s *MerchantCenterService) ApiInfo(uid uint) (*dto.MerchantApiInfo, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, maErr("商户不存在")
	}
	return &dto.MerchantApiInfo{
		UID:     m.UID,
		MDKey:   m.AppKey,
		APIURL:  "https://0538pay.com/", // 接口地址（接站点配置域后动态）
		KeyType: m.KeyType,
		HasRSA:  m.PublicKey != "",
	}, nil
}

// GenRSAKeyPair 生成商户 RSA 密钥对（V2）：仅公钥入库，私钥一次性返回（对齐 epay：私钥不落库）。
// 返回私钥（单行 base64，商户须自行保存）。
func (s *MerchantCenterService) GenRSAKeyPair(uid uint) (string, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", maErr("商户不存在")
	}
	priv, pub, err := sign.GenerateRSAKeyPair()
	if err != nil {
		return "", err
	}
	if err := s.merchants.UpdateFields(uid, map[string]interface{}{"publickey": pub}); err != nil {
		return "", err
	}
	return priv, nil
}

// SetKeyType 设置商户签名模式（0=MD5+RSA兼容 1=仅RSA安全）。仅RSA模式需已配公钥。
func (s *MerchantCenterService) SetKeyType(uid uint, keytype int8) error {
	if keytype != 0 && keytype != 1 {
		return maErr("签名模式不合法")
	}
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	if keytype == 1 && m.PublicKey == "" {
		return maErr("请先生成 RSA 密钥对再切换到仅 RSA 安全模式")
	}
	return s.merchants.UpdateFields(uid, map[string]interface{}{"keytype": keytype})
}

// ResetKey 重置商户 MD5 通信密钥，返回新密钥。对齐 epay resetKey：随机 32 位。
// 副作用：原密钥立即失效，商户对接代码需同步更新。
func (s *MerchantCenterService) ResetKey(uid uint) (string, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", maErr("商户不存在")
	}
	key, err := randomHex(32)
	if err != nil {
		return "", err
	}
	if err := s.merchants.UpdateFields(uid, map[string]interface{}{"app_key": key}); err != nil {
		return "", err
	}
	return key, nil
}

// UpdateProfile 修改商户资料（收款账号 + 联系方式 + 扣费模式，仅模型已有字段）。
func (s *MerchantCenterService) UpdateProfile(uid uint, req dto.MerchantProfileReq) error {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	if req.SettleID < 1 || req.SettleID > 5 {
		return maErr("结算方式不合法")
	}
	if req.Mode != 0 && req.Mode != 1 {
		return maErr("扣费模式不合法")
	}
	// F-1 收款账号资金安全：改动已有 account/username（结算落点）时需登录密码二次确认
	//（对齐 epay edit_settle 的 need verify；我方以登录密码替代 OTP，与换绑同构）。
	newAccount := strings.TrimSpace(req.Account)
	newUsername := strings.TrimSpace(req.Username)
	settleChanged := (m.Account != "" && newAccount != m.Account) ||
		(m.Username != "" && newUsername != m.Username) ||
		(m.SettleID != 0 && req.SettleID != m.SettleID)
	if settleChanged && m.Password != "" {
		if req.Password == "" {
			return maErr("修改收款账号需输入登录密码确认身份")
		}
		if bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(req.Password)) != nil {
			return maErr("登录密码不正确")
		}
	}
	fields := map[string]interface{}{
		"settle_id": req.SettleID,
		"account":   newAccount,
		"username":  newUsername,
		"email":     strings.TrimSpace(req.Email),
		"qq":        strings.TrimSpace(req.QQ),
		"url":       strings.TrimSpace(req.URL),
		"mode":      req.Mode,
	}
	// 对齐 epay edit_info：keylogin/refund/transfer/remain_money 提交时才更新（nil=未提交不改）。
	if req.KeyLogin != nil {
		fields["keylogin"] = normBool(*req.KeyLogin)
	}
	if req.Refund != nil {
		fields["refund"] = normBool(*req.Refund)
	}
	if req.Transfer != nil {
		fields["transfer"] = normBool(*req.Transfer)
	}
	if req.RemainMoney != nil {
		fields["remain_money"] = strings.TrimSpace(*req.RemainMoney)
	}
	return s.merchants.UpdateFields(uid, fields)
}

// normBool 把任意 int8 归一为 0/1（对齐 epay intval 后仅存 0/1 语义，非 0 一律视为 1）。
func normBool(v int8) int8 {
	if v != 0 {
		return 1
	}
	return 0
}

// Rebind 换绑手机/邮箱（D-3，对齐 epay editinfo 密保换绑）。
// epay 用旧联系方式 OTP 验证；真 OTP 待短信/邮件凭证，此处用登录密码二次确认作身份校验（等价安全）。
// field: "phone" | "email"；校验格式 + 全局唯一。
func (s *MerchantCenterService) Rebind(uid uint, field, newValue, password string) error {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	// 身份校验：已设密码则校验（未设密码的老账号允许直接换绑，与改密逻辑一致）。
	if m.Password != "" {
		if password == "" {
			return maErr("请输入登录密码确认身份")
		}
		if bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(password)) != nil {
			return maErr("登录密码不正确")
		}
	}
	newValue = strings.TrimSpace(newValue)
	if newValue == "" {
		return maErr("请输入新的" + rebindLabel(field))
	}
	switch field {
	case "phone":
		if !rebindPhoneRe.MatchString(newValue) {
			return maErr("手机号格式不正确")
		}
		if n, _ := s.merchants.CountByPhone(newValue, uid); n > 0 {
			return maErr("该手机号已被其他商户绑定")
		}
		return s.merchants.UpdateFields(uid, map[string]interface{}{"phone": newValue})
	case "email":
		if !emailRe.MatchString(newValue) {
			return maErr("邮箱格式不正确")
		}
		if n, _ := s.merchants.CountByEmail(newValue, uid); n > 0 {
			return maErr("该邮箱已被其他商户绑定")
		}
		return s.merchants.UpdateFields(uid, map[string]interface{}{"email": newValue})
	default:
		return maErr("不支持的换绑类型")
	}
}

// rebindPhoneRe 中国大陆手机号格式（换绑校验用）。
var rebindPhoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

func rebindLabel(field string) string {
	if field == "email" {
		return "邮箱"
	}
	return "手机号"
}

// GetMsgConfig 返回商户消息提醒配置 JSON（D-3，对齐 epay editinfo 消息提醒设置）。空则返回默认。
func (s *MerchantCenterService) GetMsgConfig(uid uint) (string, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", maErr("商户不存在")
	}
	if strings.TrimSpace(m.MsgConfig) == "" {
		// 默认（对齐 epay edit_msgconfig 8 项：order/settle/login/complain/mchrisk/order_money/
		// balance/balance_money，F-11 补齐 complain/mchrisk/order_money 三项）。
		return `{"order":1,"settle":1,"login":0,"complain":0,"mchrisk":0,"order_money":"","balance":0,"balance_threshold":""}`, nil
	}
	return m.MsgConfig, nil
}

// SaveMsgConfig 保存商户消息提醒配置（D-3）。校验为合法 JSON、长度≤300。
func (s *MerchantCenterService) SaveMsgConfig(uid uint, cfg string) error {
	cfg = strings.TrimSpace(cfg)
	if len(cfg) > 300 {
		return maErr("消息配置过长")
	}
	if cfg != "" && !json.Valid([]byte(cfg)) {
		return maErr("消息配置不是合法 JSON")
	}
	return s.merchants.UpdateFields(uid, map[string]interface{}{"msgconfig": cfg})
}

// ChangePassword 修改登录密码（bcrypt）。已设密码则校验旧密码。
func (s *MerchantCenterService) ChangePassword(uid uint, req dto.MerchantPwdReq) error {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return err
	}
	if m == nil {
		return maErr("商户不存在")
	}
	newPwd := strings.TrimSpace(req.NewPwd)
	if len(newPwd) < 6 {
		return maErr("新密码至少 6 位")
	}
	if m.Password != "" {
		if bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(req.OldPwd)) != nil {
			return maErr("原密码不正确")
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.merchants.UpdateFields(uid, map[string]interface{}{"password": string(hash)})
}

// randomHex 生成 n 位十六进制随机串（密钥用）。
func randomHex(n int) (string, error) {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
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

