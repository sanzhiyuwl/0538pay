package repository

import (
	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// AnnounceRepo 网站公告数据访问（对齐 epay pre_anounce）。
type AnnounceRepo struct{ db *gorm.DB }

func NewAnnounceRepo(db *gorm.DB) *AnnounceRepo { return &AnnounceRepo{db: db} }

// All 全部公告（后台管理用，按 sort 升序、id 倒序）。
func (r *AnnounceRepo) All() ([]model.Announce, error) {
	var list []model.Announce
	err := r.db.Order("sort ASC, id DESC").Find(&list).Error
	return list, err
}

// ListVisible 展示中的公告（status=1，官网/商户端读取）。
func (r *AnnounceRepo) ListVisible() ([]model.Announce, error) {
	var list []model.Announce
	err := r.db.Where("status = 1").Order("sort ASC, id DESC").Find(&list).Error
	return list, err
}

// FindByID 按 id 查。未找到返回 (nil,nil)。
func (r *AnnounceRepo) FindByID(id uint) (*model.Announce, error) {
	var a model.Announce
	err := r.db.First(&a, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AnnounceRepo) Create(a *model.Announce) error { return r.db.Create(a).Error }

func (r *AnnounceRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Announce{}).Where("id = ?", id).Updates(fields).Error
}

func (r *AnnounceRepo) Delete(id uint) error {
	return r.db.Delete(&model.Announce{}, id).Error
}
