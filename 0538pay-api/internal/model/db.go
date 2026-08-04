package model

import (
	"fmt"
	"time"

	"github.com/epvia/api/internal/config"
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
	// ConnMaxLifetime 控制连接最长存活；ConnMaxIdleTime 让空闲连接更早回收。
	// 二者取值都小于 MySQL 的 wait_timeout（默认较长，但机器休眠/服务重启会提前掐断连接），
	// 避免池里滞留已被 MySQL 端关闭的死连接，消除偶发的 "invalid connection"。
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Minute)
	return db, nil
}

// merchantUIDStart 商户号自增起始值。对齐 epay pre_user 表 AUTO_INCREMENT=1000
// （install.sql:283）——商户号从 1000 起按注册递增 +1，避免个位数号段。
const merchantUIDStart = 1000

// AutoMigrate 起步阶段用 GORM 自动建表；表结构稳定后改用 migrations/ SQL 脚本管理。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&Admin{},
		&Order{},
		&SubOrder{},
		&Merchant{},
		&PayRecord{},
		&Channel{},
		&Roll{},
		&SubChannel{},
		&RefundOrder{},
		&PayType{},
		&Weixin{},
		&Wework{},
		&WxKfAccount{},
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
		&Message{},
		&MessageRead{},
		&Announce{},
		&RegCode{},
		&ArticleCategory{},
		&Article{},
		&OperationLog{},
		&Role{},
		// 服务商通道商户进件（epay 精仿线：只走商户进件不走二清，0730 阶段1）
		&ChannelEnroll{},
		// 子商户管控快照（风控第二段：进件成功后按 sub_mchid 查微信被管控能力落地）
		&ChannelControl{},
		// 子商户管控流水（风控第三段：处置/管控订阅回调追加型时间线）
		&ChannelControlFlow{},
		// 代理进件平台（自研扩展，epay 无）
		&Agent{},
		&AgentQuotaWallet{},
		&AgentQuotaLog{},
		&SubMerchantEnroll{},
		&EnrollInvite{},
		&EnrollSettleLog{},
	); err != nil {
		return err
	}
	return ensureMerchantUIDStart(db)
}

// ensureMerchantUIDStart 把 pay_merchant 表自增起点抬到 1000（对齐 epay）。
// MySQL 规则：AUTO_INCREMENT 不能设得比 max(uid)+1 更小，故仅当表为空或最大 uid<1000
// 时该 ALTER 才实际生效；已有 uid≥1000 的行时是无害的 no-op（保持现有递增）。
func ensureMerchantUIDStart(db *gorm.DB) error {
	var maxUID uint
	// 空表时 MAX 返回 NULL → 用 COALESCE 兜底 0。
	if err := db.Model(&Merchant{}).Select("COALESCE(MAX(uid),0)").Scan(&maxUID).Error; err != nil {
		return err
	}
	if maxUID >= merchantUIDStart {
		return nil // 已进入 1000+ 号段，无需调整
	}
	// ALTER TABLE 是 DDL，MySQL 不支持占位符参数；merchantUIDStart 是内部常量（非用户输入），
	// 用 Sprintf 拼接无注入风险。
	return db.Exec(fmt.Sprintf("ALTER TABLE pay_merchant AUTO_INCREMENT = %d", merchantUIDStart)).Error
}
