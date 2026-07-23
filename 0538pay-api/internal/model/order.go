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
	BillTradeNo string          `gorm:"size:150;index" json:"bill_trade_no"`              // 对账/账单交易号(对齐 epay pre_order.bill_trade_no，A-10 回调回填，V2 回调优先此值作 api_trade_no)
	UID         uint            `gorm:"index;not null" json:"uid"`                        // 商户号
	Domain      string          `gorm:"size:128" json:"domain"`                           // 网站域名
	NotifyURL   string          `gorm:"size:512" json:"-"`                                // 商户异步通知地址
	ReturnURL   string          `gorm:"size:512" json:"-"`                                // 商户同步跳转地址
	Param       string          `gorm:"size:512" json:"-"`                                // 商户自定义透传参数
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
	Subchannel  int             `gorm:"not null;default:0" json:"-"`                      // 命中的子通道ID（0=无，对齐 epay pre_order.subchannel）
	Plugin      string          `gorm:"size:32" json:"plugin"`                            // 插件标识
	IP          string          `gorm:"size:45" json:"ip"`                                // 支付IP
	Buyer       string          `gorm:"size:128" json:"buyer"`                            // 支付账号
	AddTime     time.Time       `gorm:"index" json:"addtime"`                             // 创建时间
	EndTime     *time.Time      `json:"endtime"`                                          // 完成时间（可空）
	PayType     string          `gorm:"size:16" json:"-"`                                 // 收银台渲染方式 qrcode/redirect/html（下单后回填）
	QRCode      string          `gorm:"size:1024" json:"-"`                               // 渠道二维码内容/支付链接（下单后回填，收银台据此渲染）
	Version     int8            `gorm:"not null;default:0;index" json:"-"`                // 接口版本(对齐 epay pre_order.version)：0=V1(MD5回调) 1=V2(平台私钥RSA回调+timestamp)
	Status      int8            `gorm:"not null;default:0;index" json:"status"`           // 0未付1已付2退款3冻结4预授权
	Settle      int8            `gorm:"not null;default:0" json:"settle"`                 // 结算子状态 0/1/2/3
	Combine     int8            `gorm:"not null;default:0" json:"combine"`                // 是否合单 0/1
	Profits     uint            `gorm:"not null;default:0" json:"-"`                      // 命中的分账规则 id（0=不分账，对齐 epay pre_order.profits）
	Tid         int8            `gorm:"not null;default:0;index" json:"-"`                // 订单业务类型(对齐 epay pre_order.tid)：0普通 2充值余额 4购买会员 5充值保证金
	Notify      int8            `gorm:"not null;default:0;index" json:"-"`                // 商户通知状态/重试计数：0成功或无需 / >0 待重试第N次 / -1 放弃(对齐 epay notify 字段)
	NotifyTime  *time.Time      `json:"-"`                                                // 下次通知重试时间（可空）
	RefundTime  *time.Time      `gorm:"column:refundtime" json:"-"`                       // 退款时间（对齐 epay pre_order.refundtime，退款改单时回填，可空）
	// E-3 补齐字段（对齐 epay pre_order）
	Invite      int             `gorm:"column:invite;default:0" json:"-"`                 // 邀请返现受益商户 uid
	InviteMoney decimal.Decimal `gorm:"column:invitemoney;type:decimal(18,4);default:0" json:"-"` // 邀请返现金额
	Profits2    uint            `gorm:"column:profits2;default:0" json:"-"`               // 二级分账规则 id
	Domain2     string          `gorm:"column:domain2;size:64" json:"-"`                  // 第二域名/回调域名
	Mobile      string          `gorm:"column:mobile;size:100" json:"-"`                  // 下单手机号（pay_iplimit/buyer 维度之一）
	Ext         string          `gorm:"column:ext;type:text" json:"-"`                    // 通用扩展位(对齐 epay pre_order.ext，存 jsapi 参数/payurl 备份等杂项)
}

func (Order) TableName() string { return "pay_order" }
