package repository

import (
	"time"

	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// RiskAutoRepo 风控自动关停所需的聚合查询（对齐 epay cron do=check）。只读聚合 + 关停写。
type RiskAutoRepo struct{ db *gorm.DB }

func NewRiskAutoRepo(db *gorm.DB) *RiskAutoRepo { return &RiskAutoRepo{db: db} }

// MerchantOrderRate 统计某商户在 [start,now) 内的订单总数与已支付数（算成功率）。
func (r *RiskAutoRepo) MerchantOrderRate(uid uint, start time.Time) (total, paid int64, err error) {
	base := r.db.Model(&model.Order{}).Where("uid = ? AND add_time >= ?", uid, start)
	if err = base.Count(&total).Error; err != nil {
		return
	}
	err = r.db.Model(&model.Order{}).Where("uid = ? AND add_time >= ? AND status = 1", uid, start).Count(&paid).Error
	return
}

// MerchantsWithGivenUpNotify 取近窗口内"通知已放弃(notify=-1)"单数达阈值的商户 uid 列表。
// 对齐 epay 连续通知失败关停：这里以放弃计数近似"连续失败"。
func (r *RiskAutoRepo) MerchantsWithGivenUpNotify(minCount int, start time.Time) ([]uint, error) {
	type row struct {
		UID uint
		Cnt int64
	}
	var rows []row
	err := r.db.Model(&model.Order{}).
		Select("uid, COUNT(*) AS cnt").
		Where("notify = -1 AND add_time >= ?", start).
		Group("uid").Having("COUNT(*) >= ?", minCount).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, x := range rows {
		ids = append(ids, x.UID)
	}
	return ids, nil
}

// ActivePayMerchants 取当前支付权限开启(pay=1)且有订单活动的商户 uid（成功率检查候选）。
// 限 [start,now) 内有订单的商户，避免全表扫。
func (r *RiskAutoRepo) ActivePayMerchants(start time.Time) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.Order{}).
		Distinct("pay_order.uid").
		Joins("JOIN pay_merchant ON pay_merchant.uid = pay_order.uid AND pay_merchant.pay = 1").
		Where("pay_order.add_time >= ?", start).
		Pluck("pay_order.uid", &ids).Error
	return ids, err
}

// ClosePay 关停商户支付权限(pay=0)。仅当前 pay=1 才改（避免重复关停）。返回是否实际关停。
func (r *RiskAutoRepo) ClosePay(uid uint) (bool, error) {
	res := r.db.Model(&model.Merchant{}).Where("uid = ? AND pay = 1", uid).Update("pay", 0)
	return res.RowsAffected > 0, res.Error
}

// ActiveChannels 取近窗口内有订单的已启用通道 id（B-5 通道成功率检查候选）。
func (r *RiskAutoRepo) ActiveChannels(start time.Time) ([]int, error) {
	var ids []int
	err := r.db.Model(&model.Order{}).
		Distinct("pay_order.channel").
		Joins("JOIN pay_channel ON pay_channel.id = pay_order.channel AND pay_channel.status = 1").
		Where("pay_order.add_time >= ? AND pay_order.channel > 0", start).
		Pluck("pay_order.channel", &ids).Error
	return ids, err
}

// ChannelOrderRate 统计某通道在 [start,now) 内的订单总数与已支付数（算成功率，B-5）。
func (r *RiskAutoRepo) ChannelOrderRate(channelID int, start time.Time) (total, paid int64, err error) {
	if err = r.db.Model(&model.Order{}).Where("channel = ? AND add_time >= ?", channelID, start).Count(&total).Error; err != nil {
		return
	}
	err = r.db.Model(&model.Order{}).Where("channel = ? AND add_time >= ? AND status = 1", channelID, start).Count(&paid).Error
	return
}

// CloseChannel 关停通道(status=0)。仅当前 status=1 才改。返回是否实际关停。
func (r *RiskAutoRepo) CloseChannel(channelID int) (bool, error) {
	res := r.db.Model(&model.Channel{}).Where("id = ? AND status = 1", channelID).Update("status", 0)
	return res.RowsAffected > 0, res.Error
}

// WriteRisk 写一条风控记录。
func (r *RiskAutoRepo) WriteRisk(rec *model.RiskRecord) error {
	return r.db.Create(rec).Error
}
