package model

import "time"

// SubChannel 子通道。归属某个商户，是「同一主通道下、每个商户各自的一份参数覆盖」。
// 对齐 epay pre_subchannel（自研表名 pay_subchannel）。
//
// 用户组某支付方式分配为 -2（用户自定义子通道）时，下单在该商户该支付方式下的子通道中
// 按 UseTime 升序（最久未用优先，等价顺序轮转）取第一个可用，命中后回写 UseTime=NOW()。
//
// Info 存该商户自定义的支付参数(JSON)：主通道 config 里凡值形如 "[key]"（方括号占位）的键，
// 下单时用子通道 Info[key] 的值替换（对齐 epay Channel::getSub 的占位变量替换）。
//
// epay 的 apply_id（关联进件渠道）本项目无进件/onboarding 系统，故不移植。
type SubChannel struct {
	ID      uint       `gorm:"primaryKey" json:"id"`
	Channel int        `gorm:"index;not null" json:"channel"`          // 归属主通道ID（pay_channel.id）
	UID     uint       `gorm:"index;not null" json:"uid"`              // 归属商户ID
	Name    string     `gorm:"size:64;not null" json:"name"`           // 子通道备注（同一商户内唯一）
	Status  int8       `gorm:"not null;default:0" json:"status"`       // 0=关闭 1=开启
	Info    string     `gorm:"type:text" json:"-"`                     // 自定义支付参数 JSON（占位替换用）
	AddTime time.Time  `json:"-"`                                      // 添加时间
	UseTime *time.Time `gorm:"column:use_time;index" json:"-"`         // 上次使用时间（顺序调度用，可空）
}

func (SubChannel) TableName() string { return "pay_subchannel" }
