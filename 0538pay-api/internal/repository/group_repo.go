package repository

import (
	"github.com/epvia/api/internal/model"
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

// Create 新增用户组。GID 自增回填。
func (r *GroupRepo) Create(g *model.Group) error {
	return r.db.Create(g).Error
}

// Update 更新用户组指定字段（白名单 map）。
func (r *GroupRepo) Update(gid int, fields map[string]interface{}) error {
	return r.db.Model(&model.Group{}).Where("gid = ?", gid).Updates(fields).Error
}

// Delete 删除用户组。
func (r *GroupRepo) Delete(gid int) error {
	return r.db.Where("gid = ?", gid).Delete(&model.Group{}).Error
}

// CountByName 统计同名用户组数（组名唯一校验用）。excludeGID 非 nil 时排除自身。
func (r *GroupRepo) CountByName(name string, excludeGID *int) (int64, error) {
	tx := r.db.Model(&model.Group{}).Where("name = ?", name)
	if excludeGID != nil {
		tx = tx.Where("gid <> ?", *excludeGID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}
