package model

import "time"

// RegCode 短信/邮箱验证码（对齐 epay pre_regcode）。注册/找回密码/改绑等场景的 OTP。
type RegCode struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	UID      uint      `gorm:"index;not null;default:0" json:"uid"`
	Scene    string    `gorm:"size:20;index;not null" json:"scene"` // reg/login/find/edit
	Type     int8      `gorm:"not null;default:1" json:"type"`      // 0邮箱 1手机
	Code     string    `gorm:"size:32;not null" json:"-"`
	To       string    `gorm:"column:sendto;size:64;index;not null" json:"-"` // 手机号/邮箱（to 是保留字，列名 sendto）
	IP       string    `gorm:"size:64" json:"-"`
	Status   int8      `gorm:"not null;default:0" json:"status"`   // 0未用 1已用
	ErrCount int       `gorm:"not null;default:0" json:"-"`        // 错误次数，>=5 失效
	SendTime time.Time `gorm:"index" json:"-"`                     // 发送时间
}

func (RegCode) TableName() string { return "pay_regcode" }
