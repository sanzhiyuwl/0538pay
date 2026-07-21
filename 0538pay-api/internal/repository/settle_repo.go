package repository

import (
	"errors"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrInsufficientBalance 商户余额不足以生成结算单（生单即扣款，余额不够则拒绝）。
var ErrInsufficientBalance = errors.New("商户余额不足")

// ErrNoPending 当前不存在待结算记录（生成批次时）。对齐 epay create_batch 的报错。
var ErrNoPending = errors.New("当前不存在待结算的记录")

// SettleRepo 结算数据访问：结算明细 + 批次 CRUD，含余额扣减/退回的原子事务。
type SettleRepo struct{ db *gorm.DB }

func NewSettleRepo(db *gorm.DB) *SettleRepo { return &SettleRepo{db: db} }

// List 分页查询结算明细，支持 结算账号/姓名 关键词、商户号、方式、状态、批次筛选。
func (r *SettleRepo) List(q dto.SettleQuery) ([]model.SettleRecord, int64, error) {
	tx := r.db.Model(&model.SettleRecord{})
	if q.Keyword != "" {
		tx = tx.Where("account LIKE ? OR username LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	if q.UID != nil {
		tx = tx.Where("uid = ?", *q.UID)
	}
	if q.Type != nil && *q.Type > 0 {
		tx = tx.Where("type = ?", *q.Type)
	}
	if q.Status != nil && *q.Status >= 0 {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Batch != "" {
		tx = tx.Where("batch = ?", q.Batch)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SettleRecord
	err := tx.Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindByID 按主键查结算记录。未找到返回 (nil, nil)。
func (r *SettleRepo) FindByID(id uint) (*model.SettleRecord, error) {
	var s model.SettleRecord
	err := r.db.Where("id = ?", id).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Create 直接创建结算记录（seed / 不涉及扣款的场景用）。
func (r *SettleRepo) Create(s *model.SettleRecord) error {
	return r.db.Create(s).Error
}

// CreateWithDebit 生成结算单并从商户余额扣减 Money，事务内完成（行锁 + 余额校验 + 流水）。
// 对齐 epay：自动/手动结算生单时立即扣款。余额不足返回 ErrInsufficientBalance（不生单）。
// changeType 为资金流水类型文案（自动结算="自动结算"，手动提现="手动提现"）。
func (r *SettleRepo) CreateWithDebit(s *model.SettleRecord, changeType string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", s.UID).First(&m).Error; err != nil {
			return err
		}
		if m.Money.LessThan(s.Money) {
			return ErrInsufficientBalance
		}
		newMoney := m.Money.Sub(s.Money)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", s.UID).
			Update("money", newMoney).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: s.UID, Action: 2, Money: s.Money,
			OldMoney: m.Money, NewMoney: newMoney,
			Type: changeType, TradeNo: "", Date: time.Now(),
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		return tx.Create(s).Error
	})
}

// DeleteWithRefund 删除结算记录并把 Money 退回商户余额，事务内完成。
// 对齐 epay setStatusDo(status=4)：删除并 changeUserMoney(add,'结算失败退回')。
// 返回 refunded 表示是否发生了退款（Money>0 时退）。未找到返回 (false, gorm.ErrRecordNotFound)。
func (r *SettleRepo) DeleteWithRefund(id uint, changeType string) (bool, error) {
	var refunded bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var s model.SettleRecord
		if err := tx.Where("id = ?", id).First(&s).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.SettleRecord{}, id).Error; err != nil {
			return err
		}
		if s.Money.LessThanOrEqual(decimal.Zero) {
			return nil
		}
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", s.UID).First(&m).Error; err != nil {
			return err
		}
		newMoney := m.Money.Add(s.Money)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", s.UID).
			Update("money", newMoney).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: s.UID, Action: 1, Money: s.Money,
			OldMoney: m.Money, NewMoney: newMoney,
			Type: changeType, TradeNo: "", Date: time.Now(),
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		refunded = true
		return nil
	})
	return refunded, err
}

// SetStatus 更新单条结算记录状态相关字段（白名单 map，供状态流转/失败原因维护）。
func (r *SettleRepo) SetStatus(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.SettleRecord{}).Where("id = ?", id).Updates(fields).Error
}

// CreateBatchFromPending 把当前所有待结算(status=0)记录收进一个批次并置为"正在结算"(status=2)，
// 生成 pay_batch 记录。事务内完成。对齐 epay ajax_settle create_batch。
// 无待结算记录返回 ErrNoPending。返回收批条数与总金额(RealMoney 累加)。
func (r *SettleRepo) CreateBatchFromPending(batchNo string, now time.Time) (int, decimal.Decimal, error) {
	var count int
	allmoney := decimal.Zero
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var pending []model.SettleRecord
		if err := tx.Where("status = 0").Find(&pending).Error; err != nil {
			return err
		}
		if len(pending) == 0 {
			return ErrNoPending
		}
		ids := make([]uint, 0, len(pending))
		for i := range pending {
			allmoney = allmoney.Add(pending[i].RealMoney)
			ids = append(ids, pending[i].ID)
		}
		count = len(pending)
		if err := tx.Model(&model.SettleRecord{}).Where("id IN ?", ids).
			Updates(map[string]interface{}{"batch": batchNo, "status": 2}).Error; err != nil {
			return err
		}
		b := model.SettleBatch{Batch: batchNo, AllMoney: allmoney, Count: count, Time: now, Status: 0}
		return tx.Create(&b).Error
	})
	return count, allmoney, err
}

// CompleteBatch 把批次内"正在结算"(status=2)的记录一次性置为已完成(status=1)并写完成时间，
// 批次状态置 1。对齐 epay complete_batch。返回置完成的记录数。
func (r *SettleRepo) CompleteBatch(batch string, now time.Time) (int64, error) {
	var affected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.SettleRecord{}).
			Where("batch = ? AND status = 2", batch).
			Updates(map[string]interface{}{"status": 1, "end_time": now, "result": ""})
		if res.Error != nil {
			return res.Error
		}
		affected = res.RowsAffected
		return tx.Model(&model.SettleBatch{}).Where("batch = ?", batch).
			Update("status", 1).Error
	})
	return affected, err
}

// FindBatch 按批次号查批次。未找到返回 (nil, nil)。
func (r *SettleRepo) FindBatch(batch string) (*model.SettleBatch, error) {
	var b model.SettleBatch
	err := r.db.Where("batch = ?", batch).First(&b).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ListBatches 分页查询结算批次（按生成时间倒序）。
func (r *SettleRepo) ListBatches(page, pageSize int) ([]model.SettleBatch, int64, error) {
	tx := r.db.Model(&model.SettleBatch{})
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.SettleBatch
	err := tx.Order("time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// CountAutoSince 统计 since 之后创建的自动结算单数（自动结算的每日幂等锁用）。
func (r *SettleRepo) CountAutoSince(since time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.SettleRecord{}).
		Where("auto = 1 AND add_time >= ?", since).Count(&n).Error
	return n, err
}
