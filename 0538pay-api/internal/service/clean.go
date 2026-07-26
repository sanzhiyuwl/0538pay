package service

import (
	"time"

	"gorm.io/gorm"
)

// CleanService 数据清理（对齐 epay admin/clean.php）：按类型删除 N 天前的历史记录。
// 高风险破坏性操作，仅后台鉴权可调，且限定白名单表 + 时间列，避免误删。
type CleanService struct {
	db  *gorm.DB
	cfg *ConfigService // 用于清理设置缓存（cleancache）
}

func NewCleanService(db *gorm.DB, cfg *ConfigService) *CleanService {
	return &CleanService{db: db, cfg: cfg}
}

// cleanTarget 白名单：清理目标 → 表名 + 时间列。防止越权删任意表。
type cleanTarget struct {
	table   string
	timeCol string
}

var cleanTargets = map[string]cleanTarget{
	"order":    {"pay_order", "add_time"},
	"settle":   {"pay_settle", "add_time"},
	"record":   {"pay_record", "date"},
	"transfer": {"pay_transfer", "add_time"},
	"psorder":  {"pay_ps_order", "add_time"},
}

// Clean 删除某类型 days 天前的记录。days 最小 7（保护近期数据，对齐 epay 建议保留）。
// 返回删除条数。target 非白名单或 days 非法则返回错误。
func (s *CleanService) Clean(target string, days int) (int64, error) {
	t, ok := cleanTargets[target]
	if !ok {
		return 0, maErr("不支持的清理类型")
	}
	if days < 7 {
		return 0, maErr("为保护近期数据，清理天数不得小于 7 天")
	}
	before := time.Now().AddDate(0, 0, -days)
	res := s.db.Exec("DELETE FROM "+t.table+" WHERE "+t.timeCol+" < ?", before)
	if res.Error != nil {
		return 0, res.Error
	}
	// 删除后整理表空间（对齐 epay clean.php 每次删除后 OPTIMIZE TABLE）。
	// 失败不阻断（部分引擎/权限不支持），仅回收碎片。表名来自白名单，非用户输入。
	s.db.Exec("OPTIMIZE TABLE " + t.table)
	return res.RowsAffected, nil
}

// CleanCache 清理系统设置缓存：重载配置并通知订阅者（对齐 epay clean.php mod=cleancache）。
func (s *CleanService) CleanCache() error {
	if s.cfg == nil {
		return maErr("配置服务未就绪")
	}
	return s.cfg.Reload()
}
