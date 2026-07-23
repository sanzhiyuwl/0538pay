package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/pkg/money"
	"github.com/shopspring/decimal"
)

// PayCreate 处理 V2 REST 下单（对齐 epay Pay::create）。
// 复用 PayService.Submit 核心（验签在 Submit 内做，与老 submit 一致），
// 返回 JSON 契约：code=0 + trade_no + pay_type + pay_info（+ 回包签名由 handler 调 signResponse）。
// 注意：Submit 内部已做 pid/验签/风控/选通道/建单/dispatch，这里只做结果到 V2 契约的映射。
func (s *MapiService) PayCreate(ctx context.Context, params map[string]string) (map[string]string, error) {
	// B1-05：V2 create clientip 必填（对齐 epay Pay::create:241,262）。
	// V2 由商户服务器代传真实买家 IP，用于风控/IP限额/落库 order.ip，为空即拒。
	clientIP := strings.TrimSpace(params["clientip"])
	if clientIP == "" {
		return nil, mapiErrCode(-4, "用户IP地址(clientip)不能为空")
	}
	params["_ip"] = clientIP
	// V2 REST 入口标记（对齐 epay api.php define API_INIT）：订单 version=1，回调走平台私钥 RSA 签+timestamp。
	params["_version"] = "1"
	resp, err := s.pay.Submit(ctx, params)
	if err != nil {
		return nil, toMapiErr(err)
	}
	// pay_type 映射为 epay V2 契约（A-2，对齐 Payment::echoJson 9 种）。
	// 内部渠道 PayType(qrcode/redirect/html/wap) → V2 契约(qrcode/jump/html/urlscheme/jsapi/app/scan…)。
	// 渠道已返 V2 原生值(jsapi/app/scan/urlscheme/wxplugin/wxapp)则透传。
	payType, payInfo := mapV2PayType(resp.PayType, resp.QRCode, resp.PayURL, resp.RawHTML)
	// B1-18：未知支付形态或载荷全空时，回落收银台 URL + pay_type=jump
	// （对齐 epay Payment::echoJson default: pay_type=jump, pay_info=siteurl.'pay/submit/TRADE_NO/'）。
	if payInfo == "" {
		payType = "jump"
		payInfo = cashierFallbackURL(params["_siteurl"], resp.TradeNo)
	}
	out := map[string]string{
		"code":      "0",
		"trade_no":  resp.TradeNo,
		"pay_type":  payType,
		"pay_info":  payInfo,
	}
	return out, nil
}

// cashierFallbackURL 构造收银台回落地址（对齐 epay $siteurl.'pay/submit/TRADE_NO/'）。
// siteurl 为空时退回相对路径，保证 pay_info 恒非空可用。
func cashierFallbackURL(siteURL, tradeNo string) string {
	base := strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if base == "" {
		return "/pay/submit/" + tradeNo + "/"
	}
	return base + "/pay/submit/" + tradeNo + "/"
}

// mapV2PayType 把内部渠道 PayType + 载荷映射为 epay V2 pay_type + pay_info（A-2，对齐 Payment::echoJson）。
// 内部值 qrcode/redirect/html/wap → V2 qrcode/jump/html/urlscheme；渠道已返 V2 原生值(jsapi/app/scan/
// urlscheme/wxplugin/wxapp)则透传。pay_info 优先取 qrcode 码串，其次 HTML 载荷，再次跳转 URL。
func mapV2PayType(internal, qrcode, payURL, rawHTML string) (payType, payInfo string) {
	switch internal {
	case "qrcode":
		return "qrcode", firstNonEmpty(qrcode, payURL)
	case "html":
		return "html", firstNonEmpty(rawHTML, payURL)
	case "redirect", "jump", "":
		return "jump", firstNonEmpty(payURL, qrcode)
	case "wap":
		return "jump", firstNonEmpty(payURL, qrcode) // H5 网页跳转
	case "urlscheme", "scheme":
		return "urlscheme", firstNonEmpty(payURL, qrcode)
	case "jsapi":
		return "jsapi", firstNonEmpty(rawHTML, payURL)
	case "app":
		return "app", firstNonEmpty(rawHTML, payURL)
	case "scan":
		return "scan", firstNonEmpty(rawHTML, payURL)
	case "wxplugin":
		return "wxplugin", firstNonEmpty(rawHTML, payURL)
	case "wxapp":
		return "wxapp", firstNonEmpty(rawHTML, payURL)
	default:
		// 未知类型透传原值 + 合理载荷（向前兼容渠道新增类型）。
		return internal, firstNonEmpty(qrcode, rawHTML, payURL)
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// PayCreateClassic 处理经典 mapi.php JSON 下单（对齐 epay Payment::echoJson 的 else 分支，version=0）。
// 与 PayCreate 复用同一 Submit 核心，仅输出契约不同：
//   - code=1（成功）而非 code=0
//   - 按支付形态映射到不同顶层字段(payurl/html/qrcode/urlscheme)，而非统一 pay_type+pay_info
//   - 不做回包签名（经典契约无 sign/timestamp/sign_type）
// 订单 version=0（回调走商户 key MD5 签名，对齐 epay 未 define API_INIT 分支）。
func (s *MapiService) PayCreateClassic(ctx context.Context, params map[string]string) (map[string]interface{}, error) {
	if ip := strings.TrimSpace(params["clientip"]); ip != "" {
		params["_ip"] = ip
	}
	// 经典 mapi.php 入口：订单 version=0，回调走商户 key MD5（对齐 epay 未 define API_INIT）。
	params["_version"] = "0"
	resp, err := s.pay.Submit(ctx, params)
	if err != nil {
		// 经典契约错误：code=-2 + msg（对齐 echoJson case 'error'）。
		me := toMapiErr(err)
		msg := me.Error()
		if m2, ok := me.(*MapiError); ok {
			msg = m2.Msg
		}
		return map[string]interface{}{"code": -2, "msg": msg}, nil
	}
	out := map[string]interface{}{
		"code":     1,
		"trade_no": resp.TradeNo,
	}
	// 按内部 PayType 映射到经典顶层字段（对齐 echoJson else 分支 switch）。
	payType, payInfo := mapV2PayType(resp.PayType, resp.QRCode, resp.PayURL, resp.RawHTML)
	switch payType {
	case "qrcode":
		out["qrcode"] = payInfo
	case "html":
		out["html"] = payInfo
	case "urlscheme":
		out["urlscheme"] = payInfo
	case "jsapi", "app", "scan", "wxplugin", "wxapp":
		// 经典契约无这些专属字段，epay 统一回落跳转收银台；这里保留形态值+载荷兼容新渠道。
		out[payType] = payInfo
	default: // jump 及未知
		// B1-18：经典契约 default 同样回落收银台 URL（对齐 echoJson else 分支 default: payurl=siteurl.'pay/submit/TRADE_NO/'）。
		if payInfo == "" {
			payInfo = cashierFallbackURL(params["_siteurl"], resp.TradeNo)
		}
		out["payurl"] = payInfo
	}
	return out, nil
}

// PayQuery 查单（对齐 epay Pay::query）。入参 pid + (trade_no | out_trade_no)。
func (s *MapiService) PayQuery(m *model.Merchant, params map[string]string) (map[string]string, error) {
	o, err := s.findMerchantOrder(m.UID, params)
	if err != nil {
		return nil, err
	}
	out := map[string]string{
		"code":         "0",
		"trade_no":      o.TradeNo,
		"out_trade_no":  o.OutTradeNo,
		"api_trade_no":  o.APITradeNo,
		"bill_trade_no": o.BillTradeNo, // B1-02：对账/账单交易号（对齐 epay query 返回，array_filter 剔空）
		"type":          o.TypeName,
		"pid":           uintToStr(o.UID),
		"addtime":       o.AddTime.Format(timeLayout),
		"name":          o.Name,
		"money":         money.String(o.Money),
		"param":         o.Param,
		"buyer":         o.Buyer,
		"clientip":      o.IP,
		"status":        fmt.Sprintf("%d", o.Status),
		"refundmoney":   money.String(o.RefundMoney),
	}
	if o.EndTime != nil {
		out["endtime"] = o.EndTime.Format(timeLayout)
	}
	return pruneEmpty(out), nil
}

// PayRefund 退款（对齐 epay Pay::refund + Order::refund）。
// 入参 pid + money + (trade_no|out_trade_no) + 可选 out_refund_no(长度>5 启用幂等)。
// 复用退款金额校验与 reducemoney 语义；写 pay_refundorder；余额层退回，渠道原路退款待凭证。
func (s *MapiService) PayRefund(m *model.Merchant, params map[string]string) (map[string]string, error) {
	if !s.cfg.Bool("user_refund") {
		return nil, mapiErr("管理员未开启商户自助退款")
	}
	// per-merchant 退款开关（A-7，对齐 epay api.php:125 userrow['refund']==0）。
	if m.Refund == 0 {
		return nil, mapiErr("商户未开启订单退款API接口")
	}
	// B1-60(b)：金额校验对齐 epay is_numeric+/^[0-9.]+$/：非数字/空/负数一律拒'金额输入错误'，
	// 但允许 money=0 通过（epay 0 元退款为 no-op：不扣款仅改单，语义保留）。
	amountStr := strings.TrimSpace(params["money"])
	amount, err := money.Parse(amountStr)
	if err != nil || amountStr == "" || amount.IsNegative() {
		return nil, mapiErr("金额输入错误")
	}
	o, err := s.findMerchantOrder(m.UID, params)
	if err != nil {
		return nil, err
	}
	outRefundNo := strings.TrimSpace(params["out_refund_no"])

	// B1-03 幂等：out_refund_no 长度>5 时查已有退款单，按状态分两支（对齐 epay Order.php:474-483）：
	//   status==1(已成功) → 直接返回原结果（真正幂等，不重复退）；
	//   status==0(处理中/挂起) → 复用其 refund_no 继续往下推进一次（渠道异步退款重试）。
	// 当前余额退款即时 status=1，pending 分支要等接通道异步退款(乙类凭证)才走到。
	var reuseRefundNo string
	if len(outRefundNo) > 5 {
		if exist, e := s.refunds.FindByOutRefundNo(m.UID, outRefundNo); e == nil && exist != nil {
			if exist.Status == 1 {
				return s.refundResult(exist), nil
			}
			reuseRefundNo = exist.RefundNo // status==0 复用退款号继续处理
		}
	}

	// 校验状态与可退金额（对齐 epay Order::refund 前置校验）。
	if o.Status != 1 && o.Status != 2 && o.Status != 3 {
		return nil, mapiErr("当前订单状态不可退款")
	}
	// 必须有接口订单号才能退款（G-4，对齐 epay Order.php:76 if(!api_trade_no)）：
	// 防止对未经真实渠道成功的订单发起 API 退款。
	if strings.TrimSpace(o.APITradeNo) == "" {
		return nil, mapiErr("接口订单号不存在")
	}
	real := o.Money
	if o.RealMoney != nil && o.RealMoney.GreaterThan(decimal.Zero) {
		real = *o.RealMoney
	}
	if amount.GreaterThan(real) {
		return nil, mapiErr("退款金额不能超过订单实付金额")
	}
	if o.RefundMoney.GreaterThan(decimal.Zero) {
		if o.RefundMoney.GreaterThanOrEqual(real) {
			return nil, mapiErr("该订单已全额退款")
		}
		if amount.GreaterThan(real.Sub(o.RefundMoney).Round(2)) {
			return nil, mapiErr("退款金额超过剩余可退")
		}
	}

	// reducemoney：平台承担手续费(refund_fee_type=0)退回商户入账额；商户承担(=1)全额退。
	// 部分退款按退款额扣。冻结单/直清不扣。简化对齐 epay 主干四分支。
	reduce := s.calcRefundReduce(o, amount, real)
	if reduce.GreaterThan(m.Money) {
		return nil, mapiErr("商户余额不足")
	}

	// 幂等改单：status→2 累加 refundmoney（条件 UPDATE 防并发重复）。
	ok, err := s.orders.MarkRefundedPartial(o.TradeNo, amount)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, mapiErr("订单状态已变更，退款未执行")
	}
	// 扣商户余额（走流水，对齐 changeUserMoney）。reduce=0 时 ChangeUserMoney 自动跳过。
	if err := s.accounts.ChangeUserMoney(m.UID, reduce, false, "订单退款", o.TradeNo); err != nil {
		return nil, err
	}

	now := time.Now()
	// B1-03：status==0 挂起退款单复用其 refund_no 继续推进；否则新生成。
	refundNo := reuseRefundNo
	if refundNo == "" {
		refundNo = genTradeNo(now)
	}
	// B1-27：out_refund_no 空时回落为系统 refund_no（对齐 epay Order.php:82 $out_refund_no ?: $refund_no），
	// 保证落库与回包的 out_refund_no 恒非空，商户按 out_refund_no 对账可查。
	if outRefundNo == "" {
		outRefundNo = refundNo
	}
	ro := &model.RefundOrder{
		RefundNo:    refundNo,
		OutRefundNo: outRefundNo,
		TradeNo:     o.TradeNo,
		OutTradeNo:  o.OutTradeNo,
		UID:         m.UID,
		Money:       amount,
		ReduceMoney: reduce,
		Status:      1, // 余额层退款即时完成；渠道原路退款待凭证
		AddTime:     now,
		EndTime:     &now,
	}
	if err := s.refunds.Create(ro); err != nil {
		return nil, err
	}
	out := s.refundResult(ro)
	out["msg"] = "退款成功！退款金额¥" + money.String(amount)
	return out, nil
}

// PayRefundQuery 退款查询（对齐 epay Pay::refundquery）。入参 pid + (refund_no|out_refund_no)。
func (s *MapiService) PayRefundQuery(m *model.Merchant, params map[string]string) (map[string]string, error) {
	if !s.cfg.Bool("user_refund") {
		return nil, mapiErr("管理员未开启商户自助退款")
	}
	refundNo := strings.TrimSpace(params["refund_no"])
	outRefundNo := strings.TrimSpace(params["out_refund_no"])
	var ro *model.RefundOrder
	var err error
	if refundNo != "" {
		ro, err = s.refunds.FindByRefundNo(m.UID, refundNo)
	} else if outRefundNo != "" {
		ro, err = s.refunds.FindByOutRefundNo(m.UID, outRefundNo)
	} else {
		return nil, mapiErrCode(-4, "退款单号不能为空")
	}
	if err != nil {
		return nil, err
	}
	if ro == nil {
		return nil, mapiErr("退款记录不存在")
	}
	out := s.refundResult(ro)
	out["addtime"] = ro.AddTime.Format(timeLayout)
	if ro.EndTime != nil {
		out["endtime"] = ro.EndTime.Format(timeLayout)
	}
	return out, nil
}

// findMerchantOrder 按 pid + (trade_no|out_trade_no) 查本商户订单。
func (s *MapiService) findMerchantOrder(uid uint, params map[string]string) (*model.Order, error) {
	tradeNo := strings.TrimSpace(params["trade_no"])
	outTradeNo := strings.TrimSpace(params["out_trade_no"])
	var o *model.Order
	var err error
	if tradeNo != "" {
		o, err = s.orders.FindByTradeNoAndUID(tradeNo, uid)
	} else if outTradeNo != "" {
		o, err = s.orders.FindByOut(uid, outTradeNo)
	} else {
		return nil, mapiErrCode(-4, "订单号不能为空")
	}
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, mapiErr("订单号不存在")
	}
	return o, nil
}

// refundResult 退款单 → 返回契约（对齐 epay Order::refund 返回结构）。
func (s *MapiService) refundResult(ro *model.RefundOrder) map[string]string {
	return map[string]string{
		"code":          "0",
		"refund_no":     ro.RefundNo,
		"out_refund_no": ro.OutRefundNo,
		"trade_no":      ro.TradeNo,
		"out_trade_no":  ro.OutTradeNo,
		"uid":           uintToStr(ro.UID),
		"money":         money.String(ro.Money),
		"reducemoney":   money.String(ro.ReduceMoney),
		"status":        fmt.Sprintf("%d", ro.Status),
	}
}

// calcRefundReduce 计算退款扣商户余额金额（简化对齐 epay Order::refund reducemoney 四分支）。
// - 冻结单(status=3)或商户直清：不扣(0)，冻结资金/直清不影响余额。
// - refund_fee_type=1(商户承担手续费)：全额退，扣 realmoney 等比部分(此处扣退款额)。
// - refund_fee_type=0(平台承担)：退回商户实际入账额，按退款额占实付比例折算 getmoney。
// calcRefundReduce 计算退款需从商户余额扣回的金额 reducemoney，1:1 对齐 epay Order::refund 的四分支：
//
//	① status==3(已冻结) 或 通道 mode==1(商户直清) → 0（款项未入商户余额，无需扣回）
//	② refund_fee_type==1(商户承担手续费) 且 全额退(money==realmoney) → realmoney
//	③ refund_fee_type==0(平台承担) 且 (全额退 或 退款额≥商户实得getmoney) → getmoney
//	④ 其余(部分退) → 按退款额 money 原额扣回
//
// 注意：不再做"按比例折算"，与 epay 一致——部分退按退款额扣、达 getmoney 阈值退 getmoney。
func (s *MapiService) calcRefundReduce(o *model.Order, amount, real decimal.Decimal) decimal.Decimal {
	if o.Status == 3 || s.channelDirect(o.Channel) {
		return decimal.Zero
	}
	if s.cfg.Int("refund_fee_type", 0) == 1 {
		// 商户承担手续费：仅全额退时扣 realmoney，否则按退款额扣（对齐 epay ② + 兜底 ④）。
		if amount.Equal(real) {
			return real
		}
		return amount
	}
	// 平台承担：全额退 或 退款额≥getmoney → 扣 getmoney；否则按退款额扣（对齐 epay ③④）。
	if amount.Equal(real) || amount.GreaterThanOrEqual(o.GetMoney) {
		return o.GetMoney
	}
	return amount
}

// channelDirect 判断通道是否商户直清(mode=1)。查不到通道视为非直清。
func (s *MapiService) channelDirect(channelID int) bool {
	if channelID <= 0 {
		return false
	}
	c, err := s.channels.FindByID(uint(channelID))
	if err != nil || c == nil {
		return false
	}
	return c.Mode == 1
}

// pruneEmpty 剔除空值键（对齐 epay array_filter，避免回包含空字段影响签名/展示）。
func pruneEmpty(m map[string]string) map[string]string {
	for k, v := range m {
		if k == "code" {
			continue
		}
		if strings.TrimSpace(v) == "" {
			delete(m, k)
		}
	}
	return m
}

// toMapiErr 把 PayError 等内部错误归一为 MapiError（保留提示，码归 -1）。
func toMapiErr(err error) error {
	if me, ok := err.(*MapiError); ok {
		return me
	}
	if pe, ok := err.(*PayError); ok {
		return &MapiError{Code: -1, Msg: pe.Msg}
	}
	return &MapiError{Code: -1, Msg: err.Error()}
}
