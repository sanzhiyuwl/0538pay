package repository

import (
	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConfigRepo 系统配置 KV 存储数据访问（对齐 epay pre_config）。
type ConfigRepo struct{ db *gorm.DB }

func NewConfigRepo(db *gorm.DB) *ConfigRepo { return &ConfigRepo{db: db} }

// All 全量读出配置，返回 key→value 映射（服务层加载进内存缓存）。
func (r *ConfigRepo) All() (map[string]string, error) {
	var rows []model.Config
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]string, len(rows))
	for _, c := range rows {
		m[c.Key] = c.Value
	}
	return m, nil
}

// SetMany 批量 upsert 配置项（保存设置表单时逐键写入，对齐 epay saveSetting REPLACE INTO）。
func (r *ConfigRepo) SetMany(kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	rows := make([]model.Config, 0, len(kv))
	for k, v := range kv {
		rows = append(rows, model.Config{Key: k, Value: v})
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "k"}},
		DoUpdates: clause.AssignmentColumns([]string{"v"}),
	}).Create(&rows).Error
}
