package repository

import (
	"time"

	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// EnrollRepo 进件域数据访问：进件申请单 + 邀请链接 + 进件结算流水。
type EnrollRepo struct{ db *gorm.DB }

func NewEnrollRepo(db *gorm.DB) *EnrollRepo { return &EnrollRepo{db: db} }

// EnrollQuery 进件单列表筛选。AgentID 非空时强制按代理隔离（代理端只看自己名下）。
type EnrollQuery struct {
	Keyword  string // 商户名/进件单号 关键词
	AgentID  *uint  // 归属代理（代理端强制传自己，平台端可空看全部）
	Status   string // 本地状态机筛选
	Source   *int   // 发起方式
	Page     int
	PageSize int
}

// ListEnrolls 分页查询进件单。
func (r *EnrollRepo) ListEnrolls(q EnrollQuery) ([]model.SubMerchantEnroll, int64, error) {
	tx := r.db.Model(&model.SubMerchantEnroll{})
	if q.Keyword != "" {
		tx = tx.Where("merchant_name LIKE ? OR enroll_no LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	if q.AgentID != nil {
		tx = tx.Where("agent_id = ?", *q.AgentID)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.Source != nil {
		tx = tx.Where("source = ?", *q.Source)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SubMerchantEnroll
	err := tx.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// FindEnroll 按 id 取进件单。
func (r *EnrollRepo) FindEnroll(id uint) (*model.SubMerchantEnroll, error) {
	var e model.SubMerchantEnroll
	if err := r.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// FindEnrollByNo 按进件单号取（幂等/查询进度用）。
func (r *EnrollRepo) FindEnrollByNo(no string) (*model.SubMerchantEnroll, error) {
	var e model.SubMerchantEnroll
	if err := r.db.Where("enroll_no = ?", no).First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// ListPendingBefore 取创建时间早于 before 且仍待支付的进件单（超时关单 cron 用）。
func (r *EnrollRepo) ListPendingBefore(before time.Time) ([]model.SubMerchantEnroll, error) {
	var list []model.SubMerchantEnroll
	err := r.db.Where("status = ? AND add_time < ?", model.EnrollStatusPendingPay, before).
		Order("id ASC").Limit(200).Find(&list).Error
	return list, err
}

// CreateEnroll 新建进件单。
func (r *EnrollRepo) CreateEnroll(e *model.SubMerchantEnroll) error {
	if e.AddTime.IsZero() {
		e.AddTime = time.Now()
	}
	return r.db.Create(e).Error
}

// UpdateEnroll 更新进件单指定字段。
func (r *EnrollRepo) UpdateEnroll(id uint, fields map[string]any) error {
	return r.db.Model(&model.SubMerchantEnroll{}).Where("id = ?", id).Updates(fields).Error
}

// MarkEnrollRefunded 幂等退款改单：仅当当前状态在可退集合内时置 refunded，返回是否命中（true=本次真正执行）。
// 条件 UPDATE + 影响行数判重，防同单并发/重复退款。可退集合=paid/submitted/rejected
// （pending_pay 未收款无需退、finished 已交付硬锁、closed/refunded 已终态）。
func (r *EnrollRepo) MarkEnrollRefunded(id uint) (bool, error) {
	res := r.db.Model(&model.SubMerchantEnroll{}).
		Where("id = ? AND status IN ?", id, []string{
			model.EnrollStatusPaid, model.EnrollStatusSubmitted, model.EnrollStatusRejected,
		}).
		Update("status", model.EnrollStatusRefunded)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// CountByAgent 统计某代理名下进件单数（删代理守卫用）。
func (r *EnrollRepo) CountByAgent(agentID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.SubMerchantEnroll{}).Where("agent_id = ?", agentID).Count(&n).Error
	return n, err
}

// —— 邀请链接 ——

// ListInvites 分页查询邀请链接（AgentID 非空则按代理隔离）。
func (r *EnrollRepo) ListInvites(agentID *uint, page, pageSize int) ([]model.EnrollInvite, int64, error) {
	tx := r.db.Model(&model.EnrollInvite{})
	if agentID != nil {
		tx = tx.Where("agent_id = ?", *agentID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.EnrollInvite
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// CountInvitesByAgent 统计某代理名下邀请链接数（删代理守卫用）。
func (r *EnrollRepo) CountInvitesByAgent(agentID uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.EnrollInvite{}).Where("agent_id = ?", agentID).Count(&n).Error
	return n, err
}

// FindInviteByCode 按 code 取邀请链接（公开页校验用）。
func (r *EnrollRepo) FindInviteByCode(code string) (*model.EnrollInvite, error) {
	var v model.EnrollInvite
	if err := r.db.Where("code = ?", code).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// FindInvite 按 id 取邀请链接。
func (r *EnrollRepo) FindInvite(id uint) (*model.EnrollInvite, error) {
	var v model.EnrollInvite
	if err := r.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// CreateInvite 新建邀请链接。
func (r *EnrollRepo) CreateInvite(v *model.EnrollInvite) error {
	if v.AddTime.IsZero() {
		v.AddTime = time.Now()
	}
	return r.db.Create(v).Error
}

// UpdateInvite 更新邀请链接指定字段（停启用/备注/首次打开/过期时间等）。
func (r *EnrollRepo) UpdateInvite(id uint, fields map[string]any) error {
	return r.db.Model(&model.EnrollInvite{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteInvite 物理删除邀请链接。
func (r *EnrollRepo) DeleteInvite(id uint) error {
	return r.db.Delete(&model.EnrollInvite{}, id).Error
}

// IncInviteOpen 打开数 +1（原子），首次打开顺带记 first_access_at。公开页落地打点用。
func (r *EnrollRepo) IncInviteOpen(id uint, first bool) error {
	updates := map[string]any{"open_count": gorm.Expr("open_count + 1")}
	if first {
		updates["first_access_at"] = time.Now()
	}
	return r.db.Model(&model.EnrollInvite{}).Where("id = ?", id).Updates(updates).Error
}

// IncInviteSubmit 提交数 +1（原子）。公开页成功建单后打点用。
func (r *EnrollRepo) IncInviteSubmit(id uint) error {
	return r.db.Model(&model.EnrollInvite{}).Where("id = ?", id).
		Update("submit_count", gorm.Expr("submit_count + 1")).Error
}

// ExpireDueInvites 把 expire_at 已到且仍启用的邀请链接置为 expired（定时任务用）。
// 返回受影响行数。
func (r *EnrollRepo) ExpireDueInvites(now time.Time) (int64, error) {
	res := r.db.Model(&model.EnrollInvite{}).
		Where("status = ? AND expire_at IS NOT NULL AND expire_at <= ?", model.InviteStatusEnabled, now).
		Update("status", model.InviteStatusExpired)
	return res.RowsAffected, res.Error
}

// —— 进件结算流水 ——

// CreateSettleLog 写一笔进件结算流水。
func (r *EnrollRepo) CreateSettleLog(l *model.EnrollSettleLog) error {
	if l.SettleTime.IsZero() {
		l.SettleTime = time.Now()
	}
	return r.db.Create(l).Error
}

// ListSettleLogs 分页查询进件结算流水（AgentID 非空则按代理隔离）。
func (r *EnrollRepo) ListSettleLogs(agentID *uint, page, pageSize int) ([]model.EnrollSettleLog, int64, error) {
	tx := r.db.Model(&model.EnrollSettleLog{})
	if agentID != nil {
		tx = tx.Where("agent_id = ?", *agentID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.EnrollSettleLog
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
