package service

import (
	"context"
	"errors"
	"fmt"
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
	risk      *RiskService      // 风控关键词拦截（可空）
	blacklist *BlacklistService // 黑名单拦截（可空）
	domain    *DomainService    // 域名白名单校验（可空）
}

func NewPayService(m *repository.MerchantRepo, o *repository.OrderRepo, a *repository.AccountRepo, ch *repository.ChannelRepo) *PayService {
	return &PayService{merchants: m, orders: o, accounts: a, channels: ch}
}

// SetProfitService 注入分账服务（下单匹配规则 + 支付成功建分账单）。nil 则不启用分账。
func (s *PayService) SetProfitService(p *ProfitService) { s.profit = p }

// SetRiskServices 注入风控/黑名单/域名服务（下单拦截校验）。任一为 nil 则跳过对应校验。
func (s *PayService) SetRiskServices(r *RiskService, b *BlacklistService, d *DomainService) {
	s.risk, s.blacklist, s.domain = r, b, d
}

// blockKeywords 关键词屏蔽词（对齐 epay blockname，先固化，待 config 域迁移）。
var blockKeywords = []string{"博彩", "赌博", "违禁", "毒品", "枪支"}

// hitKeyword 返回商品名命中的第一个屏蔽词，未命中返回空串（对齐 epay strpos 子串匹配）。
func hitKeyword(name string) string {
	for _, kw := range blockKeywords {
		if strings.Contains(name, kw) {
			return kw
		}
	}
	return ""
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
	if m.Pay == 2 {
		return nil, payErr("商户未通过审核，无法支付！")
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
	// name 超长截断，对齐 epay 的 127 字节
	if len(name) > 127 {
		name = name[:127]
	}

	// 4b. 风控 / 黑名单 / 域名拦截（C4，对齐 epay Pay::submit 的下单前置校验）。
	//     命中黑名单统一返回模糊报错（不明示被拉黑）；关键词命中记风控并拦截；域名未过白拦截。
	clientIP := params["_ip"]
	notifyHost := hostOf(notifyURL)
	if s.blacklist != nil && s.blacklist.IsBlocked(1, clientIP) {
		return nil, payErr("系统异常无法完成付款")
	}
	if s.domain != nil && notifyHost != "" && !s.domain.IsAllowed(pid, notifyHost) {
		// 域名白名单：仅当该商户配置了域名校验时才拦截。无任何域名记录视为未开启，放行（向后兼容）。
		if s.domain.HasAnyDomain(pid) {
			return nil, payErr("该域名不可发起支付，请前往支付平台授权支付域名")
		}
	}
	if s.risk != nil && hitKeyword(name) != "" {
		kw := hitKeyword(name)
		s.risk.RecordKeyword(pid, notifyHost, "商品名命中屏蔽词「"+kw+"」")
		return nil, payErr("温馨提醒该商品禁止出售，如有疑问请联系网站客服！")
	}

	// 5. 幂等：同 uid+out_trade_no 已存在则复用/拦截（对齐 epay 10 天窗口内校验）
	old, err := s.orders.FindByOut(pid, outTradeNo)
	if err != nil {
		return nil, err
	}
	if old != nil {
		if old.Status > 0 {
			return nil, payErr("该订单(" + outTradeNo + ")已完成支付，请勿重复发起支付")
		}
		// 参数一致性校验：金额/名称/回调不一致视为参数变化
		if !old.Money.Equal(amount) || old.Name != name {
			return nil, payErr("该订单(" + outTradeNo + ")支付参数有变化，请更换订单号重新发起支付")
		}
		return s.dispatch(ctx, old, payType)
	}

	// 6. 选通道：按 type 定位一个已开启通道，拿到 plugin/费率（对齐 epay Channel::submit 的简化版，
	//    轮询/加权留待 pay_roll 阶段）。阶段A的 type 传 "mock"，通道表里 plugin=mock 的记录承接。
	ch, err := s.pickChannel(payType)
	if err != nil {
		return nil, err
	}

	// 7. 费率计算（平台代收 mode=0）：getmoney = money * rate / 100，四舍五入两位。
	//    商户直清 mode=1 的加费逻辑留待后续，阶段C1先支持代收。
	getMoney := decimal.Zero
	plugin := "mock"
	channelID := 0
	if ch != nil {
		plugin = ch.Plugin
		channelID = int(ch.ID)
		getMoney = amount.Mul(ch.Rate).Div(decimal.NewFromInt(100)).Round(2)
	}

	// 7b. 分账规则匹配（对齐 epay updateOrderProfits）：命中则记规则 id 到 order.profits，
	//     支付成功回调时据此按比例创建分账订单。realmoney 用订单金额（无独立实收字段时）。
	var profits uint
	if s.profit != nil {
		profits = s.profit.MatchRuleForOrder(channelID, 0, pid, amount)
	}

	// 8. 创建订单（status=0 未支付）。
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
		GetMoney:   getMoney,
		Type:       0,
		TypeName:   payType,
		Channel:    channelID,
		Plugin:     plugin,
		AddTime:    now,
		Status:     0,
		Profits:    profits,
	}
	if err := s.orders.Create(order); err != nil {
		// 并发下唯一键冲突：重查一次走幂等分支
		if again, e := s.orders.FindByOut(pid, outTradeNo); e == nil && again != nil {
			return s.dispatch(ctx, again, payType)
		}
		return nil, err
	}
	return s.dispatch(ctx, order, payType)
}

// pickChannel 按下单传入的 type 选一个可用通道。
// 阶段A/C1 简化策略：type 先当 plugin 名匹配已开启通道（mock/alipay/wxpay 等插件名）。
// 找不到通道记录时返回 (nil, nil)——保持 mock 渠道在通道表为空时仍可下单（向后兼容阶段A）。
func (s *PayService) pickChannel(payType string) (*model.Channel, error) {
	if payType == "" {
		return nil, nil
	}
	return s.channels.FindEnabledByPlugin(payType)
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

// GetCashier 返回收银台中间页所需的公开订单信息（无鉴权，仅安全字段）。
func (s *PayService) GetCashier(tradeNo string) (*dto.CashierView, error) {
	o, err := s.orders.FindByTradeNo(tradeNo)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, payErr("订单不存在")
	}
	return &dto.CashierView{
		TradeNo:    o.TradeNo,
		OutTradeNo: o.OutTradeNo,
		Name:       o.Name,
		Money:      money.String(o.Money),
		Plugin:     o.Plugin,
		PayType:    o.PayType,
		QRCode:     o.QRCode,
		Status:     o.Status,
		AddTime:    o.AddTime.Format(timeLayout),
		ReturnURL:  o.ReturnURL,
	}, nil
}

// CreateInternalOrder 创建内部业务订单（充值余额 tid=2 等）并走渠道下单，返回收银台信息。
// 对齐 epay：内部订单下到收款商户名下、回调时按 tid 分派。当前 uid 直接记发起商户，
// settle() 按 tid 决定入账流水类型。plugin 指定渠道（如 mock 可真跑；真实渠道待凭证）。
func (s *PayService) CreateInternalOrder(ctx context.Context, uid uint, tid int8, name string, amount decimal.Decimal, plugin string) (*dto.SubmitResp, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, payErr("金额不合法")
	}
	ch, err := s.channels.FindEnabledByPlugin(plugin)
	if err != nil {
		return nil, err
	}
	channelID := 0
	if ch != nil {
		channelID = int(ch.ID)
	}
	now := time.Now()
	order := &model.Order{
		TradeNo:    genTradeNo(now),
		OutTradeNo: fmt.Sprintf("IN%d%s", tid, genTradeNo(now)),
		UID:        uid,
		Name:       name,
		Money:      amount,
		Type:       0,
		TypeName:   plugin,
		Channel:    channelID,
		Plugin:     plugin,
		AddTime:    now,
		Status:     0,
		Tid:        tid,
	}
	if err := s.orders.Create(order); err != nil {
		return nil, err
	}
	return s.dispatch(ctx, order, plugin)
}

// dispatch 调用渠道下单，构造对外返回。
func (s *PayService) dispatch(ctx context.Context, o *model.Order, payType string) (*dto.SubmitResp, error) {
	ch, ok := channel.Get(o.Plugin)
	if !ok {
		return nil, payErr("支付渠道不可用：" + o.Plugin)
	}
	// 载入通道密钥配置（真实渠道用；mock 通道 config 为空返回零值 Config）。
	cfg := s.loadChannelConfig(o.Channel)
	// 渠道回调地址 = 通道配置的 notify_url 基址 + /系统订单号，命中本系统 /api/pay/notify/:trade_no。
	cfg.NotifyURL = notifyBackURL(cfg.NotifyURL, o.TradeNo)
	cr, err := ch.Create(ctx, cfg, channel.CreateReq{
		TradeNo:   o.TradeNo,
		Money:     o.Money,
		Subject:   o.Name,
		NotifyURL: cfg.NotifyURL,
	})
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
		Money:      money.String(o.Money),
	}, nil
}
