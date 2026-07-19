package repository

import (
	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// AdminRepo 管理员数据访问。
type AdminRepo struct{ db *gorm.DB }

func NewAdminRepo(db *gorm.DB) *AdminRepo { return &AdminRepo{db: db} }

// FindByUsername 按用户名查管理员，未找到返回 gorm.ErrRecordNotFound。
func (r *AdminRepo) FindByUsername(username string) (*model.Admin, error) {
	var a model.Admin
	if err := r.db.Where("username = ?", username).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// MerchantRepo 商户数据访问。
type MerchantRepo struct{ db *gorm.DB }

func NewMerchantRepo(db *gorm.DB) *MerchantRepo { return &MerchantRepo{db: db} }

// List 分页查询商户，支持字段模糊搜索与多种状态筛选。
func (r *MerchantRepo) List(q dto.MerchantQuery) ([]model.Merchant, int64, error) {
	allowedCols := map[string]string{
		"uid": "uid", "account": "account", "username": "username",
		"url": "url", "qq": "qq", "phone": "phone", "email": "email",
	}

	tx := r.db.Model(&model.Merchant{})
	if q.Keyword != "" {
		if col, ok := allowedCols[q.Column]; ok {
			tx = tx.Where(col+" LIKE ?", "%"+q.Keyword+"%")
		}
	}
	if q.GID != nil {
		tx = tx.Where("gid = ?", *q.GID)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Pay != nil {
		tx = tx.Where("pay = ?", *q.Pay)
	}
	if q.SettleSt != nil {
		tx = tx.Where("settle = ?", *q.SettleSt)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.Merchant
	err := tx.Order("uid DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// OrderRepo 订单数据访问。
type OrderRepo struct{ db *gorm.DB }

func NewOrderRepo(db *gorm.DB) *OrderRepo { return &OrderRepo{db: db} }

// List 分页查询订单，支持按 column/keyword 模糊搜索与状态筛选。
func (r *OrderRepo) List(q dto.OrderQuery) ([]model.Order, int64, error) {
	// 白名单，防 SQL 注入到列名
	allowedCols := map[string]bool{
		"trade_no": true, "out_trade_no": true, "api_trade_no": true,
		"name": true, "money": true, "realmoney": true, "domain": true,
	}

	tx := r.db.Model(&model.Order{})
	if q.Keyword != "" && allowedCols[q.Column] {
		tx = tx.Where(q.Column+" LIKE ?", "%"+q.Keyword+"%")
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.Order
	err := tx.Order("add_time DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
