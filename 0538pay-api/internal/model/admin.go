package model

import "time"

// Admin 运营后台管理员。对应 epay pre_config 里的后台账号体系，自研独立表。
type Admin struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"` // bcrypt 哈希，绝不输出
	Nickname  string    `gorm:"size:64" json:"nickname"`
	Role      string    `gorm:"size:32;not null;default:admin" json:"role"` // admin / super
	Status    int8      `gorm:"not null;default:1" json:"status"`           // 1 启用 0 停用
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Admin) TableName() string { return "sys_admin" }
