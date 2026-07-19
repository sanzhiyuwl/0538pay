package model

import (
	"time"

	"github.com/0538pay/api/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// NewDB 按配置建立 GORM 连接并配置连接池。
func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
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
	)
}
