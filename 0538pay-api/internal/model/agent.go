package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Agent 代理。代理进件平台的分销层：招特约商户、发起进件、赚佣金。
// 自研扩展表 pay_agent，epay 无对应。独立代理端 /agent 登录态按 UID 隔离数据。
//
// 权限体系（2026-07-28）：代理能用哪些功能由 Permissions 决定——权限开通啥代理就有啥。
// Permissions 存已开通权限点 key 的逗号分隔串（如 "enroll,quota,invite,refund,settlement"），
// 内置 enroll(进件代理)/acquire(收单代理占位) 收敛进来，替代早前零散的 can_enroll/can_acquire 布尔字段。
// 权限点清单常量见 service 层 AgentPermissionCatalog；新增功能只需追加一个权限点 key，本表结构不改。
type Agent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`                 // 代理名称
	Account     string    `gorm:"size:64;uniqueIndex;not null" json:"account"`  // 登录账号（唯一）
	Password    string    `gorm:"size:100;not null" json:"-"`                   // 登录密码 bcrypt
	Contact     string    `gorm:"size:64" json:"contact"`                       // 联系方式
	Status      int8      `gorm:"not null;default:1;index" json:"status"`       // 1启用 0停用
	Permissions string    `gorm:"size:255" json:"permissions"`                  // 已开通权限点 key 逗号分隔串
	Remark      string    `gorm:"size:255" json:"remark"`                       // 备注
	AddTime     time.Time `gorm:"index" json:"-"`                               // 创建时间（原始）
}

func (Agent) TableName() string { return "pay_agent" }

// AgentQuotaWallet 名额钱包（路径一：代理预购名额）。一个代理一行。
// 余额变动一律走 AgentQuotaLog 流水，不裸改（对齐项目"充扣走流水"约定）。
type AgentQuotaWallet struct {
	AgentID   uint      `gorm:"primaryKey" json:"agent_id"`               // 代理 id（主键，一代理一钱包）
	Balance   int       `gorm:"not null;default:0" json:"balance"`        // 当前可用名额数（可再建路径一进件单的额度）
	Frozen    int       `gorm:"not null;default:0" json:"frozen"`         // 冻结中名额（路径一建单预占，进件成功转消耗/失败释放回可用）
	TotalBuy  int       `gorm:"not null;default:0" json:"total_buy"`      // 累计购买名额
	TotalUsed int       `gorm:"not null;default:0" json:"total_used"`     // 累计消耗名额
	UpdatedAt time.Time `json:"-"`                                        // 更新时间（原始）
}

func (AgentQuotaWallet) TableName() string { return "pay_agent_quota" }

// AgentQuotaLog 名额流水。一行 = 一次名额变动（购买/消耗/退回）。
type AgentQuotaLog struct {
	ID       uint            `gorm:"primaryKey" json:"id"`
	AgentID  uint            `gorm:"index;not null" json:"agent_id"`                  // 代理 id
	Type     string          `gorm:"size:16;not null;index" json:"type"`              // purchase 购买 / freeze 建单冻结 / consume 进件消耗(冻结转出) / release 失败释放冻结 / refund 退回
	Change   int             `gorm:"not null" json:"change"`                          // 变动量（+购买/-消耗/+退回）
	Before   int             `gorm:"not null" json:"before"`                          // 变动前余额
	After    int             `gorm:"not null" json:"after"`                           // 变动后余额
	Amount   decimal.Decimal `gorm:"type:decimal(18,4);not null;default:0" json:"-"`  // 关联金额（购买名额的批发款，原始）
	RelNo    string          `gorm:"column:rel_no;size:64;index" json:"rel_no"`       // 关联单号（进件单号/充值收款单号）
	AddTime  time.Time       `gorm:"index" json:"-"`                                  // 时间（原始）
	Remark   string          `gorm:"size:255" json:"remark"`                          // 备注
}

func (AgentQuotaLog) TableName() string { return "pay_agent_quota_log" }
