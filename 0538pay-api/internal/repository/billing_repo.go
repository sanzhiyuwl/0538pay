package repository

import (
	"time"

	"github.com/epvia/api/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BillingRepo 平台月度对账账单聚合的数据访问（我方独有细化项，epay 无直接对应）。
// 纯只读聚合，无独立表：按账期从 订单/结算/代付/分账/退款 五张既有真实表归集平台资金收支。
type BillingRepo struct{ db *gorm.DB }

func NewBillingRepo(db *gorm.DB) *BillingRepo { return &BillingRepo{db: db} }

// SumOrderProfit 汇总 [start,end) 内已支付订单(status=1)的手续费利润(profit_money)——平台收入主项。
func (r *BillingRepo) SumOrderProfit(start, end time.Time) (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.Order{}).
		Where("status = 1 AND add_time >= ? AND add_time < ?", start, end).
		Select("COALESCE(SUM(profit_money),0)").Scan(&v).Error
	return v, err
}

// SumSettleReal 汇总 [start,end) 内已完成结算(status=1)的实际打款额(real_money)——平台支出：商户结算打款。
// 以完成时间 end_time 归账（打款真正发生的月份），未完成的不计。
func (r *BillingRepo) SumSettleReal(start, end time.Time) (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.SettleRecord{}).
		Where("status = 1 AND end_time >= ? AND end_time < ?", start, end).
		Select("COALESCE(SUM(real_money),0)").Scan(&v).Error
	return v, err
}

// SumTransferMoney 汇总 [start,end) 内成功代付(status=1)的到账金额(money)——平台支出：代付转账。
// 以付款成功时间 pay_time 归账。
func (r *BillingRepo) SumTransferMoney(start, end time.Time) (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.Transfer{}).
		Where("status = 1 AND pay_time >= ? AND pay_time < ?", start, end).
		Select("COALESCE(SUM(money),0)").Scan(&v).Error
	return v, err
}

// SumProfitMoney 汇总 [start,end) 内分账成功(status=2)的分账金额(money)——平台支出：分账划拨。
func (r *BillingRepo) SumProfitMoney(start, end time.Time) (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.ProfitOrder{}).
		Where("status = 2 AND add_time >= ? AND add_time < ?", start, end).
		Select("COALESCE(SUM(money),0)").Scan(&v).Error
	return v, err
}

// SumRefundMoney 汇总 [start,end) 内退款成功(status=1)的退款金额(money)——平台支出：订单退款。
// 以完成时间 end_time 归账。
func (r *BillingRepo) SumRefundMoney(start, end time.Time) (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.RefundOrder{}).
		Where("status = 1 AND end_time >= ? AND end_time < ?", start, end).
		Select("COALESCE(SUM(money),0)").Scan(&v).Error
	return v, err
}

// CountPaidOrders 统计 [start,end) 内已支付订单数(status=1)，供账单标注当期成交笔数。
func (r *BillingRepo) CountPaidOrders(start, end time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.Order{}).
		Where("status = 1 AND add_time >= ? AND add_time < ?", start, end).
		Count(&n).Error
	return n, err
}
