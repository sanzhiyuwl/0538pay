package repository

import (
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// DashboardRepo 后台仪表盘全平台聚合数据访问（对齐 epay admin/index.php + ajax getcount）。
// 纯只读聚合，无独立表。
type DashboardRepo struct{ db *gorm.DB }

func NewDashboardRepo(db *gorm.DB) *DashboardRepo { return &DashboardRepo{db: db} }

// CountOrders 统计 [start,end) 内订单数（status<0 表示不限状态，>=0 限定状态）。
func (r *DashboardRepo) CountOrders(status int, start, end time.Time) (int64, error) {
	tx := r.db.Model(&model.Order{}).Where("add_time >= ? AND add_time < ?", start, end)
	if status >= 0 {
		tx = tx.Where("status = ?", status)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// SumOrderMoney 汇总 [start,end) 内已支付订单金额（field 指定金额字段）。
func (r *DashboardRepo) SumOrderMoney(field string, start, end time.Time) (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.Order{}).
		Where("status = 1 AND add_time >= ? AND add_time < ?", start, end).
		Select("COALESCE(SUM(" + field + "),0)").Scan(&v).Error
	return v, err
}

// TotalOrders 订单总数（全部时间，不限状态）。
func (r *DashboardRepo) TotalOrders() (int64, error) {
	var n int64
	err := r.db.Model(&model.Order{}).Count(&n).Error
	return n, err
}

// TotalMerchants 商户总数。
func (r *DashboardRepo) TotalMerchants() (int64, error) {
	var n int64
	err := r.db.Model(&model.Merchant{}).Count(&n).Error
	return n, err
}

// SumMerchantBalance 全体商户余额总额。
func (r *DashboardRepo) SumMerchantBalance() (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.Merchant{}).Select("COALESCE(SUM(money),0)").Scan(&v).Error
	return v, err
}

// SumSettled 已结算总额（结算记录 status=1 已完成的 real_money 之和）。
func (r *DashboardRepo) SumSettled() (decimal.Decimal, error) {
	var v decimal.Decimal
	err := r.db.Model(&model.SettleRecord{}).
		Where("status = 1").Select("COALESCE(SUM(real_money),0)").Scan(&v).Error
	return v, err
}

// CountSettlePending 待结算记录数（status=0）。
func (r *DashboardRepo) CountSettlePending() (int64, error) {
	var n int64
	err := r.db.Model(&model.SettleRecord{}).Where("status = 0").Count(&n).Error
	return n, err
}

// RecentOrders 最近 n 笔订单（按创建时间倒序，供仪表盘实时订单）。
func (r *DashboardRepo) RecentOrders(n int) ([]model.Order, error) {
	var list []model.Order
	err := r.db.Model(&model.Order{}).Order("add_time DESC").Limit(n).Find(&list).Error
	return list, err
}
