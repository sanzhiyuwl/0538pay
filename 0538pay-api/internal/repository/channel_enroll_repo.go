package repository

import (
	"strconv"
	"time"

	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// ChannelEnrollRepo 服务商通道商户进件单数据访问（epay 精仿线，pay_channel_enroll）。
// 与代理进件线的 EnrollRepo（pay_submch_enroll）独立，互不混写。
type ChannelEnrollRepo struct{ db *gorm.DB }

func NewChannelEnrollRepo(db *gorm.DB) *ChannelEnrollRepo { return &ChannelEnrollRepo{db: db} }

// ChannelEnrollQuery 进件单列表筛选。UID>0 时按商户隔离（商户端只看自己）。
type ChannelEnrollQuery struct {
	Keyword   string // 商户名/进件单号/联系手机 关键词
	UID       uint   // 归属商户（商户端强制传自己，后台端传 0 看全部）
	ChannelID int    // 主通道筛选（0=不限）
	Status    string // 本地状态机筛选
	WxState   string // 微信侧细状态筛选（APPLYMENT_STATE_TO_BE_SIGNED / SIGNING / AUDITING / ...），本地 status 之上进一步细分
	Sort      string // 排序：id_asc / id_desc（默认 id_desc）
	Page      int
	PageSize  int
}

// List 分页查询进件单，按 id 倒序。
func (r *ChannelEnrollRepo) List(q ChannelEnrollQuery) ([]model.ChannelEnroll, int64, error) {
	tx := r.db.Model(&model.ChannelEnroll{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		// 按 进件单号 / 商户(名称或商户号) / 手机号(进件联系手机 + 商户账户注册手机) 联合模糊查。
		// 账户手机在 pay_merchant，用子查询命中其 uid；商户号(uid)按数字精确匹配。
		conds := "enroll_no LIKE ? OR merchant_name LIKE ? OR contact_phone LIKE ? OR uid IN (SELECT uid FROM pay_merchant WHERE phone LIKE ?)"
		args := []interface{}{kw, kw, kw, kw}
		if uid, err := strconv.ParseUint(q.Keyword, 10, 64); err == nil {
			conds += " OR uid = ?"
			args = append(args, uid)
		}
		tx = tx.Where(conds, args...)
	}
	if q.UID > 0 {
		tx = tx.Where("uid = ?", q.UID)
	}
	if q.ChannelID > 0 {
		tx = tx.Where("channel_id = ?", q.ChannelID)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.WxState != "" {
		tx = tx.Where("wx_state = ?", q.WxState)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "id DESC"
	if q.Sort == "id_asc" {
		order = "id ASC"
	}
	var list []model.ChannelEnroll
	err := tx.Order(order).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// ListApproved 取所有已开通（status=approved 且 sub_mchid 非空）的进件单。
// 风控管控总览页的数据源全集：这些是在本服务商名下已拿到子商户号、可能被管控的商户。
func (r *ChannelEnrollRepo) ListApproved() ([]model.ChannelEnroll, error) {
	var list []model.ChannelEnroll
	err := r.db.Where("status = ? AND sub_mchid <> ''", model.ChannelEnrollApproved).
		Order("id DESC").Find(&list).Error
	return list, err
}

// ListApprovedByUID 取某商户名下所有已开通（approved 且 sub_mchid 非空）进件单。
// 提现/分账等 uid 级硬锁用：商户名下任一子商户被管控即命中，无进件的纯平台商户返回空。
func (r *ChannelEnrollRepo) ListApprovedByUID(uid uint) ([]model.ChannelEnroll, error) {
	var list []model.ChannelEnroll
	err := r.db.Where("uid = ? AND status = ? AND sub_mchid <> ''", uid, model.ChannelEnrollApproved).
		Order("id DESC").Find(&list).Error
	return list, err
}

// FindApprovedBySubMchID 按子商户号反查已开通进件单（风控第三段订阅回调归属用）。
// 回调只带 sub_mchid，需据此定位本服务商名下的进件单以回填 enroll_id/uid 并触发第二段刷新。
// 未找到（含非本服务商名下）返回 (nil, nil)。
func (r *ChannelEnrollRepo) FindApprovedBySubMchID(subMchID string) (*model.ChannelEnroll, error) {
	if subMchID == "" {
		return nil, nil
	}
	var e model.ChannelEnroll
	err := r.db.Where("sub_mchid = ? AND status = ?", subMchID, model.ChannelEnrollApproved).
		Order("id DESC").First(&e).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// FindByID 按主键取进件单。未找到返回 gorm.ErrRecordNotFound。
func (r *ChannelEnrollRepo) FindByID(id uint) (*model.ChannelEnroll, error) {
	var e model.ChannelEnroll
	if err := r.db.First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// FindByNo 按进件单号取（幂等/查重用）。
func (r *ChannelEnrollRepo) FindByNo(no string) (*model.ChannelEnroll, error) {
	var e model.ChannelEnroll
	if err := r.db.Where("enroll_no = ?", no).First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// FindDraftOrRejected 取某商户在某主通道下仍可续填的进件单（draft/rejected 复用同一单，不重复建）。
// 未找到返回 (nil, nil)。
func (r *ChannelEnrollRepo) FindDraftOrRejected(uid uint, channelID int) (*model.ChannelEnroll, error) {
	var e model.ChannelEnroll
	err := r.db.Where("uid = ? AND channel_id = ? AND status IN ?", uid, channelID,
		[]string{model.ChannelEnrollDraft, model.ChannelEnrollRejected}).
		Order("id DESC").First(&e).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// Create 新建进件单。AddTime 为空时补当前时间。
func (r *ChannelEnrollRepo) Create(e *model.ChannelEnroll) error {
	if e.AddTime.IsZero() {
		e.AddTime = time.Now()
	}
	return r.db.Create(e).Error
}

// Update 更新进件单指定字段（白名单 map）。
func (r *ChannelEnrollRepo) Update(id uint, fields map[string]any) error {
	return r.db.Model(&model.ChannelEnroll{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 硬删进件单（仅 service 层校验通过后调用；只允许 draft 状态被删）。
func (r *ChannelEnrollRepo) Delete(id uint) error {
	return r.db.Delete(&model.ChannelEnroll{}, id).Error
}
