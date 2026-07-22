package model

import "github.com/shopspring/decimal"

// Group 用户组（会员等级）。对齐 epay pre_group。自研表名 pay_group。
// 商户可购买 isbuy=1 的组升级；price=单期售价，expire=有效期月数(0=永久)。
type Group struct {
	GID      int             `gorm:"primaryKey;column:gid" json:"gid"`      // 用户组ID（0=默认组）
	Name     string          `gorm:"size:64;not null" json:"name"`          // 组名
	IsBuy    int8            `gorm:"not null;default:0" json:"isbuy"`       // 是否允许购买 0/1
	Price    decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"-"` // 单期售价（原始）
	Expire   int             `gorm:"not null;default:0" json:"expire"`      // 有效期月数（0=永久）
	Sort     int             `gorm:"not null;default:0" json:"sort"`        // 列表排序（越小越前）
	Info     string          `gorm:"type:text" json:"info"`                 // 通道费率说明 JSON
	Config   string          `gorm:"type:text" json:"config"`               // 组级功能配置 JSON（结算/充值/代付/邀请等）
	Settings string          `gorm:"type:text" json:"settings"`             // 用户变量定义（对齐 epay group.settings）
	Visible  string          `gorm:"size:64" json:"visible"`                // 可见范围（GID 列表逗号分隔，空=全部可见）
}

func (Group) TableName() string { return "pay_group" }
