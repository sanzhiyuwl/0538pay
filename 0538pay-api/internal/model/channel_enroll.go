package model

import "time"

// 服务商通道商户进件申请状态机（epay 精仿线，对齐 docs/0730；阶段2 全自动 applyment4sub）。
// 本平台只走商户进件不走二清：每个商户在平台服务商名下进件，拿到自己的子商户号后才能收款。
//   draft     草稿   —— 商户建单未提交，可继续补资料
//   submitted 审核中 —— 商户资料已通过 applyment4sub 直提交微信，等微信审核（wx_state 记微信侧细状态）
//   approved  已开通 —— 微信审核完成(APPLYMENT_STATE_FINISHED)返回 sub_mchid，系统已建/更新该商户
//                       子通道并把子商户号写进占位符 key、置 status=1，该商户此通道开通（非二清落地）
//   rejected  已驳回 —— 微信驳回(APPLYMENT_STATE_REJECTED)，audit_detail 记逐字段原因，商户改料重提（复用同一单）
//
// ★ pending 为旧半自动阶段「人工审核」态，全自动化后不再产生；保留常量仅为兼容历史存量单读取显示。
const (
	ChannelEnrollDraft     = "draft"
	ChannelEnrollPending   = "pending" // 兼容历史存量（旧人工审核态），新流程不再产生
	ChannelEnrollSubmitted = "submitted"
	ChannelEnrollApproved  = "approved"
	ChannelEnrollRejected  = "rejected"
)

// ChannelEnroll 服务商通道商户进件单。一行 = 某商户在某服务商主通道下的一笔进件。
// 自研表 pay_channel_enroll（epay 进件业务本体闭源未开放，本表为精仿线自研；
// 与代理进件线的 pay_submch_enroll 独立，两条线互不混写）。
//
// 敏感字段（身份证号/银行账号）经 RSA 加密后存入 MaterialJSON 密文，不明文落库；
// 审核通过时把子商户号按【该渠道的子商户标识字段名】写入 pay_subchannel.info 对应 key
// （富友写 appmchid、微信服务商写 sub_appid/sub_mchid…，见 0730 4.3 对照表），
// 并置子通道 status=1、回写本单 SubChannelID（即 epay 的 apply_id 语义）。
type ChannelEnroll struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	EnrollNo  string `gorm:"column:enroll_no;size:32;uniqueIndex;not null" json:"enroll_no"` // 进件单号（业务唯一幂等键）
	UID       uint   `gorm:"index;not null" json:"uid"`                                     // 归属商户
	ChannelID int    `gorm:"column:channel_id;index;not null" json:"channel_id"`            // 归属服务商主通道 pay_channel.id
	Plugin    string `gorm:"size:32;not null" json:"plugin"`                                // 主通道插件 key（决定要采集哪些资料，见 0730 4.3）

	// 进件资料（对齐微信 applyment4sub / 富友二级商户进件字段）
	MerchantName string `gorm:"size:128;not null" json:"merchant_name"`  // 商户/主体名称
	SubjectType  string `gorm:"size:32" json:"subject_type"`             // 主体类型（个体户/企业/事业单位…）
	ContactPhone string `gorm:"size:20;index" json:"contact_phone"`      // 联系手机（明文，进度匹配用）
	MaterialJSON string `gorm:"column:material_json;type:text" json:"-"` // 全套进件资料 JSON（敏感字段密文），审核用
	MaterialMeta string `gorm:"column:material_meta;type:text" json:"-"` // 非敏感字段明文快照（后台回显/商户回填编辑，绝不含身份证/银行账号原文）

	// 微信 applyment4sub 直提交对接字段（全自动化，阶段2）
	BusinessCode  string `gorm:"column:business_code;size:64;index" json:"business_code"`   // 业务申请编号（提交微信幂等键，驳回重提复用）
	WxApplymentID string `gorm:"column:wx_applyment_id;size:64" json:"wx_applyment_id"`     // 微信返回的申请单号 applyment_id
	WxState       string `gorm:"column:wx_state;size:64" json:"wx_state"`                   // 微信侧申请单状态原值（APPLYMENT_STATE_xxx）
	SignURL       string `gorm:"column:sign_url;type:text" json:"sign_url"`                 // 超级管理员签约链接（微信在待签约阶段返回）
	AuditDetail   string `gorm:"column:audit_detail;type:text" json:"-"`                   // 驳回逐字段详情 JSON（精准补料用）

	// 审核与交付
	Status       string `gorm:"size:16;not null;default:'draft';index" json:"status"` // 本地状态机
	SubMchID     string `gorm:"column:sub_mchid;size:64;index" json:"sub_mchid"`      // 微信开出的子商户号（非空=已开通交付）
	SubChannelID uint   `gorm:"column:subchannel_id;index" json:"subchannel_id"`      // 交付后建/更新的子通道 id（epay apply_id 语义）
	RejectReason string `gorm:"column:reject_reason;size:512" json:"reject_reason"`   // 驳回原因（驳回时有）
	AuditAdmin   string `gorm:"column:audit_admin;size:64" json:"audit_admin"`        // 审核操作人（后台用户名，全自动阶段可空）

	AddTime    time.Time  `gorm:"index" json:"-"` // 建单时间
	SubmitTime *time.Time `json:"-"`              // 提交微信时间
	AuditTime  *time.Time `json:"-"`              // 微信审核完成时间（开通/驳回）
}

func (ChannelEnroll) TableName() string { return "pay_channel_enroll" }
