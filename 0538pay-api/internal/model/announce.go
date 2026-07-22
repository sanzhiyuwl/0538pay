package model

import "time"

// Announce 网站公告（对齐 epay pre_anounce）。展示在商户中心/首页，支持排序与文字颜色。
type Announce struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	Content string    `gorm:"type:text;not null" json:"content"`
	Color   string    `gorm:"size:20" json:"color"` // 文字颜色(hex，空=默认)
	Sort    int       `gorm:"not null;default:50" json:"sort"`
	Status  int8      `gorm:"not null;default:1" json:"status"` // 1显示 0隐藏
	AddTime time.Time `json:"-"`
}

func (Announce) TableName() string { return "pay_announce" }
