package repository

import (
	"strconv"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// atoiOrZero 把字符串转 int，失败返回 0（关键词按 ID 精确匹配时用；非数字则匹配不到任何 ID）。
func atoiOrZero(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// AdminRepo 管理员数据访问。
type AdminRepo struct{ db *gorm.DB }

func NewAdminRepo(db *gorm.DB) *AdminRepo { return &AdminRepo{db: db} }

// FindByUsername 按用户名查管理员，未找到返回 gorm.ErrRecordNotFound。
func (r *AdminRepo) FindByUsername(username string) (*model.Admin, error) {
	var a model.Admin
	if err := r.db.Where("username = ?", username).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// MerchantRepo 商户数据访问。
type MerchantRepo struct{ db *gorm.DB }

func NewMerchantRepo(db *gorm.DB) *MerchantRepo { return &MerchantRepo{db: db} }

// FindByUID 按商户号查商户，未找到返回 gorm.ErrRecordNotFound。
func (r *MerchantRepo) FindByUID(uid uint) (*model.Merchant, error) {
	var m model.Merchant
	if err := r.db.Where("uid = ?", uid).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// FindByLoginAccount 按邮箱或手机号查商户（密码登录用）。未找到返回 (nil, nil)。
func (r *MerchantRepo) FindByLoginAccount(account string) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.Where("email = ? OR phone = ?", account, account).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// FindByUIDSafe 按商户号查商户，未找到返回 (nil, nil)（区别于 FindByUID 返回 ErrRecordNotFound）。
func (r *MerchantRepo) FindByUIDSafe(uid uint) (*model.Merchant, error) {
	var m model.Merchant
	err := r.db.Where("uid = ?", uid).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// List 分页查询商户，支持字段模糊搜索与多种状态筛选。
func (r *MerchantRepo) List(q dto.MerchantQuery) ([]model.Merchant, int64, error) {
	allowedCols := map[string]string{
		"uid": "uid", "account": "account", "username": "username",
		"url": "url", "qq": "qq", "phone": "phone", "email": "email",
	}

	tx := r.db.Model(&model.Merchant{})
	if q.Keyword != "" {
		if col, ok := allowedCols[q.Column]; ok {
			tx = tx.Where(col+" LIKE ?", "%"+q.Keyword+"%")
		}
	}
	if q.GID != nil {
		tx = tx.Where("gid = ?", *q.GID)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Pay != nil {
		tx = tx.Where("pay = ?", *q.Pay)
	}
	if q.SettleSt != nil {
		tx = tx.Where("settle = ?", *q.SettleSt)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.Merchant
	err := tx.Order("uid DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindSettleable 取满足自动结算条件的商户：余额 >= 门槛、结算权限开(settle=1)、
// 状态正常(status=1)、已填结算账号+姓名。对齐 epay cron do=settle 的商户筛选。
func (r *MerchantRepo) FindSettleable(minMoney decimal.Decimal, limit int) ([]model.Merchant, error) {
	var list []model.Merchant
	err := r.db.Where("money >= ? AND settle = 1 AND status = 1 AND account <> '' AND username <> ''", minMoney).
		Order("uid ASC").Limit(limit).Find(&list).Error
	return list, err
}

// ChannelRepo 支付通道数据访问。
type ChannelRepo struct{ db *gorm.DB }

func NewChannelRepo(db *gorm.DB) *ChannelRepo { return &ChannelRepo{db: db} }

// List 分页查询通道，支持 名称/ID 关键词、插件模糊、支付方式、状态筛选。
func (r *ChannelRepo) List(q dto.ChannelQuery) ([]model.Channel, int64, error) {
	tx := r.db.Model(&model.Channel{})
	if q.Keyword != "" {
		// ID 精确 OR 名称模糊
		tx = tx.Where("id = ? OR name LIKE ?", atoiOrZero(q.Keyword), "%"+q.Keyword+"%")
	}
	if q.Plugin != "" {
		tx = tx.Where("plugin LIKE ?", "%"+q.Plugin+"%")
	}
	if q.Type != nil && *q.Type > 0 {
		tx = tx.Where("type = ?", *q.Type)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Channel
	err := tx.Order("id ASC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindByID 按主键查通道。未找到返回 (nil, nil)。
func (r *ChannelRepo) FindByID(id uint) (*model.Channel, error) {
	var c model.Channel
	err := r.db.Where("id = ?", id).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Create 新增通道。
func (r *ChannelRepo) Create(c *model.Channel) error {
	return r.db.Create(c).Error
}

// Update 更新通道的可编辑字段（白名单，避免误改 config/status，二者由专用接口维护）。
func (r *ChannelRepo) Update(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Channel{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除通道。
func (r *ChannelRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Channel{}).Error
}

// SetStatus 只更新状态字段。
func (r *ChannelRepo) SetStatus(id uint, status int8) error {
	return r.db.Model(&model.Channel{}).Where("id = ?", id).Update("status", status).Error
}

// SaveConfig 只更新 config（密钥/参数 JSON）字段。
func (r *ChannelRepo) SaveConfig(id uint, config string) error {
	return r.db.Model(&model.Channel{}).Where("id = ?", id).Update("config", config).Error
}

// FindEnabledByType 取某支付方式下第一个已开启通道（下单选通道用，最简策略；
// 轮询/加权等留待 pay_roll 阶段）。未找到返回 (nil, nil)。
func (r *ChannelRepo) FindEnabledByType(typeID int) (*model.Channel, error) {
	var c model.Channel
	err := r.db.Where("type = ? AND status = 1", typeID).Order("id ASC").First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// FindEnabledByPlugin 取某插件下第一个已开启通道（阶段A mock 下单用 plugin 定位）。
func (r *ChannelRepo) FindEnabledByPlugin(plugin string) (*model.Channel, error) {
	var c model.Channel
	err := r.db.Where("plugin = ? AND status = 1", plugin).Order("id ASC").First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// AccountRepo 商户资金账户数据访问：余额变更 + 流水落库，原子事务。
type AccountRepo struct{ db *gorm.DB }

func NewAccountRepo(db *gorm.DB) *AccountRepo { return &AccountRepo{db: db} }

// ChangeUserMoney 变更商户余额并写一条资金流水，事务内完成，行锁防并发。
// 对齐 epay changeUserMoney：add=true 入账(action=1) / false 出账(action=2)，
// money<=0 直接跳过；记录变更前后余额。幂等由调用方(回调改单)保证，这里不再去重。
func (r *AccountRepo) ChangeUserMoney(uid uint, amount decimal.Decimal, add bool, changeType, tradeNo string) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var m model.Merchant
		// 行锁读当前余额
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", uid).First(&m).Error; err != nil {
			return err
		}
		old := m.Money
		var newMoney decimal.Decimal
		var action int8
		if add {
			action = 1
			newMoney = old.Add(amount)
		} else {
			action = 2
			newMoney = old.Sub(amount)
		}
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", uid).
			Update("money", newMoney).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: uid, Action: action, Money: amount,
			OldMoney: old, NewMoney: newMoney,
			Type: changeType, TradeNo: tradeNo, Date: time.Now(),
		}
		return tx.Create(&rec).Error
	})
}

// RecordRepo 资金流水（pay_record）数据访问。
type RecordRepo struct{ db *gorm.DB }

func NewRecordRepo(db *gorm.DB) *RecordRepo { return &RecordRepo{db: db} }

// List 分页查询资金流水，支持 action 与 类型/单号 关键词筛选（商户端按 uid 限定）。
func (r *RecordRepo) List(q dto.RecordQuery) ([]model.PayRecord, int64, error) {
	tx := r.db.Model(&model.PayRecord{})
	if q.UID != nil {
		tx = tx.Where("uid = ?", *q.UID)
	}
	if q.Action != nil {
		tx = tx.Where("action = ?", *q.Action)
	}
	if q.Keyword != "" {
		tx = tx.Where("type LIKE ? OR trade_no LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.PayRecord
	err := tx.Order("id DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// OrderRepo 订单数据访问。
type OrderRepo struct{ db *gorm.DB }

func NewOrderRepo(db *gorm.DB) *OrderRepo { return &OrderRepo{db: db} }

// FindByOut 按 商户号 + 商户订单号 查订单（下单幂等用）。未找到返回 (nil, nil)。
func (r *OrderRepo) FindByOut(uid uint, outTradeNo string) (*model.Order, error) {
	var o model.Order
	err := r.db.Where("uid = ? AND out_trade_no = ?", uid, outTradeNo).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// Create 创建订单。trade_no 唯一索引作为幂等兜底（并发同 out_trade_no 时靠唯一键防重）。
func (r *OrderRepo) Create(o *model.Order) error {
	return r.db.Create(o).Error
}

// FindByTradeNo 按系统订单号查订单。未找到返回 (nil, nil)。
func (r *OrderRepo) FindByTradeNo(tradeNo string) (*model.Order, error) {
	var o model.Order
	err := r.db.Where("trade_no = ?", tradeNo).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// MarkPaid 幂等改单为已支付：仅当当前 status 属于未终态(0未付/4预授权)时才更新。
// 用条件 UPDATE + 影响行数判断并发/重复回调：返回 true 表示本次真正完成了状态翻转。
// 对齐 epay Payment::processOrder 的 `UPDATE ... SET status=1 WHERE ... status IN(0,4)`。
func (r *OrderRepo) MarkPaid(tradeNo, apiTradeNo, buyer string, endTime time.Time) (bool, error) {
	fields := map[string]interface{}{
		"status":   1,
		"end_time": endTime,
	}
	if apiTradeNo != "" {
		fields["api_trade_no"] = apiTradeNo
	}
	if buyer != "" {
		fields["buyer"] = buyer
	}
	res := r.db.Model(&model.Order{}).
		Where("trade_no = ? AND status IN (0, 4)", tradeNo).
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// SavePayInfo 回填下单后的收银台渲染信息（pay_type + 二维码/支付链接）。
func (r *OrderRepo) SavePayInfo(tradeNo, payType, qrCode string) error {
	return r.db.Model(&model.Order{}).Where("trade_no = ?", tradeNo).
		Updates(map[string]interface{}{"pay_type": payType, "qr_code": qrCode}).Error
}

// SetNotifySuccess 通知成功：notify=0，清空重试时间。
func (r *OrderRepo) SetNotifySuccess(tradeNo string) error {
	return r.db.Model(&model.Order{}).Where("trade_no = ?", tradeNo).
		Updates(map[string]interface{}{"notify": 0, "notify_time": nil}).Error
}

// SetNotifyRetry 通知失败：写入下一次重试计数 notify=n 与下次重试时间。
// n<=0 视为放弃（notify=-1，清空重试时间），交由人工/notify2 兜底。
func (r *OrderRepo) SetNotifyRetry(tradeNo string, n int, nextRetry time.Time) error {
	fields := map[string]interface{}{}
	if n <= 0 {
		fields["notify"] = -1
		fields["notify_time"] = nil
	} else {
		fields["notify"] = n
		fields["notify_time"] = nextRetry
	}
	return r.db.Model(&model.Order{}).Where("trade_no = ?", tradeNo).Updates(fields).Error
}

// FindNotifyPending 取待重试通知的订单（notify>0 且到期 且完成时间在近一天内）。
// 对齐 epay cron notify：WHERE notify>0 AND notify_time<NOW() AND endtime 近 1 天，LIMIT。
func (r *OrderRepo) FindNotifyPending(limit int) ([]model.Order, error) {
	var list []model.Order
	err := r.db.Where("notify > 0 AND notify_time IS NOT NULL AND notify_time < ? AND end_time > ?",
		time.Now(), time.Now().Add(-24*time.Hour)).
		Order("notify_time ASC").Limit(limit).Find(&list).Error
	return list, err
}

// CleanExpiredUnpaid 清理超时未支付订单：删除 status=0 且创建时间早于 before 的订单。
// 对齐 epay cron do=order 的 `delete from pre_order where status=0 and addtime<24h前`。
// 未支付订单无资金影响，直接删除；返回清理条数。
func (r *OrderRepo) CleanExpiredUnpaid(before time.Time) (int64, error) {
	res := r.db.Where("status = 0 AND add_time < ?", before).Delete(&model.Order{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// FindUnpaidForReconcile 取待对账的未支付订单（status=0 且创建在最近窗口内，排除 mock）。
// 供定时对账主动向渠道查单。窗口用创建时间避免扫全表。
func (r *OrderRepo) FindUnpaidForReconcile(since time.Time, limit int) ([]model.Order, error) {
	var list []model.Order
	err := r.db.Where("status = 0 AND plugin <> ? AND add_time > ?", "mock", since).
		Order("add_time ASC").Limit(limit).Find(&list).Error
	return list, err
}

// CountPaidByMerchant 统计商户已支付订单数（status=1）。since 非零则只统计该时刻之后创建的。
func (r *OrderRepo) CountPaidByMerchant(uid uint, since time.Time) (int64, error) {
	tx := r.db.Model(&model.Order{}).Where("uid = ? AND status = 1", uid)
	if !since.IsZero() {
		tx = tx.Where("add_time >= ?", since)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// SumPaidMoneyByMerchant 汇总商户在 [start,end) 内已支付订单的收入（按 money 求和）。
func (r *OrderRepo) SumPaidMoneyByMerchant(uid uint, start, end time.Time) (decimal.Decimal, error) {
	var result struct{ Total decimal.Decimal }
	err := r.db.Model(&model.Order{}).
		Select("COALESCE(SUM(money),0) AS total").
		Where("uid = ? AND status = 1 AND add_time >= ? AND add_time < ?", uid, start, end).
		Scan(&result).Error
	return result.Total, err
}

// FindByTradeNoAndUID 按系统订单号 + 商户号查订单（商户端退款/操作校验归属用）。未找到返回 (nil,nil)。
func (r *OrderRepo) FindByTradeNoAndUID(tradeNo string, uid uint) (*model.Order, error) {
	var o model.Order
	err := r.db.Where("trade_no = ? AND uid = ?", tradeNo, uid).First(&o).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// MarkRefunded 幂等退款改单：仅当 status=1(已付) 时转为 2(已退款)，写 refundmoney。
// 条件 UPDATE + 影响行数判重复退款：返回 true 表示本次真正执行了退款。
func (r *OrderRepo) MarkRefunded(tradeNo string, refundMoney decimal.Decimal) (bool, error) {
	res := r.db.Model(&model.Order{}).
		Where("trade_no = ? AND status = 1", tradeNo).
		Updates(map[string]interface{}{"status": 2, "refund_money": refundMoney})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// List 分页查询订单，支持按 column/keyword 模糊搜索与状态筛选。
func (r *OrderRepo) List(q dto.OrderQuery) ([]model.Order, int64, error) {
	// 白名单，防 SQL 注入到列名
	allowedCols := map[string]bool{
		"trade_no": true, "out_trade_no": true, "api_trade_no": true,
		"name": true, "money": true, "realmoney": true, "domain": true,
	}

	tx := r.db.Model(&model.Order{})
	if q.UID != nil {
		tx = tx.Where("uid = ?", *q.UID)
	}
	if q.Keyword != "" && allowedCols[q.Column] {
		tx = tx.Where(q.Column+" LIKE ?", "%"+q.Keyword+"%")
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.Order
	err := tx.Order("add_time DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
