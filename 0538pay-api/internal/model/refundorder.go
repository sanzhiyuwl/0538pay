package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// RefundOrder 退款单。对齐 epay pre_refundorder：记录每次退款申请，支持 out_refund_no 幂等。
// 自研表名 pay_refundorder。
type RefundOrder struct {
	ID          uint            `gorm:"primaryKey" json:"-"`
	RefundNo    string          `gorm:"size:32;uniqueIndex;not null" json:"refund_no"`  // 系统退款单号
	OutRefundNo string          `gorm:"size:64;index" json:"out_refund_no"`             // 商户退款单号（长度>5 时启用幂等）
	TradeNo     string          `gorm:"size:32;index;not null" json:"trade_no"`         // 系统订单号
	OutTradeNo  string          `gorm:"size:64" json:"out_trade_no"`                    // 商户订单号
	UID         uint            `gorm:"index;not null" json:"uid"`                      // 商户号
	Money       decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"money"`       // 退款金额
	ReduceMoney decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"reduce_money"` // 扣商户余额金额
	Status      int8            `gorm:"not null;default:0;index" json:"status"`         // 0处理中 1成功 2失败
	AddTime     time.Time       `json:"-"`                                              // 申请时间
	EndTime     *time.Time      `json:"-"`                                              // 完成时间
}

func (RefundOrder) TableName() string { return "pay_refundorder" }
