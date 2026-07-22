package model

import "time"

// Message 站内信（我方新增功能，epay 无此实体——epay 的 MsgNotice 是对外推送非站内收件箱）。
// 管理员下发给商户（UID>0 定向 / UID=0 全体广播），商户端收件箱查看 + 标记已读。
type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UID       uint      `gorm:"index;not null;default:0" json:"uid"` // 收件商户号，0=全体广播
	Title     string    `gorm:"size:128;not null" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	IsRead    int8      `gorm:"column:is_read;not null;default:0" json:"is_read"` // 0未读 1已读（广播信的已读态由 MessageRead 记录）
	CreatedAt time.Time `gorm:"index" json:"-"`
}

func (Message) TableName() string { return "pay_message" }

// MessageRead 广播信（UID=0）的已读回执（每个商户读一次记一条）。定向信直接改 Message.IsRead。
type MessageRead struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	MsgID     uint      `gorm:"column:msg_id;index;not null" json:"msg_id"`
	UID       uint      `gorm:"index;not null" json:"uid"`
	CreatedAt time.Time `json:"-"`
}

func (MessageRead) TableName() string { return "pay_message_read" }
