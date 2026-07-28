package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// 进件申请状态机（对齐 docs-代理进件/01 第五节）。本地 status 取值：
//   pending_pay 待支付   —— 已建单未付款，创建即进入；30 分钟未付自动关单
//   paid        已支付待完善 —— 付款回调确认到账，放行填全套资料
//   submitted   已提交/审核中 —— 提交微信后（映射微信申请单状态：待账户验证/待签约/开通权限中）
//   finished    已完成   —— 拿到 sub_mchid，触发扣名额(路径一)或结算(路径二)；终态且不可退
//   rejected    已驳回   —— 可改资料复用 business_code 重提，或走退款
//   closed      已关单   —— 待支付超时关单（终态事件）
//   refunded    已退款   —— 已支付后放弃或驳回退款（终态事件）
const (
	EnrollStatusPendingPay = "pending_pay"
	EnrollStatusPaid       = "paid"
	EnrollStatusSubmitted  = "submitted"
	EnrollStatusFinished   = "finished"
	EnrollStatusRejected   = "rejected"
	EnrollStatusClosed     = "closed"
	EnrollStatusRefunded   = "refunded"
)

// 进件发起方式（source）。
const (
	EnrollSourcePlatform = 1 // 平台代填
	EnrollSourceAgent    = 2 // 代理代填
	EnrollSourceSelf     = 3 // 客户自助（邀请链接）
)

// 资金路径（path）。
const (
	EnrollPathQuota = 1 // 路径一：代理预购名额，进件成功扣 1 名额
	EnrollPathSelf  = 2 // 路径二：商户自付，进件成功后分账（平台一份/代理一份）
)

// SubMerchantEnroll 特约商户进件申请。一行 = 一笔进件单。
// 自研表 pay_submch_enroll。敏感字段（身份证号/银行账号/手机号）经 RSA-OAEP 加密后存密文，不明文落库。
type SubMerchantEnroll struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	EnrollNo string `gorm:"column:enroll_no;size:32;uniqueIndex;not null" json:"enroll_no"` // 进件单号（幂等键，业务唯一）
	AgentID  uint   `gorm:"index;not null" json:"agent_id"`                                // 归属代理 id（0=平台自己发起）

	// 主体资料（对齐微信进件接口字段，含敏感字段密文）
	MerchantName string `gorm:"size:128;not null" json:"merchant_name"`  // 商户名称/主体名称
	SubjectType  string `gorm:"size:32" json:"subject_type"`             // 主体类型（个体/企业/事业单位等）
	ContactPhone string `gorm:"size:20;index" json:"contact_phone"`      // 联系手机（明文，用于进度查询匹配）
	MaterialJSON string `gorm:"column:material_json;type:text" json:"-"` // 全套资料 JSON（敏感字段已加密），提交微信前暂存

	// 资金与路径
	Path         int             `gorm:"not null;default:2" json:"path"`                     // 1 预购名额 / 2 商户自付
	RetailAmount decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"-"`     // 商户付的开户零售价（原始）
	PayOrderNo   string          `gorm:"column:pay_order_no;size:64;index" json:"pay_order_no"` // 开户费/名额批发的收款单号

	// 微信侧
	BusinessCode string `gorm:"column:business_code;size:64;index" json:"business_code"` // 服务商自定义业务申请编号（驳回可复用覆盖）
	WxApplymentID string `gorm:"column:wx_applyment_id;size:64;index" json:"wx_applyment_id"` // 微信申请单号
	WxSubMchID   string `gorm:"column:wx_sub_mchid;size:32;index" json:"wx_sub_mchid"`   // 成功后的 sub_mchid（非空=已交付，硬锁不可退）
	WxState      string `gorm:"column:wx_state;size:64" json:"wx_state"`                 // 微信申请单最新状态原值

	Status     string     `gorm:"size:16;not null;default:'pending_pay';index" json:"status"` // 本地状态机
	RejectReason string   `gorm:"column:reject_reason;size:512" json:"reject_reason"`         // 驳回原因
	Source     int        `gorm:"not null;default:1" json:"source"`                          // 1平台代填 2代理代填 3客户自助
	InviteCode string     `gorm:"column:invite_code;size:32;index" json:"invite_code"`       // 自助进件来源邀请码（回填便于追溯）

	AddTime    time.Time  `gorm:"index" json:"-"` // 创建时间（原始）
	SubmitTime *time.Time `json:"-"`              // 提交微信时间（原始）
	FinishTime *time.Time `json:"-"`              // 完成时间（拿到 sub_mchid，原始）
}

func (SubMerchantEnroll) TableName() string { return "pay_submch_enroll" }

// 邀请链接状态。
const (
	InviteStatusEnabled  = 1 // 启用
	InviteStatusDisabled = 0 // 停用
	InviteStatusExpired  = 2 // 已失效（终态事件 24h 后定时任务置）
)

// EnrollInvite 进件邀请链接/二维码。客户自助进件入口，code 绑定 agent_id。
// 自研表 pay_enroll_invite。二维码由前端按 code 对应公开页 URL 生成，不单独存图。
//
// 有效期锚定"终态事件"（不从打开死算）：ExpireAt 为空=不倒计时仍有效；
// 出现终态事件（待支付超时关单 / 已支付后驳回 / 退款完成）时 service 置 ExpireAt=事件时刻+24h（可配）；
// 定时任务扫过期置 status=expired。客户无法自助复活，重开须代理/平台重新生成。
type EnrollInvite struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Code          string     `gorm:"size:32;uniqueIndex;not null" json:"code"`        // URL 里用的唯一短码
	AgentID       uint       `gorm:"index;not null" json:"agent_id"`                  // 归属代理 id（0=平台自己发）
	Name          string     `gorm:"size:64" json:"name"`                             // 备注（便于区分给了哪个客户）
	Status        int8       `gorm:"not null;default:1;index" json:"status"`          // 1启用 0停用 2已失效
	FirstAccessAt *time.Time `gorm:"column:first_access_at" json:"-"`                 // 首次打开时间（为空=还没人点）
	ExpireAt      *time.Time `gorm:"column:expire_at;index" json:"-"`                 // 失效时间（为空=不倒计时仍有效）
	OpenCount     int        `gorm:"not null;default:0" json:"open_count"`            // 累计打开数
	SubmitCount   int        `gorm:"not null;default:0" json:"submit_count"`          // 累计提交数
	AddTime       time.Time  `gorm:"index" json:"-"`                                  // 创建时间（原始）
}

func (EnrollInvite) TableName() string { return "pay_enroll_invite" }

// EnrollSettleLog 进件结算流水（路径二 200 分账 / 路径一全额过账给代理）。
// 一行 = 一笔进件成功后的佣金结算。自研表 pay_enroll_settle_log。
type EnrollSettleLog struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	EnrollID       uint            `gorm:"index;not null" json:"enroll_id"`                 // 关联进件单 id
	AgentID        uint            `gorm:"index;not null" json:"agent_id"`                  // 代理 id
	AgentAmount    decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"-"`  // 代理所得（原始）
	PlatformAmount decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"-"`  // 平台所得（原始）
	Path           int             `gorm:"not null;default:2" json:"path"`                  // 结算对应的资金路径
	PayOrderNo     string          `gorm:"column:pay_order_no;size:64;index" json:"pay_order_no"` // 关联收款单号
	SettleTime     time.Time       `gorm:"index" json:"-"`                                  // 结算时间（原始）
}

func (EnrollSettleLog) TableName() string { return "pay_enroll_settle_log" }
