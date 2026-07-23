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

// MerchantsWithRecentNotifyFail 候选商户：近窗口内有 notify>0(重试中1~5 或已放弃-1) 订单的 pay=1 商户。
// 对齐 epay cron auto_check_notify 的触发时机（回调重试失败后检查该商户），供上层再逐个判「最近N单全失败」。
func (r *RiskAutoRepo) MerchantsWithRecentNotifyFail(start time.Time) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.Order{}).
		Distinct("pay_order.uid").
		Joins("JOIN pay_merchant ON pay_merchant.uid = pay_order.uid AND pay_merchant.pay = 1").
		Where("pay_order.notify > 0 AND pay_order.status > 0 AND pay_order.add_time >= ?", start).
		Pluck("pay_order.uid", &ids).Error
	return ids, err
}

// LastNOrdersAllNotifyFail 判断商户最近 count 个 status>0 订单是否【全部】notify>0（重试中或已放弃）。
// 1:1 对齐 epay cron.php:188-193：取 trade_no desc limit count，逐单数 notify>0，failcount>=count 即关。
// 不足 count 单则不满足（epay failcount 恒 < count，不关）。
func (r *RiskAutoRepo) LastNOrdersAllNotifyFail(uid uint, count int) (bool, error) {
	if count <= 0 {
		return false, nil
	}
	var notifies []int
	err := r.db.Model(&model.Order{}).
		Where("uid = ? AND status > 0", uid).
		Order("trade_no DESC").Limit(count).
		Pluck("notify", &notifies).Error
	if err != nil {
		return false, err
	}
	if len(notifies) < count {
		return false, nil
	}
	fail := 0
	for _, n := range notifies {
		if n > 0 {
			fail++
		}
	}
	return fail >= count, nil
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

// EnabledChannelsForCheck 取用于自动关停检查的启用通道（status=1）。ids 非空则仅取其中的通道
//（对齐 epay check_channel_ids：留空为全部）。返回 id + config（判断是否有子通道占位）。
func (r *RiskAutoRepo) EnabledChannelsForCheck(ids []int) ([]model.Channel, error) {
	var chs []model.Channel
	tx := r.db.Model(&model.Channel{}).Where("status = 1")
	if len(ids) > 0 {
		tx = tx.Where("id IN ?", ids)
	}
	err := tx.Order("id ASC").Find(&chs).Error
	return chs, err
}

// EnabledSubchannelsOfChannel 取某主通道下启用中的子通道（对齐 epay pre_subchannel status=1）。
func (r *RiskAutoRepo) EnabledSubchannelsOfChannel(channelID int) ([]model.SubChannel, error) {
	var subs []model.SubChannel
	err := r.db.Where("channel = ? AND status = 1", channelID).Order("id ASC").Find(&subs).Error
	return subs, err
}

// LastNOrdersAllUnpaidByChannel 判断某通道近窗口内最近 failCount 个订单是否【全部未支付】(status<=0)。
// 1:1 对齐 epay cron.php:262-268：limit failCount，succount==0(无 status>0) 才关；不足 failCount 单不关。
func (r *RiskAutoRepo) LastNOrdersAllUnpaidByChannel(channelID int, start time.Time, failCount int) (bool, error) {
	return r.lastNAllUnpaid(map[string]interface{}{"channel": channelID}, start, failCount)
}

// LastNOrdersAllUnpaidBySubchannel 同上，但限定 channel+subchannel（对齐 epay cron.php:247-259）。
func (r *RiskAutoRepo) LastNOrdersAllUnpaidBySubchannel(channelID, subchannelID int, start time.Time, failCount int) (bool, error) {
	return r.lastNAllUnpaid(map[string]interface{}{"channel": channelID, "subchannel": subchannelID}, start, failCount)
}

func (r *RiskAutoRepo) lastNAllUnpaid(where map[string]interface{}, start time.Time, failCount int) (bool, error) {
	if failCount <= 0 {
		return false, nil
	}
	var statuses []int8
	err := r.db.Model(&model.Order{}).
		Where(where).Where("add_time >= ?", start).
		Order("trade_no DESC").Limit(failCount).
		Pluck("status", &statuses).Error
	if err != nil {
		return false, err
	}
	if len(statuses) < failCount {
		return false, nil
	}
	for _, st := range statuses {
		if st > 0 {
			return false, nil // 有成功单，不关
		}
	}
	return true, nil
}

// CloseChannel 关停通道(status=0)。仅当前 status=1 才改。返回是否实际关停。
func (r *RiskAutoRepo) CloseChannel(channelID int) (bool, error) {
	res := r.db.Model(&model.Channel{}).Where("id = ? AND status = 1", channelID).Update("status", 0)
	return res.RowsAffected > 0, res.Error
}

// CloseSubchannel 关停子通道(status=0)。仅当前 status=1 才改。返回是否实际关停（F-5）。
func (r *RiskAutoRepo) CloseSubchannel(subchannelID uint) (bool, error) {
	res := r.db.Model(&model.SubChannel{}).Where("id = ? AND status = 1", subchannelID).Update("status", 0)
	return res.RowsAffected > 0, res.Error
}

// WriteRisk 写一条风控记录。
func (r *RiskAutoRepo) WriteRisk(rec *model.RiskRecord) error {
	return r.db.Create(rec).Error
}
