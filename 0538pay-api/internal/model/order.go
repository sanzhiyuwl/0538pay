package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Order 订单。对齐前端 mock/orders.ts 的 Order interface 字段（json tag 保持一致，
// 保证前端零改动接入）。自研表名 pay_order，不复制 epay 的 pre_order。
//
// 金额字段用 decimal.Decimal（DB DECIMAL(18,4)）；对外 JSON 由 decimal 自动序列化为字符串。
type Order struct {
	ID          uint            `gorm:"primaryKey" json:"-"`
	TradeNo     string          `gorm:"size:32;uniqueIndex;not null" json:"trade_no"`     // 系统订单号
	OutTradeNo  string          `gorm:"size:64;index" json:"out_trade_no"`                // 商户订单号
	APITradeNo  string          `gorm:"size:64" json:"api_trade_no"`                      // 接口订单号
	UID         uint            `gorm:"index;not null" json:"uid"`                        // 商户号
	Domain      string          `gorm:"size:128" json:"domain"`                           // 网站域名
	Name        string          `gorm:"size:255" json:"name"`                             // 商品名称
	Money       decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"money"`         // 订单金额
	RealMoney   *decimal.Decimal `gorm:"type:decimal(18,4)" json:"realmoney"`             // 实际支付（可空）
	GetMoney    decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"getmoney"`     // 商户分成
	RefundMoney decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"refundmoney"`  // 已退款金额
	ProfitMoney decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"profitmoney"`  // 手续费利润
	Type        int             `gorm:"not null" json:"type"`                             // 支付方式ID
	TypeName    string          `gorm:"size:32" json:"typename"`                          // 支付方式英文名
	TypeShow    string          `gorm:"size:32" json:"typeshowname"`                      // 支付方式中文名
	Channel     int             `gorm:"index" json:"channel"`                             // 通道ID
	Plugin      string          `gorm:"size:32" json:"plugin"`                            // 插件标识
	IP          string          `gorm:"size:45" json:"ip"`                                // 支付IP
	Buyer       string          `gorm:"size:128" json:"buyer"`                            // 支付账号
	AddTime     time.Time       `gorm:"index" json:"addtime"`                             // 创建时间
	EndTime     *time.Time      `json:"endtime"`                                          // 完成时间（可空）
	Status      int8            `gorm:"not null;default:0;index" json:"status"`           // 0未付1已付2退款3冻结4预授权
	Settle      int8            `gorm:"not null;default:0" json:"settle"`                 // 结算子状态 0/1/2/3
	Combine     int8            `gorm:"not null;default:0" json:"combine"`                // 是否合单 0/1
}

func (Order) TableName() string { return "pay_order" }
