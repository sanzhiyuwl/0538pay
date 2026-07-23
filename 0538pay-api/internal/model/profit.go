package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// ProfitReceiver 分账规则（接收方）。一条 = 一个接收方 + 匹配条件。
// 对齐 epay pre_psreceiver。自研表名 pay_ps_receiver，不复制 epay 表。
//
// 匹配优先级（下单时 updateOrderProfits）：channel+uid+subchannel > channel+uid > channel+uid IS NULL(通道级全局)。
// UID 非空且通道 mode=0 时：分账成功从该商户余额扣除分账金额（changeUserMoney '订单分账'）。
type ProfitReceiver struct {
	ID         uint            `gorm:"primaryKey" json:"id"`
	Channel    int             `gorm:"index;not null" json:"channel"`         // 匹配的支付通道 id
	SubChannel int             `gorm:"default:0" json:"subchannel"`           // 子通道 id（0=不限）
	UID        *uint           `gorm:"index" json:"uid"`                      // 绑定商户（空=该通道全局；填=仅该商户且从其余额扣款）
	Account    string          `gorm:"size:255;not null" json:"account"`      // 接收方账号（| 分隔多接收方）
	Name       string          `gorm:"size:255" json:"name"`                  // 接收方姓名（可空；| 分隔）
	Rate       string          `gorm:"size:64;not null;default:30" json:"-"`  // 分账比例 %（对齐 epay varchar，| 分隔多接收方，段数缺省复用首段）
	MinMoney   decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"-"` // 订单最小金额门槛（0=不限）
	Status     int8            `gorm:"not null;default:0;index" json:"status"` // 0关闭 1开启
	AddTime    time.Time       `json:"-"`                                     // 创建时间（原始）
}

func (ProfitReceiver) TableName() string { return "pay_ps_receiver" }

// ProfitOrder 分账订单（明细）。一笔支付订单按规则拆出的分账记录。
// 对齐 epay pre_psorder。自研表名 pay_ps_order。
//
// 支付成功回调时按比例 round(floor(realmoney*rate)/100,2) 创建，初始 status=0 待分账。
// 状态：0待分账 1已提交 2成功 3失败 4取消。成功且规则绑定商户+mode=0 时从商户余额扣分账金额；
// 取消/回退时若已扣则解冻退回。真实渠道分账 API 待凭证，此处走本地状态流转。
type ProfitOrder struct {
	ID         uint            `gorm:"primaryKey" json:"id"`
	RID        uint            `gorm:"index;not null" json:"rid"`                    // 分账规则 id
	TradeNo    string          `gorm:"size:32;index;not null" json:"trade_no"`       // 系统订单号
	APITradeNo string          `gorm:"size:150" json:"api_trade_no"`                 // 上游渠道订单号
	SettleNo   string          `gorm:"size:150" json:"settle_no"`                    // 渠道返回分账单号
	PsUID      *uint           `json:"-"`                                            // 分账扣款商户（规则 uid+mode=0 时有值；决定是否扣余额）
	Money      decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"-"`         // 分账金额（原始）
	Status     int8            `gorm:"not null;default:0;index" json:"status"`       // 0待分账 1已提交 2成功 3失败 4取消
	Result     string          `gorm:"type:text" json:"result"`                      // 失败原因
	AddTime    time.Time       `gorm:"index" json:"-"`                               // 创建时间（原始）
	Debited    int8            `gorm:"not null;default:0" json:"-"`                  // 是否已扣商户余额（1=已扣，用于取消/回退时判断是否解冻退回）
	Delay      int8            `gorm:"not null;default:0;index" json:"-"`            // 延迟分账标记（E-6，对齐 epay pre_psorder.delay，供定时扫描）
}

func (ProfitOrder) TableName() string { return "pay_ps_order" }
