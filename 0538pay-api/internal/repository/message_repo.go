package repository

import (
	"time"

	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// MessageRepo 站内信数据访问（我方新增）。
type MessageRepo struct{ db *gorm.DB }

func NewMessageRepo(db *gorm.DB) *MessageRepo { return &MessageRepo{db: db} }

// Create 新建站内信（管理员下发）。
func (r *MessageRepo) Create(m *model.Message) error { return r.db.Create(m).Error }

// AdminList 后台分页列出所有已下发的站内信（按时间倒序）。
func (r *MessageRepo) AdminList(page, pageSize int) ([]model.Message, int64, error) {
	tx := r.db.Model(&model.Message{})
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Message
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// Delete 后台删除站内信 + 其已读回执。
func (r *MessageRepo) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.Message{}, id).Error; err != nil {
			return err
		}
		return tx.Where("msg_id = ?", id).Delete(&model.MessageRead{}).Error
	})
}

// MerchantList 商户收件箱：定向本人(uid) + 全体广播(uid=0)，按时间倒序分页。
func (r *MessageRepo) MerchantList(uid uint, page, pageSize int) ([]model.Message, int64, error) {
	tx := r.db.Model(&model.Message{}).Where("uid = ? OR uid = 0", uid)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Message
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// FindByID 按 ID 查单条。未找到返回 (nil,nil)。
func (r *MessageRepo) FindByID(id uint) (*model.Message, error) {
	var m model.Message
	err := r.db.First(&m, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// ReadUIDs 取某广播信已读的商户 uid 集合（用于收件箱标注广播信已读态）。
func (r *MessageRepo) ReadUIDs(uid uint) (map[uint]bool, error) {
	var reads []model.MessageRead
	if err := r.db.Where("uid = ?", uid).Find(&reads).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]bool, len(reads))
	for _, x := range reads {
		out[x.MsgID] = true
	}
	return out, nil
}

// MarkRead 商户标记已读：定向信直接改 is_read=1；广播信(uid=0)写一条已读回执（幂等）。
func (r *MessageRepo) MarkRead(uid, msgID uint) error {
	var m model.Message
	if err := r.db.First(&m, msgID).Error; err != nil {
		return err
	}
	if m.UID == 0 {
		// 广播信：查回执是否已存在，不存在则插入。
		var cnt int64
		if err := r.db.Model(&model.MessageRead{}).
			Where("msg_id = ? AND uid = ?", msgID, uid).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			return nil
		}
		return r.db.Create(&model.MessageRead{MsgID: msgID, UID: uid, CreatedAt: time.Now()}).Error
	}
	// 定向信：限本人 scope。
	if m.UID != uid {
		return nil
	}
	return r.db.Model(&model.Message{}).Where("id = ?", msgID).Update("is_read", 1).Error
}

// UnreadCount 商户未读数：定向未读 + 广播中未在回执表出现的。
func (r *MessageRepo) UnreadCount(uid uint) (int64, error) {
	var directUnread int64
	if err := r.db.Model(&model.Message{}).
		Where("uid = ? AND is_read = 0", uid).Count(&directUnread).Error; err != nil {
		return 0, err
	}
	var broadcastTotal int64
	if err := r.db.Model(&model.Message{}).Where("uid = 0").Count(&broadcastTotal).Error; err != nil {
		return 0, err
	}
	var broadcastRead int64
	if err := r.db.Model(&model.MessageRead{}).Where("uid = ?", uid).Count(&broadcastRead).Error; err != nil {
		return 0, err
	}
	return directUnread + (broadcastTotal - broadcastRead), nil
}
