package model

import "time"

// SiteConfig 官网 CMS 内容的 KV 存储（自研，非 epay 功能）。
// 一个 key 存一份整体 JSON 文档：content(首页营销板块) / docs(开发者文档) / settings(网站设置外壳)。
// 后台写、官网读。value 用 longtext 容纳大 JSON。自研表名 pay_site_config。
type SiteConfig struct {
	Key       string    `gorm:"column:k;primaryKey;size:32" json:"key"`
	Value     string    `gorm:"column:v;type:longtext" json:"value"` // 整份 JSON 文档
	UpdatedAt time.Time `json:"-"`
}

func (SiteConfig) TableName() string { return "pay_site_config" }
