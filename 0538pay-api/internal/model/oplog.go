package model

import "time"

// OperationLog 操作日志（我方独有安全审计增强，epay 无此概念）。
// 通用表：scope 区分商户端/管理端操作，本期只写 merchant，将来管理员操作审计可复用同表。
// 记录商户在 /m 端的写操作（改资料/提现/退款/绑域名/改密钥等），供管理端查看/导出。
// 自研表名 pay_oplog。
type OperationLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Scope     string    `gorm:"size:10;index:idx_scope_uid;not null;default:merchant" json:"scope"` // merchant / admin
	UID       uint      `gorm:"index:idx_scope_uid;not null;default:0" json:"uid"`                  // 操作者ID（商户uid或管理员id）
	Operator  string    `gorm:"size:64" json:"operator"`                                            // 操作者名（冗余存，列表免JOIN）
	Action    string    `gorm:"size:40;index" json:"action"`                                        // 动作key（apply_withdraw…）
	Category  string    `gorm:"size:20;index" json:"category"`                                      // 分类 account/fund/auth/config
	Level     string    `gorm:"size:10;index" json:"level"`                                         // 级别 normal/warning/danger
	Target    string    `gorm:"size:255" json:"target"`                                             // 操作对象摘要
	Detail    string    `gorm:"type:text" json:"detail"`                                            // JSON明细（详情展开用）
	Result    string    `gorm:"size:10;not null;default:ok" json:"result"`                          // ok / fail
	IP        string    `gorm:"size:50" json:"ip"`                                                  // 操作IP
	CreatedAt time.Time `gorm:"index" json:"-"`                                                     // 时间（原始，格式化在service层）
}

func (OperationLog) TableName() string { return "pay_oplog" }
