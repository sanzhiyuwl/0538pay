package model

import "time"

// Role 后台角色（RBAC 增强，epay 原版为单管理员无角色体系，此为我方独有）。
// Code 与 Admin.Role 字符串对应（super/admin/operator/finance/service…），是账号与角色的关联键。
// Permissions 存可访问的功能模块 key（逗号分隔），特殊值 "*" 表示全部权限。
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:32;uniqueIndex;not null" json:"code"` // 角色代码，关联 Admin.Role
	Name        string    `gorm:"size:64;not null" json:"name"`             // 角色名
	Description string    `gorm:"size:255" json:"desc"`                     // 说明
	Permissions string    `gorm:"type:text" json:"-"`                       // 权限模块 key（逗号分隔，"*"=全部）
	Builtin     bool      `gorm:"not null;default:0" json:"builtin"`        // 内置角色不可删除
	Sort        int       `gorm:"not null;default:0" json:"sort"`           // 列表排序（越小越前）
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

func (Role) TableName() string { return "pay_role" }
