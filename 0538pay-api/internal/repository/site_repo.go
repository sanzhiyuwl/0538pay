package repository

import (
	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SiteConfigRepo 官网 CMS KV 存储数据访问。
type SiteConfigRepo struct{ db *gorm.DB }

func NewSiteConfigRepo(db *gorm.DB) *SiteConfigRepo { return &SiteConfigRepo{db: db} }

// Get 按 key 取 JSON 文档。未找到返回 ("", nil)（官网读时回退前端默认值）。
func (r *SiteConfigRepo) Get(key string) (string, error) {
	var c model.SiteConfig
	err := r.db.Where("k = ?", key).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return c.Value, nil
}

// Set 保存/覆盖某 key 的 JSON 文档（upsert）。
func (r *SiteConfigRepo) Set(key, value string) error {
	c := model.SiteConfig{Key: key, Value: value}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "k"}},
		DoUpdates: clause.AssignmentColumns([]string{"v", "updated_at"}),
	}).Create(&c).Error
}
