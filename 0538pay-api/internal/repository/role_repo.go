package repository

import (
	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// RoleRepo 后台角色数据访问（RBAC 增强）。
type RoleRepo struct{ db *gorm.DB }

func NewRoleRepo(db *gorm.DB) *RoleRepo { return &RoleRepo{db: db} }

// All 列出全部角色，按 sort 升序、id 升序。
func (r *RoleRepo) All() ([]model.Role, error) {
	var list []model.Role
	err := r.db.Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

// FindByID 按 id 查角色。未找到返回 (nil,nil)。
func (r *RoleRepo) FindByID(id uint) (*model.Role, error) {
	var role model.Role
	err := r.db.Where("id = ?", id).First(&role).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// Create 新增角色。id 自增回填。
func (r *RoleRepo) Create(role *model.Role) error {
	return r.db.Create(role).Error
}

// Update 更新角色指定字段（白名单 map）。
func (r *RoleRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Role{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除角色。
func (r *RoleRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Role{}).Error
}

// CountByCode 统计同 code 角色数（唯一校验用）。excludeID 非 0 时排除自身。
func (r *RoleRepo) CountByCode(code string, excludeID uint) (int64, error) {
	tx := r.db.Model(&model.Role{}).Where("code = ?", code)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// Count 角色总数（用于判断是否需要播种内置角色）。
func (r *RoleRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&model.Role{}).Count(&n).Error
	return n, err
}

// CountAdminsByRole 统计使用某角色 code 的管理员数（删除角色前防悬挂校验）。
func (r *RoleRepo) CountAdminsByRole(code string) (int64, error) {
	var n int64
	err := r.db.Model(&model.Admin{}).Where("role = ?", code).Count(&n).Error
	return n, err
}
