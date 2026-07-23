package model

import "github.com/shopspring/decimal"

// SubOrder 合单支付的子订单（L-5，对齐 epay pre_suborder）。
//
// 微信服务商合单支付（wxpayn combineNativePay 等）把一笔大额订单拆成多个子单一起下单，
// 每个子单独立走渠道，规避单笔限额、便于分账。主单 pay_order.combine=1 标记为合单，
// 子单在此表按 trade_no（主单号）归集。拆单金额由 CombineSubMoneys 计算（见 service 层）。
//
// 自研表名 pay_suborder，不复制 epay 的 pre_suborder。渠道下单/回调/退款的真实对接待凭证（乙类），
// 本模型与拆单算法（甲类）先就绪，凭证到位后由 wxpayn 等渠道消费。
type SubOrder struct {
	ID         uint            `gorm:"primaryKey" json:"-"`
	TradeNo    string          `gorm:"size:32;index;not null" json:"trade_no"`      // 主单号（关联 pay_order.trade_no）
	SubTradeNo string          `gorm:"size:40;uniqueIndex;not null" json:"sub_trade_no"` // 子单号（主单号+序号）
	APITradeNo string          `gorm:"size:64" json:"api_trade_no"`                 // 渠道子单流水号（回调回填）
	Money      decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"money"`    // 子单金额（元）
	RefundMoney decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"refundmoney"` // 子单已退金额
	Settle     int8            `gorm:"not null;default:0" json:"settle"`            // 子单结算状态（对齐 epay suborder.settle）
	Status     int8            `gorm:"not null;default:0" json:"status"`            // 0=未支付 1=已支付 2=已退款（对齐 epay）
}

// TableName 自研表名。
func (SubOrder) TableName() string { return "pay_suborder" }
