package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// SettleRecord 结算明细。一行 = 一笔商户结算/提现单。
// 对齐 epay pre_settle + 前端 mock/settle.ts 的 SettleRecord（json tag 一致）。
// 自研表名 pay_settle，不复制 epay 的 pre_settle。
//
// 资金语义：生成结算单时从商户余额扣减 Money（出账，pay_record type=自动结算）；
// 单条删除(status=4)时把 Money 退回商户余额（type=结算失败退回）。RealMoney=Money-手续费。
// 真实打款(transfer_*)不在本域，归代付域(C3)。
type SettleRecord struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	UID       uint            `gorm:"index;not null" json:"uid"`                        // 商户号
	Batch     string          `gorm:"size:20;index" json:"batch"`                       // 所属批次号（手动申请为空）
	Auto      int8            `gorm:"not null;default:1" json:"auto"`                   // 1=自动结算 0=手动申请
	Type      int8            `gorm:"not null;default:1" json:"type"`                   // 结算方式 1支付宝2微信3QQ4银行卡5支付机构
	Account   string          `gorm:"size:128" json:"account"`                          // 结算账号（下单时从商户快照）
	Username  string          `gorm:"size:128" json:"username"`                         // 收款人姓名（快照）
	Money     decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"-"`             // 结算金额（从余额扣减额，原始）
	RealMoney decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"-"`             // 实际到账（Money-手续费，原始）
	AddTime   time.Time       `gorm:"index" json:"-"`                                   // 创建时间（原始）
	EndTime   *time.Time      `json:"-"`                                                // 完成时间（status=1 时写入，原始）
	Status    int8            `gorm:"not null;default:0;index" json:"status"`           // 0待结算 1已完成 2正在结算 3结算失败
	Result    string          `gorm:"size:255" json:"result"`                           // 结算失败原因
}

func (SettleRecord) TableName() string { return "pay_settle" }

// SettleBatch 结算批次。一行 = 一次"生成结算批次"操作，汇总一批待结算记录。
// 对齐 epay pre_batch + 前端 mock/settle.ts 的 SettleBatch。自研表名 pay_batch。
type SettleBatch struct {
	Batch    string          `gorm:"primaryKey;size:20" json:"batch"`      // 批次号
	AllMoney decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"-"` // 批次总金额（各单 RealMoney 累加，原始）
	Count    int             `gorm:"not null;default:0" json:"count"`      // 批次内记录数
	Time     time.Time       `json:"-"`                                    // 生成时间（原始）
	Status   int8            `gorm:"not null;default:0" json:"status"`     // 0处理中 1已完成 2部分完成
}

func (SettleBatch) TableName() string { return "pay_batch" }
