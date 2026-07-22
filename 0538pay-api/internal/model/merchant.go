package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Merchant 商户。对齐前端 mock/merchants.ts 的 Merchant interface（json tag 一致）。
// 自研表名 pay_merchant，不复制 epay 的 pre_user。
// 用户组名 groupname 在查询时 JOIN 用户组表派生，不直接存本表。
type Merchant struct {
	UID       uint            `gorm:"primaryKey;column:uid" json:"uid"`             // 商户号
	GID       int             `gorm:"column:gid;index;default:0" json:"gid"`        // 用户组ID
	AppKey    string          `gorm:"column:app_key;size:64" json:"-"`              // 商户通信密钥（MD5 验签，对齐 epay pre_user.key）
	KeyType   int8            `gorm:"column:keytype;default:0" json:"-"`            // 签名类型 0=MD5 1=RSA（RSA 留待 V2）
	PublicKey string          `gorm:"column:publickey;type:text" json:"-"`          // 商户 RSA 公钥（V2 用）
	GroupEnd  *time.Time      `gorm:"column:group_end" json:"-"`                    // 用户组到期时间（原始）
	Money     decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"-"`        // 余额（原始）
	SettleID  int             `gorm:"column:settle_id;default:1" json:"settle_id"`  // 结算方式
	Account   string          `gorm:"size:128" json:"account"`                      // 结算账号
	Username  string          `gorm:"size:64" json:"username"`                      // 结算姓名
	QQ        string          `gorm:"size:20" json:"qq"`
	Phone     string          `gorm:"size:20" json:"phone"`
	Email     string          `gorm:"size:128" json:"email"`
	URL       string          `gorm:"size:255" json:"url"`                          // 域名
	AddTime   time.Time       `gorm:"index" json:"-"`                               // 添加时间（原始）
	Status    int8            `gorm:"not null;default:2;index" json:"status"`       // 0封禁1正常2未审核
	Cert      int8            `gorm:"default:0" json:"cert"`                        // 实名 0未认证/审核中 1已认证
	CertType  int8            `gorm:"default:0" json:"certtype"`                    // 0个人 1企业
	CertName  string          `gorm:"size:64" json:"-"`                             // 真实姓名（脱敏后输出）
	CertNo    string          `gorm:"size:32" json:"-"`                             // 证件号（脱敏后输出）
	CertCorp  string          `gorm:"size:128" json:"-"`                            // 企业名称
	CertTime  *time.Time      `gorm:"column:cert_time" json:"-"`                    // 认证通过时间
	Pay       int8            `gorm:"default:2" json:"pay"`                         // 支付权限 0关1开2未审核
	Settle    int8            `gorm:"default:0" json:"settle"`                      // 结算权限 0/1
	UpID      int             `gorm:"column:upid;default:0" json:"upid"`            // 邀请方
	Mode      int8            `gorm:"default:0" json:"mode"`                        // 手续费模式 0/1
	Deposit   decimal.Decimal `gorm:"type:decimal(18,4);default:0" json:"-"`        // 保证金（原始）
	Password  string          `gorm:"size:255" json:"-"`                            // 登录密码哈希，不输出
	CodeName  string          `gorm:"column:codename;size:64" json:"-"`             // 聚合收款码收款方名称（对齐 epay pre_user.codename）
	OpenCode  int8            `gorm:"column:open_code;default:0" json:"-"`          // 单独开启聚合收款 0/1（对齐 epay pre_user.open_code）
	QQUID     string          `gorm:"column:qq_uid;size:64;index" json:"-"`         // QQ 快捷登录 openid（对齐 epay pre_user.qq_uid）
	WxUID     string          `gorm:"column:wx_uid;size:64;index" json:"-"`         // 微信快捷登录 openid（对齐 epay pre_user.wx_uid）
	AlipayUID string          `gorm:"column:alipay_uid;size:64;index" json:"-"`     // 支付宝快捷登录 user_id（对齐 epay pre_user.alipay_uid）
}

func (Merchant) TableName() string { return "pay_merchant" }
