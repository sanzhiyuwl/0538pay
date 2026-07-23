package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/money"
	"github.com/0538pay/api/pkg/sign"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// PayError 携带业务错误，handler 据此返回 code+msg。
type PayError struct {
	Code int
	Msg  string
}

func (e *PayError) Error() string { return e.Msg }

func payErr(msg string) *PayError { return &PayError{Code: 1100, Msg: msg} }

// out_trade_no 合法字符，对齐 epay：字母数字 . _ - |
var outTradeNoRe = regexp.MustCompile(`^[a-zA-Z0-9._\-|]+$`)

// PayService 收单下单业务。阶段 A：MD5 验签 + 幂等 + mock 渠道下单。
// 尚未接入的（费率/通道轮询/风控/域名白名单/保证金）留待 B/C 阶段，
// 此处逻辑顺序对齐 epay lib/api/Pay::submit，便于后续逐段补齐。
type PayService struct {
	merchants *repository.MerchantRepo
	orders    *repository.OrderRepo
	accounts  *repository.AccountRepo
	channels  *repository.ChannelRepo
	profit    *ProfitService    // 分账（可空；SetProfitService 注入，避免构造顺序耦合）
	invite    *InviteRewardService // 邀请返现（可空；SetInviteReward 注入）
	risk      *RiskService      // 风控关键词拦截（可空）
	blacklist *BlacklistService // 黑名单拦截（可空）
	domain    *DomainService    // 域名白名单校验（可空）
	selector  *ChannelSelector  // 选通道分发（可空；SetSelector 注入。nil 则退回 plugin 名直配）
	cfg       *ConfigService    // 系统配置（可空；SetConfigService 注入。回调 RSA 签名/全局限额/加费兜底用）
	notice    *NoticeService    // 对外通知中枢（可空；SetNoticeService 注入。order 支付成功商户通知）
	paytypes  *repository.PayTypeRepo // 支付方式（可空；SetPayTypeRepo 注入。device 分端过滤用）
	subchannels *repository.SubChannelRepo // 子通道（可空；SetSubChannelRepo 注入。B1-34 下单/退款子通道占位覆盖用）
	regPayHook func(param string) error   // B1-51 付费注册 tid=1 回调建号钩子（可空；SetRegPayHook 注入，避免与 reg 服务循环依赖）
}

// SetRegPayHook 注入付费注册回调建号钩子（B1-51）。tid=1 注册费订单支付成功后调用，参数为订单 param（注册信息 JSON）。
// nil 则 tid=1 订单支付后不建号（向后兼容 reg_pay=0）。
func (s *PayService) SetRegPayHook(f func(param string) error) { s.regPayHook = f }

// SetNoticeService 注入对外通知中枢（K-1）。支付成功后发 order 场景通知（微信/邮件/短信）。
// nil 则不发对外通知（不影响商户异步回调 do_notify）。
func (s *PayService) SetNoticeService(n *NoticeService) { s.notice = n }

// SetConfigService 注入系统配置域（V2 回调平台私钥签名、notifyordername、全局金额限额、mode=1 加费兜底）。
// nil 则回调恒 MD5、无全局限额（向后兼容）。
func (s *PayService) SetConfigService(c *ConfigService) { s.cfg = c }

func NewPayService(m *repository.MerchantRepo, o *repository.OrderRepo, a *repository.AccountRepo, ch *repository.ChannelRepo) *PayService {
	return &PayService{merchants: m, orders: o, accounts: a, channels: ch}
}

// SetSelector 注入选通道分发器（用户组通道分配 / 轮询组 / 子通道 / 组级费率覆盖）。
// nil 则 Submit 退回旧版按 plugin 名定位单一通道（向后兼容阶段A/B）。
func (s *PayService) SetSelector(sel *ChannelSelector) { s.selector = sel }

// SetPayTypeRepo 注入支付方式仓储（device 分端过滤：移动端不出仅PC通道，反之亦然，对齐 epay Channel::submit）。
func (s *PayService) SetPayTypeRepo(r *repository.PayTypeRepo) { s.paytypes = r }

// SetSubChannelRepo 注入子通道仓储（B1-34：下单/退款按 order.subchannel>0 用子通道 info 覆盖主通道 config 占位）。
func (s *PayService) SetSubChannelRepo(r *repository.SubChannelRepo) { s.subchannels = r }

// SetProfitService 注入分账服务（下单匹配规则 + 支付成功建分账单）。nil 则不启用分账。
func (s *PayService) SetProfitService(p *ProfitService) { s.profit = p }

// SetInviteReward 注入邀请返现服务（支付成功后按比例返现到下单商户上级余额）。nil 则不启用。
func (s *PayService) SetInviteReward(ir *InviteRewardService) { s.invite = ir }

// SetRiskServices 注入风控/黑名单/域名服务（下单拦截校验）。任一为 nil 则跳过对应校验。
func (s *PayService) SetRiskServices(r *RiskService, b *BlacklistService, d *DomainService) {
	s.risk, s.blacklist, s.domain = r, b, d
}

// blockKeywords 关键词屏蔽词（对齐 epay blockname，| 分隔）。config 域加载后刷新。
var blockKeywords = []string{"博彩", "赌博", "违禁", "毒品", "枪支"}

// blockAlert 命中屏蔽词时的提示文案（对齐 epay blockalert）。
var blockAlert = "温馨提醒该商品禁止出售"

// reloadPayConfig 从 config 域刷新屏蔽词与提示。blockname 为空则不屏蔽。
func reloadPayConfig(cfg *ConfigService) {
	raw := strings.TrimSpace(cfg.Str("blockname"))
	if raw == "" {
		blockKeywords = nil
	} else {
		parts := strings.Split(raw, "|")
		kws := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				kws = append(kws, p)
			}
		}
		blockKeywords = kws
	}
	if a := strings.TrimSpace(cfg.Str("blockalert")); a != "" {
		blockAlert = a
	}
}

// hitKeyword 返回商品名命中的第一个屏蔽词，未命中返回空串（对齐 epay strpos 子串匹配）。
func hitKeyword(name string) string {
	for _, kw := range blockKeywords {
		if strings.Contains(name, kw) {
			return kw
		}
	}
	return ""
}

// calcFee 计算商户实得(getmoney)与实际支付金额(realmoney)，1:1 对齐 epay Pay.php:141-183。
// (hundred 常量 100 在 settle.go 同包已声明，复用。)
//   - mode=1 商户直清(加费)：realmoney = money*(200-rate)/100（买家多付手续费）、getmoney = money（商户全额到账）；
//     最低手续费兜底 payfee_lessthan/mincost：手续费 money*(100-rate)/100 低于阈值时 realmoney = money+mincost。
//   - mode=0 平台代收：realmoney = money、getmoney = money*rate/100；
//     兜底：手续费低于阈值时 getmoney = money-mincost（不小于 0）。
//   - realmoney 命中 pay_payaddstart 起始金额时，加 randomFloat(min,max) 随机微调（防同额并单，A-3）。
// cfg 为 nil 时退回最简：mode=0、无兜底、无随机（向后兼容）。
func (s *PayService) calcFee(m *model.Merchant, money, rate decimal.Decimal) (getMoney, realMoney decimal.Decimal) {
	feeLessThan, feeMinCost := decimal.Zero, decimal.Zero
	if s.cfg != nil {
		feeLessThan = s.cfg.Dec("payfee_lessthan", decimal.Zero)
		feeMinCost = s.cfg.Dec("payfee_mincost", decimal.Zero)
	}
	feeEnabled := feeLessThan.GreaterThan(decimal.Zero) && feeMinCost.GreaterThan(decimal.Zero)
	// 手续费 = money*(100-rate)/100（epay feemoney）。
	feeMoney := money.Mul(hundred.Sub(rate)).Div(hundred).Round(2)

	if m.Mode == 1 { // 订单加费模式
		realMoney = money.Mul(hundred.Add(hundred.Sub(rate))).Div(hundred).Round(2)
		getMoney = money
		if feeEnabled && feeMoney.LessThan(feeLessThan.Round(2)) {
			realMoney = money.Add(feeMinCost).Round(2)
		}
	} else { // 平台代收
		realMoney = money
		getMoney = money.Mul(rate).Div(hundred).Round(2)
		if feeEnabled && feeMoney.LessThan(feeLessThan.Round(2)) {
			getMoney = money.Sub(feeMinCost).Round(2)
			if getMoney.LessThan(decimal.Zero) {
				getMoney = decimal.Zero
			}
		}
	}
	realMoney = s.applyRandomFloat(realMoney)
	return getMoney, realMoney
}

// applyRandomFloat 随机增减 realmoney 防同额并单（A-3，对齐 epay Pay.php:183 + functions.php:848）。
// 仅当 pay_payaddstart/min/max 均非 0 且 realmoney≥起始金额时生效：realmoney += randomFloat(min,max)。
func (s *PayService) applyRandomFloat(realMoney decimal.Decimal) decimal.Decimal {
	if s.cfg == nil {
		return realMoney
	}
	start := s.cfg.Dec("pay_payaddstart", decimal.Zero)
	min := s.cfg.Dec("pay_payaddmin", decimal.Zero)
	max := s.cfg.Dec("pay_payaddmax", decimal.Zero)
	if start.LessThanOrEqual(decimal.Zero) || min.LessThanOrEqual(decimal.Zero) || max.LessThanOrEqual(decimal.Zero) {
		return realMoney
	}
	if realMoney.LessThan(start) {
		return realMoney
	}
	// randomFloat(min,max)：min + rand*(max-min)，两位小数（对齐 epay randomFloat）。
	minF, _ := min.Float64()
	maxF, _ := max.Float64()
	add := minF + rand.Float64()*(maxF-minF)
	return realMoney.Add(decimal.NewFromFloat(add)).Round(2)
}

// truncateRunes 按 UTF-8 边界截断到最多 maxBytes 字节（A-11，对齐 epay mb_strcut($s,0,127,'utf-8')）。
// mb_strcut 按字节上限截断但不切碎多字节字符：累加每个 rune 的字节数，超出即停，保留完整字符。
func truncateRunes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	total := 0
	for i, r := range s {
		size := len(string(r))
		if total+size > maxBytes {
			return s[:i]
		}
		total += size
	}
	return s
}

// calcProfitMoney 计算平台利润（对齐 epay processOrder）：reducemoney = realmoney - getmoney（负则 clamp 0），
// profitmoney = reducemoney - realmoney*通道成本费率costrate/100。realmoney 空则用 money。
// 通道不存在或 costrate=0 时 profit = reducemoney。成本高于毛利时返回负值（不 clamp，对齐 epay B1-50）。
// isDirectChannel 判断通道是否商户直清模式(mode==1)。查不到通道按非直清处理（安全默认：走平台代收加钱路径）。
func (s *PayService) isDirectChannel(channelID int) bool {
	if s.channels == nil || channelID <= 0 {
		return false
	}
	c, err := s.channels.FindByID(uint(channelID))
	if err != nil || c == nil {
		return false
	}
	return c.Mode == 1
}

// calcReduceMoneyOnSettle 计算直清通道回调入账时应从商户余额扣除的订单服务费 reducemoney。
// 对齐 epay functions.php：reducemoney = realmoney - getmoney（商户已直接收到全额，平台回收服务费）。
// getmoney<=0（阶段A无费率）时服务费为 0，不扣。
func (s *PayService) calcReduceMoneyOnSettle(o *model.Order) decimal.Decimal {
	realMoney := o.Money
	if o.RealMoney != nil && o.RealMoney.GreaterThan(decimal.Zero) {
		realMoney = *o.RealMoney
	}
	reduce := realMoney.Sub(o.GetMoney)
	if reduce.LessThan(decimal.Zero) {
		return decimal.Zero
	}
	return reduce
}

func (s *PayService) calcProfitMoney(o *model.Order) decimal.Decimal {
	realMoney := o.Money
	if o.RealMoney != nil && o.RealMoney.GreaterThan(decimal.Zero) {
		realMoney = *o.RealMoney
	}
	// reducemoney = realmoney - getmoney，负值 clamp 到 0（对齐 epay processOrder:561 if($reducemoney<0)$reducemoney=0）。
	reduce := realMoney.Sub(o.GetMoney)
	if reduce.LessThan(decimal.Zero) {
		reduce = decimal.Zero
	}
	// 扣通道成本费率后不再 clamp：成本高于毛利时利润可为负（对齐 epay，profitmoney 允许写负值）。
	if o.Channel > 0 {
		if ch, _ := s.channels.FindByID(uint(o.Channel)); ch != nil && ch.CostRate.GreaterThan(decimal.Zero) {
			cost := realMoney.Mul(ch.CostRate).Div(hundred).Round(2)
			reduce = reduce.Sub(cost)
		}
	}
	return reduce
}

// orderVersion 判定订单接口版本（对齐 epay `defined('API_INIT')?1:0`）。
// V2 REST 入口（mapi PayCreate）注入 _version=1 → 回调用平台私钥 RSA 签+timestamp；
// 表单页入口（submit.php 等价）未注入 → 0 → 回调用商户 key MD5 签。
func orderVersion(params map[string]string) int8 {
	if params["_version"] == "1" {
		return 1
	}
	return 0
}

// verifySubmitSign 校验下单签名（对齐 epay ApiHelper::api_verify）。
//   - keytype=1（安全模式）：强制 sign_type=RSA，否则拒绝。
//   - keytype=0（兼容模式）：按请求 sign_type 选 MD5(默认) 或 RSA。
//   - RSA：用商户公钥验签 + 校验 timestamp ±300s（防重放）。MD5：md5(str+key)。
func (s *PayService) verifySubmitSign(m *model.Merchant, params map[string]string) error {
	signType := params["sign_type"]
	if signType == "" {
		signType = "MD5"
	}
	if m.KeyType == 1 && signType != "RSA" {
		return &PayError{Code: 1103, Msg: "该商户仅支持 RSA 签名类型"}
	}
	if signType == "RSA" {
		if m.PublicKey == "" {
			return &PayError{Code: 1103, Msg: "该商户未配置 RSA 公钥，无法用 RSA 验签"}
		}
		// V2 时间戳窗口校验（±300s），防重放。
		if err := checkTimestamp(params["timestamp"]); err != nil {
			return err
		}
		if !sign.VerifyRSA(params, m.PublicKey) {
			return &PayError{Code: 1103, Msg: "RSA签名校验失败"}
		}
		return nil
	}
	if !sign.VerifyMD5(params, m.AppKey) {
		return &PayError{Code: 1103, Msg: "MD5签名校验失败"}
	}
	return nil
}

// checkTimestamp 校验请求时间戳在当前时间 ±300 秒内（对齐 epay V2 5 分钟窗口）。
func checkTimestamp(ts string) error {
	if ts == "" {
		return &PayError{Code: 1103, Msg: "时间戳(timestamp)不能为空"}
	}
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return &PayError{Code: 1103, Msg: "时间戳(timestamp)格式不正确"}
	}
	diff := time.Now().Unix() - n
	if diff < 0 {
		diff = -diff
	}
	if diff > 300 {
		return &PayError{Code: 1103, Msg: "时间戳(timestamp)已过期"}
	}
	return nil
}

// Submit 处理下单请求。params 为原始请求参数（用于验签，含 sign/pid/type/... 全量）。
func (s *PayService) Submit(ctx context.Context, params map[string]string) (*dto.SubmitResp, error) {
	// 1. 商户存在性
	pid := parseUint(params["pid"])
	if pid == 0 {
		return nil, payErr("商户ID(pid)不能为空")
	}
	m, err := s.merchants.FindByUID(pid)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, payErr("商户不存在！")
	}
	if err != nil {
		return nil, err
	}

	// 2. 验签（对齐 epay ApiHelper::api_verify → verifySign）。
	//    keytype=0 兼容模式：按请求 sign_type 选 MD5/RSA；keytype=1 安全模式：强制 RSA。
	//    RSA(V2) 用商户公钥验商户私钥签名，并校验 timestamp ±300s（对齐 epay）。
	if err := s.verifySubmitSign(m, params); err != nil {
		return nil, err
	}

	// 3. 商户状态（对齐 epay：status/pay）
	if m.Status == 0 || m.Pay == 0 {
		return nil, payErr("商户已被封禁，无法支付！")
	}
	// pay==2(待审核) 仅在开启注册审核 user_review==1 时才拒付（G-6，对齐 epay Pay.php:37,233）。
	// 未开审核时 pay==2 视为可收款，放行。
	if m.Pay == 2 && (s.cfg == nil || s.cfg.Bool("user_review")) {
		return nil, payErr("商户未通过审核，无法支付！")
	}

	// 3b. 下单前置合规校验（A-6，对齐 epay Pay.php:64-77）。cfg 为 nil 时全部跳过（向后兼容）。
	if s.cfg != nil {
		if s.cfg.Bool("cert_force") && m.Cert == 0 {
			return nil, payErr("当前商户未完成实名认证，无法收款")
		}
		if s.cfg.Bool("forceqq") && strings.TrimSpace(m.QQ) == "" {
			return nil, payErr("当前商户未填写联系QQ，无法收款")
		}
		if s.cfg.Bool("user_deposit") {
			if min := s.cfg.Dec("user_deposit_min", decimal.Zero); min.GreaterThan(decimal.Zero) && m.Deposit.LessThan(min) {
				return nil, payErr("商户保证金不足，请前往支付平台充值保证金后再发起支付")
			}
		}
	}

	// 4. 参数校验（对齐 epay submit 的必填与格式）
	payType := params["type"]
	outTradeNo := params["out_trade_no"]
	notifyURL := params["notify_url"]
	returnURL := params["return_url"]
	name := params["name"]
	moneyStr := params["money"]

	if outTradeNo == "" {
		return nil, payErr("订单号(out_trade_no)不能为空")
	}
	if notifyURL == "" {
		return nil, payErr("通知地址(notify_url)不能为空")
	}
	// 表单页入口(version=0)要求 return_url 必填；V2 create(_version=1)不要求
	// （G-6，对齐 epay Pay.php:51 submit() 必填 return_url、create() 不校验）。
	if params["_version"] != "1" && returnURL == "" {
		return nil, payErr("跳转地址(return_url)不能为空")
	}
	if name == "" {
		return nil, payErr("商品名称(name)不能为空")
	}
	if moneyStr == "" {
		return nil, payErr("金额(money)不能为空")
	}
	if !outTradeNoRe.MatchString(outTradeNo) {
		return nil, payErr("订单号(out_trade_no)格式不正确")
	}
	amount, err := money.Parse(moneyStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return nil, payErr("金额不合法")
	}
	// 全局金额上下限（A-6，对齐 epay Pay.php:55-56 pay_maxmoney/pay_minmoney）。cfg 为 nil 或值≤0 时不限。
	if s.cfg != nil {
		if mx := s.cfg.Dec("pay_maxmoney", decimal.Zero); mx.GreaterThan(decimal.Zero) && amount.GreaterThan(mx) {
			return nil, payErr("支付金额超过最大限额 " + money.String(mx) + " 元")
		}
		if mn := s.cfg.Dec("pay_minmoney", decimal.Zero); mn.GreaterThan(decimal.Zero) && amount.LessThan(mn) {
			return nil, payErr("支付金额低于最小限额 " + money.String(mn) + " 元")
		}
	}
	// name 按 UTF-8 rune 安全截断（A-11，对齐 epay mb_strcut，避免字节切碎中文）。
	name = truncateRunes(name, 127)

	// 4b. 风控 / 黑名单 / 域名拦截（C4，对齐 epay Pay::submit 的下单前置校验）。
	//     命中黑名单统一返回模糊报错（不明示被拉黑）；关键词命中记风控并拦截；域名未过白拦截。
	clientIP := params["_ip"]
	notifyHost := hostOf(notifyURL)
	if s.blacklist != nil && s.blacklist.IsBlocked(1, clientIP) {
		return nil, payErr("系统异常无法完成付款")
	}
	// 域名白名单 gating（G-3，对齐 epay Pay.php:70-74）：仅当全局开关 pay_domain_forbid==1 才校验；
	// 开且商户无匹配 status=1 授权域名 → 拦截（含商户根本没配域名的情形，epay 也拦）。
	if s.domain != nil && notifyHost != "" && s.cfg != nil && s.cfg.Bool("pay_domain_forbid") {
		if !s.domain.IsAllowed(pid, notifyHost) {
			return nil, payErr("该域名不可发起支付，请前往支付平台授权支付域名")
		}
	}
	if s.risk != nil && hitKeyword(name) != "" {
		kw := hitKeyword(name)
		s.risk.RecordKeyword(pid, notifyHost, "商品名命中屏蔽词「"+kw+"」")
		return nil, payErr(blockAlert)
	}
	// pay_iplimit：同 IP 当日成功单数上限（A-6，对齐 epay Pay.php:92-97）。0=不限。
	if s.cfg != nil && clientIP != "" {
		if limit := s.cfg.Int("pay_iplimit", 0); limit > 0 {
			today := dayStart(time.Now())
			cnt, _ := s.orders.CountPaidByIPRange(clientIP, today, today.AddDate(0, 0, 1))
			if cnt >= int64(limit) {
				return nil, payErr("你今天已无法再发起支付，请明天再试")
			}
		}
	}

	// 5. 幂等：同 uid+out_trade_no 已存在且在 10 天窗口内则复用/拦截（对齐 epay Pay.php:110
	//    `time()-strtotime(addtime)<864000`）。超 10 天的旧单视为过期，允许用同 out_trade_no 重新建单。
	old, err := s.orders.FindByOut(pid, outTradeNo)
	if err != nil {
		return nil, err
	}
	if old != nil && time.Since(old.AddTime) < 10*24*time.Hour {
		if old.Status > 0 {
			return nil, payErr("该订单(" + outTradeNo + ")已完成支付，请勿重复发起支付")
		}
		// 参数一致性校验：金额/名称/回调/透传参数不一致视为参数变化（对齐 epay Pay.php:114 全字段比对）。
		if !old.Money.Equal(amount) || old.Name != name ||
			old.NotifyURL != notifyURL || old.ReturnURL != returnURL || old.Param != params["param"] {
			return nil, payErr("该订单(" + outTradeNo + ")支付参数有变化，请更换订单号重新发起支付")
		}
		// B1-04：裸单(收银台建的空 type 单，channel=0)复发起时若带上了具体 type，此处补选通道并回填订单，
		// 再走正常下单（epay 收银台选方式后 firstGetChannel 重新取通道信息）。仍空 type 则再跳收银台。
		if old.Channel == 0 && old.Plugin == "" {
			if strings.TrimSpace(payType) == "" {
				return &dto.SubmitResp{
					TradeNo: old.TradeNo, OutTradeNo: old.OutTradeNo, PayType: "jump",
					PayURL: cashierSelectURL(params["_siteurl"], old.TradeNo), Money: money.String(old.Money),
				}, nil
			}
			return s.upgradeBareOrder(ctx, m, old, params, payType, amount)
		}
		// B1-01：幂等复用单也要过通道单笔限额硬校验（epay Pay.php:169-174 在 firstGetChannel 之外，复用路径同样跑）。
		// 复用旧单的已定通道，用旧单金额比对；通道查不到则跳过（向后兼容 mock/阶段A）。
		if s.channels != nil && old.Channel > 0 {
			if och, _ := s.channels.FindByID(uint(old.Channel)); och != nil {
				if err := checkChannelPayLimit(och.PayMin, och.PayMax, old.Money); err != nil {
					return nil, err
				}
			}
		}
		return s.dispatch(ctx, old, payType, sceneFromParams(params))
	}

	// 6-0. 空 type（未指定支付方式）→ 建裸单(未定通道) + 跳收银台聚合选方式（B1-04，对齐 epay Pay.php:129-132）。
	//      epay：type 为空时不选通道，直接把用户带到 /cashier.php?trade_no=..，由收银台自选方式后再回来定通道。
	//      我方等价：建 channel=0/plugin="" 的裸单（realmoney=money，未加费），返回 PayType="jump" 指向收银台。
	//      用户在收银台选定方式后，前端以该 trade_no 复发起下单（携带 type），走幂等复用分支定通道下单。
	if strings.TrimSpace(payType) == "" {
		return s.createBareOrderForCashier(ctx, m, params, outTradeNo, notifyURL, returnURL, name, amount)
	}

	// 6. 选通道：按 商户用户组(gid) 对该支付方式的分配选出主通道 + 可选子通道 + 组级费率覆盖
	//    （对齐 epay Channel::getSubmitInfo：0关/-1随机/-2子通道/正整数固定或轮询组）。
	//    selector 未注入时退回旧版按 plugin 名定位单一通道（向后兼容阶段A/B）。
	sel, err := s.resolveChannel(m, payType, params["device"], amount)
	if err != nil {
		return nil, err
	}
	plugin, channelID, subchannelID, rate := sel.plugin, sel.channelID, sel.subchannelID, sel.rate

	// 6b. 选定通道后单笔限额硬拒绝（B1-01/B1-56，对齐 epay Pay.php:169-174）：
	//     epay 在选通道之后（firstGetChannel 之外）还有一段独立的通道级 paymin/paymax 硬校验，
	//     固定通道/随机兜底/子通道全过滤兜底等场景选出的越限通道都会在此被拒（channelFitsMoney 仅用于候选过滤，
	//     兜底可能返回越限通道）。用订单原额 amount 比对，与 epay 一致。
	if err := checkChannelPayLimit(sel.payMin, sel.payMax, amount); err != nil {
		return nil, err
	}

	// 7. 费率计算 + 实收金额（对齐 epay Pay.php:141-183）。
	//    - mode=0 平台代收：getmoney = money*rate/100（商户实得手续费差额），realmoney = money（可随机微调）。
	//    - mode=1 商户直清：realmoney = money*(200-rate)/100（加费，买家多付手续费），getmoney = money（商户实得全额）；
	//      并有最低手续费兜底 payfee_lessthan/payfee_mincost。
	//    rate 已含组级覆盖。realmoney 命中 pay_payaddstart 时随机微调防同额并单。
	getMoney, realMoney := s.calcFee(m, amount, rate)

	// 7a. 直清 mode=1 余额校验（对齐 epay Pay.php:177）：注意 epay 此处判的是「所选通道」的 mode
	//     (submitData['mode'])，而非商户 mode——费率计算用商户 mode(calcFee)，余额扣减门槛用通道 mode。
	//     直清通道下 realmoney-getmoney(手续费) 需从商户余额扣，余额不足则拦截。
	if s.channelModeIs(channelID, 1) && realMoney.Sub(getMoney).GreaterThan(m.Money) {
		return nil, payErr("当前商户余额不足，无法完成支付，请商户登录用户中心充值余额")
	}

	// 7b. 分账规则匹配（A-11，对齐 epay updateOrderProfits 三级优先 subchannel→uid→NULL）：
	//     传订单实际命中的 subchannelID，让 MatchRule 能按 channel+uid+subchannel 最精确匹配。
	//     命中则记规则 id 到 order.profits，支付成功回调时据此按比例创建分账订单。
	//     B1-55 传 plugin 做支持列表前置门；B1-19/54 用 realMoney（实收额）而非原始 amount 判 minmoney。
	var profits uint
	if s.profit != nil {
		profits = s.profit.MatchRuleForOrder(plugin, channelID, subchannelID, pid, realMoney)
	}

	// 8. 创建订单（status=0 未支付）。
	//    version：对齐 epay `defined('API_INIT')?1:0`——V2 REST 入口(mapi 注入 _version=1)记 1，
	//    回调时用平台私钥 RSA 签+timestamp；表单页入口(submit.php 等价)记 0，回调用商户 key MD5 签。
	now := time.Now()
	order := &model.Order{
		TradeNo:    genTradeNo(now),
		OutTradeNo: outTradeNo,
		UID:        pid,
		Domain:     hostOf(notifyURL),
		NotifyURL:  notifyURL,
		ReturnURL:  returnURL,
		Param:      params["param"],
		Name:       name,
		Money:      amount,
		RealMoney:  &realMoney,
		GetMoney:   getMoney,
		IP:         clientIP,
		Type:       0,
		TypeName:   payType,
		Channel:    channelID,
		Subchannel: subchannelID,
		Plugin:     plugin,
		AddTime:    now,
		Status:     0,
		Version:    orderVersion(params),
		Profits:    profits,
	}
	// 下单场景：把选通道时确定的 apptype（含子通道 info 覆盖）带给 dispatch，供插件按子形态分派。
	scene := sceneFromParams(params)
	scene.AppType = sel.apptype
	if err := s.orders.Create(order); err != nil {
		// 并发下唯一键冲突：重查一次走幂等分支
		if again, e := s.orders.FindByOut(pid, outTradeNo); e == nil && again != nil {
			return s.dispatch(ctx, again, payType, scene)
		}
		return nil, err
	}
	return s.dispatch(ctx, order, payType, scene)
}

// createBareOrderForCashier 建裸单(未定通道) + 返回跳收银台的 jump 响应（B1-04，对齐 epay Pay.php 空 type 分支）。
// 裸单：channel=0/subchannel=0/plugin=""/realmoney=money（未加费，待收银台选定方式后由复发起下单定通道）。
// 返回 PayType="jump"，PayURL 指向收银台聚合选方式页 /pay/cashier/<trade_no>（前端 replace 跳转）。
func (s *PayService) createBareOrderForCashier(ctx context.Context, m *model.Merchant, params map[string]string, outTradeNo, notifyURL, returnURL, name string, amount decimal.Decimal) (*dto.SubmitResp, error) {
	now := time.Now()
	realMoney := amount // 裸单未选通道、不加费：realmoney=money
	order := &model.Order{
		TradeNo:    genTradeNo(now),
		OutTradeNo: outTradeNo,
		UID:        m.UID,
		Domain:     hostOf(notifyURL),
		NotifyURL:  notifyURL,
		ReturnURL:  returnURL,
		Param:      params["param"],
		Name:       name,
		Money:      amount,
		RealMoney:  &realMoney,
		GetMoney:   decimal.Zero,
		IP:         params["_ip"],
		Type:       0,
		TypeName:   "", // 空 type：未定支付方式
		Channel:    0,  // 未定通道
		AddTime:    now,
		Status:     0,
		Version:    orderVersion(params),
	}
	if err := s.orders.Create(order); err != nil {
		// 并发唯一键冲突：重查复用（已存在的同单可能已定通道，直接返回其收银台跳转）。
		if again, e := s.orders.FindByOut(m.UID, outTradeNo); e == nil && again != nil {
			order = again
		} else {
			return nil, err
		}
	}
	return &dto.SubmitResp{
		TradeNo:    order.TradeNo,
		OutTradeNo: order.OutTradeNo,
		PayType:    "jump",
		PayURL:     cashierSelectURL(params["_siteurl"], order.TradeNo),
		Money:      money.String(order.Money),
	}, nil
}

// ChooseCashierPay 收银台选定支付方式（B1-04，对齐 epay cashier.php 选方式后对既有订单取通道信息）。
// 无需商户签名：订单已在空 type 下单时验签建号，此处仅凭 trade_no 对既有裸单补选通道并下单。
// device 为请求端标识（PC/mobile），用于设备过滤。仅允许对未支付(status=0)裸单(channel=0)操作。
func (s *PayService) ChooseCashierPay(ctx context.Context, tradeNo, payType, device string) (*dto.SubmitResp, error) {
	if strings.TrimSpace(payType) == "" {
		return nil, payErr("请选择支付方式")
	}
	o, err := s.orders.FindByTradeNo(tradeNo)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, payErr("订单不存在")
	}
	if o.Status != 0 {
		return nil, payErr("该订单已处理，请勿重复支付")
	}
	m, err := s.merchants.FindByUID(o.UID)
	if err != nil || m == nil {
		return nil, payErr("商户不存在")
	}
	params := map[string]string{"device": device}
	return s.upgradeBareOrder(ctx, m, o, params, payType, o.Money)
}

// upgradeBareOrder 裸单(收银台空 type 单)在收银台选定方式后补选通道 + 费率 + 分账规则并回填订单，
// 再走渠道下单（B1-04，对齐 epay 收银台选方式后 firstGetChannel 重新取通道信息）。
func (s *PayService) upgradeBareOrder(ctx context.Context, m *model.Merchant, old *model.Order, params map[string]string, payType string, amount decimal.Decimal) (*dto.SubmitResp, error) {
	sel, err := s.resolveChannel(m, payType, params["device"], amount)
	if err != nil {
		return nil, err
	}
	// 选定通道后单笔限额硬拒绝（对齐 epay Pay.php:169-174）。
	if err := checkChannelPayLimit(sel.payMin, sel.payMax, amount); err != nil {
		return nil, err
	}
	getMoney, realMoney := s.calcFee(m, amount, sel.rate)
	// 直清 mode=1 余额校验（对齐 epay Pay.php:177）。
	if s.channelModeIs(sel.channelID, 1) && realMoney.Sub(getMoney).GreaterThan(m.Money) {
		return nil, payErr("当前商户余额不足，无法完成支付，请商户登录用户中心充值余额")
	}
	var profits uint
	if s.profit != nil {
		profits = s.profit.MatchRuleForOrder(sel.plugin, sel.channelID, sel.subchannelID, m.UID, realMoney)
	}
	if err := s.orders.UpdateChannelInfo(old.TradeNo, sel.plugin, payType, sel.channelID, sel.subchannelID, realMoney, getMoney, profits); err != nil {
		return nil, err
	}
	// 回填内存态供 dispatch 用（避免再查一次库）。
	old.Plugin = sel.plugin
	old.TypeName = payType
	old.Channel = sel.channelID
	old.Subchannel = sel.subchannelID
	old.RealMoney = &realMoney
	old.GetMoney = getMoney
	old.Profits = profits
	scene := sceneFromParams(params)
	scene.AppType = sel.apptype
	return s.dispatch(ctx, old, payType, scene)
}

// cashierSelectURL 构造收银台聚合选方式页地址（对齐 epay /cashier.php?trade_no=..）。
// siteURL 为空时返回相对路径 /pay/cashier/<trade_no>（同源前端路由可直接消费）。
func cashierSelectURL(siteURL, tradeNo string) string {
	path := "/pay/cashier/" + tradeNo
	siteURL = strings.TrimSpace(siteURL)
	if siteURL == "" {
		return path
	}
	return strings.TrimRight(siteURL, "/") + path
}

// resolveChannel 选出下单主通道 + 可选子通道 + 最终费率。
// 优先走 selector（用户组通道分配/轮询/子通道/组级费率覆盖，对齐 epay getSubmitInfo）；
// selector 未注入或该 type 无法解析为支付方式ID时，退回旧版按 plugin 名定位单一通道
// （向后兼容阶段A mock 与阶段B epay/真实渠道：type 传插件名、通道表 plugin 匹配）。
// 返回 (plugin, channelID, subchannelID, rate)。
func (s *PayService) resolveChannel(m *model.Merchant, payType, device string, amount decimal.Decimal) (channelSelection, error) {
	if s.selector != nil {
		if typeID, ok := s.resolveTypeID(payType); ok {
			// device 分端过滤（对齐 epay Channel::submit L108-116）：移动端不出仅PC(device=1)方式，
			// PC 端不出仅移动(device=2)方式；device=0 通用。paytypes 未注入时跳过（向后兼容）。
			if err := s.checkPayTypeDevice(typeID, device); err != nil {
				return channelSelection{}, err
			}
			res, err := s.selector.Select(m.UID, m.GID, typeID, amount)
			if err != nil {
				return channelSelection{}, err
			}
			return channelSelection{
				plugin: res.Plugin, channelID: res.ChannelID,
				subchannelID: res.Subchannel, rate: res.Rate, apptype: res.AppType,
				payMin: res.PayMin, payMax: res.PayMax,
			}, nil
		}
	}
	// 退回旧版：type 当 plugin 名定位单一已开启通道。找不到则记 mock/零费率（阶段A向后兼容）。
	sel := channelSelection{plugin: "mock", rate: decimal.Zero}
	if payType != "" {
		ch, err := s.channels.FindEnabledByPlugin(payType)
		if err != nil {
			return channelSelection{}, err
		}
		if ch != nil {
			sel.plugin = ch.Plugin
			sel.channelID = int(ch.ID)
			sel.rate = ch.Rate
			sel.apptype = ch.AppType
			sel.payMin = ch.PayMin
			sel.payMax = ch.PayMax
		}
	}
	return sel, nil
}

// channelSelection 选通道结果（含 apptype，供下单场景按子形态分派 JSAPI/H5/scan 等，对齐 epay getSubmitInfo）。
type channelSelection struct {
	plugin       string
	channelID    int
	subchannelID int
	rate         decimal.Decimal
	apptype      string
	payMin       string // 选定通道单笔最小限额（选后硬拒绝，对齐 epay submitData['paymin']）
	payMax       string // 选定通道单笔最大限额（选后硬拒绝，对齐 epay submitData['paymax']）
}

// channelModeIs 判断所选通道的 mode 是否等于 want（对齐 epay submitData['mode']）。
// channelID<=0（阶段A退回 mock 无真实通道）或查不到通道时返回 false（视为非直清）。
func (s *PayService) channelModeIs(channelID, want int) bool {
	if channelID <= 0 {
		return false
	}
	ch, err := s.channels.FindByID(uint(channelID))
	if err != nil || ch == nil {
		return false
	}
	return ch.Mode == int8(want)
}

// resolveTypeID 把下单 type 参数解析为支付方式ID（pay_channel.type）。
//   - 纯数字：直接当 typeID。
//   - 插件名（如 alipay/wxpay/mock）：取该 plugin 下第一个已开启通道的 type。
//
// 无法解析（无匹配通道）返回 ok=false，调用方退回旧版 plugin 名直配。
func (s *PayService) resolveTypeID(payType string) (int, bool) {
	payType = strings.TrimSpace(payType)
	if payType == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(payType); err == nil {
		return n, true
	}
	ch, err := s.channels.FindEnabledByPlugin(payType)
	if err != nil || ch == nil {
		return 0, false
	}
	return ch.Type, true
}

// checkPayTypeDevice 校验支付方式的 device 与请求端匹配（对齐 epay Channel::submit L108-116）。
//   - 移动端请求(device∈{mobile,qq,wechat,alipay})：仅允许 device∈{0,2} 的支付方式
//   - PC 端请求(device 空或其它)：仅允许 device∈{0,1}
//   - 支付方式 device=0（通用）永远允许
//
// paytypes 未注入时不做校验（向后兼容）。命中不匹配返回错误，对齐 epay sysmsg('支付方式(type)不存在')。
func (s *PayService) checkPayTypeDevice(typeID int, device string) error {
	if s.paytypes == nil {
		return nil
	}
	pt, err := s.paytypes.FindByID(uint(typeID))
	if err != nil || pt == nil {
		return nil // 查不到不在此拦截，交由后续选通道处理
	}
	if pt.Device == 0 {
		return nil // 通用方式，PC/移动皆可
	}
	mobile := isMobileDevice(device)
	if mobile && pt.Device != 2 {
		return payErr("支付方式(type)不存在") // 移动端选到仅PC方式
	}
	if !mobile && pt.Device != 1 {
		return payErr("支付方式(type)不存在") // PC端选到仅移动方式
	}
	return nil
}

// isMobileDevice 判断下单请求是否移动端场景（对齐 epay device 判定 + checkmobile）。
func isMobileDevice(device string) bool {
	switch strings.ToLower(strings.TrimSpace(device)) {
	case "mobile", "qq", "wechat", "alipay", "wap", "jsapi", "h5":
		return true
	}
	return false
}

// QueryStatus 收银台轮询查单：先看本地订单状态，未付则主动问渠道 Query。
// 若渠道确认已支付，则走与回调一致的改单+入账+通知流程（幂等）。返回最终 status。
// 对齐微信"不能仅依赖回调，需结合查询接口"的要求，也为对账兜底。
func (s *PayService) QueryStatus(ctx context.Context, tradeNo string) (int8, error) {
	order, err := s.orders.FindByTradeNo(tradeNo)
	if err != nil {
		return 0, err
	}
	if order == nil {
		return 0, payErr("订单不存在")
	}
	// 已是终态直接返回，不再问渠道
	if order.Status != 0 {
		return order.Status, nil
	}
	// mock 渠道不主动查（以模拟支付回调为准），直接返回未付
	if order.Plugin == "mock" {
		return order.Status, nil
	}

	ch, ok := channel.Get(order.Plugin)
	if !ok {
		return order.Status, nil
	}
	cfg := s.loadChannelConfig(order.Channel)
	paid, err := ch.Query(ctx, cfg, order.TradeNo)
	if err != nil {
		// 查单失败不改变状态，返回当前未付（收银台继续轮询）
		return order.Status, nil
	}
	if !paid {
		return order.Status, nil
	}

	// 渠道确认已支付：走幂等改单 + 入账 + 通知（与回调同一路径）。
	flipped, err := s.orders.MarkPaid(order.TradeNo, "", "", time.Now())
	if err != nil {
		return order.Status, err
	}
	if flipped {
		if err := s.settle(ctx, order.TradeNo); err != nil {
			return 1, err
		}
	}
	return 1, nil
}

// loadChannelConfig 按通道 ID 载入其密钥配置。通道不存在或无 config 时返回零值 Config。
func (s *PayService) loadChannelConfig(channelID int) channel.Config {
	if channelID <= 0 {
		return channel.Config{Extra: map[string]string{}}
	}
	c, err := s.channels.FindByID(uint(channelID))
	if err != nil || c == nil {
		return channel.Config{Extra: map[string]string{}}
	}
	return buildChannelConfig(c)
}

// loadChannelConfigForOrder 载入订单所用通道配置，subchannel>0 时用子通道 info 覆盖主通道 config 占位
// （B1-34，对齐 epay Plugin::loadForPay/refund 的 subchannel>0 ? getSub : get）。
// 子通道仓储未注入 / subchannel<=0 / 子通道无 info 时退回主通道 config（向后兼容）。
func (s *PayService) loadChannelConfigForOrder(channelID, subchannelID int) channel.Config {
	if channelID <= 0 {
		return channel.Config{Extra: map[string]string{}}
	}
	c, err := s.channels.FindByID(uint(channelID))
	if err != nil || c == nil {
		return channel.Config{Extra: map[string]string{}}
	}
	if subchannelID <= 0 || s.subchannels == nil {
		return buildChannelConfig(c)
	}
	sub, err := s.subchannels.FindByID(uint(subchannelID))
	if err != nil || sub == nil || strings.TrimSpace(sub.Info) == "" {
		return buildChannelConfig(c)
	}
	kv := mergeSubChannelConfig(c.Config, sub.Info)
	if kv == nil {
		return buildChannelConfig(c)
	}
	return buildChannelConfigFromKV(kv)
}

// RefundViaChannel 尝试通过渠道原路退款（对齐 epay Order::refund 的渠道退款）。
// 返回 (是否已由渠道处理, error)。渠道未实现 Refunder 或插件不可用 → (false, nil)，由调用方走余额层。
// 渠道退款失败 → (false, err)，调用方据此决定是否中止。
func (s *PayService) RefundViaChannel(ctx context.Context, o *model.Order, money decimal.Decimal, outRefundNo string) (bool, error) {
	ch, ok := channel.Get(o.Plugin)
	if !ok {
		return false, nil // mock 等无真实渠道，走余额层
	}
	refunder, ok := ch.(channel.Refunder)
	if !ok {
		return false, nil // 该渠道不支持原路退款
	}
	// B1-34：退款侧同样按 subchannel>0 取子通道配置（对齐 epay Plugin::refund L145 getSub）。
	cfg := s.loadChannelConfigForOrder(o.Channel, o.Subchannel)
	total := o.Money
	if o.RealMoney != nil && o.RealMoney.GreaterThan(decimal.Zero) {
		total = *o.RealMoney
	}
	resp, err := refunder.Refund(ctx, cfg, channel.RefundReq{
		TradeNo:     o.TradeNo,
		ChannelNo:   o.APITradeNo,
		OutRefundNo: outRefundNo,
		Money:       money,
		TotalMoney:  total,
		Reason:      "订单退款",
	})
	if err != nil {
		return false, err
	}
	// 渠道请求无传输错误但明确未受理退款（Success=false）视为失败，
	// 不能据此翻转订单/扣商户余额（对齐 epay：渠道退款成功才继续）。
	if !resp.Success {
		return false, payErr("渠道未受理退款")
	}
	return true, nil
}

// GetCashier 返回收银台中间页所需的公开订单信息（无鉴权，仅安全字段）。
func (s *PayService) GetCashier(tradeNo string) (*dto.CashierView, error) {
	o, err := s.orders.FindByTradeNo(tradeNo)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, payErr("订单不存在")
	}
	// 收银台展示实际待付金额（realmoney，含加费/随机微调）；为空退回订单额。
	payMoney := o.Money
	if o.RealMoney != nil && o.RealMoney.GreaterThan(decimal.Zero) {
		payMoney = *o.RealMoney
	}
	// pageordername==1 时收银台商品名强制为 onlinepay（G-2，对齐 epay Payment.php:82/97）。
	dispName := o.Name
	if s.cfg != nil && s.cfg.Bool("pageordername") {
		dispName = "onlinepay"
	}
	view := &dto.CashierView{
		TradeNo:    o.TradeNo,
		OutTradeNo: o.OutTradeNo,
		Name:       dispName,
		Money:      money.String(payMoney),
		OrderMoney: money.String(o.Money), // B1-65：原始订单额，前端据 money≠order_money 显示'含X元手续费'
		Plugin:     o.Plugin,
		PayType:    o.PayType,
		QRCode:     o.QRCode,
		Status:     o.Status,
		AddTime:    o.AddTime.Format(timeLayout),
		ReturnURL:  s.buildReturnURL(o),
	}
	// B1-67：已付/终态订单服务端不再下发可支付信息（对齐 epay cashier.php:13 已付即中断，防重复支付）。
	//   epay 是整页 sysmsg 中断；我方 SPA 仍需 status+return_url 跳成功页，故保留这两项，
	//   仅抹掉二维码/支付方式/可选方式等「可继续支付」字段——直接请求 API 也拿不到已付单的可支付信息。
	if o.Status != 0 {
		view.QRCode = ""
		view.PayType = ""
		view.PayTypes = nil
		return view, nil
	}
	// B1-04：裸单(空 type 未定通道)返回可选支付方式，供收银台聚合选方式后以本 trade_no 复发起下单。
	if o.Channel == 0 && o.Plugin == "" {
		view.PayTypes = s.payTypeOptionsForOrder(o)
	}
	return view, nil
}

// payTypeOptionsForOrder 列出裸单可选支付方式（B1-04 收银台聚合选方式用，对齐 epay cashier 渲染 getTypes）。
// 复用商户所在组的可见性(IsTypeAvailable)+设备通用性+插件已实现判定；paytypes 未注入时返回空。
func (s *PayService) payTypeOptionsForOrder(o *model.Order) []dto.PayTypeOption {
	if s.paytypes == nil {
		return nil
	}
	types, err := s.paytypes.All()
	if err != nil {
		return nil
	}
	gid := -1
	if m, e := s.merchants.FindByUID(o.UID); e == nil && m != nil {
		gid = m.GID
	}
	out := make([]dto.PayTypeOption, 0, len(types))
	seen := map[string]bool{}
	for i := range types {
		t := &types[i]
		if t.Status != 1 {
			continue
		}
		// 组可见性过滤（channel=0 隐藏 / -1 有启用通道 / -2 有子通道 / 正整数校验），selector+gid 有效才做。
		if s.selector != nil && gid >= 0 {
			if !s.selector.IsTypeAvailable(o.UID, gid, int(t.ID)) {
				continue
			}
		}
		ch, err := s.channels.FindEnabledByPlugin(t.Name)
		if err != nil || ch == nil {
			continue
		}
		if _, ok := channel.Get(ch.Plugin); !ok {
			continue // 插件未实现（seed 显示名），不作为可选项
		}
		if seen[ch.Plugin] {
			continue
		}
		seen[ch.Plugin] = true
		out = append(out, dto.PayTypeOption{Type: ch.Plugin, ShowName: t.ShowName})
	}
	return out
}

// buildReturnURL 构造支付完成后跳回商户的 return_url（A-10，对齐 epay processOrder !isnotify 分支）：
//   - status==2(退款) → 空(前端跳 payerr)；
//   - 已付且完成超 5 分钟 → 裸 return_url(不带签名参数，等价 epay payok)；
//   - tid>0(内部订单：充值/购组/保证金/测试) → 裸 return_url(epay tid>0 不带 query)；
//   - 其余已付 → return_url 带签名回调参数(商户 return 页可验签)；
//   - 未付或无 return_url → 原样返回 return_url。
func (s *PayService) buildReturnURL(o *model.Order) string {
	if o.ReturnURL == "" {
		return ""
	}
	if o.Status == 2 {
		return "" // 退款单，前端跳失败页
	}
	if o.Status != 1 {
		return o.ReturnURL // 未付：原样（前端此时不会跳转）
	}
	if o.Tid > 0 {
		return o.ReturnURL // 内部订单不带签名参数
	}
	if o.EndTime != nil && time.Since(*o.EndTime) > 5*time.Minute {
		return o.ReturnURL // 完成超 5 分钟：裸跳（对齐 epay 5 分钟禁带参跳转）
	}
	// 已付且在 5 分钟内：带签名回调参数回跳，供商户 return 页验签。
	m, err := s.merchants.FindByUID(o.UID)
	if err != nil {
		return o.ReturnURL
	}
	params := s.buildCallbackParams(o, m)
	return appendQuery(o.ReturnURL, params)
}

// CreateInternalOrder 创建内部业务订单（充值余额 tid=2 等）并走渠道下单，返回收银台信息。
// 对齐 epay：内部订单下到收款商户名下、回调时按 tid 分派。当前 uid 直接记发起商户，
// settle() 按 tid 决定入账流水类型。plugin 指定渠道（如 mock 可真跑；真实渠道待凭证）。
func (s *PayService) CreateInternalOrder(ctx context.Context, uid uint, tid int8, name string, amount decimal.Decimal, plugin string, param ...string) (*dto.SubmitResp, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, payErr("金额不合法")
	}
	ch, err := s.channels.FindEnabledByPlugin(plugin)
	if err != nil {
		return nil, err
	}
	channelID := 0
	rate := decimal.Zero
	if ch != nil {
		channelID = int(ch.ID)
		rate = ch.Rate
	}
	// 费率计算（B1-53，对齐 epay submit2.php:35）：内部订单同样按 tid 维度加费——
	//   · tid==2 余额充值：无论商户 mode 都走加费（买家多付手续费 realmoney=money*(200-rate)/100、getmoney=money 全额到账）；
	//   · tid==4 购买用户组：即便 mode==1 也强制平台代收（不加费）；
	//   · 其余（tid=3 聚合收款等）：按下单商户自身 mode 判定加费/代收。
	// 收款商户按 tid 语义解析：tid=2/4/5 收款方是 param.uid（内部业务方），否则就是下单 uid。
	payUID := uid
	if len(param) > 0 && param[0] != "" {
		if v := parseParamUID(param[0]); v > 0 {
			payUID = v
		}
	}
	m, _ := s.merchants.FindByUID(payUID)
	getMoney, realMoney := s.calcInternalFee(m, tid, amount, rate)
	now := time.Now()
	order := &model.Order{
		TradeNo:    genTradeNo(now),
		OutTradeNo: fmt.Sprintf("IN%d%s", tid, genTradeNo(now)),
		UID:        payUID,
		Name:       name,
		Money:      amount,
		RealMoney:  &realMoney,
		GetMoney:   getMoney,
		Type:       0,
		TypeName:   plugin,
		Channel:    channelID,
		Plugin:     plugin,
		AddTime:    now,
		Status:     0,
		Tid:        tid,
	}
	if len(param) > 0 {
		order.Param = param[0]
	}
	if err := s.orders.Create(order); err != nil {
		return nil, err
	}
	return s.dispatch(ctx, order, plugin)
}

// calcInternalFee 内部订单费率计算，对齐 epay submit2.php:35 的 tid 维度加费判定。
// 条件 (mode==1 && tid!=4) || tid==2 为真 → 加费（买家多付）；否则平台代收。
// m 为 nil（收款商户查不到）时按非加费(平台代收)安全默认处理。
func (s *PayService) calcInternalFee(m *model.Merchant, tid int8, money, rate decimal.Decimal) (getMoney, realMoney decimal.Decimal) {
	mode := int8(0)
	if m != nil {
		mode = m.Mode
	}
	if (mode == 1 && tid != 4) || tid == 2 {
		// 加费模式：买家多付手续费，getmoney=全额到账。
		return s.calcFee(&model.Merchant{Mode: 1}, money, rate)
	}
	// 平台代收：realmoney=money，getmoney=money*rate/100。
	return s.calcFee(&model.Merchant{Mode: 0}, money, rate)
}

// internalOrderParam 内部订单 param（JSON）承载的业务字段（对齐 epay processOrder 从 param 解析 uid/gid/endtime）。
// uid/gid 兼容 JSON 数字与字符串两种编码（json.Number 统一解析）。
type internalOrderParam struct {
	UID     json.Number `json:"uid"`
	GID     json.Number `json:"gid"`
	EndTime string      `json:"endtime"`
}

// parseInternalParam 解析内部订单 param JSON，非法则返回零值结构（各字段按缺省处理）。
func parseInternalParam(raw string) internalOrderParam {
	var p internalOrderParam
	if raw == "" {
		return p
	}
	_ = json.Unmarshal([]byte(raw), &p)
	return p
}

// parseParamUID 从内部订单 param JSON 取收款商户 uid（tid=2/4/5 的实际入账/操作对象）。
func parseParamUID(raw string) uint {
	if v, err := parseInternalParam(raw).UID.Int64(); err == nil && v > 0 {
		return uint(v)
	}
	return 0
}

// parseParamEndTime 解析 param 里的用户组到期时间（tid=4 购组），空串/非法返回 nil（永久组，对齐 epay endtime null）。
// 支持 "2006-01-02 15:04:05" 与 "2006-01-02" 两种格式。
func parseParamEndTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t
		}
	}
	return nil
}

// CreateTestOrderByChannel 后台测试支付：指定通道 ID(+可选子通道)下一笔真实测试单走收单链
// （1:1 对齐 epay admin/ajax_pay.php act=testpay：固定 channel/subchannel、tid=3、收款方 test_pay_uid）。
// 与 CreateInternalOrder 的区别是这里锁定具体通道而非按 plugin 名选首个启用通道，
// 让运营能定向验证某个通道（含子通道）密钥配置是否可用。
func (s *PayService) CreateTestOrderByChannel(ctx context.Context, channelID, subchannel int, name string, amount decimal.Decimal) (*dto.SubmitResp, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, payErr("金额不合法")
	}
	ch, err := s.channels.FindByID(uint(channelID))
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, payErr("当前支付通道不存在")
	}
	// 全局金额限额（对齐 epay pay_maxmoney/pay_minmoney）。
	if s.cfg != nil {
		if minM := s.cfg.Dec("pay_minmoney", decimal.Zero); minM.GreaterThan(decimal.Zero) && amount.LessThan(minM) {
			return nil, payErr("最小支付金额是 " + minM.String() + " 元")
		}
		if maxM := s.cfg.Dec("pay_maxmoney", decimal.Zero); maxM.GreaterThan(decimal.Zero) && amount.GreaterThan(maxM) {
			return nil, payErr("最大支付金额是 " + maxM.String() + " 元")
		}
	}
	// 收款方：test_pay_uid 配置了则用固定测试商户，否则下到 uid=1（对齐 epay test_pay_uid）。
	payUID := uint(1)
	if s.cfg != nil {
		if tp := s.cfg.Int("test_pay_uid", 0); tp > 0 {
			payUID = uint(tp)
		}
	}
	if name == "" {
		name = "支付测试"
	}
	now := time.Now()
	order := &model.Order{
		TradeNo:    genTradeNo(now),
		OutTradeNo: fmt.Sprintf("TEST%s", genTradeNo(now)),
		UID:        payUID,
		Name:       name,
		Money:      amount,
		RealMoney:  &amount,
		GetMoney:   amount,
		Type:       ch.Type,
		TypeName:   ch.Plugin,
		Channel:    channelID,
		Subchannel: subchannel,
		Plugin:     ch.Plugin,
		AddTime:    now,
		Status:     0,
		Tid:        3,
	}
	if err := s.orders.Create(order); err != nil {
		return nil, err
	}
	return s.dispatch(ctx, order, ch.Plugin)
}

// resolveApptype 解析本单生效的 apptype（对齐 epay getSubmitInfo/getSub apptype）：
// 优先取 scene 里选通道时确定的 apptype（含子通道 info 覆盖）；否则读所选通道的 apptype
//（补单/查单复用 dispatch 但无 scene 时的兜底，等价 epay Channel::get(apptype)）。
func (s *PayService) resolveApptype(o *model.Order, scene ...*sceneParams) string {
	if len(scene) > 0 && scene[0] != nil && strings.TrimSpace(scene[0].AppType) != "" {
		return scene[0].AppType
	}
	if o.Channel <= 0 || s.channels == nil {
		return ""
	}
	ch, err := s.channels.FindByID(uint(o.Channel))
	if err != nil || ch == nil {
		return ""
	}
	return ch.AppType
}

// sceneParams 下单场景参数（A-2，对齐 epay device/method/sub_openid/auth_code + apptype）。
type sceneParams struct {
	Method, Device, SubOpenID, SubAppID, AuthCode, AppType string
}

// sceneFromParams 从原始请求参数提取下单场景参数。
func sceneFromParams(params map[string]string) *sceneParams {
	return &sceneParams{
		Method:    params["method"],
		Device:    params["device"],
		SubOpenID: params["sub_openid"],
		SubAppID:  params["sub_appid"],
		AuthCode:  params["auth_code"],
	}
}

// dispatch 调用渠道下单，构造对外返回。scene 可空（内部订单/收银台无场景参数）。
func (s *PayService) dispatch(ctx context.Context, o *model.Order, payType string, scene ...*sceneParams) (*dto.SubmitResp, error) {
	ch, ok := channel.Get(o.Plugin)
	if !ok {
		return nil, payErr("支付渠道不可用：" + o.Plugin)
	}
	// 载入通道密钥配置（真实渠道用；mock 通道 config 为空返回零值 Config）。
	// B1-34：subchannel>0 时用子通道 info 覆盖主通道 config 占位（对齐 epay Plugin::loadForPay getSub）。
	cfg := s.loadChannelConfigForOrder(o.Channel, o.Subchannel)
	// 渠道回调地址 = 通道配置的 notify_url 基址 + /系统订单号，命中本系统 /api/pay/notify/:trade_no。
	cfg.NotifyURL = notifyBackURL(cfg.NotifyURL, o.TradeNo)
	// 发给渠道的是实际待付金额 realmoney（含 randomFloat 随机微调 + mode=1 加费），而非原始订单额 money
	//（对齐 epay 插件一律用 $order['realmoney'] 下单）。realmoney 为空/≤0 时退回 money（mock/内部单）。
	payAmount := o.Money
	if o.RealMoney != nil && o.RealMoney.GreaterThan(decimal.Zero) {
		payAmount = *o.RealMoney
	}
	// 发给渠道的商品名 subject：优先商户级 ordername 模板，退回全局 conf.ordername，均空则原样订单名
	//（G-2，对齐 epay Plugin.php:58-59 + functions.php:720 ordername_replace）。
	subject := s.resolveOrderName(o)
	req := channel.CreateReq{
		TradeNo:   o.TradeNo,
		Money:     payAmount,
		Subject:   subject,
		NotifyURL: cfg.NotifyURL,
		ReturnURL: o.ReturnURL,
		ClientIP:  o.IP,
		// 支付方式英文名（alipay/wxpay/qqpay/bank）透传给聚合渠道，用于其内部 paytype 映射
		//（对齐 epay 插件 $order['typename']；vmq/epay 等据此选上游收单方式）。
		Extra: map[string]string{"typename": payType},
		// L-7 渠道侧分账标记：本单命中分账规则（Profits>0）时置位，让支持分账的渠道下单带
		// settle_info.profit_sharing（对齐 epay wxpayn/adapay）；不支持的渠道忽略，走本地余额层分账。
		ProfitSharing: o.Profits > 0,
	}
	// apptype 透传（对齐 epay getSubmitInfo 返回 apptype → 插件按 in_array($apptype) 分派 JSAPI/H5/扫码）。
	// 优先取 scene 里选通道时定的 apptype；否则读所选通道的 apptype（Fill/Query 复用路径）。
	if at := s.resolveApptype(o, scene...); at != "" {
		req.Extra["apptype"] = at
	}
	// 注入下单场景参数（A-2）：渠道按需消费（JSAPI/付款码/APP 等），mock/不支持则忽略。
	if len(scene) > 0 && scene[0] != nil {
		sp := scene[0]
		req.Method, req.Device = sp.Method, sp.Device
		req.SubOpenID, req.SubAppID, req.AuthCode = sp.SubOpenID, sp.SubAppID, sp.AuthCode
	}
	cr, err := ch.Create(ctx, cfg, req)
	if err != nil {
		return nil, payErr("渠道下单失败：" + err.Error())
	}
	// 回填收银台渲染信息（二维码/支付链接），供 GET /pay/order/:trade_no 展示。
	// 优先存 QRCode，其次 PayURL；失败不阻断下单（仅影响收银台展示，可重新下单）。
	payInfo := cr.QRCode
	if payInfo == "" {
		payInfo = cr.PayURL
	}
	_ = s.orders.SavePayInfo(o.TradeNo, string(cr.PayType), payInfo)
	return &dto.SubmitResp{
		TradeNo:    o.TradeNo,
		OutTradeNo: o.OutTradeNo,
		PayType:    string(cr.PayType),
		PayURL:     cr.PayURL,
		QRCode:     cr.QRCode,
		RawHTML:    cr.RawHTML,
		Money:      money.String(o.Money),
	}, nil
}

// resolveOrderName 计算发给渠道的商品名（对齐 epay Plugin loadForSubmit + ordername_replace）：
// 优先商户级 ordername 模板，退回全局 conf.ordername；模板非空时替换占位符
// [name]/[order]/[outorder]/[qq]/[phone]，否则退回订单原始名。
func (s *PayService) resolveOrderName(o *model.Order) string {
	tpl := ""
	var m *model.Merchant
	if s.merchants != nil {
		m, _ = s.merchants.FindByUID(o.UID)
		if m != nil {
			tpl = strings.TrimSpace(m.OrderName)
		}
	}
	if tpl == "" && s.cfg != nil {
		tpl = strings.TrimSpace(s.cfg.Str("ordername"))
	}
	if tpl == "" {
		return o.Name
	}
	name := tpl
	name = strings.ReplaceAll(name, "[name]", o.Name)
	name = strings.ReplaceAll(name, "[order]", o.TradeNo)
	name = strings.ReplaceAll(name, "[outorder]", o.OutTradeNo)
	if m != nil {
		name = strings.ReplaceAll(name, "[qq]", m.QQ)
		name = strings.ReplaceAll(name, "[phone]", m.Phone)
	}
	return name
}
