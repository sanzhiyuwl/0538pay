package model

// Config 系统配置的 KV 存储（对齐 epay pre_config：一行一个配置项，值一律按字符串存）。
// 键名对齐 epay set.php 表单 name（如 settle_rate / transfer_rate / refund_fee_type 等）。
// 自研表名 pay_config。
type Config struct {
	Key   string `gorm:"column:k;primaryKey;size:64" json:"k"`
	Value string `gorm:"column:v;type:text" json:"v"`
}

func (Config) TableName() string { return "pay_config" }
