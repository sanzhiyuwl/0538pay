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

// RecordSettleApply 记录一次「代办修改结算账户」留痕（application_no + 时间）。
// 快照不存在（从未刷新过管控状态）时先建一条空快照壳，仅为承载留痕字段，不影响管控态判定。
func (r *ChannelControlRepo) RecordSettleApply(enrollID, uid uint, channelID int, subMchID, applyNo string) error {
	return r.recordApply(enrollID, uid, channelID, subMchID, map[string]any{
		"last_settle_apply_no": applyNo,
		"last_settle_apply_at": time.Now(),
	})
}

// RecordSubjectApply 记录一次「代办修改主体资料」留痕（apply_id + 时间）。
func (r *ChannelControlRepo) RecordSubjectApply(enrollID, uid uint, channelID int, subMchID, applyID string) error {
	return r.recordApply(enrollID, uid, channelID, subMchID, map[string]any{
		"last_subject_apply_no": applyID,
		"last_subject_apply_at": time.Now(),
	})
}

func (r *ChannelControlRepo) recordApply(enrollID, uid uint, channelID int, subMchID string, fields map[string]any) error {
	var existing model.ChannelControl
	err := r.db.Where("enroll_id = ?", enrollID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		c := &model.ChannelControl{EnrollID: enrollID, UID: uid, ChannelID: channelID, SubMchID: subMchID, State: model.ChannelControlNormal}
		if err := r.db.Create(c).Error; err != nil {
			return err
		}
		return r.db.Model(&model.ChannelControl{}).Where("id = ?", c.ID).Updates(fields).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.ChannelControl{}).Where("id = ?", existing.ID).Updates(fields).Error
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
