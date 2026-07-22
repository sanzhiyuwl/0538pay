package repository

import (
	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// RefundOrderRepo 退款单数据访问（对齐 epay pre_refundorder）。
type RefundOrderRepo struct{ db *gorm.DB }

func NewRefundOrderRepo(db *gorm.DB) *RefundOrderRepo { return &RefundOrderRepo{db: db} }

// FindByOutRefundNo 按 商户号+商户退款单号 查退款单（幂等用）。未找到返回 (nil,nil)。
func (r *RefundOrderRepo) FindByOutRefundNo(uid uint, outRefundNo string) (*model.RefundOrder, error) {
	var m model.RefundOrder
	err := r.db.Where("uid = ? AND out_refund_no = ?", uid, outRefundNo).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// FindByRefundNo 按 商户号+系统退款单号 查退款单（退款查询用）。未找到返回 (nil,nil)。
func (r *RefundOrderRepo) FindByRefundNo(uid uint, refundNo string) (*model.RefundOrder, error) {
	var m model.RefundOrder
	err := r.db.Where("uid = ? AND refund_no = ?", uid, refundNo).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create 新增退款单。
func (r *RefundOrderRepo) Create(m *model.RefundOrder) error {
	return r.db.Create(m).Error
}
