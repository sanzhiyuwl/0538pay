package model

import "time"

// ArticleCategory 文章分类（epay pre_article 无分类，我方官网首页「最新动态」按分类分列展示，属独有扩展）。
// 官网 NewsSection/ClassicNews 依赖分类分列，故建行表时保留。
type ArticleCategory struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	Name    string    `gorm:"size:64;not null" json:"name"`   // 中文名，如「产品动态」
	EnName  string    `gorm:"size:128" json:"enName"`         // 英文小标题，如「Function Update」
	Cover   string    `gorm:"size:255" json:"cover"`          // 该列顶部深色头图（留空用占位）
	Sort    int       `gorm:"not null;default:0" json:"sort"` // 越小越靠前
	AddTime time.Time `json:"-"`
}

func (ArticleCategory) TableName() string { return "pay_article_category" }

// Article 文章（对齐 epay pre_article 行表 CRUD：id/title/content/addtime/top/active/count）。
// 自研表名 pay_article，不复制 epay 表名。
//   - Top    置顶（对齐 epay top，首页头条优先）
//   - Active 显示/隐藏（对齐 epay active，0 隐藏不在官网展示）
//   - Count  浏览量（对齐 epay count，详情页查看时 +1）
//
// 分类 CategoryID、摘要 Summary、封面 Cover、IsNew、Sort、Tags 为我方官网展示扩展字段（epay 无）。
type Article struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CategoryID uint      `gorm:"index;not null;default:0" json:"categoryId"` // 所属分类（我方扩展）
	Title      string    `gorm:"size:255;not null" json:"title"`             // 文章标题（唯一）
	Summary    string    `gorm:"size:512" json:"summary"`                    // 摘要（首页头条展示，我方扩展）
	Content    string    `gorm:"type:longtext" json:"content"`               // 正文 HTML
	Cover      string    `gorm:"size:255" json:"cover"`                      // 封面（我方扩展）
	Tags       string    `gorm:"size:255" json:"tags"`                       // 文章标签，英文逗号分隔（我方扩展：列表 meta pill + 侧栏热门标签云聚合）
	IsNew      bool      `gorm:"not null;default:0" json:"isNew"`            // 是否标 new（我方扩展）
	Top        bool      `gorm:"not null;default:0" json:"isTop"`            // 置顶（对齐 epay top）
	Active     int8      `gorm:"not null;default:1;index" json:"status"`     // 1 显示 0 隐藏（对齐 epay active）
	Sort       int       `gorm:"not null;default:0" json:"sort"`             // 越小越靠前（我方扩展）
	Count      int       `gorm:"not null;default:0" json:"views"`            // 浏览量（对齐 epay count）
	AddTime    time.Time `gorm:"index" json:"-"`                             // 发布时间
}

func (Article) TableName() string { return "pay_article" }
