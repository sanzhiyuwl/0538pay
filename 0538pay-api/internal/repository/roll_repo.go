package repository

import (
	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// RollRepo 通道轮询组数据访问。
type RollRepo struct{ db *gorm.DB }

func NewRollRepo(db *gorm.DB) *RollRepo { return &RollRepo{db: db} }

// All 返回全部轮询组（按 id 升序）。
func (r *RollRepo) All() ([]model.Roll, error) {
	var list []model.Roll
	err := r.db.Order("id ASC").Find(&list).Error
	return list, err
}

// FindByID 按主键查轮询组。未找到返回 (nil, nil)。
func (r *RollRepo) FindByID(id uint) (*model.Roll, error) {
	var m model.Roll
	err := r.db.Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create 新增轮询组。
func (r *RollRepo) Create(m *model.Roll) error {
	return r.db.Create(m).Error
}

// Update 更新轮询组指定字段（白名单 map）。
func (r *RollRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Roll{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除轮询组。
func (r *RollRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Roll{}).Error
}

// SetStatus 只更新状态字段。
func (r *RollRepo) SetStatus(id uint, status int8) error {
	return r.db.Model(&model.Roll{}).Where("id = ?", id).Update("status", status).Error
}

// AdvanceIndex 顺序轮询命中后回写游标（对齐 epay UPDATE pre_roll SET index=...）。
// 用条件更新，避免读改写竞态覆盖：仅在当前游标仍等于 from 时改为 to。
func (r *RollRepo) AdvanceIndex(id uint, from, to int) error {
	return r.db.Model(&model.Roll{}).
		Where("id = ? AND idx = ?", id, from).
		Update("idx", to).Error
}
