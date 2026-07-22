package model

import (
	"github.com/shopspring/decimal"
)

// Channel 支付通道。一个通道 = 一个支付方式(type) + 一个支付插件(plugin) 的对接实例。
// 对齐 epay pre_channel 字段 + 前端 mock/channels.ts 的 Channel（json tag 一致）。
// 自研表名 pay_channel，不复制 epay 的 pre_channel。
//
// Config 存渠道密钥/参数的 JSON（如 appid、商户号、私钥路径等），是阶段 B 真实渠道的地基。
type Channel struct {
	ID       uint            `gorm:"primaryKey" json:"id"`
	Name     string          `gorm:"size:64;not null" json:"name"`                 // 显示名称
	Type     int             `gorm:"index;not null" json:"type"`                   // 支付方式ID
	TypeName string          `gorm:"size:32" json:"typename"`                      // 支付方式英文名（图标）
	TypeShow string          `gorm:"size:32" json:"typeshowname"`                  // 支付方式中文名
	Plugin   string          `gorm:"size:32;not null" json:"plugin"`               // 支付插件标识（对应 channel.registry 的 key）
	Mode     int8            `gorm:"not null;default:0" json:"mode"`               // 0=平台代收 1=商户直清
	Rate     decimal.Decimal `gorm:"type:decimal(5,2);not null;default:0" json:"-"` // 分成比例 %（原始，输出时格式化）
	CostRate decimal.Decimal `gorm:"type:decimal(5,2);default:0" json:"-"`         // 通道成本 %（原始）
	DayTop   int             `gorm:"default:0" json:"daytop"`                      // 单日限额（0=无）
	PayMin   string          `gorm:"size:16" json:"paymin"`                        // 单笔最小
	PayMax   string          `gorm:"size:16" json:"paymax"`                        // 单笔最大
	Config   string          `gorm:"type:text" json:"-"`                           // 渠道密钥/参数 JSON（不对外输出）
	Status   int8            `gorm:"not null;default:0;index" json:"status"`       // 0=关闭 1=开启
	// E-5 补齐字段（对齐 epay pre_channel，JSAPI/限额相关，真实渠道消费）
	AppWxMp   int    `gorm:"column:appwxmp;default:0" json:"appwxmp"`     // 绑定的微信公众号 id（JSAPI 需要）
	AppWxA    int    `gorm:"column:appwxa;default:0" json:"appwxa"`       // 绑定的微信小程序 id
	AppType   string `gorm:"column:apptype;size:50" json:"apptype"`       // 通道支持的 APP 支付子类型
	DayStatus int8   `gorm:"column:daystatus;default:0" json:"daystatus"` // 单日限额开关
}

func (Channel) TableName() string { return "pay_channel" }
