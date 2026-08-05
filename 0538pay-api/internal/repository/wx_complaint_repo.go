package repository

import (
	"errors"

	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// WxComplaintRepo 消费者投诉单主表数据访问（pay_wx_complaint，自研扩展）。
// 主表按 complaint_id 唯一，回调/查询到明细后 Upsert（存在则更新，不存在则插入）。
type WxComplaintRepo struct{ db *gorm.DB }

func NewWxComplaintRepo(db *gorm.DB) *WxComplaintRepo { return &WxComplaintRepo{db: db} }

// WxComplaintQuery 投诉单列表筛选。ComplaintedMchID 非空时按子商户隔离（/m 商户端二期按 sub_mchid 过滤复用）。
type WxComplaintQuery struct {
	Keyword          string // 投诉单号 / 投诉内容 / 商户名 关键词
	ComplaintedMchID string // 按被诉子商户号过滤（后台按子商户筛选 / 商户端隔离）
	MerchantID       uint   // 按本地商户过滤（0=不限）
	State            string // 投诉单状态筛选
	Page             int
	PageSize         int
}

// List 分页查询投诉单，按最近更新倒序（新投诉/新动作在上）。
func (r *WxComplaintRepo) List(q WxComplaintQuery) ([]model.WxComplaint, int64, error) {
	tx := r.db.Model(&model.WxComplaint{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("complaint_id LIKE ? OR complaint_detail LIKE ? OR merchant_name LIKE ? OR complainted_mchid LIKE ?", kw, kw, kw, kw)
	}
	if q.ComplaintedMchID != "" {
		tx = tx.Where("complainted_mchid = ?", q.ComplaintedMchID)
	}
	if q.MerchantID > 0 {
		tx = tx.Where("merchant_id = ?", q.MerchantID)
	}
	if q.State != "" {
		tx = tx.Where("complaint_state = ?", q.State)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.WxComplaint
	err := tx.Order("updated_at DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// FindByComplaintID 按微信投诉单号取。未找到返回 (nil, nil)。
func (r *WxComplaintRepo) FindByComplaintID(complaintID string) (*model.WxComplaint, error) {
	if complaintID == "" {
		return nil, nil
	}
	var c model.WxComplaint
	err := r.db.Where("complaint_id = ?", complaintID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// FindByID 按主键取。未找到返回 gorm.ErrRecordNotFound。
func (r *WxComplaintRepo) FindByID(id uint) (*model.WxComplaint, error) {
	var c model.WxComplaint
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Upsert 按 complaint_id 落库：存在则按白名单字段更新，不存在则插入。
// 明细以微信查询详情为准，回调只提供 complaint_id 触发拉取。
func (r *WxComplaintRepo) Upsert(c *model.WxComplaint) error {
	var existing model.WxComplaint
	err := r.db.Where("complaint_id = ?", c.ComplaintID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(c).Error
	}
	if err != nil {
		return err
	}
	c.ID = existing.ID
	// 保留原建单时间，仅更新业务字段（GORM Save 会刷新 UpdatedAt）。
	c.CreatedAt = existing.CreatedAt
	return r.db.Save(c).Error
}

// Stats 各状态计数（列表页概览小卡）。
func (r *WxComplaintRepo) Stats(complaintedMchID string, merchantID uint) (map[string]int64, error) {
	type row struct {
		ComplaintState string
		N              int64
	}
	tx := r.db.Model(&model.WxComplaint{}).Select("complaint_state, COUNT(*) AS n")
	if complaintedMchID != "" {
		tx = tx.Where("complainted_mchid = ?", complaintedMchID)
	}
	if merchantID > 0 {
		tx = tx.Where("merchant_id = ?", merchantID)
	}
	var rows []row
	if err := tx.Group("complaint_state").Scan(&rows).Error; err != nil {
		return nil, err
	}
	m := map[string]int64{}
	for _, x := range rows {
		m[x.ComplaintState] = x.N
	}
	return m, nil
}

// —— 投诉回调流水（pay_wx_complaint_notify）——

// WxComplaintNotifyRepo 投诉回调流水数据访问（追加型，NotifyID 唯一索引做幂等）。
type WxComplaintNotifyRepo struct{ db *gorm.DB }

func NewWxComplaintNotifyRepo(db *gorm.DB) *WxComplaintNotifyRepo {
	return &WxComplaintNotifyRepo{db: db}
}

// ErrComplaintNotifyDuplicate 表示该回调（NotifyID）已处理过（幂等命中），应答成功不重复处理。
var ErrComplaintNotifyDuplicate = errors.New("投诉回调已存在（重复通知）")

// Create 落一条回调流水。NotifyID 唯一冲突时返回 ErrComplaintNotifyDuplicate。
func (r *WxComplaintNotifyRepo) Create(n *model.WxComplaintNotify) error {
	err := r.db.Create(n).Error
	if err != nil && errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrComplaintNotifyDuplicate
	}
	return err
}

// ListByComplaintID 取某投诉单的全部回调流水，按落库时间倒序（详情抽屉时间线：新在上）。
func (r *WxComplaintNotifyRepo) ListByComplaintID(complaintID string) ([]model.WxComplaintNotify, error) {
	if complaintID == "" {
		return nil, nil
	}
	var list []model.WxComplaintNotify
	err := r.db.Where("complaint_id = ?", complaintID).Order("id DESC").Find(&list).Error
	return list, err
}
