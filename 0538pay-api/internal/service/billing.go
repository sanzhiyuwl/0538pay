package service

import (
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/repository"
)

// BillingService 平台月度对账账单聚合（我方独有细化项，epay 无）。
// 按账期从 订单/结算/代付/分账/退款 五张既有真实表归集平台资金收支，生成可对账的月度账单。
type BillingService struct {
	repo *repository.BillingRepo
}

func NewBillingService(repo *repository.BillingRepo) *BillingService {
	return &BillingService{repo: repo}
}

// defaultBillMonths 账单中心默认回溯的账期数（含当月）。
const defaultBillMonths = 6

// Billing 生成近 months 期（含当月）的平台月度账单，倒序返回（最新账期在前）。
// months<=0 时用默认值 defaultBillMonths。
func (s *BillingService) Billing(months int) (*dto.BillingResult, error) {
	if months <= 0 {
		months = defaultBillMonths
	}
	now := time.Now()
	// 当月月初（本地时区）。
	curMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	res := &dto.BillingResult{}
	for i := 0; i < months; i++ {
		start := curMonthStart.AddDate(0, -i, 0)
		end := start.AddDate(0, 1, 0)
		bill, err := s.buildMonth(start, end)
		if err != nil {
			return nil, err
		}
		// 当月(i==0)标进行中，历史月份标已归档。
		if i == 0 {
			bill.Status = 0
		} else {
			bill.Status = 1
		}
		res.Bills = append(res.Bills, *bill)
	}
	return res, nil
}

// buildMonth 归集单个账期 [start,end) 的收支。
func (s *BillingService) buildMonth(start, end time.Time) (*dto.MonthlyBill, error) {
	profit, err := s.repo.SumOrderProfit(start, end)
	if err != nil {
		return nil, err
	}
	settleOut, err := s.repo.SumSettleReal(start, end)
	if err != nil {
		return nil, err
	}
	transferOut, err := s.repo.SumTransferMoney(start, end)
	if err != nil {
		return nil, err
	}
	profitOut, err := s.repo.SumProfitMoney(start, end)
	if err != nil {
		return nil, err
	}
	refundOut, err := s.repo.SumRefundMoney(start, end)
	if err != nil {
		return nil, err
	}
	orders, err := s.repo.CountPaidOrders(start, end)
	if err != nil {
		return nil, err
	}

	incomes := []dto.BillItem{
		{Label: "订单手续费利润", Amount: profit.StringFixed(2), Kind: "income"},
	}
	expenses := []dto.BillItem{
		{Label: "商户结算打款", Amount: settleOut.StringFixed(2), Kind: "expense"},
		{Label: "代付转账", Amount: transferOut.StringFixed(2), Kind: "expense"},
		{Label: "分账划拨", Amount: profitOut.StringFixed(2), Kind: "expense"},
		{Label: "订单退款", Amount: refundOut.StringFixed(2), Kind: "expense"},
	}

	incomeSum := profit
	expenseSum := settleOut.Add(transferOut).Add(profitOut).Add(refundOut)
	net := incomeSum.Sub(expenseSum)

	return &dto.MonthlyBill{
		Period:   start.Format("2006-01"),
		Orders:   orders,
		Incomes:  incomes,
		Expenses: expenses,
		Income:   incomeSum.StringFixed(2),
		Expense:  expenseSum.StringFixed(2),
		Net:      net.StringFixed(2),
	}, nil
}
