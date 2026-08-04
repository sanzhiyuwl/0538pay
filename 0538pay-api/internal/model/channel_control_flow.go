package model

import "time"

// 管控流水机制（风控第三段：处置/管控订阅回调）。两套独立机制各自建模，落同一张追加型流水表。
//   violation       —— (A) 商户平台处置通知（service_notify_url，event_type=VIOLATION.*）
//   merchant_notify —— (B) 合作伙伴订阅·商户新增管控流水通知（topic 20000，event_type=MERCHANT_NOTIFY.NOTIFY）
const (
	ChannelControlMechViolation = "violation"
	ChannelControlMechMerchant  = "merchant_notify"
)

// 管控流水业务状态（B 机制 business_state，源：产品介绍 4012165270，已核实）。
const (
	ChannelFlowStatePunishment          = "PUNISHMENT"          // 特约商户有一条新的管控流水
	ChannelFlowStateRecovery            = "RECOVERY"            // 特约商户有一条新的管控解除流水
	ChannelFlowStateDelayPunishment     = "DELAYPUNISHMENT"     // 即将被管控（延迟管控提前通知）
	ChannelFlowStatePunishmentCancelled = "PUNISHMENTCANCELLED" // 即将被管控的单据已撤销
)

// ChannelControlFlow 子商户管控流水表 pay_channel_control_flow（自研，服务商进件风控第三段）。
// 追加型（append-only）：每条微信处置/管控订阅回调落一行，随时间累积成时间线，不覆盖。
// 与 pay_channel_control（按 enroll_id 唯一的最新快照）分工：本表记「变化过程」，快照记「当前态」。
//
// 幂等：NotifyID（回调 envelope 顶层 id，同一通知重推 id 不变）唯一索引去重，重复回调直接放行应答 200。
// 关联：SubMchID 反查 channel_enroll(approved) → 回填 EnrollID/UID；回调落库后异步触发第二段查询接口
//       核实明细并刷新快照（★官方要求不能只依赖回调，回调 + 查询 + 兜底轮询三者并用）。
type ChannelControlFlow struct {
	ID uint `gorm:"primaryKey" json:"id"`

	NotifyID  string `gorm:"column:notify_id;size:64;uniqueIndex;not null" json:"notify_id"` // 回调唯一ID（幂等键）
	Mechanism string `gorm:"size:16;index;not null" json:"mechanism"`                       // violation / merchant_notify
	EventType string `gorm:"column:event_type;size:48;not null" json:"event_type"`          // VIOLATION.* / MERCHANT_NOTIFY.NOTIFY
	Summary   string `gorm:"size:128" json:"summary"`                                       // 回调摘要

	// 关联主体（A=resource.sub_mchid，B=message_content.merchant_code；均为子商户号）。
	SubMchID string `gorm:"column:sub_mchid;size:64;index;not null" json:"sub_mchid"`
	EnrollID uint   `gorm:"column:enroll_id;index" json:"enroll_id"` // 反查命中的进件单（0=未匹配到本服务商名下）
	UID      uint   `gorm:"index" json:"uid"`                        // 归属商户（反查回填）

	// —— (A) 商户平台处置通知解密字段（4012079216）——
	RecordID          string `gorm:"column:record_id;size:128;index" json:"record_id"` // 处置通知唯一标识（A 业务去重键）
	CompanyName       string `gorm:"column:company_name;size:128" json:"company_name"` // 子商户公司名称（A/B 共用，B 为 merchant_company_name）
	PunishPlan        string `gorm:"column:punish_plan;type:text" json:"punish_plan"`  // 处罚方案文本
	PunishTime        string `gorm:"column:punish_time;size:64" json:"punish_time"`
	PunishDescription string `gorm:"column:punish_description;size:256" json:"punish_description"`
	RiskType          string `gorm:"column:risk_type;size:128" json:"risk_type"`        // 风险类型枚举
	RiskDescription   string `gorm:"column:risk_description;type:text" json:"risk_description"`

	// —— (B) 合作伙伴订阅·管控流水解密字段（4016022266 / 4012165270）——
	BusinessCode  string `gorm:"column:business_code;size:128;index" json:"business_code"`   // ↔ 第二段 limitation_case_id
	BusinessState string `gorm:"column:business_state;size:32" json:"business_state"`        // PUNISHMENT/RECOVERY/DELAYPUNISHMENT/PUNISHMENTCANCELLED
	BusinessTime  string `gorm:"column:business_time;size:64" json:"business_time"`

	RawJSON   string    `gorm:"column:raw_json;type:text" json:"-"` // 解密后业务报文原文（审计留痕）
	CreatedAt time.Time `json:"created_at"`                          // 落库时间（= 时间线排序键）
}

func (ChannelControlFlow) TableName() string { return "pay_channel_control_flow" }
