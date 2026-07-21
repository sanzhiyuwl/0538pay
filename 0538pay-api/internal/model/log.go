package model

import "time"

// LoginLog 登录日志。对齐 epay pre_log（本质是登录日志，非全量操作审计）。
// 记录后台/商户登录事件（成功/失败/快捷登录），只读。自研表名 pay_log。
type LoginLog struct {
	ID   uint      `gorm:"primaryKey" json:"id"`
	UID  uint      `gorm:"index;not null;default:0" json:"uid"` // 商户ID，0=管理员
	Type string    `gorm:"size:20" json:"type"`                 // 操作类型（登录后台/登录失败/普通登录…）
	Date time.Time `gorm:"index" json:"-"`                      // 时间（原始）
	IP   string    `gorm:"size:50" json:"ip"`                   // 操作IP
	City string    `gorm:"size:20" json:"city"`                 // 归属地（可空）
}

func (LoginLog) TableName() string { return "pay_log" }

// InviteCode 邀请码。对齐 epay pre_invitecode。reg_open=仅邀请码模式时注册需填。
// 自研表名 pay_invitecode。
type InviteCode struct {
	ID      uint       `gorm:"primaryKey" json:"id"`
	Code    string     `gorm:"size:40;uniqueIndex;not null" json:"code"` // 邀请码
	Status  int8       `gorm:"not null;default:0" json:"status"`         // 0未使用 1已使用
	AddTime time.Time  `json:"-"`                                        // 生成时间（原始）
	UseTime *time.Time `json:"-"`                                        // 使用时间（原始）
	UID     *uint      `json:"uid"`                                      // 使用者商户ID
}

func (InviteCode) TableName() string { return "pay_invitecode" }
