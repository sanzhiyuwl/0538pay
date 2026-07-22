package repository

import (
	"time"

	"github.com/0538pay/api/internal/model"
	"gorm.io/gorm"
)

// RegCodeRepo 短信/邮箱验证码数据访问（对齐 epay pre_regcode）。
type RegCodeRepo struct{ db *gorm.DB }

func NewRegCodeRepo(db *gorm.DB) *RegCodeRepo { return &RegCodeRepo{db: db} }

func (r *RegCodeRepo) Create(c *model.RegCode) error { return r.db.Create(c).Error }

// CountByToSince 统计某接收方(手机/邮箱)在 since 之后发送的条数（单号频控）。
func (r *RegCodeRepo) CountByToSince(to string, since time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.RegCode{}).Where("sendto = ? AND send_time > ?", to, since).Count(&n).Error
	return n, err
}

// CountByIPSince 统计某 IP 在 since 之后发送的条数（IP 频控）。
func (r *RegCodeRepo) CountByIPSince(ip string, since time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.RegCode{}).Where("ip = ? AND send_time > ?", ip, since).Count(&n).Error
	return n, err
}

// LatestValid 取某接收方+场景最新一条验证码（校验用，按 id 倒序）。未找到 (nil,nil)。
func (r *RegCodeRepo) Latest(to, scene string) (*model.RegCode, error) {
	var c model.RegCode
	err := r.db.Where("sendto = ? AND scene = ?", to, scene).Order("id DESC").First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// IncrErr 错误次数 +1。
func (r *RegCodeRepo) IncrErr(id uint) error {
	return r.db.Model(&model.RegCode{}).Where("id = ?", id).UpdateColumn("err_count", gorm.Expr("err_count + 1")).Error
}

// MarkUsed 标记已用（status=1）。
func (r *RegCodeRepo) MarkUsed(id uint) error {
	return r.db.Model(&model.RegCode{}).Where("id = ?", id).Update("status", 1).Error
}
