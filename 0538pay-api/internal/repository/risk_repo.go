package repository

import (
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// ===== 风控记录（只读 + 系统写入）=====

// RiskRepo 风控记录数据访问。
type RiskRepo struct{ db *gorm.DB }

func NewRiskRepo(db *gorm.DB) *RiskRepo { return &RiskRepo{db: db} }

// Create 写入一条风控记录（系统触发，如关键词拦截）。
func (r *RiskRepo) Create(rec *model.RiskRecord) error {
	return r.db.Create(rec).Error
}

// List 分页查询风控记录。搜索为精确等值（对齐 epay riskList，非模糊）。
func (r *RiskRepo) List(q dto.RiskQuery) ([]model.RiskRecord, int64, error) {
	tx := r.db.Model(&model.RiskRecord{})
	if q.Value != "" {
		allowed := map[string]bool{"uid": true, "url": true, "content": true}
		if allowed[q.Column] {
			tx = tx.Where(q.Column+" = ?", q.Value)
		}
	}
	if q.Type != nil && *q.Type >= 0 {
		tx = tx.Where("type = ?", *q.Type)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.RiskRecord
	err := tx.Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// ===== 黑名单 =====

// BlacklistRepo 黑名单数据访问。
type BlacklistRepo struct{ db *gorm.DB }

func NewBlacklistRepo(db *gorm.DB) *BlacklistRepo { return &BlacklistRepo{db: db} }

// List 分页查询黑名单。kw 为 content 精确等值（对齐 epay blackList）。
func (r *BlacklistRepo) List(q dto.BlacklistQuery) ([]model.Blacklist, int64, error) {
	tx := r.db.Model(&model.Blacklist{})
	if q.Keyword != "" {
		tx = tx.Where("content = ?", q.Keyword)
	}
	if q.Type != nil && *q.Type >= 0 {
		tx = tx.Where("type = ?", *q.Type)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Blacklist
	err := tx.Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// ExistActive 判断某类型+内容是否命中未过期黑名单（下单拦截用）。
func (r *BlacklistRepo) ExistActive(typ int8, content string, now time.Time) (bool, error) {
	var n int64
	err := r.db.Model(&model.Blacklist{}).
		Where("type = ? AND content = ? AND (end_time IS NULL OR end_time > ?)", typ, content, now).
		Count(&n).Error
	return n > 0, err
}

// CountByType 按类型统计（供概况）。typ<0 统计全部。
func (r *BlacklistRepo) CountByType(typ int) (int64, error) {
	tx := r.db.Model(&model.Blacklist{})
	if typ >= 0 {
		tx = tx.Where("type = ?", typ)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// CountPermanent 统计永久黑名单（end_time IS NULL）。
func (r *BlacklistRepo) CountPermanent() (int64, error) {
	var n int64
	err := r.db.Model(&model.Blacklist{}).Where("end_time IS NULL").Count(&n).Error
	return n, err
}

// Exist 判断 (type,content) 是否已存在（添加查重）。
func (r *BlacklistRepo) Exist(typ int8, content string) (bool, error) {
	var n int64
	err := r.db.Model(&model.Blacklist{}).Where("type = ? AND content = ?", typ, content).Count(&n).Error
	return n > 0, err
}

// Create 添加黑名单。
func (r *BlacklistRepo) Create(b *model.Blacklist) error {
	return r.db.Create(b).Error
}

// Delete 删除黑名单（单条）。
func (r *BlacklistRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Blacklist{}).Error
}

// BatchDelete 批量删除，返回删除条数。
func (r *BlacklistRepo) BatchDelete(ids []uint) (int64, error) {
	res := r.db.Where("id IN ?", ids).Delete(&model.Blacklist{})
	return res.RowsAffected, res.Error
}

// ===== 授权域名 =====

// DomainRepo 授权域名数据访问。
type DomainRepo struct{ db *gorm.DB }

func NewDomainRepo(db *gorm.DB) *DomainRepo { return &DomainRepo{db: db} }

// List 分页查询域名。uid + kw(domain 精确等值) + 状态筛选（对齐 epay domainList）。
func (r *DomainRepo) List(q dto.DomainQuery) ([]model.Domain, int64, error) {
	tx := r.db.Model(&model.Domain{})
	if q.UID != nil {
		tx = tx.Where("uid = ?", *q.UID)
	}
	if q.Keyword != "" {
		tx = tx.Where("domain = ?", q.Keyword)
	}
	if q.Status != nil && *q.Status >= 0 {
		tx = tx.Where("status = ?", *q.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Domain
	err := tx.Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// FindByID 按主键查域名。未找到返回 (nil,nil)。
func (r *DomainRepo) FindByID(id uint) (*model.Domain, error) {
	var d model.Domain
	err := r.db.Where("id = ?", id).First(&d).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Exist 判断 (uid,domain) 是否已存在（添加查重）。
func (r *DomainRepo) Exist(uid uint, domain string) (bool, error) {
	var n int64
	err := r.db.Model(&model.Domain{}).Where("uid = ? AND domain = ?", uid, domain).Count(&n).Error
	return n > 0, err
}

// Create 添加域名。
func (r *DomainRepo) Create(d *model.Domain) error {
	return r.db.Create(d).Error
}

// SetStatus 审核/状态变更，刷新审核时间 end_time。
func (r *DomainRepo) SetStatus(id uint, status int8, now time.Time) error {
	return r.db.Model(&model.Domain{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "end_time": now}).Error
}

// Delete 删除域名。
func (r *DomainRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Domain{}).Error
}

// BatchOp 批量：status=3 删除，否则批量改状态+刷新审核时间。返回影响条数。
func (r *DomainRepo) BatchOp(ids []uint, status int8, now time.Time) (int64, error) {
	if status == 3 {
		res := r.db.Where("id IN ?", ids).Delete(&model.Domain{})
		return res.RowsAffected, res.Error
	}
	res := r.db.Model(&model.Domain{}).Where("id IN ?", ids).
		Updates(map[string]interface{}{"status": status, "end_time": now})
	return res.RowsAffected, res.Error
}

// CountByStatus 按状态统计（概况）。status<0 全部。
func (r *DomainRepo) CountByStatus(status int) (int64, error) {
	tx := r.db.Model(&model.Domain{})
	if status >= 0 {
		tx = tx.Where("status = ?", status)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// CountByUID 统计商户配置的授权域名数（判断是否启用白名单校验）。
func (r *DomainRepo) CountByUID(uid uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.Domain{}).Where("uid = ?", uid).Count(&n).Error
	return n, err
}

// MatchWhitelist 下单域名白名单校验：uid 下有 status=1 且 domain 精确匹配 或 *.主域名 通配匹配。
// wildcard 为 "*."+主域名。命中返回 true。
func (r *DomainRepo) MatchWhitelist(uid uint, domain, wildcard string) (bool, error) {
	var n int64
	err := r.db.Model(&model.Domain{}).
		Where("uid = ? AND status = 1 AND (domain = ? OR domain = ?)", uid, domain, wildcard).
		Count(&n).Error
	return n > 0, err
}
