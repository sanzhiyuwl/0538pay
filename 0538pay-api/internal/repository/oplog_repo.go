package repository

import (
	"time"

	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// OpLogRepo 操作日志数据访问（我方独有安全审计）。
type OpLogRepo struct{ db *gorm.DB }

func NewOpLogRepo(db *gorm.DB) *OpLogRepo { return &OpLogRepo{db: db} }

// Create 写入一条操作日志（fire-and-forget，调用方吞错不阻断主流程）。
func (r *OpLogRepo) Create(l *model.OperationLog) error {
	return r.db.Create(l).Error
}

// OpLogFilter 多维筛选条件。空字段忽略。
type OpLogFilter struct {
	Scope    string     // merchant / admin
	UID      uint       // 操作者ID，0 表示不限
	Action   string     // 动作key
	Category string     // 分类
	Level    string     // 级别
	Result   string     // ok / fail
	Keyword  string     // 模糊：operator / target
	Start    *time.Time // 时间范围起
	End      *time.Time // 时间范围止
}

// buildQuery 按筛选条件组装查询（列表与导出共用）。
func (r *OpLogRepo) buildQuery(f OpLogFilter) *gorm.DB {
	tx := r.db.Model(&model.OperationLog{})
	if f.Scope != "" {
		tx = tx.Where("scope = ?", f.Scope)
	}
	if f.UID > 0 {
		tx = tx.Where("uid = ?", f.UID)
	}
	if f.Action != "" {
		tx = tx.Where("action = ?", f.Action)
	}
	if f.Category != "" {
		tx = tx.Where("category = ?", f.Category)
	}
	if f.Level != "" {
		tx = tx.Where("level = ?", f.Level)
	}
	if f.Result != "" {
		tx = tx.Where("result = ?", f.Result)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		tx = tx.Where("operator LIKE ? OR target LIKE ?", kw, kw)
	}
	if f.Start != nil {
		tx = tx.Where("created_at >= ?", *f.Start)
	}
	if f.End != nil {
		tx = tx.Where("created_at <= ?", *f.End)
	}
	return tx
}

// List 分页查询。
func (r *OpLogRepo) List(f OpLogFilter, page, pageSize int) ([]model.OperationLog, int64, error) {
	tx := r.buildQuery(f)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.OperationLog
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ExportAll 导出全量（按筛选，无分页；上限保护交由 service 层）。
func (r *OpLogRepo) ExportAll(f OpLogFilter, limit int) ([]model.OperationLog, error) {
	var list []model.OperationLog
	err := r.buildQuery(f).Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

// DistinctActions 取某 scope 下已落库的去重动作名（供前端动作下拉，只列真实发生过的）。
func (r *OpLogRepo) DistinctActions(scope string) ([]string, error) {
	var actions []string
	err := r.db.Model(&model.OperationLog{}).
		Where("scope = ?", scope).
		Distinct().Order("action").Pluck("action", &actions).Error
	return actions, err
}
