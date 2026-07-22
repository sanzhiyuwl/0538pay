package service

import (
	"time"

	"gorm.io/gorm"
)

// CleanService 数据清理（对齐 epay admin/clean.php）：按类型删除 N 天前的历史记录。
// 高风险破坏性操作，仅后台鉴权可调，且限定白名单表 + 时间列，避免误删。
type CleanService struct {
	db *gorm.DB
}

func NewCleanService(db *gorm.DB) *CleanService {
	return &CleanService{db: db}
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
	return res.RowsAffected, nil
}
