package repository

import (
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ===== 统计（实时聚合，无独立表）=====

// StatRepo 统计聚合数据访问：对 pay_order / pay_transfer 实时聚合。
type StatRepo struct{ db *gorm.DB }

func NewStatRepo(db *gorm.DB) *StatRepo { return &StatRepo{db: db} }

// StatCell 聚合结果一格：商户 uid + 分组维度键 + 金额。
type StatCell struct {
	UID     uint
	GroupBy string          // 支付方式名(type_name) 或 通道id字符串
	Amount  decimal.Decimal
}

// AggregateOrders 按 商户 × (支付方式 type_name | 通道 channel) 聚合订单金额。
// field 为金额字段：money/real_money/get_money/profit_money。仅统计已支付(status=1)。
// byChannel=true 按 channel 分组，否则按 type_name。时间范围 [start,end)。
func (r *StatRepo) AggregateOrders(field string, byChannel bool, start, end time.Time) ([]StatCell, error) {
	groupCol := "type_name"
	if byChannel {
		groupCol = "channel"
	}
	type row struct {
		UID     uint
		GroupBy string
		Amount  decimal.Decimal
	}
	var rows []row
	err := r.db.Model(&model.Order{}).
		Select("uid, "+groupCol+" AS group_by, COALESCE(SUM("+field+"),0) AS amount").
		Where("status = 1 AND add_time >= ? AND add_time < ?", start, end).
		Group("uid, " + groupCol).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	cells := make([]StatCell, 0, len(rows))
	for _, x := range rows {
		cells = append(cells, StatCell{UID: x.UID, GroupBy: x.GroupBy, Amount: x.Amount})
	}
	return cells, nil
}

// AggregateTransfers 按 商户 × (方式 type | 通道 channel) 聚合代付成功金额(status=1)。
// 用于统计口径 type=4 代付金额。时间范围按 pay_time [start,end)。
func (r *StatRepo) AggregateTransfers(byChannel bool, start, end time.Time) ([]StatCell, error) {
	groupCol := "type"
	if byChannel {
		groupCol = "channel"
	}
	type row struct {
		UID     uint
		GroupBy string
		Amount  decimal.Decimal
	}
	var rows []row
	err := r.db.Model(&model.Transfer{}).
		Select("uid, "+groupCol+" AS group_by, COALESCE(SUM(money),0) AS amount").
		Where("status = 1 AND pay_time >= ? AND pay_time < ?", start, end).
		Group("uid, " + groupCol).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	cells := make([]StatCell, 0, len(rows))
	for _, x := range rows {
		cells = append(cells, StatCell{UID: x.UID, GroupBy: x.GroupBy, Amount: x.Amount})
	}
	return cells, nil
}

// ===== 登录日志 =====

// LogRepo 登录日志数据访问。
type LogRepo struct{ db *gorm.DB }

func NewLogRepo(db *gorm.DB) *LogRepo { return &LogRepo{db: db} }

// Create 写入一条登录日志。
func (r *LogRepo) Create(l *model.LoginLog) error {
	return r.db.Create(l).Error
}

// List 分页查询登录日志。column 精确等值(uid/ip)。
func (r *LogRepo) List(column, value string, page, pageSize int) ([]model.LoginLog, int64, error) {
	tx := r.db.Model(&model.LoginLog{})
	if value != "" && (column == "uid" || column == "ip") {
		tx = tx.Where(column+" = ?", value)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.LoginLog
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// CountRecentFail 统计近 since 内某 uid 的失败登录次数（防爆破用）。
func (r *LogRepo) CountRecentFail(uid uint, failType string, since time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.LoginLog{}).
		Where("uid = ? AND type = ? AND date > ?", uid, failType, since).Count(&n).Error
	return n, err
}

// ===== 邀请码 =====

// InviteRepo 邀请码数据访问。
type InviteRepo struct{ db *gorm.DB }

func NewInviteRepo(db *gorm.DB) *InviteRepo { return &InviteRepo{db: db} }

// List 分页查询邀请码。kw 为 code 精确等值。
func (r *InviteRepo) List(kw string, status *int, page, pageSize int) ([]model.InviteCode, int64, error) {
	tx := r.db.Model(&model.InviteCode{})
	if kw != "" {
		tx = tx.Where("code = ?", kw)
	}
	if status != nil && *status >= 0 {
		tx = tx.Where("status = ?", *status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.InviteCode
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// BatchCreate 批量插入邀请码。
func (r *InviteRepo) BatchCreate(codes []model.InviteCode) error {
	return r.db.Create(&codes).Error
}

// Delete 删除单个邀请码。
func (r *InviteRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.InviteCode{}).Error
}

// ClearAll 清空全部邀请码，返回删除条数。
func (r *InviteRepo) ClearAll() (int64, error) {
	res := r.db.Where("1 = 1").Delete(&model.InviteCode{})
	return res.RowsAffected, res.Error
}

// ClearUsed 清空已使用邀请码，返回删除条数。
func (r *InviteRepo) ClearUsed() (int64, error) {
	res := r.db.Where("status = 1").Delete(&model.InviteCode{})
	return res.RowsAffected, res.Error
}

// FindUsable 查一个可用邀请码（存在且未使用）。未找到返回 (nil,nil)。
func (r *InviteRepo) FindUsable(code string) (*model.InviteCode, error) {
	var c model.InviteCode
	err := r.db.Where("code = ? AND status = 0", code).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// MarkUsed 标记邀请码已使用（注册成功时）。
func (r *InviteRepo) MarkUsed(id, uid uint, now time.Time) error {
	return r.db.Model(&model.InviteCode{}).Where("id = ? AND status = 0", id).
		Updates(map[string]interface{}{"status": 1, "uid": uid, "use_time": now}).Error
}
