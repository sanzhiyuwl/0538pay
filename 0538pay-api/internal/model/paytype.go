package model

// PayType 支付方式。对齐 epay pre_type（自研表名 pay_type）。
// 支付方式 = 前端可选的收款方式（支付宝/微信…），关联多个支付通道(pay_channel.type)。
// epay pre_type 无 icon/sort 列：图标按 /assets/icon/{name}.ico 约定，排序按 id。
type PayType struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"size:30;index;not null" json:"name"`     // 调用值（英文，与支付文档一致）
	ShowName string `gorm:"size:30;not null" json:"showname"`       // 显示名称
	Device   int8   `gorm:"not null;default:0" json:"device"`       // 支持设备 0=PC+Mobile 1=PC 2=Mobile
	Status   int8   `gorm:"not null;default:0" json:"status"`       // 0关闭 1开启
}

func (PayType) TableName() string { return "pay_type" }
