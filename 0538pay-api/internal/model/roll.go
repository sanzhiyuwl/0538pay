package model

// Roll 通道轮询组。把多个同支付方式(type)的通道编成一组，按策略分流流量。
// 对齐 epay pre_roll（自研表名 pay_roll，不复制 epay 表结构）。
//
// Info 存组内通道配置串：
//   - kind=1（权重随机）：形如 "12:3,15:7"（通道ID:权重，逗号分隔）
//   - 其它 kind：形如 "12,15,18"（仅通道ID，权重被丢弃）
//
// Idx 是顺序轮询(kind=0)的游标，命中后 +1 取模持久化（对齐 epay pre_roll.index，
// index 为 SQL 保留字，改列名 idx 规避）。
type Roll struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Type   int    `gorm:"index;not null" json:"type"`             // 支付方式ID
	Name   string `gorm:"size:64;not null" json:"name"`           // 显示名称
	Kind   int8   `gorm:"not null;default:0" json:"kind"`         // 轮询方式 0=顺序 1=权重随机 2=首个启用
	Info   string `gorm:"type:text" json:"-"`                     // 组内通道配置串（见上）
	Status int8   `gorm:"not null;default:0;index" json:"status"` // 0=关闭 1=开启
	Idx    int    `gorm:"column:idx;not null;default:0" json:"-"` // 顺序轮询游标（对齐 epay pre_roll.index）
}

func (Roll) TableName() string { return "pay_roll" }
