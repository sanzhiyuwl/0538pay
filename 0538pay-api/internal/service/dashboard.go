package service

import (
	"strconv"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// itoa64 int64 转字符串。
func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

// DashboardService 后台仪表盘聚合（对齐 epay admin/index.php + ajax getcount）。
type DashboardService struct {
	repo    *repository.DashboardRepo
	domains *repository.DomainRepo
	profit  *repository.ProfitRepo
	admins  *repository.AdminRepo // 弱密码/默认密码告警（可空，SetAdminRepo 注入）
}

func NewDashboardService(repo *repository.DashboardRepo, domains *repository.DomainRepo, profit *repository.ProfitRepo) *DashboardService {
	return &DashboardService{repo: repo, domains: domains, profit: profit}
}

// SetAdminRepo 注入管理员 repo，启用弱密码/默认密码安全告警。nil 则不告警。
func (s *DashboardService) SetAdminRepo(a *repository.AdminRepo) { s.admins = a }

// Overview 汇总全平台仪表盘数据。
func (s *DashboardService) Overview() (*dto.AdminDashboard, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)
	ydayStart := todayStart.AddDate(0, 0, -1)
	farFuture := todayStart.AddDate(100, 0, 0)
	farPast := todayStart.AddDate(-100, 0, 0)

	d := &dto.AdminDashboard{}

	// —— 概况卡 ——
	ordToday, _ := s.repo.CountOrders(-1, todayStart, todayEnd)
	ordYday, _ := s.repo.CountOrders(-1, ydayStart, todayStart)
	ordTotal, _ := s.repo.TotalOrders()
	paidToday, _ := s.repo.CountOrders(1, todayStart, todayEnd)
	paidYday, _ := s.repo.CountOrders(1, ydayStart, todayStart)
	amtToday, _ := s.repo.SumOrderMoney("money", todayStart, todayEnd)
	amtYday, _ := s.repo.SumOrderMoney("money", ydayStart, todayStart)
	amtTotal, _ := s.repo.SumOrderMoney("money", farPast, farFuture)
	profitToday, _ := s.repo.SumOrderMoney("profit_money", todayStart, todayEnd)
	profitYday, _ := s.repo.SumOrderMoney("profit_money", ydayStart, todayStart)
	profitTotal, _ := s.repo.SumOrderMoney("profit_money", farPast, farFuture)

	d.Overview = []dto.DashOverviewCard{
		{Label: "订单数", Today: itoa64(ordToday), Yesterday: itoa64(ordYday), TotalLabel: "累计订单", Total: itoa64(ordTotal)},
		{Label: "成功订单", Today: itoa64(paidToday), Yesterday: itoa64(paidYday), TotalLabel: "累计成功", Total: itoa64(ordTotal)},
		{Label: "交易额(元)", Today: amtToday.StringFixed(2), Yesterday: amtYday.StringFixed(2), TotalLabel: "累计交易额", Total: amtTotal.StringFixed(2)},
		{Label: "手续费利润(元)", Today: profitToday.StringFixed(2), Yesterday: profitYday.StringFixed(2), TotalLabel: "累计利润", Total: profitTotal.StringFixed(2)},
	}

	// —— 今日成功率 ——
	rate := decimal.Zero
	if ordToday > 0 {
		rate = decimal.NewFromInt(paidToday).Mul(decimal.NewFromInt(100)).Div(decimal.NewFromInt(ordToday)).Round(1)
	}
	d.SuccessRate = rate.String()

	// —— 汇总数字 ——
	d.Merchants, _ = s.repo.TotalMerchants()
	d.OrdersTotal = ordTotal
	bal, _ := s.repo.SumMerchantBalance()
	d.TotalMoney = bal.StringFixed(2)
	settled, _ := s.repo.SumSettled()
	d.SettledSum = settled.StringFixed(2)

	// —— 待办 ——
	d.Todo.PendingSettle, _ = s.repo.CountSettlePending()
	d.Todo.PendingDomain, _ = s.domains.CountByStatus(0)
	d.Todo.PendingProfit = s.countPendingProfit()
	d.Todo.UnpaidOrders, _ = s.repo.CountOrders(0, todayStart, todayEnd)

	// —— 近 7 日趋势 ——
	d.Trend = s.trend(todayStart)

	// —— 最近订单 ——
	orders, _ := s.repo.RecentOrders(8)
	for i := range orders {
		o := &orders[i]
		d.Recent = append(d.Recent, dto.DashRecentOrder{
			TradeNo:  o.TradeNo,
			UID:      o.UID,
			TypeShow: o.TypeShow,
			Money:    o.Money.StringFixed(2),
			Status:   o.Status,
			Time:     o.AddTime.Format(timeLayout),
		})
	}

	// —— 支付方式手续费利润交叉表（今+近6日 × 各支付方式）——
	d.FeeProfit = s.feeProfitTable(todayStart)

	// —— 安全告警（弱密码/默认密码）——
	d.Alerts = s.securityAlerts()

	return d, nil
}

// feeProfitTable 构建近 7 日（含今日）× 各支付方式的收入/利润交叉表（对齐 epay profit_paytype）。
func (s *DashboardService) feeProfitTable(todayStart time.Time) dto.DashFeeProfit {
	ft := dto.DashFeeProfit{
		Income: map[string][]string{},
		Profit: map[string][]string{},
	}
	// 收集列（支付方式显示名，保持出现顺序稳定）。
	typeOrder := []string{}
	seen := map[string]bool{}
	// 每日聚合缓存：dayIdx → (typeshow → (income,profit))
	type cell struct{ income, profit decimal.Decimal }
	daily := make([]map[string]cell, 7)

	for i := 6; i >= 0; i-- {
		idx := 6 - i
		dayStart := todayStart.AddDate(0, 0, -i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		ft.Days = append(ft.Days, dayStart.Format("01-02"))
		daily[idx] = map[string]cell{}
		rows, _ := s.repo.SumByPayType(dayStart, dayEnd)
		for _, r := range rows {
			name := r.TypeShow
			if name == "" {
				name = "未知"
			}
			if !seen[name] {
				seen[name] = true
				typeOrder = append(typeOrder, name)
			}
			daily[idx][name] = cell{income: r.Income, profit: r.Profit}
		}
	}
	ft.PayTypes = typeOrder
	// 按列填充每日数组（缺失填 0.00）。
	for _, name := range typeOrder {
		inc := make([]string, 7)
		pro := make([]string, 7)
		for idx := 0; idx < 7; idx++ {
			c := daily[idx][name]
			inc[idx] = c.income.StringFixed(2)
			pro[idx] = c.profit.StringFixed(2)
		}
		ft.Income[name] = inc
		ft.Profit[name] = pro
	}
	return ft
}

// securityAlerts 检测管理员弱密码/默认密码，返回告警文案（对齐 epay 后台安全提醒）。
// 密码为 bcrypt 哈希，只能对常见弱口令候选逐一 Compare 命中。
func (s *DashboardService) securityAlerts() []string {
	if s.admins == nil {
		return nil
	}
	list, err := s.admins.List()
	if err != nil {
		return nil
	}
	weakCandidates := []string{"admin888", "admin123", "123456", "admin", "888888", "123456789", "12345678"}
	// 分别检测：是否存在弱登录密码、支付密码是否仍为默认值，最后合并为一条单行提示。
	weakLogin := false
	for i := range list {
		a := &list[i]
		for _, cand := range weakCandidates {
			if bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(cand)) == nil {
				weakLogin = true
				break
			}
		}
		if weakLogin {
			break
		}
	}
	defaultPayPwd := payPwdVerifier != nil && payPwdVerifier.VerifyPayPwd(defaultAdminPayPwd) == nil

	var alerts []string
	switch {
	case weakLogin && defaultPayPwd:
		alerts = append(alerts, "安全提醒：登录密码过于简单、支付密码仍为默认 123456，请尽快在右上角菜单分别修改。")
	case weakLogin:
		alerts = append(alerts, "安全提醒：登录密码过于简单，请尽快在右上角菜单修改登录密码。")
	case defaultPayPwd:
		alerts = append(alerts, "安全提醒：支付密码仍为默认 123456（转账/结算/退款校验用），请尽快在右上角菜单修改支付密码。")
	}
	return alerts
}

// trend 近 7 日（含今日）每日已支付订单数与交易额。
func (s *DashboardService) trend(todayStart time.Time) dto.DashTrend {
	t := dto.DashTrend{}
	for i := 6; i >= 0; i-- {
		dayStart := todayStart.AddDate(0, 0, -i)
		dayEnd := dayStart.AddDate(0, 0, 1)
		cnt, _ := s.repo.CountOrders(1, dayStart, dayEnd)
		amt, _ := s.repo.SumOrderMoney("money", dayStart, dayEnd)
		t.Labels = append(t.Labels, dayStart.Format("01-02"))
		t.Orders = append(t.Orders, cnt)
		t.Amounts = append(t.Amounts, amt.StringFixed(2))
	}
	return t
}

// countPendingProfit 待分账单数（status=0）。复用 ProfitRepo.ListPendingIDs。
func (s *DashboardService) countPendingProfit() int64 {
	ids, err := s.profit.ListPendingIDs(10000)
	if err != nil {
		return 0
	}
	return int64(len(ids))
}
