package service

import (
	"strconv"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
)

// itoa64 int64 转字符串。
func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

// DashboardService 后台仪表盘聚合（对齐 epay admin/index.php + ajax getcount）。
type DashboardService struct {
	repo    *repository.DashboardRepo
	domains *repository.DomainRepo
	profit  *repository.ProfitRepo
}

func NewDashboardService(repo *repository.DashboardRepo, domains *repository.DomainRepo, profit *repository.ProfitRepo) *DashboardService {
	return &DashboardService{repo: repo, domains: domains, profit: profit}
}

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
	return d, nil
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
