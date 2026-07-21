package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// PayRecord 资金明细流水。每条 = 商户余额的一次变更。
// 对齐前端 mock/records.ts 的 FundRecord + epay pre_record 语义。
// 自研表名 pay_record，不复制 epay 的 pre_record。
type PayRecord struct {
	ID       uint            `gorm:"primaryKey" json:"id"`
	UID      uint            `gorm:"index;not null" json:"uid"`                        // 商户号
	Action   int8            `gorm:"not null" json:"action"`                           // 1=增加 2=减少
	Money    decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"money"`         // 变更金额
	OldMoney decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"oldmoney"`      // 变更前余额
	NewMoney decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"newmoney"`      // 变更后余额
	Type     string          `gorm:"size:32" json:"type"`                              // 操作类型文案
	TradeNo  string          `gorm:"size:32;index" json:"trade_no"`                    // 关联订单号（可空）
	Date     time.Time       `gorm:"index" json:"date"`                                // 时间
}

func (PayRecord) TableName() string { return "pay_record" }
