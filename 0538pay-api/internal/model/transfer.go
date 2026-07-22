package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transfer 代付/转账记录。一行 = 一笔对外打款（后台主动发起 或 商户发起）。
// 对齐 epay pre_transfer + 前端 mock/transfer.ts。自研表名 pay_transfer，不复制 epay 表。
//
// 主键为 BizNo（19 位交易号），既是业务号也是幂等键：同一交易号只能落一次（对齐 epay）。
// 资金语义（对齐 epay lib/Transfer）：
//   - 商户发起(UID>0)：落库时即时从余额扣减 CostMoney（含手续费），失败/退回时退回 CostMoney。
//   - 后台发起(UID=0)：CostMoney=Money，不收手续费、不扣款、不涉及退款。
//   - 退款只在 status=0(处理中) 时执行，用条件 UPDATE 防重复退款。
//   - Money=到账金额（打给收款方），CostMoney=商户实际被扣（Money+手续费）。
type Transfer struct {
	BizNo      string          `gorm:"primaryKey;column:biz_no;size:19" json:"biz_no"`   // 交易号（19位数字，幂等键）
	PayOrderNo string          `gorm:"size:80" json:"pay_order_no"`                      // 第三方转账单号（渠道返回）
	UID        uint            `gorm:"index;not null" json:"uid"`                        // 商户号（0=后台管理员发起）
	Type       string          `gorm:"size:10;not null" json:"type"`                     // 付款方式 alipay/wxpay/qqpay/bank
	Channel    int             `gorm:"not null;default:0" json:"channel"`                // 使用的支付通道 id
	Account    string          `gorm:"size:128;not null" json:"account"`                 // 收款账号
	Username   string          `gorm:"size:128" json:"username"`                         // 收款人姓名（选填，留空不校验）
	Money      decimal.Decimal `gorm:"type:decimal(18,4);not null" json:"-"`             // 到账金额（原始）
	CostMoney  decimal.Decimal `gorm:"type:decimal(18,4)" json:"-"`                      // 商户扣款=到账+手续费（原始）
	AddTime    time.Time       `gorm:"index" json:"-"`                                   // 提交时间（原始）
	PayTime    *time.Time      `json:"-"`                                                // 付款成功时间（status=1 时写入，原始）
	Status     int8            `gorm:"not null;default:0;index" json:"status"`           // 0处理中 1成功 2失败
	Desc       string          `gorm:"size:80" json:"desc"`                              // 转账备注（≤32字）
	Result     string          `gorm:"size:255" json:"result"`                           // 结果消息（失败原因/撤销说明）
	Ext        string          `gorm:"type:text" json:"-"`                               // 扩展（微信确认收款 jumpurl 等）
	API        int8            `gorm:"column:api;not null;default:0" json:"-"`           // 发起来源 0后台 1API（E-6，对齐 epay pre_transfer.api）
}

func (Transfer) TableName() string { return "pay_transfer" }
