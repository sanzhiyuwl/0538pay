package repository

import (
	"time"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProfitRepo 分账数据访问：规则匹配 + 分账订单 CRUD + 成功扣款/取消退回的原子事务。
type ProfitRepo struct{ db *gorm.DB }

func NewProfitRepo(db *gorm.DB) *ProfitRepo { return &ProfitRepo{db: db} }

// MatchRule 下单时按优先级匹配一条已开启(status=1)的分账规则（对齐 epay updateOrderProfits）：
// ① channel+uid+subchannel ② channel+uid ③ channel + uid IS NULL(通道级全局)。未命中返回 (nil,nil)。
func (r *ProfitRepo) MatchRule(channelID, subChannel int, uid uint) (*model.ProfitReceiver, error) {
	// ① 最精确：channel + uid + subchannel
	if subChannel > 0 {
		var rule model.ProfitReceiver
		err := r.db.Where("status = 1 AND channel = ? AND uid = ? AND sub_channel = ?", channelID, uid, subChannel).
			Order("id ASC").First(&rule).Error
		if err == nil {
			return &rule, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	// ② channel + uid
	var rule model.ProfitReceiver
	err := r.db.Where("status = 1 AND channel = ? AND uid = ?", channelID, uid).
		Order("id ASC").First(&rule).Error
	if err == nil {
		return &rule, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	// ③ 通道级全局（uid IS NULL）
	err = r.db.Where("status = 1 AND channel = ? AND uid IS NULL", channelID).
		Order("id ASC").First(&rule).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// FindRule 按 id 查规则。未找到返回 (nil,nil)。
func (r *ProfitRepo) FindRule(id uint) (*model.ProfitReceiver, error) {
	var rule model.ProfitReceiver
	err := r.db.Where("id = ?", id).First(&rule).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// CreateRule 新增分账规则。
func (r *ProfitRepo) CreateRule(rule *model.ProfitReceiver) error {
	return r.db.Create(rule).Error
}

// ListRules 列出全部规则（供 seed 校验/后续管理页；当前无独立分页需求）。
func (r *ProfitRepo) ListRules() ([]model.ProfitReceiver, error) {
	var list []model.ProfitReceiver
	err := r.db.Order("id ASC").Find(&list).Error
	return list, err
}

// UpdateRule 更新分账规则的可编辑字段（channel/subchannel/uid/account/name/rate/minmoney）。
func (r *ProfitRepo) UpdateRule(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.ProfitReceiver{}).Where("id = ?", id).Updates(fields).Error
}

// SetRuleStatus 切换规则开关（0关/1开）。
func (r *ProfitRepo) SetRuleStatus(id uint, status int8) error {
	return r.db.Model(&model.ProfitReceiver{}).Where("id = ?", id).Update("status", status).Error
}

// DeleteRule 删除分账规则。
func (r *ProfitRepo) DeleteRule(id uint) error {
	return r.db.Delete(&model.ProfitReceiver{}, id).Error
}

// RuleExistsByChannelUID 判断某 channel+uid 是否已存在规则（对齐 epay：每次支付只能给1人分账，唯一约束）。
// uid 为 nil 表示通道级全局（uid IS NULL）。excludeID>0 时排除自身（编辑用）。
func (r *ProfitRepo) RuleExistsByChannelUID(channelID int, uid *uint, excludeID uint) (bool, error) {
	tx := r.db.Model(&model.ProfitReceiver{}).Where("channel = ?", channelID)
	if uid == nil {
		tx = tx.Where("uid IS NULL")
	} else {
		tx = tx.Where("uid = ?", *uid)
	}
	if excludeID > 0 {
		tx = tx.Where("id != ?", excludeID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n > 0, err
}

// CreateOrder 创建分账订单（支付成功回调触发）。
func (r *ProfitRepo) CreateOrder(o *model.ProfitOrder) error {
	return r.db.Create(o).Error
}

// ExistOrderByTradeNo 判断某订单是否已生成分账单（回调幂等，避免重复回调重复建分账）。
func (r *ProfitRepo) ExistOrderByTradeNo(tradeNo string) (bool, error) {
	var n int64
	err := r.db.Model(&model.ProfitOrder{}).Where("trade_no = ?", tradeNo).Count(&n).Error
	return n > 0, err
}

// ListPendingIDs 取到期可提交的待分账(status=0)订单 id 列表（自动执行用），按 id 升序，限 limit 条。
// B1-42/B1-70：加提交时间窗口，1:1 对齐 epay ProfitSharing\CommUtil::task()：
//
//	(add_time<=NOW-60秒 AND delay=0) OR (add_time<=NOW-24小时 AND delay=1)
//
// 即普通分账(delay=0)建单后需等 60 秒冷却、延迟分账(delay=1)需等 24 小时才可提交，
// 避免订单尚未稳定(如可能退款)就分账。时间基准用 DB 的 NOW() 与 epay 对齐。
func (r *ProfitRepo) ListPendingIDs(limit int) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&model.ProfitOrder{}).
		Where("status = 0").
		Where("(add_time <= DATE_SUB(NOW(), INTERVAL 60 SECOND) AND delay = 0) OR (add_time <= DATE_SUB(NOW(), INTERVAL 24 HOUR) AND delay = 1)").
		Order("id ASC").Limit(limit).Pluck("id", &ids).Error
	return ids, err
}

// FindOrder 按 id 查分账订单。未找到返回 (nil,nil)。
func (r *ProfitRepo) FindOrder(id uint) (*model.ProfitOrder, error) {
	var o model.ProfitOrder
	err := r.db.Where("id = ?", id).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// List 分页查询分账订单，支持 rid/status/column+value/时间范围筛选。
func (r *ProfitRepo) List(q dto.PsOrderQuery) ([]model.ProfitOrder, int64, error) {
	tx := r.profitFilters(q)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.ProfitOrder
	err := tx.Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// profitFilters 套用列表/统计共用的筛选条件。
func (r *ProfitRepo) profitFilters(q dto.PsOrderQuery) *gorm.DB {
	tx := r.db.Model(&model.ProfitOrder{})
	if q.RID != nil {
		tx = tx.Where("rid = ?", *q.RID)
	}
	if q.Status != nil && *q.Status >= 0 {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Value != "" {
		allowed := map[string]bool{"trade_no": true, "api_trade_no": true, "money": true}
		if allowed[q.Column] {
			tx = tx.Where(q.Column+" LIKE ?", "%"+q.Value+"%")
		}
	}
	if t, ok := parseDate(q.StartTime); ok {
		tx = tx.Where("add_time >= ?", t)
	}
	if t, ok := parseDate(q.EndTime); ok {
		tx = tx.Where("add_time < ?", t.Add(24*time.Hour))
	}
	return tx
}

// Stats 当前筛选条件下的分账统计（总额/成功额/失败额 + 各计数）。
func (r *ProfitRepo) Stats(q dto.PsOrderQuery) (totalMoney, successMoney, failMoney decimal.Decimal, totalCnt, successCnt, failCnt int64, err error) {
	type agg struct {
		Status int8
		Sum    decimal.Decimal
		Cnt    int64
	}
	var rows []agg
	err = r.profitFilters(q).
		Select("status, COALESCE(SUM(money),0) AS sum, COUNT(*) AS cnt").
		Group("status").Scan(&rows).Error
	if err != nil {
		return
	}
	for _, row := range rows {
		totalMoney = totalMoney.Add(row.Sum)
		totalCnt += row.Cnt
		switch row.Status {
		case 2:
			successMoney, successCnt = row.Sum, row.Cnt
		case 3:
			failMoney, failCnt = row.Sum, row.Cnt
		}
	}
	return
}

// UpdateOrder 更新分账订单字段（状态流转，白名单 map）。
func (r *ProfitRepo) UpdateOrder(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.ProfitOrder{}).Where("id = ?", id).Updates(fields).Error
}

// DeleteOrder 删除分账订单。
func (r *ProfitRepo) DeleteOrder(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.ProfitOrder{}).Error
}

// MarkSuccessWithDebit 分账成功：置 status=2；若规则绑定商户(PsUID 非空)且未扣过，
// 则从该商户余额扣分账金额并写流水（'订单分账'），标记 Debited=1。事务内完成，条件 UPDATE 防重复。
// 对齐 epay：分账成功 changeUserMoney(psuid, money, false, '订单分账')。返回是否本次真正成功翻转。
func (r *ProfitRepo) MarkSuccessWithDebit(id uint, settleNo string) (bool, error) {
	var flipped bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 仅 status ∈ {0,1}(待分账/已提交) 可翻成功，条件 UPDATE 防并发重复
		res := tx.Model(&model.ProfitOrder{}).
			Where("id = ? AND status IN (0,1)", id).
			Updates(map[string]interface{}{"status": 2, "settle_no": settleNo, "result": ""})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil
		}
		flipped = true
		var o model.ProfitOrder
		if err := tx.Where("id = ?", id).First(&o).Error; err != nil {
			return err
		}
		// 需扣款：规则绑定商户 + 尚未扣过 + 金额>0
		if o.PsUID == nil || *o.PsUID == 0 || o.Debited == 1 || o.Money.LessThanOrEqual(decimal.Zero) {
			return nil
		}
		// B1-44：扣款与否在【执行期】按当时通道 mode==0 现场判定（对齐 epay process_item），
		// 而非建单期定死。mode!=0（商户直清）不从商户余额扣分账，仅置成功。
		mode, err := currentOrderChannelMode(tx, o.TradeNo)
		if err != nil {
			return err
		}
		if mode != 0 {
			return nil
		}
		if err := debitMerchant(tx, *o.PsUID, o.Money, "订单分账", o.TradeNo); err != nil {
			return err
		}
		return tx.Model(&model.ProfitOrder{}).Where("id = ?", id).Update("debited", 1).Error
	})
	return flipped, err
}

// CancelOrRefund 取消(status 0/3→4)或回退(status 2→4)分账订单。
// 若之前已扣商户余额(Debited=1)，则解冻退回该商户并写流水（'分账退回'），清 Debited。
// 事务内完成，条件 UPDATE 防重复退回。fromStatuses 指定允许的当前状态。返回是否本次真正翻转。
func (r *ProfitRepo) CancelOrRefund(id uint, fromStatuses []int, changeType string) (bool, error) {
	var flipped bool
	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.ProfitOrder{}).
			Where("id = ? AND status IN ?", id, fromStatuses).
			Update("status", 4)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil
		}
		flipped = true
		var o model.ProfitOrder
		if err := tx.Where("id = ?", id).First(&o).Error; err != nil {
			return err
		}
		// 已扣款则解冻退回
		if o.Debited != 1 || o.PsUID == nil || *o.PsUID == 0 || o.Money.LessThanOrEqual(decimal.Zero) {
			return nil
		}
		if err := creditMerchant(tx, *o.PsUID, o.Money, changeType, o.TradeNo); err != nil {
			return err
		}
		return tx.Model(&model.ProfitOrder{}).Where("id = ?", id).Update("debited", 0).Error
	})
	return flipped, err
}

// currentOrderChannelMode 事务内按订单号解析当前所属通道的 mode（0=平台代收 1=商户直清）。
// 对齐 epay process_item：subchannel>0 用 getSub、否则 get(channel)，两者 mode 均取自父通道
//（pay_subchannel 无独立 mode），故直接查订单 channel 对应 pay_channel.mode 即可 1:1。
// 订单/通道查不到时返回 mode=1（非代收），保守不扣商户余额，避免误扣。
func currentOrderChannelMode(tx *gorm.DB, tradeNo string) (int8, error) {
	var channelID int
	err := tx.Model(&model.Order{}).Where("trade_no = ?", tradeNo).
		Limit(1).Pluck("channel", &channelID).Error
	if err != nil {
		return 1, err
	}
	if channelID <= 0 {
		return 1, nil
	}
	var ch model.Channel
	err = tx.Select("id, mode").Where("id = ?", channelID).First(&ch).Error
	if err == gorm.ErrRecordNotFound {
		return 1, nil // 通道已删/查不到：保守不扣
	}
	if err != nil {
		return 1, err
	}
	return ch.Mode, nil
}

// debitMerchant 事务内从商户余额扣款并写出账流水（行锁+余额校验）。
func debitMerchant(tx *gorm.DB, uid uint, amount decimal.Decimal, changeType, tradeNo string) error {
	var m model.Merchant
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uid = ?", uid).First(&m).Error; err != nil {
		return err
	}
	if m.Money.LessThan(amount) {
		return ErrInsufficientBalance
	}
	newMoney := m.Money.Sub(amount)
	if err := tx.Model(&model.Merchant{}).Where("uid = ?", uid).Update("money", newMoney).Error; err != nil {
		return err
	}
	rec := model.PayRecord{
		UID: uid, Action: 2, Money: amount,
		OldMoney: m.Money, NewMoney: newMoney,
		Type: changeType, TradeNo: tradeNo, Date: time.Now(),
	}
	return tx.Create(&rec).Error
}

// creditMerchant 事务内向商户余额退回并写入账流水（行锁）。
func creditMerchant(tx *gorm.DB, uid uint, amount decimal.Decimal, changeType, tradeNo string) error {
	var m model.Merchant
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uid = ?", uid).First(&m).Error; err != nil {
		return err
	}
	newMoney := m.Money.Add(amount)
	if err := tx.Model(&model.Merchant{}).Where("uid = ?", uid).Update("money", newMoney).Error; err != nil {
		return err
	}
	rec := model.PayRecord{
		UID: uid, Action: 1, Money: amount,
		OldMoney: m.Money, NewMoney: newMoney,
		Type: changeType, TradeNo: tradeNo, Date: time.Now(),
	}
	return tx.Create(&rec).Error
}
