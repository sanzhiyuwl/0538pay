package repository

import (
	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// GroupRepo 用户组数据访问。
type GroupRepo struct{ db *gorm.DB }

func NewGroupRepo(db *gorm.DB) *GroupRepo { return &GroupRepo{db: db} }

// FindByID 按 gid 查用户组。未找到返回 (nil,nil)。
func (r *GroupRepo) FindByID(gid int) (*model.Group, error) {
	var g model.Group
	err := r.db.Where("gid = ?", gid).First(&g).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// ListBuyable 列出可购买(isbuy=1)的用户组，按 sort 升序。
func (r *GroupRepo) ListBuyable() ([]model.Group, error) {
	var list []model.Group
	err := r.db.Where("is_buy = 1").Order("sort ASC, gid ASC").Find(&list).Error
	return list, err
}

// FindByGID 别名（供内部使用）。
func (r *GroupRepo) All() ([]model.Group, error) {
	var list []model.Group
	err := r.db.Order("sort ASC, gid ASC").Find(&list).Error
	return list, err
}
