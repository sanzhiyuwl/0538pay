package model

import "time"

// RiskRecord 风控命中记录。只读——由系统自动写入（关键词实时拦截 / 定时任务成功率·通知失败·投诉率），
// 后台不增删改。对齐 epay pre_risk。自研表名 pay_risk。
type RiskRecord struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	UID     uint      `gorm:"index;not null;default:0" json:"uid"` // 商户号
	Type    int8      `gorm:"not null;default:0" json:"type"`      // 0关键词屏蔽 1订单成功率 2连续通知失败 3订单投诉率
	URL     string    `gorm:"size:64" json:"url"`                  // 风控网址（关键词屏蔽记来源域名）
	Content string    `gorm:"size:64" json:"content"`              // 风控内容（命中词/成功率描述/订单数）
	Date    time.Time `gorm:"index" json:"-"`                      // 触发时间（原始）
	Status  int8      `gorm:"not null;default:0" json:"status"`    // 处理状态 0未处理 1已处理（E-6，对齐 epay pre_risk.status）
}

func (RiskRecord) TableName() string { return "pay_risk" }

// Blacklist 支付黑名单。命中则拦截支付。对齐 epay pre_blacklist。自研表名 pay_blacklist。
// 维度仅 IP + 支付账号两种；有效期用 EndTime 表达（NULL=永久），过期由 cron 清理，无 status。
type Blacklist struct {
	ID      uint       `gorm:"primaryKey" json:"id"`
	Type    int8       `gorm:"not null;default:0;uniqueIndex:uk_content_type" json:"type"` // 0支付账号 1IP地址
	Content string     `gorm:"size:64;not null;uniqueIndex:uk_content_type" json:"content"` // 账号/IP
	AddTime time.Time  `json:"-"`                                                          // 添加时间（原始）
	EndTime *time.Time `json:"-"`                                                          // 过期时间（NULL=永久，原始）
	Remark  string     `gorm:"size:80" json:"remark"`                                      // 备注
}

func (Blacklist) TableName() string { return "pay_blacklist" }

// Domain 授权支付域名（白名单）。带审核流转。对齐 epay pre_domain。自研表名 pay_domain。
// EndTime 语义是"审核时间"（非过期），每次状态变更刷新。下单校验含 *.主域名 通配 + status=1。
type Domain struct {
	ID      uint       `gorm:"primaryKey" json:"id"`
	UID     uint       `gorm:"index;not null;default:0" json:"uid"` // 商户号
	Domain  string     `gorm:"size:128;index;not null" json:"domain"` // 域名（支持 *. 通配）
	Status  int8       `gorm:"not null;default:0" json:"status"`      // 0待审核 1正常 2拒绝
	AddTime time.Time  `json:"-"`                                     // 添加时间（原始）
	EndTime *time.Time `json:"-"`                                     // 审核时间（原始）
}

func (Domain) TableName() string { return "pay_domain" }
