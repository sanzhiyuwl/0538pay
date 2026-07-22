package model

import (
	"time"

	"github.com/0538pay/api/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewDB 按配置建立 GORM 连接并配置连接池。
func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	// TranslateError=true 让 GORM 把驱动错误归一为 gorm.ErrDuplicatedKey 等哨兵错误，
	// 供 repo 层判主键/唯一键冲突（如代付交易号幂等）。
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}

// AutoMigrate 起步阶段用 GORM 自动建表；表结构稳定后改用 migrations/ SQL 脚本管理。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Admin{},
		&Order{},
		&Merchant{},
		&PayRecord{},
		&Channel{},
		&Roll{},
		&SubChannel{},
		&SettleRecord{},
		&SettleBatch{},
		&Transfer{},
		&ProfitReceiver{},
		&ProfitOrder{},
		&RiskRecord{},
		&Blacklist{},
		&Domain{},
		&LoginLog{},
		&InviteCode{},
		&Group{},
		&SiteConfig{},
		&Config{},
	)
}
