package repository

import (
	"errors"

	"github.com/epvia/api/internal/model"
	"gorm.io/gorm"
)

// ChannelControlFlowRepo 子商户管控流水数据访问（pay_channel_control_flow，风控第三段）。
// 追加型：每条微信处置/管控订阅回调落一行；NotifyID 唯一索引做幂等去重。
type ChannelControlFlowRepo struct{ db *gorm.DB }

func NewChannelControlFlowRepo(db *gorm.DB) *ChannelControlFlowRepo {
	return &ChannelControlFlowRepo{db: db}
}

// ErrFlowDuplicate 表示该回调（NotifyID）已处理过（幂等命中），调用方据此直接应答成功。
var ErrFlowDuplicate = errors.New("管控流水已存在（重复回调）")

// Create 落一条管控流水。NotifyID 唯一冲突时返回 ErrFlowDuplicate（幂等，非真错误）。
func (r *ChannelControlFlowRepo) Create(f *model.ChannelControlFlow) error {
	err := r.db.Create(f).Error
	if err != nil && errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrFlowDuplicate
	}
	return err
}

// ExistsByNotifyID 判断某回调是否已落库（幂等预检，Create 唯一索引为最终防线）。
func (r *ChannelControlFlowRepo) ExistsByNotifyID(notifyID string) (bool, error) {
	if notifyID == "" {
		return false, nil
	}
	var n int64
	err := r.db.Model(&model.ChannelControlFlow{}).Where("notify_id = ?", notifyID).Count(&n).Error
	return n > 0, err
}

// ListByEnrollID 取某进件单名下全部管控流水，按落库时间倒序（时间线：新在上）。
func (r *ChannelControlFlowRepo) ListByEnrollID(enrollID uint) ([]model.ChannelControlFlow, error) {
	var list []model.ChannelControlFlow
	err := r.db.Where("enroll_id = ?", enrollID).Order("id DESC").Find(&list).Error
	return list, err
}

// ListBySubMchID 取某子商户号全部管控流水（含未匹配到进件单的历史），按落库时间倒序。
func (r *ChannelControlFlowRepo) ListBySubMchID(subMchID string) ([]model.ChannelControlFlow, error) {
	if subMchID == "" {
		return nil, nil
	}
	var list []model.ChannelControlFlow
	err := r.db.Where("sub_mchid = ?", subMchID).Order("id DESC").Find(&list).Error
	return list, err
}
