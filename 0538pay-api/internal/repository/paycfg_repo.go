package repository

import (
	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// ===== 支付方式 pay_type =====

type PayTypeRepo struct{ db *gorm.DB }

func NewPayTypeRepo(db *gorm.DB) *PayTypeRepo { return &PayTypeRepo{db: db} }

// All 全部支付方式，按 id 升序（对齐 epay ORDER BY id ASC）。
func (r *PayTypeRepo) All() ([]model.PayType, error) {
	var list []model.PayType
	err := r.db.Order("id ASC").Find(&list).Error
	return list, err
}

func (r *PayTypeRepo) FindByID(id uint) (*model.PayType, error) {
	var m model.PayType
	err := r.db.Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CountByNameDevice 统计同 name+device 的记录数（唯一校验）。excludeID>0 排除自身。
func (r *PayTypeRepo) CountByNameDevice(name string, device int8, excludeID uint) (int64, error) {
	tx := r.db.Model(&model.PayType{}).Where("name = ? AND device = ?", name, device)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

func (r *PayTypeRepo) Create(m *model.PayType) error { return r.db.Create(m).Error }

func (r *PayTypeRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.PayType{}).Where("id = ?", id).Updates(fields).Error
}

func (r *PayTypeRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.PayType{}).Error
}

// ===== 微信公众号/小程序 pay_weixin =====

type WeixinRepo struct{ db *gorm.DB }

func NewWeixinRepo(db *gorm.DB) *WeixinRepo { return &WeixinRepo{db: db} }

func (r *WeixinRepo) All() ([]model.Weixin, error) {
	var list []model.Weixin
	err := r.db.Order("id ASC").Find(&list).Error
	return list, err
}

func (r *WeixinRepo) FindByID(id uint) (*model.Weixin, error) {
	var m model.Weixin
	err := r.db.Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CountByName / CountByAppID 唯一校验。excludeID>0 排除自身。
func (r *WeixinRepo) CountByName(name string, excludeID uint) (int64, error) {
	return countUnique(r.db, &model.Weixin{}, "name", name, excludeID)
}
func (r *WeixinRepo) CountByAppID(appid string, excludeID uint) (int64, error) {
	return countUnique(r.db, &model.Weixin{}, "app_id", appid, excludeID)
}

func (r *WeixinRepo) Create(m *model.Weixin) error { return r.db.Create(m).Error }

func (r *WeixinRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Weixin{}).Where("id = ?", id).Updates(fields).Error
}

func (r *WeixinRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Weixin{}).Error
}

// ===== 企业微信 pay_wework (+ 子表 pay_wxkfaccount) =====

type WeworkRepo struct{ db *gorm.DB }

func NewWeworkRepo(db *gorm.DB) *WeworkRepo { return &WeworkRepo{db: db} }

func (r *WeworkRepo) All() ([]model.Wework, error) {
	var list []model.Wework
	err := r.db.Order("id ASC").Find(&list).Error
	return list, err
}

func (r *WeworkRepo) FindByID(id uint) (*model.Wework, error) {
	var m model.Wework
	err := r.db.Where("id = ?", id).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *WeworkRepo) CountByName(name string, excludeID uint) (int64, error) {
	return countUnique(r.db, &model.Wework{}, "name", name, excludeID)
}
func (r *WeworkRepo) CountByAppID(appid string, excludeID uint) (int64, error) {
	return countUnique(r.db, &model.Wework{}, "app_id", appid, excludeID)
}

func (r *WeworkRepo) Create(m *model.Wework) error { return r.db.Create(m).Error }

func (r *WeworkRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Wework{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除企业微信并级联删其客服账号（对齐 epay delWework 级联 wxkfaccount）。
func (r *WeworkRepo) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("wid = ?", id).Delete(&model.WxKfAccount{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.Wework{}).Error
	})
}

// CountKfByWork 统计某企业微信下的客服账号数（列表展示用）。返回 wid→count。
func (r *WeworkRepo) CountKfByWork() (map[uint]int64, error) {
	type row struct {
		WID uint
		Cnt int64
	}
	var rows []row
	if err := r.db.Model(&model.WxKfAccount{}).
		Select("wid, COUNT(*) AS cnt").Group("wid").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]int64, len(rows))
	for _, x := range rows {
		out[x.WID] = x.Cnt
	}
	return out, nil
}

// countUnique 通用唯一计数：某列等值计数，excludeID>0 时排除自身主键 id。
func countUnique(db *gorm.DB, model interface{}, col, val string, excludeID uint) (int64, error) {
	tx := db.Model(model).Where(col+" = ?", val)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}
