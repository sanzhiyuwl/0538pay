package model

import "time"

// 微信支付「消费者投诉2.0」（自研扩展，epay 无此功能；挂服务商进件线，复用 sub_mchid + 服务商证书/APIv3密钥）。
//
// 角色：普通服务商（境内），V3。服务商用 sp_mchid 签名调用，业务参数带 complainted_mchid（被诉子商户号）。
// 微信回调只会打到服务商这一个回调地址，数据必然「平台先收 → 按 complainted_mchid 映射到本地商户 → 分发」。
// 本地一套表 pay_wx_complaint（主表，按 complaint_id 唯一 Upsert）+ pay_wx_complaint_notify（回调流水，按通知ID幂等）。

// 投诉单状态 complaint_state（源：服务商版错误码 268440133 排查段）。状态机 PENDING → PROCESSING → PROCESSED。
const (
	WxComplaintStatePending    = "PENDING"    // 待处理
	WxComplaintStateProcessing = "PROCESSING" // 处理中
	WxComplaintStateProcessed  = "PROCESSED"  // 已处理完成（终态；PROCESSED 后禁 complete，需幂等当成功）
)

// 回调通知类型 event_type（外层信封）。
const (
	WxComplaintEventCreate      = "COMPLAINT.CREATE"       // 产生新投诉
	WxComplaintEventStateChange = "COMPLAINT.STATE_CHANGE" // 投诉状态变化
)

// 回调解密后 action_type（触发本次回调的具体动作，13 种，源：服务商版投诉通知回调 4012076174）。
const (
	WxComplaintActionCreate                 = "CREATE_COMPLAINT"              // 用户提交投诉
	WxComplaintActionContinue               = "CONTINUE_COMPLAINT"           // 用户继续投诉
	WxComplaintActionUserResponse           = "USER_RESPONSE"                // 用户新留言
	WxComplaintActionResponseByPlatform     = "RESPONSE_BY_PLATFORM"         // 平台新留言
	WxComplaintActionSellerRefund           = "SELLER_REFUND"                // 商户发起全额退款
	WxComplaintActionMerchantResponse       = "MERCHANT_RESPONSE"            // 商户新回复
	WxComplaintActionMerchantConfirm        = "MERCHANT_CONFIRM_COMPLETE"    // 商户反馈处理完成
	WxComplaintActionUserApplyService       = "USER_APPLY_PLATFORM_SERVICE"  // 用户申请平台协助
	WxComplaintActionUserCancelService      = "USER_CANCEL_PLATFORM_SERVICE" // 用户取消平台协助
	WxComplaintActionPlatformServiceFinish  = "PLATFORM_SERVICE_FINISHED"    // 客服结束平台协助
	WxComplaintActionMerchantApproveRefund  = "MERCHANT_APPROVE_REFUND"      // 商户同意退款
	WxComplaintActionMerchantRejectRefund   = "MERCHANT_REJECT_REFUND"       // 商户驳回退款
	WxComplaintActionRefundSuccess          = "REFUND_SUCCESS"               // 退款到账
)

// WxComplaint 消费者投诉单主表 pay_wx_complaint（自研扩展）。一行 = 一个微信投诉单（complaint_id 唯一）。
//
// 数据来源：回调只带 {complaint_id, action_type}，明细一律以「查询投诉单详情」为准回填（★官方铁律：
// 回调不能作唯一数据源，必须配主动查询 + 兜底轮询）。complainted_mchid 反查本服务商名下已开通进件单
// 回填 MerchantID/EnrollID，未匹配到（非本平台名下）也照落，但后台代处理时按越权拦截拒绝。
//
// 敏感字段：payer_phone 微信用平台证书 RSA-OAEP 加密下发，PayerPhoneEnc 存密文（只有平台私钥能解），
// PayerPhoneMask 存脱敏明文供回显（对齐进件线敏感字段规范：加密落库 + 回显脱敏）。
type WxComplaint struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ComplaintID string `gorm:"column:complaint_id;size:64;uniqueIndex;not null" json:"complaint_id"` // 微信投诉单号（唯一幂等键）

	// 归属（complainted_mchid = 被诉子商户号；反查本地商户/进件单）。
	ComplaintedMchID string `gorm:"column:complainted_mchid;size:64;index;not null" json:"complainted_mchid"`
	MerchantID       uint   `gorm:"column:merchant_id;index" json:"merchant_id"` // 本地商户 uid（反查回填，0=未匹配到本平台名下）
	EnrollID         uint   `gorm:"column:enroll_id;index" json:"enroll_id"`     // 命中的进件单 id（0=未匹配）
	MerchantName     string `gorm:"size:128" json:"merchant_name"`               // 商户名称（反查回填，便于列表展示）

	// 投诉核心
	ComplaintState  string `gorm:"column:complaint_state;size:16;index;not null" json:"complaint_state"` // PENDING/PROCESSING/PROCESSED
	ComplaintTime   string `gorm:"column:complaint_time;size:40" json:"complaint_time"`                  // 投诉时间（rfc3339 原值）
	ComplaintDetail string `gorm:"column:complaint_detail;type:text" json:"complaint_detail"`            // 投诉内容
	ProblemType     string `gorm:"column:problem_type;size:32" json:"problem_type"`                      // REFUND/SERVICE_NOT_WORK/OTHERS
	ProblemDesc     string `gorm:"column:problem_description;type:text" json:"problem_description"`       // 问题描述

	// 关联订单 / 退款
	ApplyRefundAmount     int    `gorm:"column:apply_refund_amount" json:"apply_refund_amount"`     // 用户申请退款金额（分）
	ComplaintFullRefunded bool   `gorm:"column:complaint_full_refunded" json:"complaint_full_refunded"` // 是否已全额退款
	ComplaintOrderInfo    string `gorm:"column:complaint_order_info;type:text" json:"complaint_order_info"` // 关联订单列表 JSON

	// 投诉人（敏感）
	PayerPhoneEnc  string `gorm:"column:payer_phone_enc;type:text" json:"-"`          // 手机号密文（平台私钥可解，不回原文）
	PayerPhoneMask string `gorm:"column:payer_phone_mask;size:32" json:"payer_phone"` // 手机号脱敏（回显用）
	PayerOpenID    string `gorm:"column:payer_openid;size:128" json:"payer_openid"`

	// 协商/服务标记
	UserComplaintTimes   int    `gorm:"column:user_complaint_times" json:"user_complaint_times"`       // 用户投诉次数
	IncomingUserResponse bool   `gorm:"column:incoming_user_response" json:"incoming_user_response"`   // 是否有用户新留言待回复
	InPlatformService    bool   `gorm:"column:in_platform_service" json:"in_platform_service"`         // 是否处于平台协助中
	NeedImmediateService bool   `gorm:"column:need_immediate_service" json:"need_immediate_service"`   // 是否需即时服务
	ComplaintMediaList   string `gorm:"column:complaint_media_list;type:text" json:"complaint_media_list"` // 投诉资料图片列表 JSON
	UserTagList          string `gorm:"column:user_tag_list;type:text" json:"user_tag_list"`               // 用户标签列表 JSON

	// 回调足迹
	LastActionType string `gorm:"column:last_action_type;size:48" json:"last_action_type"` // 最近一次回调 action_type
	LastEventType  string `gorm:"column:last_event_type;size:48" json:"last_event_type"`   // 最近一次回调 event_type

	RawJSON   string    `gorm:"column:raw_json;type:text" json:"-"` // 查询详情原始应答（审计留痕）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"` // 最近更新（列表按此倒序，新投诉/新动作在上）
}

func (WxComplaint) TableName() string { return "pay_wx_complaint" }

// WxComplaintNotify 投诉回调流水表 pay_wx_complaint_notify（追加型）。每条微信投诉回调落一行。
// 幂等：NotifyID（回调 envelope 顶层 id，同一通知重推 id 不变）唯一索引去重，重复回调直接应答成功。
// 用途：幂等去重 + 该投诉单的回调动作时间线（详情抽屉展示「谁在什么时候做了什么」）。
type WxComplaintNotify struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	NotifyID    string `gorm:"column:notify_id;size:64;uniqueIndex;not null" json:"notify_id"` // 回调唯一ID（幂等键）
	ComplaintID string `gorm:"column:complaint_id;size:64;index;not null" json:"complaint_id"`
	EventType   string `gorm:"column:event_type;size:48" json:"event_type"`   // COMPLAINT.CREATE / COMPLAINT.STATE_CHANGE
	ActionType  string `gorm:"column:action_type;size:48" json:"action_type"` // 解密后动作类型
	Summary     string `gorm:"size:128" json:"summary"`
	RawJSON     string `gorm:"column:raw_json;type:text" json:"-"` // 解密后业务报文原文
	CreatedAt   time.Time `json:"created_at"`                      // 落库时间（= 时间线排序键）
}

func (WxComplaintNotify) TableName() string { return "pay_wx_complaint_notify" }

// WxComplaintStateText 投诉单状态中文名（后台 Badge 展示）。
func WxComplaintStateText(s string) string {
	switch s {
	case WxComplaintStatePending:
		return "待处理"
	case WxComplaintStateProcessing:
		return "处理中"
	case WxComplaintStateProcessed:
		return "已完成"
	default:
		return s
	}
}
