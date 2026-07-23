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

// parseDate 解析 yyyy-mm-dd 日期（时间范围筛选用）。空串或格式不符返回 ok=false。
func parseDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
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

// FindByID 按主键查管理员（代付发起时二次校验密码用），未找到返回 gorm.ErrRecordNotFound。
func (r *AdminRepo) FindByID(id uint) (*model.Admin, error) {
	var a model.Admin
	if err := r.db.Where("id = ?", id).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// List 全部管理员（按 id 升序；管理员数量少，不分页）。
func (r *AdminRepo) List() ([]model.Admin, error) {
	var list []model.Admin
	err := r.db.Order("id ASC").Find(&list).Error
	return list, err
}

// CountByUsername 统计用户名出现次数（新增/改名查重）。exceptID>0 时排除该 id 自身。
func (r *AdminRepo) CountByUsername(username string, exceptID uint) (int64, error) {
	tx := r.db.Model(&model.Admin{}).Where("username = ?", username)
	if exceptID > 0 {
		tx = tx.Where("id <> ?", exceptID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// Create 新增管理员。
func (r *AdminRepo) Create(a *model.Admin) error { return r.db.Create(a).Error }

// UpdateFields 更新管理员的白名单字段（避免误改密码/主键）。
func (r *AdminRepo) UpdateFields(id uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Admin{}).Where("id = ?", id).Updates(fields).Error
}

// UpdatePassword 只更新密码哈希。
func (r *AdminRepo) UpdatePassword(id uint, hash string) error {
	return r.db.Model(&model.Admin{}).Where("id = ?", id).Update("password", hash).Error
}

// SetStatus 只更新启停状态。
func (r *AdminRepo) SetStatus(id uint, status int8) error {
	return r.db.Model(&model.Admin{}).Where("id = ?", id).Update("status", status).Error
}

// Delete 删除管理员。
func (r *AdminRepo) Delete(id uint) error {
	return r.db.Where("id = ?", id).Delete(&model.Admin{}).Error
}

// Count 管理员总数（删除最后一个超管前的护栏用）。
func (r *AdminRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&model.Admin{}).Count(&n).Error
	return n, err
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

// UpdateFields 更新商户指定字段（白名单 map，供资料/密钥/密码更新）。
func (r *MerchantRepo) UpdateFields(uid uint, fields map[string]interface{}) error {
	return r.db.Model(&model.Merchant{}).Where("uid = ?", uid).Updates(fields).Error
}

// Create 新增商户。UID 由 GORM 自增回填（起始由建表 AUTO_INCREMENT 决定）。
func (r *MerchantRepo) Create(m *model.Merchant) error {
	return r.db.Create(m).Error
}

// Delete 删除商户。
func (r *MerchantRepo) Delete(uid uint) error {
	return r.db.Where("uid = ?", uid).Delete(&model.Merchant{}).Error
}

// CountByPhone 统计手机号占用数（新增查重用）。excludeUID>0 时排除自身。
func (r *MerchantRepo) CountByPhone(phone string, excludeUID uint) (int64, error) {
	tx := r.db.Model(&model.Merchant{}).Where("phone = ?", phone)
	if excludeUID > 0 {
		tx = tx.Where("uid <> ?", excludeUID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// CountByEmail 统计邮箱占用数（新增查重用）。excludeUID>0 时排除自身。
func (r *MerchantRepo) CountByEmail(email string, excludeUID uint) (int64, error) {
	tx := r.db.Model(&model.Merchant{}).Where("email = ?", email)
	if excludeUID > 0 {
		tx = tx.Where("uid <> ?", excludeUID)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// ResetGroupToDefault 把某用户组下的所有商户回落默认组（删组级联，对齐 epay delGroup）。
func (r *MerchantRepo) ResetGroupToDefault(gid int) error {
	return r.db.Model(&model.Merchant{}).Where("gid = ?", gid).
		Updates(map[string]interface{}{"gid": 0, "group_end": nil}).Error
}

// CountByGroup 统计各用户组的商户数（用户组列表展示覆盖数用）。返回 gid→count。
func (r *MerchantRepo) CountByGroup() (map[int]int64, error) {
	type row struct {
		GID int
		Cnt int64
	}
	var rows []row
	if err := r.db.Model(&model.Merchant{}).
		Select("gid, COUNT(*) AS cnt").Group("gid").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, x := range rows {
		out[x.GID] = x.Cnt
	}
	return out, nil
}

// FindSettleable 取满足自动结算条件的商户：余额 >= 门槛、结算权限开(settle=1)、
// 状态正常(status=1)、已填结算账号+姓名。对齐 epay cron do=settle 的商户筛选。
func (r *MerchantRepo) FindSettleable(minMoney decimal.Decimal, limit int) ([]model.Merchant, error) {
	var list []model.Merchant
	err := r.db.Where("money >= ? AND settle = 1 AND status = 1 AND account <> '' AND username <> ''", minMoney).
		Order("uid ASC").Limit(limit).Find(&list).Error
	return list, err
}

// FindByOAuth 按第三方 openid 查商户（快捷登录）。col 限 qq_uid/wx_uid/alipay_uid。未找到 (nil,nil)。
func (r *MerchantRepo) FindByOAuth(col, openid string) (*model.Merchant, error) {
	allowed := map[string]bool{"qq_uid": true, "wx_uid": true, "alipay_uid": true}
	if !allowed[col] || openid == "" {
		return nil, nil
	}
	var m model.Merchant
	err := r.db.Where(col+" = ?", openid).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// BindOAuth 把第三方 openid 绑定到指定商户（col 限 qq_uid/wx_uid/alipay_uid）。
func (r *MerchantRepo) BindOAuth(uid uint, col, openid string) error {
	allowed := map[string]bool{"qq_uid": true, "wx_uid": true, "alipay_uid": true}
	if !allowed[col] {
		return nil
	}
	return r.db.Model(&model.Merchant{}).Where("uid = ?", uid).Update(col, openid).Error
}

// CountByUpID 统计某商户名下已邀请的下级商户数（upid=uid）。对齐 epay 邀请人数统计。
func (r *MerchantRepo) CountByUpID(uid uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.Merchant{}).Where("upid = ?", uid).Count(&n).Error
	return n, err
}

// ListByUpID 分页列出某商户名下已邀请的下级商户（按注册时间倒序）。
func (r *MerchantRepo) ListByUpID(uid uint, page, pageSize int) ([]model.Merchant, int64, error) {
	tx := r.db.Model(&model.Merchant{}).Where("upid = ?", uid)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Merchant
	err := tx.Order("uid DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
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

// SetDayStatus 只更新 daystatus 字段（单日限额触发暂停/次日重置用；对齐 epay daystatus）。
func (r *ChannelRepo) SetDayStatus(id uint, daystatus int8) error {
	return r.db.Model(&model.Channel{}).Where("id = ?", id).Update("daystatus", daystatus).Error
}

// ResetAllDayStatus 把所有通道 daystatus 重置为 0（cron 每日重置，对齐 epay cron.php:152）。
func (r *ChannelRepo) ResetAllDayStatus() (int64, error) {
	res := r.db.Model(&model.Channel{}).Where("daystatus <> 0").Update("daystatus", 0)
	return res.RowsAffected, res.Error
}

// SaveConfig 只更新 config（密钥/参数 JSON）字段。
func (r *ChannelRepo) SaveConfig(id uint, config string) error {
	return r.db.Model(&model.Channel{}).Where("id = ?", id).Update("config", config).Error
}

// CountByType 统计使用某支付方式(type)的通道数（删支付方式前的引用校验，对齐 epay delPayType）。
func (r *ChannelRepo) CountByType(typeID int) (int64, error) {
	var n int64
	err := r.db.Model(&model.Channel{}).Where("type = ?", typeID).Count(&n).Error
	return n, err
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

// ListEnabledByType 取某支付方式下全部已开启且未达单日限额的通道（用户组 -1 随机分配用）。
// 对齐 epay getSubmitInfo -1 分支的 `SELECT ... WHERE type AND status=1 AND daystatus=0`。
func (r *ChannelRepo) ListEnabledByType(typeID int) ([]model.Channel, error) {
	var list []model.Channel
	err := r.db.Where("type = ? AND status = 1 AND daystatus = 0", typeID).Order("id ASC").Find(&list).Error
	return list, err
}

// FindManyByIDs 按 ID 列表批量查通道（轮询组内成员金额过滤用）。返回 id→通道。
func (r *ChannelRepo) FindManyByIDs(ids []int) (map[int]*model.Channel, error) {
	out := map[int]*model.Channel{}
	if len(ids) == 0 {
		return out, nil
	}
	var list []model.Channel
	if err := r.db.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		out[int(list[i].ID)] = &list[i]
	}
	return out, nil
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
		// B1-23：type=='订单退款' 记录级幂等（对齐 epay changeUserMoney:678-681）——
		// 同 trade_no 已有一条'订单退款'流水则整单不再二次扣款（epay 语义：一个订单退款只扣一次）。
		if changeType == "订单退款" && tradeNo != "" {
			var cnt int64
			if err := tx.Model(&model.PayRecord{}).
				Where("type = ? AND trade_no = ?", "订单退款", tradeNo).Count(&cnt).Error; err != nil {
				return err
			}
			if cnt > 0 {
				return nil
			}
		}
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

// DepositFromBalance 保证金充值(余额支付路径)：从 money 扣 amount 转入 deposit，事务内完成。
// 对齐 epay deposit_recharge typeid=0：changeUserMoney(减,'充值保证金') + deposit 累加。
// 余额不足返回 ErrInsufficientBalance。写一条余额减少流水（type=充值保证金）。
func (r *AccountRepo) DepositFromBalance(uid uint, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", uid).First(&m).Error; err != nil {
			return err
		}
		if m.Money.LessThan(amount) {
			return ErrInsufficientBalance
		}
		newMoney := m.Money.Sub(amount)
		newDeposit := m.Deposit.Add(amount)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", uid).
			Updates(map[string]interface{}{"money": newMoney, "deposit": newDeposit}).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: uid, Action: 2, Money: amount,
			OldMoney: m.Money, NewMoney: newMoney,
			Type: "充值保证金", Date: time.Now(),
		}
		return tx.Create(&rec).Error
	})
}

// DepositWithdraw 保证金提取：从 deposit 扣 amount 转回 money，事务内完成。
// 对齐 epay deposit_withdraw：deposit 减 + changeUserMoney(加,'提取保证金')。
// 保证金不足返回 ErrInsufficientDeposit。写一条余额增加流水。
func (r *AccountRepo) DepositWithdraw(uid uint, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", uid).First(&m).Error; err != nil {
			return err
		}
		if m.Deposit.LessThan(amount) {
			return ErrInsufficientDeposit
		}
		newDeposit := m.Deposit.Sub(amount)
		newMoney := m.Money.Add(amount)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", uid).
			Updates(map[string]interface{}{"money": newMoney, "deposit": newDeposit}).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: uid, Action: 1, Money: amount,
			OldMoney: m.Money, NewMoney: newMoney,
			Type: "提取保证金", Date: time.Now(),
		}
		return tx.Create(&rec).Error
	})
}

// BuyGroupWithBalance 余额支付购买会员：从 money 扣 price 并改用户组 gid + 到期时间，事务内完成。
// 对齐 epay groupbuy typeid=0：changeUserMoney(减,'购买会员') + changeUserGroup。
// 余额不足返回 ErrInsufficientBalance。endTime 为 nil 表示永久组。写一条余额减少流水。
func (r *AccountRepo) BuyGroupWithBalance(uid uint, price decimal.Decimal, gid int, endTime *time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", uid).First(&m).Error; err != nil {
			return err
		}
		if m.Money.LessThan(price) {
			return ErrInsufficientBalance
		}
		newMoney := m.Money.Sub(price)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", uid).
			Updates(map[string]interface{}{"money": newMoney, "gid": gid, "group_end": endTime}).Error; err != nil {
			return err
		}
		if price.GreaterThan(decimal.Zero) {
			rec := model.PayRecord{
				UID: uid, Action: 2, Money: price,
				OldMoney: m.Money, NewMoney: newMoney,
				Type: "购买会员", Date: time.Now(),
			}
			return tx.Create(&rec).Error
		}
		return nil
	})
}

// ChangeUserGroup 改用户组 + 到期时间（B1-46 tid=4 渠道支付回调路径，对齐 epay changeUserGroup:697-700）。
// 钱已从渠道收取，此处只改组不扣余额、不写流水。endTime 为 nil 表示永久组。
func (r *AccountRepo) ChangeUserGroup(uid uint, gid int, endTime *time.Time) error {
	return r.db.Model(&model.Merchant{}).Where("uid = ?", uid).
		Updates(map[string]interface{}{"gid": gid, "group_end": endTime}).Error
}

// DepositAdd 保证金累加（B1-46 tid=5 渠道支付回调路径，对齐 epay processOrder:607-611 deposit=deposit+money）。
// 钱从渠道收取转入保证金，deposit 累加 amount 并写一条收入流水（type=充值保证金），不动 money 余额。
func (r *AccountRepo) DepositAdd(uid uint, amount decimal.Decimal, tradeNo string) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		var m model.Merchant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uid = ?", uid).First(&m).Error; err != nil {
			return err
		}
		newDeposit := m.Deposit.Add(amount)
		if err := tx.Model(&model.Merchant{}).Where("uid = ?", uid).
			Update("deposit", newDeposit).Error; err != nil {
			return err
		}
		rec := model.PayRecord{
			UID: uid, Action: 1, Money: amount,
			OldMoney: m.Money, NewMoney: m.Money,
			Type: "充值保证金", TradeNo: tradeNo, Date: time.Now(),
		}
		return tx.Create(&rec).Error
	})
}

// RecordRepo 资金流水（pay_record）数据访问。
type RecordRepo struct{ db *gorm.DB }

func NewRecordRepo(db *gorm.DB) *RecordRepo { return &RecordRepo{db: db} }

// recordFilters 把 RecordQuery 的筛选条件套到查询上（列表与统计共用，保证口径一致）。
// column+value 走字段白名单模糊；时间范围按 date 列 [start, end+1d) 半开区间。
func (r *RecordRepo) recordFilters(q dto.RecordQuery) *gorm.DB {
	tx := r.db.Model(&model.PayRecord{})
	if q.UID != nil {
		tx = tx.Where("uid = ?", *q.UID)
	}
	if q.Action != nil {
		tx = tx.Where("action = ?", *q.Action)
	}
	if q.Type != "" {
		tx = tx.Where("type = ?", q.Type)
	}
	if q.Keyword != "" {
		tx = tx.Where("type LIKE ? OR trade_no LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	if q.Value != "" {
		allowed := map[string]bool{"type": true, "money": true, "trade_no": true}
		if allowed[q.Column] {
			tx = tx.Where(q.Column+" LIKE ?", "%"+q.Value+"%")
		}
	}
	if t, ok := parseDate(q.StartTime); ok {
		tx = tx.Where("date >= ?", t)
	}
	if t, ok := parseDate(q.EndTime); ok {
		tx = tx.Where("date < ?", t.Add(24*time.Hour)) // 含结束日当天
	}
	return tx
}

// List 分页查询资金流水，支持 action / 类型 / column+value / 时间范围筛选（商户端按 uid 限定）。
func (r *RecordRepo) List(q dto.RecordQuery) ([]model.PayRecord, int64, error) {
	tx := r.recordFilters(q)
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

// Stats 在当前筛选条件下汇总增/减金额与笔数（对齐 epay record_stats）。
func (r *RecordRepo) Stats(q dto.RecordQuery) (incMoney, decMoney decimal.Decimal, incCount, decCount int64, err error) {
	type agg struct {
		Action int8
		Sum    decimal.Decimal
		Cnt    int64
	}
	var rows []agg
	err = r.recordFilters(q).
		Select("action, COALESCE(SUM(money),0) AS sum, COUNT(*) AS cnt").
		Group("action").Scan(&rows).Error
	if err != nil {
		return
	}
	for _, row := range rows {
		if row.Action == 1 {
			incMoney, incCount = row.Sum, row.Cnt
		} else if row.Action == 2 {
			decMoney, decCount = row.Sum, row.Cnt
		}
	}
	return
}

// SumInviteReward 汇总某商户"邀请返现"流水：今日/昨日/累计。对齐 epay inviteStat。
func (r *RecordRepo) SumInviteReward(uid uint) (today, yesterday, total decimal.Decimal, err error) {
	sum := func(tx *gorm.DB) (decimal.Decimal, error) {
		var v decimal.Decimal
		e := tx.Model(&model.PayRecord{}).
			Where("uid = ? AND type = ?", uid, "邀请返现").
			Select("COALESCE(SUM(money),0)").Scan(&v).Error
		return v, e
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	ydayStart := todayStart.AddDate(0, 0, -1)

	if total, err = sum(r.db); err != nil {
		return
	}
	if today, err = sum(r.db.Where("date >= ?", todayStart)); err != nil {
		return
	}
	yesterday, err = sum(r.db.Where("date >= ? AND date < ?", ydayStart, todayStart))
	return
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

// SumPaidMoneyByChannel 汇总 [start,end) 内各通道已支付订单的 money 总额（通道列表今/昨收款列用）。
// 返回 channelID → 金额。对齐 epay pay_channel.php 刷新时按通道聚合。
func (r *OrderRepo) SumPaidMoneyByChannel(start, end time.Time) (map[int]decimal.Decimal, error) {
	type row struct {
		Channel int
		Total   decimal.Decimal
	}
	var rows []row
	err := r.db.Model(&model.Order{}).
		Select("channel, COALESCE(SUM(money),0) AS total").
		Where("status = 1 AND channel > 0 AND add_time >= ? AND add_time < ?", start, end).
		Group("channel").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int]decimal.Decimal, len(rows))
	for _, x := range rows {
		m[x.Channel] = x.Total
	}
	return m, nil
}

// SumTodayPaidRealMoneyByChannel 汇总某通道「今日」已付订单的实付额(realmoney，为空退回 money)。
// 用于 daytop 单日限额累计（对齐 epay functions.php 的 daytop 缓存累计 realmoney）。
func (r *OrderRepo) SumTodayPaidRealMoneyByChannel(channelID int, dayStart, dayEnd time.Time) (decimal.Decimal, error) {
	var result struct{ Total decimal.Decimal }
	err := r.db.Model(&model.Order{}).
		Select("COALESCE(SUM(COALESCE(NULLIF(realmoney,0), money)),0) AS total").
		Where("channel = ? AND status = 1 AND add_time >= ? AND add_time < ?", channelID, dayStart, dayEnd).
		Scan(&result).Error
	return result.Total, err
}

// ChannelPaidRate 统计各通道在 [start,end) 内的订单总数与已付数（成功率列用）。返回 channelID → (total,paid)。
func (r *OrderRepo) ChannelPaidRate(start, end time.Time) (map[int][2]int64, error) {
	type row struct {
		Channel int
		Total   int64
		Paid    int64
	}
	var rows []row
	err := r.db.Model(&model.Order{}).
		Select("channel, COUNT(*) AS total, SUM(CASE WHEN status=1 THEN 1 ELSE 0 END) AS paid").
		Where("channel > 0 AND add_time >= ? AND add_time < ?", start, end).
		Group("channel").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int][2]int64, len(rows))
	for _, x := range rows {
		m[x.Channel] = [2]int64{x.Total, x.Paid}
	}
	return m, nil
}

// SetProfitMoney 写订单平台利润（支付成功入账后，对齐 epay processOrder 写 profitmoney）。
func (r *OrderRepo) SetProfitMoney(tradeNo string, profit decimal.Decimal) error {
	return r.db.Model(&model.Order{}).Where("trade_no = ?", tradeNo).Update("profit_money", profit).Error
}

// BackfillCallbackFields 重复回调时补写订单缺失的 api_trade_no/buyer/bill_trade_no（A-10，
// 对齐 epay processOrder：订单已终态但缺这些字段时补写，仅填当前为空的列）。
func (r *OrderRepo) BackfillCallbackFields(tradeNo, apiTradeNo, buyer, billTradeNo string) error {
	fields := map[string]interface{}{}
	if apiTradeNo != "" {
		fields["api_trade_no"] = apiTradeNo
	}
	if buyer != "" {
		fields["buyer"] = buyer
	}
	if billTradeNo != "" {
		fields["bill_trade_no"] = billTradeNo
	}
	if len(fields) == 0 {
		return nil
	}
	// 只补当前为空的列：用 COALESCE(NULLIF(col,''), :val) 语义——这里用条件 UPDATE 逐列判空。
	return r.db.Transaction(func(tx *gorm.DB) error {
		var o model.Order
		if err := tx.Where("trade_no = ?", tradeNo).First(&o).Error; err != nil {
			return err
		}
		upd := map[string]interface{}{}
		if apiTradeNo != "" && o.APITradeNo == "" {
			upd["api_trade_no"] = apiTradeNo
		}
		if buyer != "" && o.Buyer == "" {
			upd["buyer"] = buyer
		}
		if billTradeNo != "" && o.BillTradeNo == "" {
			upd["bill_trade_no"] = billTradeNo
		}
		if len(upd) == 0 {
			return nil
		}
		return tx.Model(&model.Order{}).Where("trade_no = ?", tradeNo).Updates(upd).Error
	})
}

// SavePayInfo 回填下单后的收银台渲染信息（pay_type + 二维码/支付链接）。
func (r *OrderRepo) SavePayInfo(tradeNo, payType, qrCode string) error {
	return r.db.Model(&model.Order{}).Where("trade_no = ?", tradeNo).
		Updates(map[string]interface{}{"pay_type": payType, "qr_code": qrCode}).Error
}

// UpdateChannelInfo 回填裸单(收银台空 type 单)选定通道后的下单信息（B1-04）：
// 通道/子通道/插件/支付方式名/费率结果 realmoney、getmoney + 命中的分账规则 profits。
// 仅允许对未支付(status=0)单更新，避免并发下覆盖已翻转单。
func (r *OrderRepo) UpdateChannelInfo(tradeNo, plugin, payType string, channelID, subchannelID int, realMoney, getMoney decimal.Decimal, profits uint) error {
	return r.db.Model(&model.Order{}).Where("trade_no = ? AND status = 0", tradeNo).
		Updates(map[string]interface{}{
			"channel":    channelID,
			"subchannel": subchannelID,
			"plugin":     plugin,
			"type_name":  payType,
			"real_money": realMoney,
			"get_money":  getMoney,
			"profits":    profits,
		}).Error
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

// FindNotifyPending 取待重试通知的订单（notify>0 且到期 且完成时间在近日历天窗内）。
// 对齐 epay cron notify（cron.php:158）：WHERE (TO_DAYS(NOW())-TO_DAYS(endtime)<=1) AND notify>0 AND notifytime<NOW()。
// B1-71：完成窗口用【日历天差<=1】(endtime 是今天或昨天，最长约 48h)，非严格滚动 24h，与 epay 边界一致。
func (r *OrderRepo) FindNotifyPending(limit int) ([]model.Order, error) {
	var list []model.Order
	err := r.db.Where("(TO_DAYS(NOW()) - TO_DAYS(end_time) <= 1) AND notify > 0 AND notify_time IS NOT NULL AND notify_time < ?",
		time.Now()).
		Order("notify_time ASC").Limit(limit).Find(&list).Error
	return list, err
}

// FindAbandonedNotify 取已放弃通知(notify=-1)且完成时间在近日历天窗内的订单（notify2 兜底重发用）。
// 对齐 epay cron notify2（cron.php:211）：WHERE (TO_DAYS(NOW())-TO_DAYS(endtime)<=1) AND notify=-1。
// B1-71：完成窗口用【日历天差<=1】而非严格滚动 24h，与 epay 边界一致。
func (r *OrderRepo) FindAbandonedNotify(limit int) ([]model.Order, error) {
	var list []model.Order
	err := r.db.Where("(TO_DAYS(NOW()) - TO_DAYS(end_time) <= 1) AND notify = -1").
		Order("end_time DESC").Limit(limit).Find(&list).Error
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

// CountAllByMerchant 统计商户全部订单数（不限状态），对齐 epay Merchant::info 的
// order_num = count(*)（今/昨订单数才按 status=1 过滤）。
func (r *OrderRepo) CountAllByMerchant(uid uint) (int64, error) {
	var n int64
	err := r.db.Model(&model.Order{}).Where("uid = ?", uid).Count(&n).Error
	return n, err
}

// ListByMerchantV1 V1 api.php orders 列表：按 uid(+可选 status)，trade_no 降序，limit/offset 分页（A-5）。
func (r *OrderRepo) ListByMerchantV1(uid uint, status *int, limit, offset int) ([]model.Order, error) {
	tx := r.db.Where("uid = ?", uid)
	if status != nil {
		tx = tx.Where("status = ?", *status)
	}
	var list []model.Order
	err := tx.Order("trade_no DESC").Limit(limit).Offset(offset).Find(&list).Error
	return list, err
}

// SumTodayPaidRealMoney 汇总商户当日(自 dayStart 起)已成功订单的 realmoney(tid≠2 排除充值余额)。
// A-8：代付 settle_type=1 可用余额 = 余额 - 当日已收 realmoney（对齐 epay Transfer.php:19-22）。
func (r *OrderRepo) SumTodayPaidRealMoney(uid uint, dayStart time.Time) (decimal.Decimal, error) {
	var result struct{ Total decimal.Decimal }
	err := r.db.Model(&model.Order{}).
		Select("COALESCE(SUM(real_money),0) AS total").
		Where("uid = ? AND status = 1 AND tid <> 2 AND end_time >= ?", uid, dayStart).
		Scan(&result).Error
	return result.Total, err
}

// CountPaidByIPRange 统计某 IP 在 [start,end) 内已支付订单数（status>0，对齐 epay pay_iplimit 的 status>0）。
func (r *OrderRepo) CountPaidByIPRange(ip string, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.Order{}).
		Where("ip = ? AND status > 0 AND add_time >= ? AND add_time < ?", ip, start, end).
		Count(&n).Error
	return n, err
}

// CountPaidByBuyerRange 统计某买家(buyer/openid)在 [start,end) 内已支付订单数（status>0，
// 对齐 epay checkBlockUser 的 status>0 当日限单口径）。
func (r *OrderRepo) CountPaidByBuyerRange(buyer string, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.Order{}).
		Where("buyer = ? AND status > 0 AND add_time >= ? AND add_time < ?", buyer, start, end).
		Count(&n).Error
	return n, err
}

// CountPaidByMerchantRange 统计商户在 [start,end) 内已支付订单数（status=1）。
// 用于精确的今日/昨日订单数统计（对齐 epay Merchant::info 按 date 精确统计，避免差值近似跨天出错）。
func (r *OrderRepo) CountPaidByMerchantRange(uid uint, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.Model(&model.Order{}).
		Where("uid = ? AND status = 1 AND add_time >= ? AND add_time < ?", uid, start, end).
		Count(&n).Error
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

// SetStatus 裸改订单状态（对齐 epay setStatus：不触发资金/入账，仅改字段）。
func (r *OrderRepo) SetStatus(tradeNo string, status int8) error {
	return r.db.Model(&model.Order{}).Where("trade_no = ?", tradeNo).
		Update("status", status).Error
}

// SetStatusFrom 条件改状态：仅当当前 status==from 时改为 to，返回是否命中（冻结/解冻幂等用）。
func (r *OrderRepo) SetStatusFrom(tradeNo string, from, to int8) (bool, error) {
	res := r.db.Model(&model.Order{}).
		Where("trade_no = ? AND status = ?", tradeNo, from).
		Update("status", to)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// Delete 物理删除订单（对齐 epay setStatus=5 / operation=4：DELETE，无级联）。
func (r *OrderRepo) Delete(tradeNo string) error {
	return r.db.Where("trade_no = ?", tradeNo).Delete(&model.Order{}).Error
}

// MarkRefundedPartial 退款改单：status→2 并把 refundmoney 累加 addRefund（对齐 epay API 退款）。
// 仅当 status∈(1已付,2已退,3冻结) 时执行，返回是否命中。
func (r *OrderRepo) MarkRefundedPartial(tradeNo string, addRefund decimal.Decimal) (bool, error) {
	res := r.db.Model(&model.Order{}).
		Where("trade_no = ? AND status IN (1,2,3)", tradeNo).
		Updates(map[string]interface{}{
			"status":       2,
			"refund_money": gorm.Expr("refund_money + ?", addRefund),
			"refundtime":   time.Now(), // 回填退款时间（对齐 epay pre_order.refundtime）
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// FillOrder 手动补单改单：仅当 status==0(未付) 时置为已付并写完成时间，返回是否命中。
// 入账/通知由 service 复用支付成功链路完成（对齐 epay fillorder → processOrder）。
func (r *OrderRepo) FillOrder(tradeNo string, endTime time.Time) (bool, error) {
	res := r.db.Model(&model.Order{}).
		Where("trade_no = ? AND status = 0", tradeNo).
		Updates(map[string]interface{}{"status": 1, "end_time": endTime})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// List 分页查询订单，支持按 column/keyword 模糊搜索与状态筛选。
// orderQueryColumns 订单搜索字段白名单（防 SQL 注入到列名）。
var orderQueryColumns = map[string]bool{
	"trade_no": true, "out_trade_no": true, "api_trade_no": true,
	"name": true, "money": true, "realmoney": true, "domain": true,
}

// applyOrderQuery 按 OrderQuery 构造统一 where（List/ExportAll/Stats 共用，对齐 epay 列表与统计同一套筛选）。
// 支持：商户端强制 UID、后台商户 uid、支付方式 type、通道 channel、状态 status、时间范围、column+keyword 模糊。
func (r *OrderRepo) applyOrderQuery(tx *gorm.DB, q dto.OrderQuery) *gorm.DB {
	if q.UID != nil { // 商户端强制注入，越权保护，优先级最高
		tx = tx.Where("uid = ?", *q.UID)
	} else if q.UIDFilter > 0 { // 后台按商户筛选
		tx = tx.Where("uid = ?", q.UIDFilter)
	}
	if q.Type > 0 {
		tx = tx.Where("type = ?", q.Type)
	}
	if q.Channel > 0 {
		tx = tx.Where("channel = ?", q.Channel)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.StartTime != "" {
		tx = tx.Where("add_time >= ?", q.StartTime+" 00:00:00")
	}
	if q.EndTime != "" {
		tx = tx.Where("add_time <= ?", q.EndTime+" 23:59:59")
	}
	if q.Keyword != "" && orderQueryColumns[q.Column] {
		tx = tx.Where(q.Column+" LIKE ?", "%"+q.Keyword+"%")
	}
	return tx
}

func (r *OrderRepo) List(q dto.OrderQuery) ([]model.Order, int64, error) {
	tx := r.applyOrderQuery(r.db.Model(&model.Order{}), q)

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

// Stats 按与 List 相同的过滤条件全量聚合订单统计（对齐 epay ajax_order.php?act=statistics）。
// 一次 SQL 用 CASE WHEN 分状态聚合金额/计数，不受列表分页 ≤100 限制。
func (r *OrderRepo) Stats(q dto.OrderQuery) (dto.OrderStats, error) {
	tx := r.applyOrderQuery(r.db.Model(&model.Order{}), q)

	var row struct {
		TotalMoney     float64
		SuccessMoney   float64
		UnpaidMoney    float64
		RefundMoney    float64
		PlatformProfit float64
		TotalCount     int64
		SuccessCount   int64
		UnpaidCount    int64
		RefundCount    int64
	}
	err := tx.Select(
		"COALESCE(SUM(money),0) AS total_money, " +
			"COALESCE(SUM(CASE WHEN status = 1 THEN money ELSE 0 END),0) AS success_money, " +
			"COALESCE(SUM(CASE WHEN status = 0 THEN money ELSE 0 END),0) AS unpaid_money, " +
			"COALESCE(SUM(CASE WHEN status = 2 THEN refund_money ELSE 0 END),0) AS refund_money, " +
			"COALESCE(SUM(CASE WHEN status = 1 THEN profit_money ELSE 0 END),0) AS platform_profit, " +
			"COUNT(*) AS total_count, " +
			"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END),0) AS success_count, " +
			"COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END),0) AS unpaid_count, " +
			"COALESCE(SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END),0) AS refund_count",
	).Scan(&row).Error
	if err != nil {
		return dto.OrderStats{}, err
	}
	return dto.OrderStats{
		TotalMoney:     row.TotalMoney,
		SuccessMoney:   row.SuccessMoney,
		UnpaidMoney:    row.UnpaidMoney,
		RefundMoney:    row.RefundMoney,
		PlatformProfit: row.PlatformProfit,
		TotalCount:     row.TotalCount,
		SuccessCount:   row.SuccessCount,
		UnpaidCount:    row.UnpaidCount,
		RefundCount:    row.RefundCount,
	}, nil
}

// ExportAll 按与 List 相同的过滤条件取全量订单（不分页，供后台流式导出）。
// 限 limit 条上限（对齐 epay download.php limit 100000）防内存爆。UID 支持商户端限定。
func (r *OrderRepo) ExportAll(q dto.OrderQuery, limit int) ([]model.Order, error) {
	tx := r.applyOrderQuery(r.db.Model(&model.Order{}), q)
	var list []model.Order
	err := tx.Order("add_time DESC").Limit(limit).Find(&list).Error
	return list, err
}
