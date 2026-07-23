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

// ErrDuplicateBizNo 交易号已存在（biz_no 主键冲突，同一交易号只能发起一次，幂等）。
var ErrDuplicateBizNo = errors.New("交易号已存在")

// TransferRepo 代付数据访问：记录 CRUD + 发起即时扣款 / 退回退款的原子事务。
// 资金安全对齐 epay lib/Transfer：扣 CostMoney、退 CostMoney、退款仅在 status=0 时执行。
type TransferRepo struct{ db *gorm.DB }

func NewTransferRepo(db *gorm.DB) *TransferRepo { return &TransferRepo{db: db} }

// List 分页查询代付记录，支持 交易号/账号/姓名 关键词、商户号、方式、状态筛选。
// 按 biz_no 倒序（前缀是时间戳，等价按提交时间倒序，对齐 epay order by biz_no desc）。
func (r *TransferRepo) List(q dto.TransferQuery) ([]model.Transfer, int64, error) {
	tx := r.db.Model(&model.Transfer{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("biz_no LIKE ? OR account LIKE ? OR username LIKE ?", kw, kw, kw)
	}
	if q.UID != nil {
		tx = tx.Where("uid = ?", *q.UID)
	}
	if q.Type != "" {
		tx = tx.Where("type = ?", q.Type)
	}
	if q.Status != nil && *q.Status >= 0 {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Transfer
	err := tx.Order("biz_no DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindByBizNo 按交易号查代付记录。未找到返回 (nil, nil)。
func (r *TransferRepo) FindByBizNo(bizNo string) (*model.Transfer, error) {
	var t model.Transfer
	err := r.db.Where("biz_no = ?", bizNo).First(&t).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateAdmin 后台管理员发起代付（UID=0）：不收手续费、不校验/扣减余额，直接落库。
// biz_no 主键冲突返回 ErrDuplicateBizNo（幂等）。
func (r *TransferRepo) CreateAdmin(t *model.Transfer) error {
	if err := r.db.Create(t).Error; err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicateBizNo
		}
		return err
	}
	return nil
}

// CreateWithDebit 商户发起代付：事务内从余额扣减 CostMoney 并落库（行锁 + 余额校验 + 流水）。
// 对齐 epay：下单成功后立即 changeUserMoney(uid, need_money, false, '代付')。
// 余额不足返回 ErrInsufficientBalance；交易号重复返回 ErrDuplicateBizNo。均不落库。
func (r *TransferRepo) CreateWithDebit(t *model.Transfer) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先查交易号是否已存在（幂等前置校验，避免扣款后才发现主键冲突）
		var cnt int64
		if err := tx.Model(&model.Transfer{}).Where("biz_no = ?", t.BizNo).Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			return ErrDuplicateBizNo
		}
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", t.UID).First(&m).Error; err != nil {
			return err
		}
		if m.Money.LessThan(t.CostMoney) {
			return ErrInsufficientBalance
		}
		newMoney := m.Money.Sub(t.CostMoney)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", t.UID).
			Update("money", newMoney).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: t.UID, Action: 2, Money: t.CostMoney,
			OldMoney: m.Money, NewMoney: newMoney,
			Type: "代付", TradeNo: t.BizNo, Date: time.Now(),
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		if err := tx.Create(t).Error; err != nil {
			if isDuplicateKey(err) {
				return ErrDuplicateBizNo
			}
			return err
		}
		return nil
	})
}

// SetStatus 手动改状态（不动资金，对齐 epay setTransferStatus/operation）。
// 白名单字段：status + result + paytime。管理员 1↔2 互改用。
func (r *TransferRepo) SetStatus(bizNo string, fields map[string]interface{}) error {
	return r.db.Model(&model.Transfer{}).Where("biz_no = ?", bizNo).Updates(fields).Error
}

// Delete 删除代付记录（不退款，对齐 epay delTransfer/operation delete）。
func (r *TransferRepo) Delete(bizNo string) error {
	return r.db.Where("biz_no = ?", bizNo).Delete(&model.Transfer{}).Error
}

// FailWithRefund 代付失败/退回：把处理中(status=0)记录置为失败(2)并退回 CostMoney 给商户。
// 对齐 epay refundTransfer / status 失败分支：条件 UPDATE status=0→2 防重复退款；
// 仅 UID>0（商户发起）才退款，UID=0（管理员发起）只改状态不退款。
// 返回 refunded 表示本次是否真的执行了退款（并发/重复调用只有第一次为 true）。
func (r *TransferRepo) FailWithRefund(bizNo, result string) (refunded bool, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// 条件 UPDATE：仅当前处于处理中(0)才翻转为失败(2)。RowsAffected 判并发/重复。
		res := tx.Model(&model.Transfer{}).
			Where("biz_no = ? AND status = 0", bizNo).
			Updates(map[string]interface{}{"status": 2, "result": result})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil // 已是终态或不存在：不重复退款
		}
		// 取记录判断是否需退款
		var t model.Transfer
		if err := tx.Where("biz_no = ?", bizNo).First(&t).Error; err != nil {
			return err
		}
		if t.UID == 0 || t.CostMoney.LessThanOrEqual(decimal.Zero) {
			return nil // 管理员发起或无扣款：不退
		}
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", t.UID).First(&m).Error; err != nil {
			return err
		}
		newMoney := m.Money.Add(t.CostMoney)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", t.UID).
			Update("money", newMoney).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: t.UID, Action: 1, Money: t.CostMoney,
			OldMoney: m.Money, NewMoney: newMoney,
			Type: "代付退回", TradeNo: t.BizNo, Date: time.Now(),
		}
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
		refunded = true
		return nil
	})
	return
}

// MarkSuccess 把处理中(0)记录置为成功(1)并写付款时间。条件 UPDATE 防重复。
// 对齐 epay 成功回调/手动改为成功（成功不涉及资金变动，钱在发起时已扣）。
func (r *TransferRepo) MarkSuccess(bizNo, payOrderNo string, payTime time.Time) (bool, error) {
	fields := map[string]interface{}{
		"status": 1, "pay_time": payTime, "result": "",
	}
	if payOrderNo != "" {
		fields["pay_order_no"] = payOrderNo
	}
	res := r.db.Model(&model.Transfer{}).
		Where("biz_no = ? AND status = 0", bizNo).
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// Stats 当前筛选条件下的代付统计（总额/成功额/各状态笔数）。
func (r *TransferRepo) Stats(q dto.TransferQuery) (totalMoney, successMoney decimal.Decimal, successCnt, processCnt, failCnt int64, err error) {
	type agg struct {
		Status int8
		Sum    decimal.Decimal
		Cnt    int64
	}
	var rows []agg
	// 统计不分页，套用同样的筛选条件
	tx := r.db.Model(&model.Transfer{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		tx = tx.Where("biz_no LIKE ? OR account LIKE ? OR username LIKE ?", kw, kw, kw)
	}
	if q.UID != nil {
		tx = tx.Where("uid = ?", *q.UID)
	}
	if q.Type != "" {
		tx = tx.Where("type = ?", q.Type)
	}
	if q.Status != nil && *q.Status >= 0 {
		tx = tx.Where("status = ?", *q.Status)
	}
	err = tx.Select("status, COALESCE(SUM(money),0) AS sum, COUNT(*) AS cnt").
		Group("status").Scan(&rows).Error
	if err != nil {
		return
	}
	for _, row := range rows {
		totalMoney = totalMoney.Add(row.Sum)
		switch row.Status {
		case 0:
			processCnt = row.Cnt
		case 1:
			successMoney, successCnt = row.Sum, row.Cnt
		case 2:
			failCnt = row.Cnt
		}
	}
	return
}

// CountTodayByAccount 统计商户当天向同一账号同一方式的代付笔数（次数限制用）。
// B1-10/36：1:1 对齐 epay transfer_maxlimit（Transfer.php:50）—— 统计 pay_time>=今日00:00 的同
// account+type 记录数。pay_time 仅 status=1(成功)时写入，故只计【当日已成功】笔数，处理中/失败不占额度。
func (r *TransferRepo) CountTodayByAccount(uid uint, transferType, account string, dayStart time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.Transfer{}).
		Where("uid = ? AND type = ? AND account = ? AND pay_time >= ?", uid, transferType, account, dayStart).
		Count(&n).Error
	return n, err
}

// isDuplicateKey 判断是否为主键/唯一键冲突（MySQL 1062）。
func isDuplicateKey(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
