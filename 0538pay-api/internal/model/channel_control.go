package model

import "time"

// 子商户管控状态（风控第二段：进件成功后按 sub_mchid 查微信「商户被管控能力及原因」的落地快照）。
// 一行 = 某已开通子商户的最新管控快照，随每次刷新覆盖更新。
//   normal    正常     —— 查询无任何被管控能力
//   controlled 被管控   —— 命中被管控能力且已实际生效（立即管控，或延迟管控已到点）
//   delayed   延迟管控 —— 命中延迟管控且尚未到点（预计将被管控，可提前引导解脱）
const (
	ChannelControlNormal     = "normal"
	ChannelControlControlled = "controlled"
	ChannelControlDelayed    = "delayed"
)

// ChannelControl 子商户管控快照表 pay_channel_control（自研，服务商进件风控子能力）。
// 数据源：channel_enroll(status=approved) 的 sub_mchid；由 admin/risk-controls 单个/批量刷新触发查询后落库。
// 硬锁读取本表快照（收单/退款/提现/分账拦截），不每次现查微信（避免 429 限频、下单链路阻塞）。
type ChannelControl struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	EnrollID  uint   `gorm:"column:enroll_id;uniqueIndex;not null" json:"enroll_id"` // 关联进件单（一进件单一快照）
	UID       uint   `gorm:"index;not null" json:"uid"`                             // 归属商户
	ChannelID int    `gorm:"column:channel_id;index" json:"channel_id"`             // 归属服务商主通道
	SubMchID  string `gorm:"column:sub_mchid;size:64;index;not null" json:"sub_mchid"`

	// 管控快照（源自微信 4012803072 应答）
	State                  string `gorm:"size:16;not null;default:'normal';index" json:"state"`     // normal/controlled/delayed
	LimitedFunctions       string `gorm:"column:limited_functions;type:text" json:"-"`             // 顶层被管控能力枚举 JSON([]string)
	OtherLimitedFunctions  string `gorm:"column:other_limited_functions;type:text" json:"-"`      // 枚举外能力（自由文本）
	RecoverySpecifications string `gorm:"column:recovery_specifications;type:text" json:"-"`      // 原因+解脱路径列表 JSON
	RawJSON                string `gorm:"column:raw_json;type:text" json:"-"`                     // 微信应答原文（审计留痕）

	LastQueryAt *time.Time `gorm:"column:last_query_at" json:"-"` // 最近一次刷新时间
	LastError   string     `gorm:"column:last_error;size:512" json:"-"` // 最近一次刷新失败原因（成功时清空）

	// 解脱路径主动代办留痕（0804 方案补遗：recover_way=修改主体资料/修改结算账户时，服务商直接调 API 代办）。
	// ★同一商户同一时间只能有一笔流程中的申请单（微信语义），此处只留「最近一次」，与快照表逐商户覆盖式设计一致。
	LastSettleApplyNo string     `gorm:"column:last_settle_apply_no;size:64" json:"-"` // 最近一次改结算账户申请单号（application_no）
	LastSettleApplyAt *time.Time `gorm:"column:last_settle_apply_at" json:"-"`
	LastSubjectApplyNo string     `gorm:"column:last_subject_apply_no;size:64" json:"-"` // 最近一次改主体资料申请单号（apply_id）
	LastSubjectApplyAt *time.Time `gorm:"column:last_subject_apply_at" json:"-"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (ChannelControl) TableName() string { return "pay_channel_control" }
