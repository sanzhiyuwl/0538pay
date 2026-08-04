package repository

import (
	"time"

	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// ChannelControlRepo 子商户管控快照数据访问（pay_channel_control）。
// 快照按 enroll_id 唯一，每次刷新覆盖更新（Upsert）。
type ChannelControlRepo struct{ db *gorm.DB }

func NewChannelControlRepo(db *gorm.DB) *ChannelControlRepo { return &ChannelControlRepo{db: db} }

// Upsert 按 enroll_id 存/更新管控快照。存在则更新，不存在则插入。
func (r *ChannelControlRepo) Upsert(c *model.ChannelControl) error {
	now := time.Now()
	c.LastQueryAt = &now
	var existing model.ChannelControl
	err := r.db.Where("enroll_id = ?", c.EnrollID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(c).Error
	}
	if err != nil {
		return err
	}
	c.ID = existing.ID
	c.CreatedAt = existing.CreatedAt
	return r.db.Model(&model.ChannelControl{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"uid":                     c.UID,
		"channel_id":              c.ChannelID,
		"sub_mchid":               c.SubMchID,
		"state":                   c.State,
		"limited_functions":       c.LimitedFunctions,
		"other_limited_functions": c.OtherLimitedFunctions,
		"recovery_specifications": c.RecoverySpecifications,
		"raw_json":                c.RawJSON,
		"last_query_at":           c.LastQueryAt,
		"last_error":              c.LastError,
	}).Error
}

// FindByEnrollID 取某进件单的管控快照。未找到返回 (nil, nil)。
func (r *ChannelControlRepo) FindByEnrollID(enrollID uint) (*model.ChannelControl, error) {
	var c model.ChannelControl
	err := r.db.Where("enroll_id = ?", enrollID).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// FindBySubMchID 取某子商户号的管控快照（硬锁读取用）。未找到返回 (nil, nil)。
func (r *ChannelControlRepo) FindBySubMchID(subMchID string) (*model.ChannelControl, error) {
	if subMchID == "" {
		return nil, nil
	}
	var c model.ChannelControl
	err := r.db.Where("sub_mchid = ?", subMchID).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// MapByEnrollIDs 批量取快照，返回 enroll_id → 快照（列表页拼装用）。
func (r *ChannelControlRepo) MapByEnrollIDs(ids []uint) (map[uint]*model.ChannelControl, error) {
	out := make(map[uint]*model.ChannelControl, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var list []model.ChannelControl
	if err := r.db.Where("enroll_id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		out[list[i].EnrollID] = &list[i]
	}
	return out, nil
}
