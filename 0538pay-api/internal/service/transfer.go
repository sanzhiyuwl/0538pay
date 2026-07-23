package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// 代付计费与限额配置（对齐 epay pre_config 的 transfer_* 键，先固化为常量，待 config 域迁移）。
// transfer_rate 为空时 epay 回退到 settle_rate；此处直接复用结算费率常量保持一致。
var (
	transferRate     = settleRate                         // transfer_rate 代付手续费率（%），空则复用 settle_rate
	transferMinMoney = decimal.RequireFromString("1")     // transfer_minmoney 单笔最小
	transferMaxMoney = decimal.RequireFromString("20000") // transfer_maxmoney 单笔最大
	transferMaxLimit = 10                                 // transfer_maxlimit 同账号每日代付次数上限（0=不限）

	transferSettleTypeOrder = true // settle_type=1(订单结算)：代付可用余额扣当日已收 realmoney（A-8）
)

// reloadTransferConfig 从 config 域刷新代付常量。transfer_rate 留空时复用 settle_rate（对齐 epay）。
func reloadTransferConfig(cfg *ConfigService) {
	if r := strings.TrimSpace(cfg.Str("transfer_rate")); r != "" {
		transferRate = cfg.Dec("transfer_rate", settleRate)
	} else {
		transferRate = settleRate
	}
	transferMinMoney = cfg.Dec("transfer_minmoney", transferMinMoney)
	transferMaxMoney = cfg.Dec("transfer_maxmoney", transferMaxMoney)
	transferMaxLimit = cfg.Int("transfer_maxlimit", transferMaxLimit)
	transferSettleTypeOrder = cfg.Str("settle_type") != "0" // 非 "0" 即订单结算(D+1)，代付可用余额扣当日已收
}

// 付款方式 → 默认通道 id（对齐 epay transfer_alipay/wxpay/qqpay/bank，0=未开启该方式）。
// 待 config/通道域联动后改为读配置；当前给出占位默认，真实打款待渠道凭证。
var transferDefaultChannel = map[string]int{
	"alipay": 1,
	"wxpay":  2,
	"qqpay":  3,
	"bank":   4,
}

var transferTypes = map[string]bool{"alipay": true, "wxpay": true, "qqpay": true, "bank": true}

// TransferService 代付业务：列表/统计、发起（后台免费 / 商户扣款）、状态流转、退回。
type TransferService struct {
	repo      *repository.TransferRepo
	merchants *repository.MerchantRepo
	admins    *repository.AdminRepo
	orders    *repository.OrderRepo    // 可空；SetOrderRepo 注入。settle_type=1 可用余额扣当日已收 realmoney 用
	cfg       *ConfigService           // 可空；SetConfigService 注入。读 user_transfer/transfer_* 通道配置
	groups    *repository.GroupRepo    // 可空；SetGroupRepo 注入。按商户 gid 组级覆盖 transfer_* 配置(B1-09)
}

func NewTransferService(
	repo *repository.TransferRepo,
	merchants *repository.MerchantRepo,
	admins *repository.AdminRepo,
) *TransferService {
	return &TransferService{repo: repo, merchants: merchants, admins: admins}
}

// SetOrderRepo 注入订单 repo（A-8：settle_type=1 时代付可用余额 = 余额 - 当日已成功订单 realmoney）。
// nil 则可用余额直接取商户余额（settle_type=0 语义，向后兼容）。
func (s *TransferService) SetOrderRepo(o *repository.OrderRepo) { s.orders = o }

// SetConfigService 注入配置服务（读 user_transfer 全局开关 + transfer_alipay 等通道配置）。
func (s *TransferService) SetConfigService(c *ConfigService) { s.cfg = c }

// SetGroupRepo 注入用户组仓储，使代付按商户 gid 做组级 transfer_* 配置覆盖（B1-09）。
func (s *TransferService) SetGroupRepo(g *repository.GroupRepo) { s.groups = g }

// CheckUserTransfer 校验某商户(gid)代付全局开关（B1-08，供 query/proof/balance 等非创建路径复用）。
// 未开启返回错误；cfg 未注入时默认放行（向后兼容）。
func (s *TransferService) CheckUserTransfer(gid int) error {
	if !s.resolveTransferConf(gid).userTransfer {
		return tfErr("管理员未开启代付功能")
	}
	return nil
}

// transferConf 某商户生效的代付配置（全局为底 + 组级非空覆盖，对齐 epay array_merge(conf,getGroupConfig)）。
type transferConf struct {
	userTransfer bool            // user_transfer 全局总开关
	rate         decimal.Decimal // transfer_rate（空复用 settle_rate）
	minMoney     decimal.Decimal // transfer_minmoney
	maxMoney     decimal.Decimal // transfer_maxmoney
	maxLimit     int             // transfer_maxlimit
	channel      map[string]int  // 付款方式→通道 id（0=未开启该方式）
}

// resolveTransferConf 计算某商户(gid)生效的代付配置：以全局 config 为底，组级 config 非空键覆盖
// （B1-08 user_transfer / B1-09 组级覆盖 / B1-11 通道按方式取配置）。cfg 为 nil 时回退包级全局常量。
func (s *TransferService) resolveTransferConf(gid int) transferConf {
	c := transferConf{
		userTransfer: true, // cfg 未注入时不拦（向后兼容旧行为）
		rate:         transferRate,
		minMoney:     transferMinMoney,
		maxMoney:     transferMaxMoney,
		maxLimit:     transferMaxLimit,
		channel:      map[string]int{"alipay": transferDefaultChannel["alipay"], "wxpay": transferDefaultChannel["wxpay"], "qqpay": transferDefaultChannel["qqpay"], "bank": transferDefaultChannel["bank"]},
	}
	if s.cfg == nil {
		return c
	}
	// 全局配置为底
	c.userTransfer = s.cfg.Int("user_transfer", 0) == 1
	if r := strings.TrimSpace(s.cfg.Str("transfer_rate")); r != "" {
		c.rate = s.cfg.Dec("transfer_rate", settleRate)
	} else {
		c.rate = settleRate
	}
	c.minMoney = s.cfg.Dec("transfer_minmoney", transferMinMoney)
	c.maxMoney = s.cfg.Dec("transfer_maxmoney", transferMaxMoney)
	c.maxLimit = s.cfg.Int("transfer_maxlimit", transferMaxLimit)
	c.channel = map[string]int{
		"alipay": s.cfg.Int("transfer_alipay", 0),
		"wxpay":  s.cfg.Int("transfer_wxpay", 0),
		"qqpay":  s.cfg.Int("transfer_qqpay", 0),
		"bank":   s.cfg.Int("transfer_bank", 0),
	}
	// 组级非空覆盖（对齐 epay getGroupConfig）
	if s.groups == nil || gid <= 0 {
		return c
	}
	g, err := s.groups.FindByID(gid)
	if err != nil || g == nil || strings.TrimSpace(g.Config) == "" {
		return c
	}
	var gc map[string]interface{}
	if json.Unmarshal([]byte(g.Config), &gc) != nil {
		return c
	}
	if v, ok := groupConfStr(gc, "user_transfer"); ok {
		c.userTransfer = v == "1"
	}
	if v, ok := groupConfStr(gc, "transfer_rate"); ok {
		if d, err := decimal.NewFromString(v); err == nil {
			c.rate = d
		}
	}
	if v, ok := groupConfStr(gc, "transfer_minmoney"); ok {
		if d, err := decimal.NewFromString(v); err == nil {
			c.minMoney = d
		}
	}
	if v, ok := groupConfStr(gc, "transfer_maxmoney"); ok {
		if d, err := decimal.NewFromString(v); err == nil {
			c.maxMoney = d
		}
	}
	if v, ok := groupConfStr(gc, "transfer_maxlimit"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			c.maxLimit = n
		}
	}
	for _, t := range []string{"alipay", "wxpay", "qqpay", "bank"} {
		if v, ok := groupConfStr(gc, "transfer_"+t); ok {
			if n, err := strconv.Atoi(v); err == nil {
				c.channel[t] = n
			}
		}
	}
	return c
}

// enableMoney 计算代付可用余额（A-8，对齐 epay Transfer.php:17-25）。
// settle_type=1(订单结算)：可用 = 余额 - 当日已成功订单 realmoney(tid≠2)，不小于 0；否则=全部余额。
func (s *TransferService) enableMoney(m *model.Merchant) decimal.Decimal {
	if s.orders == nil || !transferSettleTypeOrder {
		return m.Money
	}
	today := dayStart(time.Now())
	todayPaid, err := s.orders.SumTodayPaidRealMoney(m.UID, today)
	if err != nil {
		return m.Money // 查询失败退回全额，不阻断代付
	}
	enable := m.Money.Sub(todayPaid)
	if enable.LessThan(decimal.Zero) {
		enable = decimal.Zero
	}
	return enable
}

// TransferError 携带业务错误码与提示。
type TransferError struct {
	Code int
	Msg  string
}

func (e *TransferError) Error() string { return e.Msg }

func tfErr(msg string) *TransferError { return &TransferError{Code: 1106, Msg: msg} }

// calcTransferFee 计算代付手续费（对齐 epay：need = money + money*rate/100，round 2）。返回手续费。
func calcTransferFee(money decimal.Decimal) decimal.Decimal {
	return calcTransferFeeWith(transferRate, money)
}

// calcTransferFeeWith 按指定费率计算代付手续费（B1-09：费率可组级覆盖，故显式传入）。
func calcTransferFeeWith(rate, money decimal.Decimal) decimal.Decimal {
	if rate.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return money.Mul(rate).Div(hundred).Round(2)
}

// List 后台代付列表（分页 + 筛选）。
func (s *TransferService) List(q dto.TransferQuery) ([]dto.TransferView, int64, error) {
	q.Normalize()
	list, total, err := s.repo.List(q)
	if err != nil {
		return nil, 0, err
	}
	views := make([]dto.TransferView, 0, len(list))
	for i := range list {
		views = append(views, toTransferView(&list[i]))
	}
	return views, total, nil
}

// ListByMerchant 商户端代付列表：强制注入当前商户 uid，防越权。
func (s *TransferService) ListByMerchant(uid uint, q dto.TransferQuery) ([]dto.TransferView, int64, error) {
	q.UID = &uid
	return s.List(q)
}

// Stats 后台代付概况统计。
func (s *TransferService) Stats(q dto.TransferQuery) (dto.TransferStats, error) {
	q.Normalize()
	tm, sm, sc, pc, fc, err := s.repo.Stats(q)
	if err != nil {
		return dto.TransferStats{}, err
	}
	return dto.TransferStats{
		Total:           sc + pc + fc,
		TotalMoney:      tm.InexactFloat64(),
		SuccessMoney:    sm.InexactFloat64(),
		SuccessCount:    sc,
		ProcessingCount: pc,
		FailCount:       fc,
	}, nil
}

// CreateByAdmin 后台管理员发起代付（uid=0）：校验管理员密码 → 不收费不扣款直接落库(处理中)。
// 真实渠道打款待凭证，此处仅落库进入处理中状态（对齐 epay admin/transfer_add：uid=0 免费）。
func (s *TransferService) CreateByAdmin(adminID uint, req dto.TransferCreateReq) (string, error) {
	if err := s.verifyAdminPwd(adminID, req.Password); err != nil {
		return "", err
	}
	conf := s.resolveTransferConf(0) // 管理员发起：全局配置，无组级覆盖
	money, bizNo, err := s.validateCommon(req, conf)
	if err != nil {
		return "", err
	}
	t := &model.Transfer{
		BizNo:     bizNo,
		UID:       0, // 管理员发起哨兵值
		Type:      req.Type,
		Channel:   s.resolveChannel(req, conf),
		Account:   strings.TrimSpace(req.Account),
		Username:  strings.TrimSpace(req.Username),
		Money:     money,
		CostMoney: money, // 后台不收费，扣款额=到账额（且不实际扣）
		AddTime:   time.Now(),
		Status:    0,
		Desc:      req.Desc,
	}
	if err := s.repo.CreateAdmin(t); err != nil {
		if err == repository.ErrDuplicateBizNo {
			return "", tfErr("交易号已存在，请勿重复提交")
		}
		return "", err
	}
	return bizNo, nil
}

// BatchItemResult 批量代付单条结果（C-2）。
type BatchItemResult struct {
	Index    int    `json:"index"`
	Account  string `json:"account"`
	BizNo    string `json:"biz_no,omitempty"`
	Success  bool   `json:"success"`
	Msg      string `json:"msg,omitempty"`
}

// CreateBatchByAdmin 后台批量代付（C-2，对齐 epay transfer_batch）：一次校验管理员密码，
// 逐条走 CreateByAdmin 的落库逻辑，返回每条成功/失败。任一条失败不影响其余（逐条独立）。
// 真实渠道打款同单笔待凭证，此处批量落库进处理中。
func (s *TransferService) CreateBatchByAdmin(adminID uint, password string, items []dto.TransferCreateReq) ([]BatchItemResult, error) {
	if err := s.verifyAdminPwd(adminID, password); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, tfErr("批量代付列表为空")
	}
	if len(items) > 200 {
		return nil, tfErr("单次批量代付上限 200 条")
	}
	conf := s.resolveTransferConf(0) // 管理员批量：全局配置，无组级覆盖
	results := make([]BatchItemResult, 0, len(items))
	for i, it := range items {
		r := BatchItemResult{Index: i, Account: strings.TrimSpace(it.Account)}
		money, bizNo, err := s.validateCommon(it, conf)
		if err != nil {
			r.Msg = err.Error()
			results = append(results, r)
			continue
		}
		t := &model.Transfer{
			BizNo: bizNo, UID: 0, Type: it.Type, Channel: s.resolveChannel(it, conf),
			Account: strings.TrimSpace(it.Account), Username: strings.TrimSpace(it.Username),
			Money: money, CostMoney: money, AddTime: time.Now(), Status: 0, Desc: it.Desc,
		}
		if err := s.repo.CreateAdmin(t); err != nil {
			if err == repository.ErrDuplicateBizNo {
				r.Msg = "交易号已存在"
			} else {
				r.Msg = err.Error()
			}
			results = append(results, r)
			continue
		}
		r.Success = true
		r.BizNo = bizNo
		results = append(results, r)
	}
	return results, nil
}

// CreateByMerchant 商户发起代付：校验登录密码/结算权限/限额/次数 → 计费 → 即时扣款落库(处理中)。
// 对齐 epay user/transfer_add：走完整校验 + changeUserMoney 扣 need_money。真实打款待渠道凭证。
func (s *TransferService) CreateByMerchant(uid uint, req dto.TransferCreateReq) (string, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", tfErr("商户不存在")
	}
	if m.Settle != 1 {
		return "", tfErr("结算功能未开启，无法发起代付")
	}
	// 身份校验：商户登录密码（对齐 epay user 端校验登录密码）
	if m.Password == "" {
		return "", tfErr("请先设置登录密码后再发起代付")
	}
	if bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(req.Password)) != nil {
		return "", tfErr("登录密码不正确")
	}
	return s.createByMerchantCore(uid, req)
}

// CreateByMerchantSigned 供 mapi(V2 REST)发起代付：请求已由签名鉴权，不再校验登录密码。
// 其余校验(结算开关/限额/余额/幂等)与 CreateByMerchant 完全一致。
func (s *TransferService) CreateByMerchantSigned(uid uint, req dto.TransferCreateReq) (string, error) {
	m, err := s.merchants.FindByUIDSafe(uid)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", tfErr("商户不存在")
	}
	// B1-08：全局代付开关在最前（对齐 epay Transfer.php:14 先 user_transfer 再 userrow['transfer']）。
	if err := s.CheckUserTransfer(m.GID); err != nil {
		return "", err
	}
	// per-merchant 代付API开关（A-7，对齐 epay Transfer.php:15 userrow['transfer']==0）。
	if m.Transfer == 0 {
		return "", tfErr("商户未开启代付API接口")
	}
	if m.Settle != 1 {
		return "", tfErr("结算功能未开启，无法发起代付")
	}
	// B1-07/37：V2 API 路径强制收款人姓名非空（对齐 epay api/Transfer.php:38）。
	// user 页(CreateByMerchant)姓名选填，不在此校验，故只在 signed 路径拦。
	if strings.TrimSpace(req.Username) == "" {
		return "", tfErr("收款人姓名(name)不能为空")
	}
	return s.createByMerchantCore(uid, req)
}

// createByMerchantCore 商户代付核心（鉴权后共用：限额/费率/余额扣减/幂等）。
func (s *TransferService) createByMerchantCore(uid uint, req dto.TransferCreateReq) (string, error) {
	// 解析该商户生效的代付配置（全局 + 组级覆盖）。先取商户拿 gid。
	m, _ := s.merchants.FindByUIDSafe(uid)
	gid := 0
	if m != nil {
		gid = m.GID
	}
	conf := s.resolveTransferConf(gid)

	// B1-08：全局代付总开关（管理员未开启则一律拒，与 per-merchant Transfer 开关是两道独立闸门）。
	if !conf.userTransfer {
		return "", tfErr("管理员未开启代付功能")
	}

	money, bizNo, err := s.validateCommon(req, conf)
	if err != nil {
		return "", err
	}

	// 同一收款账号+方式每日次数限制（B1-10/36：CountTodayByAccount 已改按 pay_time 仅计成功单）
	if conf.maxLimit > 0 {
		cnt, err := s.repo.CountTodayByAccount(uid, req.Type, strings.TrimSpace(req.Account), dayStart(time.Now()))
		if err != nil {
			return "", err
		}
		if int(cnt) >= conf.maxLimit {
			return "", tfErr("该收款账号今日代付已达次数上限")
		}
	}

	// B1-11：按付款方式取通道配置，channel=0(未开启该方式)则拒（对齐 epay '未开启此转账方式'）。
	channelID := s.resolveChannel(req, conf)
	if channelID <= 0 {
		return "", tfErr("未开启此转账方式")
	}

	fee := calcTransferFeeWith(conf.rate, money)
	cost := money.Add(fee)

	// settle_type=1 可用余额校验（A-8）：需支付金额 > 可用余额(余额-当日已收 realmoney)则拒。
	// CreateWithDebit 内另有余额行锁校验兜底；此处按 epay 语义提前拦并给准确文案。
	if m != nil {
		if cost.GreaterThan(s.enableMoney(m)) {
			return "", tfErr("需支付金额大于可转账余额")
		}
	}

	t := &model.Transfer{
		BizNo:     bizNo,
		UID:       uid,
		Type:      req.Type,
		Channel:   channelID,
		Account:   strings.TrimSpace(req.Account),
		Username:  strings.TrimSpace(req.Username),
		Money:     money,
		CostMoney: cost,
		AddTime:   time.Now(),
		Status:    0,
		Desc:      req.Desc,
	}
	if err := s.repo.CreateWithDebit(t); err != nil {
		switch err {
		case repository.ErrInsufficientBalance:
			return "", tfErr("余额不足，需 ¥" + cost.StringFixed(2))
		case repository.ErrDuplicateBizNo:
			return "", tfErr("交易号已存在，请勿重复提交")
		}
		return "", err
	}
	return bizNo, nil
}

// SetStatus 后台手动改状态（1成功/2失败，不动资金，对齐 epay setTransferStatus）。
// 改为成功写付款时间；改为失败写失败原因。资金退回请用 Refund（仅处理中可退）。
func (s *TransferService) SetStatus(bizNo string, req dto.TransferStatusReq) error {
	t, err := s.repo.FindByBizNo(bizNo)
	if err != nil {
		return err
	}
	if t == nil {
		return tfErr("代付记录不存在")
	}
	switch req.Status {
	case 1:
		return s.repo.SetStatus(bizNo, map[string]interface{}{
			"status": 1, "pay_time": time.Now(), "result": "",
		})
	case 2:
		return s.repo.SetStatus(bizNo, map[string]interface{}{
			"status": 2, "result": req.Result,
		})
	default:
		return tfErr("状态值不合法")
	}
}

// Refund 退回代付：仅处理中(status=0)可退，置失败并把 CostMoney 退回商户（管理员发起不退）。
// 对齐 epay refundTransfer：条件 UPDATE 防重复退款。
func (s *TransferService) Refund(bizNo string) error {
	t, err := s.repo.FindByBizNo(bizNo)
	if err != nil {
		return err
	}
	if t == nil {
		return tfErr("代付记录不存在")
	}
	if t.Status != 0 {
		return tfErr("仅处理中的代付可退回")
	}
	refunded, err := s.repo.FailWithRefund(bizNo, "转账已退回")
	if err != nil {
		return err
	}
	if !refunded && t.UID > 0 {
		// 并发下已被其它请求处理：视为状态已变更
		return tfErr("代付状态已变更，退回未执行")
	}
	return nil
}

// Delete 删除代付记录（不退款，对齐 epay delTransfer）。
func (s *TransferService) Delete(bizNo string) error {
	t, err := s.repo.FindByBizNo(bizNo)
	if err != nil {
		return err
	}
	if t == nil {
		return tfErr("代付记录不存在")
	}
	return s.repo.Delete(bizNo)
}

// validateCommon 校验发起代付的公共入参并生成/校验交易号，返回解析后的到账金额与交易号。
// conf 为该商户生效的代付配置（min/max 限额来自其中，支持组级覆盖，B1-09）。
func (s *TransferService) validateCommon(req dto.TransferCreateReq, conf transferConf) (decimal.Decimal, string, error) {
	if !transferTypes[req.Type] {
		return decimal.Zero, "", tfErr("付款方式不合法")
	}
	acct := strings.TrimSpace(req.Account)
	if acct == "" {
		return decimal.Zero, "", tfErr("请填写收款账号")
	}
	// QQ 钱包收款账号需 6~10 位纯数字（A-8，对齐 epay Transfer.php:63）。
	if req.Type == "qqpay" && (!isNumeric(acct) || len(acct) < 6 || len(acct) > 10) {
		return decimal.Zero, "", tfErr("QQ号码格式错误")
	}
	if l := len([]rune(req.Desc)); l > 32 {
		return decimal.Zero, "", tfErr("备注最多 32 字")
	}
	money, err := decimal.NewFromString(strings.TrimSpace(req.Money))
	if err != nil || money.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, "", tfErr("请输入有效的转账金额")
	}
	// 保留两位（对齐 epay 金额精度），拒绝更细粒度输入以免与扣款不一致
	if !money.Equal(money.Round(2)) {
		return decimal.Zero, "", tfErr("金额最多两位小数")
	}
	if conf.minMoney.GreaterThan(decimal.Zero) && money.LessThan(conf.minMoney) {
		return decimal.Zero, "", tfErr("单笔最低 ¥" + conf.minMoney.StringFixed(2))
	}
	if conf.maxMoney.GreaterThan(decimal.Zero) && money.GreaterThan(conf.maxMoney) {
		return decimal.Zero, "", tfErr("单笔最高 ¥" + conf.maxMoney.StringFixed(2))
	}
	bizNo, err := s.resolveBizNo(req.BizNo)
	if err != nil {
		return decimal.Zero, "", err
	}
	return money, bizNo, nil
}

// resolveBizNo 校验或生成 19 位数字交易号（对齐 epay：strlen==19 && is_numeric）。
func (s *TransferService) resolveBizNo(in string) (string, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return genBizNo(), nil
	}
	if len(in) != 19 || !isNumeric(in) {
		return "", tfErr("交易号必须为 19 位数字")
	}
	return in, nil
}

// resolveChannel 取本次代付使用的通道 id：入参优先，否则按付款方式从配置取（B1-11）。
// 返回 0 表示该方式未开启（管理员未配置 transfer_<type> 通道），调用方据此拒绝。
func (s *TransferService) resolveChannel(req dto.TransferCreateReq, conf transferConf) int {
	if req.Channel > 0 {
		return req.Channel
	}
	return conf.channel[req.Type]
}

// verifyAdminPwd 校验管理员支付密码（对齐 epay admin_paypwd，独立于登录密码）。
// adminID 保留以兼容调用签名，实际校验走全局支付密码。
func (s *TransferService) verifyAdminPwd(_ uint, pwd string) error {
	if pwd == "" {
		return tfErr("请输入支付密码")
	}
	if err := verifyPayPwd(pwd); err != nil {
		return tfErr("支付密码不正确")
	}
	return nil
}

// genBizNo 生成 19 位数字交易号：YmdHis(14) + 5 位纳秒派生随机（对齐 epay YmdHis+rand(11111,99999)）。
func genBizNo() string {
	now := time.Now()
	rand5 := 11111 + int(now.UnixNano()%88888)
	return now.Format("20060102150405") + fmt.Sprintf("%05d", rand5)
}

func toTransferView(t *model.Transfer) dto.TransferView {
	var payTime *string
	if t.PayTime != nil {
		s := t.PayTime.Format(timeLayout)
		payTime = &s
	}
	return dto.TransferView{
		BizNo:      t.BizNo,
		PayOrderNo: t.PayOrderNo,
		UID:        t.UID,
		Type:       t.Type,
		Channel:    t.Channel,
		Account:    t.Account,
		Username:   t.Username,
		Money:      t.Money.StringFixed(2),
		CostMoney:  t.CostMoney.StringFixed(2),
		Desc:       t.Desc,
		AddTime:    t.AddTime.Format(timeLayout),
		PayTime:    payTime,
		Status:     t.Status,
		Result:     t.Result,
	}
}
