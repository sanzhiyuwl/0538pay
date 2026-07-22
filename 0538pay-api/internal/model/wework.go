package model

import "time"

// Wework 企业微信账号。对齐 epay pre_wework（自研表名 pay_wework）。
// 表单维护 name/appid(企业ID corpid)/appsecret；有状态开关。删除级联子表 pay_wxkfaccount。
type Wework struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"size:30;not null" json:"name"`   // 名称（仅显示，不重复）
	AppID       string     `gorm:"size:150" json:"appid"`          // 企业ID corpid（不重复）
	AppSecret   string     `gorm:"size:250" json:"-"`              // Secret（脱敏输出）
	AccessToken string     `gorm:"size:300" json:"-"`              // 运行时缓存 token
	Status      int8       `gorm:"not null;default:0" json:"status"` // 1开启 0关闭
	AddTime     *time.Time `json:"-"`
	UpdateTime  *time.Time `gorm:"column:update_time" json:"-"`
	ExpireTime  *time.Time `gorm:"column:expire_time" json:"-"`
}

func (Wework) TableName() string { return "pay_wework" }

// WxKfAccount 企业微信客服账号。对齐 epay pre_wxkfaccount（自研表名 pay_wxkfaccount）。
// 非手工 CRUD：只能通过「刷新」从企业微信 API 同步（依赖真实凭证）。列表页只显示每企微的客服数。
type WxKfAccount struct {
	ID       uint       `gorm:"primaryKey" json:"id"`
	WID      uint       `gorm:"column:wid;index;not null" json:"wid"`         // 关联 pay_wework.id
	OpenKfID string     `gorm:"size:60;uniqueIndex" json:"openkfid"` // 客服账号ID（唯一）
	URL      string     `gorm:"size:100" json:"url"`
	Cursor   string     `gorm:"size:30" json:"-"`
	Name     string     `gorm:"size:300" json:"name"`              // 客服名称
	AddTime  *time.Time `json:"-"`
	UseTime  *time.Time `gorm:"column:use_time" json:"-"`
}

func (WxKfAccount) TableName() string { return "pay_wxkfaccount" }
