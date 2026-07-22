package model

import "time"

// Weixin 微信公众号/小程序账号。对齐 epay pre_weixin（自研表名 pay_weixin）。
// 配置 APPID/APPSECRET，用于 JSAPI 支付、网页授权等。appsecret 明文存储(对齐 epay)。
// 表单只维护 type/name/appid/appsecret；access_token/时间戳由系统运行时管理。
// 本页无状态开关(对齐 epay pay_weixin.php 无 status 列展示/编辑)。
type Weixin struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Type        int8       `gorm:"not null;default:0" json:"type"`          // 0=微信服务号 1=微信小程序
	Name        string     `gorm:"size:30;not null" json:"name"`            // 名称（仅显示，不重复）
	AppID       string     `gorm:"size:150" json:"appid"`                   // APPID（不重复）
	AppSecret   string     `gorm:"size:250" json:"-"`                       // APPSECRET（脱敏输出）
	AccessToken string     `gorm:"size:300" json:"-"`                       // 运行时缓存 token
	Status      int8       `gorm:"not null;default:1" json:"-"`             // 系统内部管理，默认开
	AddTime     *time.Time `json:"-"`                                       // 添加时间
	UpdateTime  *time.Time `gorm:"column:update_time" json:"-"`             // token 刷新时间
	ExpireTime  *time.Time `gorm:"column:expire_time" json:"-"`             // token 过期时间
}

func (Weixin) TableName() string { return "pay_weixin" }
