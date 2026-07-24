package repository

import (
	"time"

	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// SubChannelRepo 子通道数据访问。
type SubChannelRepo struct{ db *gorm.DB }

func NewSubChannelRepo(db *gorm.DB) *SubChannelRepo { return &SubChannelRepo{db: db} }

// ListByMerchant 列出某商户的全部子通道（后台管理用），按 id 升序。
func (r *SubChannelRepo) ListByMerchant(uid uint) ([]model.SubChannel, error) {
	var list []model.SubChannel
	err := r.db.Where("uid = ?", uid).Order("id ASC").Find(&list).Error
	return list, err
}

// FindByID 按主键查子通道。未找到返回 (nil, nil)。
func (r *SubChannelRepo) FindByID(id uint) (*model.SubChannel, error) {
	var m model.SubChannel
	err := r.db.Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CountByName 统计同商户下同名子通道数（组内唯一校验）。excludeID>0 时排除自身。
func (r *SubChannelRepo) CountByName(uid uint, name string, excludeID uint) (int64, error) {
	tx := r.db.Model(&model.SubChannel{}).Where("uid = ? AND name = ?", uid, name)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// Create 新增子通道。
func (r *SubChannelRepo) Create(m *model.SubChannel) error {
	return r.db.Create(m).Error
}

// Update 更新子通道指定字段（白名单 map）。
func (r *SubChannelRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.SubChannel{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除子通道。
func (r *SubChannelRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.SubChannel{}).Error
}

// DeleteByMerchant 删除某商户的全部子通道（删商户级联，对齐 epay ajax_user 删用户级联）。
func (r *SubChannelRepo) DeleteByMerchant(uid uint) error {
	return r.db.Where("uid = ?", uid).Delete(&model.SubChannel{}).Error
}

// DeleteByChannel 删除某主通道下的全部子通道（删通道级联，对齐 epay ajax_pay 删通道级联）。
func (r *SubChannelRepo) DeleteByChannel(channel int) error {
	return r.db.Where("channel = ?", channel).Delete(&model.SubChannel{}).Error
}

// SubChannelPick 子通道调度候选：JOIN 主通道后的下单可用信息（对齐 epay getSubmitInfo -2 分支）。
type SubChannelPick struct {
	SubID     uint   // 子通道ID
	ChannelID int    // 主通道ID
	Plugin    string // 主通道插件
	Rate      string // 主通道费率（decimal 转字符串）
	PayMin    string
	PayMax    string
	Info      string // 子通道自定义参数 JSON
	AppType   string // 主通道 apptype（子通道 info.apptype 非空则覆盖，对齐 epay getSub L44-46）
}

// FindPickable 取某商户在某支付方式下、按 use_time 升序（最久未用优先）排列的可用子通道。
// 条件：子通道 status=1、主通道 status=1 且 type 匹配（对齐 epay:
//
//	SELECT ... FROM pre_subchannel B INNER JOIN pre_channel A ON B.channel=A.id
//	WHERE B.uid AND A.type AND A.status=1 AND B.status=1 ORDER BY B.usetime ASC）。
//
// use_time 为 NULL（从未使用）排在最前（MySQL NULL 升序在前），符合最久未用优先语义。
func (r *SubChannelRepo) FindPickable(uid uint, typeID int) ([]SubChannelPick, error) {
	var rows []SubChannelPick
	err := r.db.Table("pay_subchannel AS B").
		Select("B.id AS sub_id, A.id AS channel_id, A.plugin AS plugin, A.rate AS rate, A.pay_min AS pay_min, A.pay_max AS pay_max, B.info AS info, A.apptype AS app_type").
		Joins("INNER JOIN pay_channel AS A ON B.channel = A.id").
		Where("B.uid = ? AND A.type = ? AND A.status = 1 AND B.status = 1 AND A.daystatus = 0", uid, typeID).
		Order("B.use_time ASC").
		Scan(&rows).Error
	return rows, err
}

// TouchUseTime 子通道命中后回写 use_time=NOW()（对齐 epay UPDATE pre_subchannel SET usetime=NOW()）。
func (r *SubChannelRepo) TouchUseTime(subID uint, now time.Time) error {
	return r.db.Model(&model.SubChannel{}).Where("id = ?", subID).
		Update("use_time", now).Error
}
