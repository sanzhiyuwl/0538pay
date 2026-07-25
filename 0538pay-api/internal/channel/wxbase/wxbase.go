// Package wxbase 提供微信支付 APIv3 各支付方式（Native/JSAPI/H5/APP/小程序/退款）共用的
// 请求签名、网关调用、应答验签、回调验签+解密、退款逻辑。
//
// 所有 V3 方式同源：同一 api.mch.weixin.qq.com、同一 Authorization 签名(pkg/wxpayv3)、
// 同一回调验签+AES-GCM 解密。各方式仅下单 path/请求体/应答不同。
package wxbase

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/wxpayv3"
	"github.com/shopspring/decimal"
)

const (
	APIHost     = "https://api.mch.weixin.qq.com"
	httpTimeout = 15 * time.Second
)

// 回调保留键（复用 channel 通用常量）。
const (
	KeyBody      = channel.RawBody
	KeySignature = channel.RawSignature
	KeyTimestamp = channel.RawTimestamp
	KeyNonce     = channel.RawNonce
)

// YuanToFen 元→分（四舍五入）。
func YuanToFen(yuan decimal.Decimal) int64 {
	return yuan.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

// 服务商（合作伙伴）模式相关（对齐 epay wxpaynp/wxpaysl）：
// config Extra 含 sub_mchid 即服务商模式——下单走 /v3/pay/partner/transactions/*，
// 请求体用 sp_appid/sp_mchid/sub_appid/sub_mchid 替代直连的 appid/mchid，
// payer 用 sp_openid（配了 sub_appid 则用 sub_openid）。签名/验签/解密与直连完全相同。
const (
	extraSubMchID = "sub_mchid" // 子商户号（服务商模式标志）
	extraSubAppID = "sub_appid" // 子商户应用 appid（可空）
)

// SubMchID 返回配置的子商户号（多个逗号分隔时随机取一个，对齐 epay PartnerPaymentService 构造）。
func SubMchID(cfg channel.Config) string {
	v := cfg.ExtraOr(extraSubMchID, "")
	if v == "" {
		return ""
	}
	if parts := strings.Split(v, ","); len(parts) > 1 {
		// 随机取一个子商户号（对齐 epay array_rand）；用订单无关的伪随机避免引入 time 依赖。
		return strings.TrimSpace(parts[randIndex(len(parts))])
	}
	return strings.TrimSpace(v)
}

// SubAppID 返回配置的子商户 appid（可空）。
func SubAppID(cfg channel.Config) string { return cfg.ExtraOr(extraSubAppID, "") }

// IsPartner 判断是否服务商模式（配了子商户号）。
func IsPartner(cfg channel.Config) bool { return SubMchID(cfg) != "" }

// randIndex 返回 [0,n) 的伪随机下标（多子商户号随机分配用）。
func randIndex(n int) int {
	if n <= 1 {
		return 0
	}
	b := make([]byte, 1)
	if _, err := crand.Read(b); err != nil {
		return 0
	}
	return int(b[0]) % n
}

// AmountInfo 金额（单位分）。
type AmountInfo struct {
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
}

// DoRequest 通用 APIv3 请求：签名 → 发请求 → 验应答签名 → 返回 body。
// method POST/GET；path 含 query；body 为请求体 JSON（GET 传空串）。
func DoRequest(ctx context.Context, cfg channel.Config, method, path, body string) ([]byte, int, error) {
	priv, err := wxpayv3.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, 0, fmt.Errorf("解析商户私钥失败: %w", err)
	}
	nonce, err := wxpayv3.NonceStr(32)
	if err != nil {
		return nil, 0, err
	}
	auth, err := wxpayv3.BuildAuthorization(wxpayv3.AuthParams{
		MchID:        cfg.MchID,
		SerialNo:     cfg.SerialNo,
		PrivateKey:   priv,
		Method:       method,
		CanonicalURL: path,
		Body:         body,
		Timestamp:    wxpayv3.NowUnix(),
		Nonce:        nonce,
	})
	if err != nil {
		return nil, 0, err
	}
	var reader io.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequestWithContext(ctx, method, APIHost+path, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", auth)
	resp, err := (&http.Client{Timeout: httpTimeout}).Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求微信网关失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	// 验应答签名（配置了平台公钥才验）。
	if cfg.PublicKey != "" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		pub, e := wxpayv3.ParsePublicKey(cfg.PublicKey)
		if e != nil {
			return nil, resp.StatusCode, fmt.Errorf("解析平台公钥失败: %w", e)
		}
		if e := wxpayv3.VerifySignature(pub, resp.Header.Get("Wechatpay-Timestamp"),
			resp.Header.Get("Wechatpay-Nonce"), string(respBody), resp.Header.Get("Wechatpay-Signature")); e != nil {
			return nil, resp.StatusCode, e
		}
	}
	return respBody, resp.StatusCode, nil
}

// Prepay 统一预下单：调 /v3/pay/transactions/{method}，返回应答体。
// bodyMap 为业务请求体（由各方式组装，含 appid/mchid/description/out_trade_no/notify_url/amount + 方式特有字段）。
func Prepay(ctx context.Context, cfg channel.Config, payMethod string, bodyMap map[string]interface{}) ([]byte, error) {
	if cfg.AppID == "" || cfg.MchID == "" {
		return nil, fmt.Errorf("微信通道缺少 appid/mch_id 配置")
	}
	b, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}
	// 服务商模式走合作伙伴下单路径（对齐 epay PartnerPaymentService：/v3/pay/partner/transactions/*）。
	path := "/v3/pay/transactions/" + payMethod
	if IsPartner(cfg) {
		path = "/v3/pay/partner/transactions/" + payMethod
	}
	respBody, status, err := DoRequest(ctx, cfg, "POST", path, string(b))
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("微信下单返回 %d: %s", status, string(respBody))
	}
	return respBody, nil
}

// BasePrepayBody 组装各方式公共的下单请求体字段。notify 空则报错。
func BasePrepayBody(cfg channel.Config, req channel.CreateReq) (map[string]interface{}, error) {
	notify := req.NotifyURL
	if notify == "" {
		notify = cfg.NotifyURL
	}
	if notify == "" {
		return nil, fmt.Errorf("微信通道缺少 notify_url 回调地址")
	}
	desc := req.Subject
	if desc == "" {
		desc = "商品支付"
	}
	body := map[string]interface{}{
		"description":  desc,
		"out_trade_no": req.TradeNo,
		"notify_url":   notify,
		"amount":       AmountInfo{Total: YuanToFen(req.Money), Currency: "CNY"},
	}
	// 商户身份字段：服务商模式用 sp_appid/sp_mchid + sub_appid/sub_mchid（对齐 epay PartnerPaymentService
	// publicParams）；直连用 appid/mchid。
	if IsPartner(cfg) {
		body["sp_appid"] = cfg.AppID
		body["sp_mchid"] = cfg.MchID
		body["sub_mchid"] = SubMchID(cfg)
		if sub := SubAppID(cfg); sub != "" {
			body["sub_appid"] = sub
		}
	} else {
		body["appid"] = cfg.AppID
		body["mchid"] = cfg.MchID
	}
	// scene_info.payer_client_ip：对齐 epay wxpayn 下单带用户终端 IP（风控必备，缺失易被拦截）。
	if ip := req.ClientIP; ip != "" {
		body["scene_info"] = map[string]interface{}{"payer_client_ip": ip}
	}
	// settle_info.profit_sharing：本单需分账时置分账标记，把资金冻结在渠道待后续分账
	// （对齐 epay wxpayn：$order['profits']>0 → settle_info.profit_sharing=true）。
	if req.ProfitSharing {
		body["settle_info"] = map[string]interface{}{"profit_sharing": true}
	}
	return body, nil
}

// QueryPaid 主动查单，判 trade_state=SUCCESS。各方式通用。
func QueryPaid(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	if cfg.MchID == "" {
		return false, fmt.Errorf("微信通道缺少 mch_id 配置")
	}
	// 服务商模式查单走合作伙伴路径，query 带 sp_mchid + sub_mchid（对齐 epay PartnerPaymentService::orderQuery）。
	path := "/v3/pay/transactions/out-trade-no/" + tradeNo + "?mchid=" + cfg.MchID
	if IsPartner(cfg) {
		path = "/v3/pay/partner/transactions/out-trade-no/" + tradeNo + "?sp_mchid=" + cfg.MchID + "&sub_mchid=" + SubMchID(cfg)
	}
	body, status, err := DoRequest(ctx, cfg, "GET", path, "")
	if err != nil {
		return false, err
	}
	if status < 200 || status >= 300 {
		return false, fmt.Errorf("微信查单返回 %d: %s", status, string(body))
	}
	var r struct {
		TradeState string `json:"trade_state"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return false, err
	}
	return r.TradeState == "SUCCESS", nil
}

type notifyEnvelope struct {
	EventType string `json:"event_type"`
	Resource  struct {
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

type notifyResource struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	Amount        struct {
		Total int64 `json:"total"`
	} `json:"amount"`
}

// ParseNotify 通用回调解析：验签 + AES-GCM 解密 + 判 trade_state。各 V3 方式通用。
func ParseNotify(cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	body := raw[KeyBody]
	if body == "" {
		return channel.NotifyResult{}, fmt.Errorf("回调报文为空")
	}
	// 回调验签强制化（对齐微信支付官方「合理应对签名探测流量」规范 doc 4013053249）：
	// 未配置平台公钥 = 无法验签，回调不可信，必须拒绝而非放行（旧实现公钥空时跳过验签存在伪造风险）。
	// 微信会下发 WECHATPAY/SIGNTEST/ 前缀的错误签名探测流量，我方一律走验签、失败即拒绝，天然合规。
	if cfg.PublicKey == "" {
		return channel.NotifyResult{}, fmt.Errorf("微信通道未配置平台公钥，无法验证回调签名，拒绝处理")
	}
	pub, err := wxpayv3.ParsePublicKey(cfg.PublicKey)
	if err != nil {
		return channel.NotifyResult{}, fmt.Errorf("解析平台公钥失败: %w", err)
	}
	if err := wxpayv3.VerifySignature(pub, raw[KeyTimestamp], raw[KeyNonce], body, raw[KeySignature]); err != nil {
		return channel.NotifyResult{}, err
	}
	var env notifyEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return channel.NotifyResult{}, fmt.Errorf("回调报文解析失败: %w", err)
	}
	if env.Resource.Ciphertext == "" {
		return channel.NotifyResult{}, fmt.Errorf("回调缺少密文")
	}
	plain, err := wxpayv3.DecryptAESGCM(cfg.Key, env.Resource.Nonce, env.Resource.AssociatedData, env.Resource.Ciphertext)
	if err != nil {
		return channel.NotifyResult{}, err
	}
	var res notifyResource
	if err := json.Unmarshal(plain, &res); err != nil {
		return channel.NotifyResult{}, fmt.Errorf("回调业务对象解析失败: %w", err)
	}
	success := env.EventType == "TRANSACTION.SUCCESS" && res.TradeState == "SUCCESS"
	money := decimal.NewFromInt(res.Amount.Total).Div(decimal.NewFromInt(100))
	return channel.NotifyResult{
		TradeNo:    res.OutTradeNo,
		ChannelNo:  res.TransactionID,
		Money:      money,
		Success:    success,
		AckContent: "",
	}, nil
}

// Refund 通用退款：POST /v3/refund/domestic/refunds。各 V3 方式通用。
func Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	if cfg.MchID == "" {
		return channel.RefundResp{}, fmt.Errorf("微信通道缺少 mch_id 配置")
	}
	notify := cfg.NotifyURL
	bodyMap := map[string]interface{}{
		"out_trade_no":  req.TradeNo,
		"out_refund_no": req.OutRefundNo,
		"amount": map[string]interface{}{
			"refund":   YuanToFen(req.Money),
			"total":    YuanToFen(req.TotalMoney),
			"currency": "CNY",
		},
	}
	// 服务商模式退款请求体带 sub_mchid（对齐 epay PartnerPaymentService::refund）。
	if IsPartner(cfg) {
		bodyMap["sub_mchid"] = SubMchID(cfg)
	}
	if req.ChannelNo != "" {
		bodyMap["transaction_id"] = req.ChannelNo
	}
	if req.Reason != "" {
		bodyMap["reason"] = req.Reason
	}
	if notify != "" {
		bodyMap["notify_url"] = notify
	}
	b, _ := json.Marshal(bodyMap)
	respBody, status, err := DoRequest(ctx, cfg, "POST", "/v3/refund/domestic/refunds", string(b))
	if err != nil {
		return channel.RefundResp{}, err
	}
	if status < 200 || status >= 300 {
		return channel.RefundResp{}, fmt.Errorf("微信退款返回 %d: %s", status, string(respBody))
	}
	var r struct {
		RefundID string `json:"refund_id"`
		Status   string `json:"status"`
		Amount   struct {
			Refund int64 `json:"refund"`
		} `json:"amount"`
	}
	_ = json.Unmarshal(respBody, &r)
	money := decimal.NewFromInt(r.Amount.Refund).Div(decimal.NewFromInt(100))
	// status: SUCCESS/PROCESSING/ABNORMAL/CLOSED. 受理成功(SUCCESS/PROCESSING)视为成功。
	return channel.RefundResp{
		RefundNo: r.RefundID,
		Money:    money,
		Success:  r.Status == "SUCCESS" || r.Status == "PROCESSING",
	}, nil
}

// CombinePrepay 合单支付下单（对齐 epay V3 PaymentService::combineNativePay/JsapiPay/H5Pay/AppPay，
// POST /v3/combine-transactions/{native,jsapi,h5,app}）。combineBody 需含 combine_out_trade_no、sub_orders。
// 公共参数 combine_appid/combine_mchid 与各子单 mchid 由本函数注入。返回原始应答体（native→code_url、jsapi/app→prepay_id、h5→h5_url）。
func CombinePrepay(ctx context.Context, cfg channel.Config, payMethod string, combineBody map[string]interface{}) ([]byte, error) {
	if cfg.AppID == "" || cfg.MchID == "" {
		return nil, fmt.Errorf("微信合单缺少 appid/mch_id 配置")
	}
	combineBody["combine_appid"] = cfg.AppID
	combineBody["combine_mchid"] = cfg.MchID
	// 各子单补 mchid（对齐 epay combineXxxPay：foreach sub_orders $order['mchid']=mchId）。
	if subs, ok := combineBody["sub_orders"].([]map[string]interface{}); ok {
		for _, s := range subs {
			if _, has := s["mchid"]; !has {
				s["mchid"] = cfg.MchID
			}
		}
	}
	b, _ := json.Marshal(combineBody)
	respBody, status, err := DoRequest(ctx, cfg, "POST", "/v3/combine-transactions/"+payMethod, string(b))
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("微信合单下单返回 %d: %s", status, string(respBody))
	}
	return respBody, nil
}

// CombineQuery 合单查单（对齐 combineQueryOrder，GET /v3/combine-transactions/out-trade-no/{no}）。
// 任一子单 trade_state=SUCCESS 视为该子单已付；此处返回整单是否全部成功。
func CombineQuery(ctx context.Context, cfg channel.Config, combineOutTradeNo string) (bool, error) {
	path := "/v3/combine-transactions/out-trade-no/" + combineOutTradeNo
	respBody, status, err := DoRequest(ctx, cfg, "GET", path, "")
	if err != nil {
		return false, err
	}
	if status < 200 || status >= 300 {
		return false, fmt.Errorf("微信合单查单返回 %d: %s", status, string(respBody))
	}
	var r struct {
		SubOrders []struct {
			TradeState string `json:"trade_state"`
		} `json:"sub_orders"`
	}
	if err := json.Unmarshal(respBody, &r); err != nil {
		return false, err
	}
	if len(r.SubOrders) == 0 {
		return false, nil
	}
	for _, s := range r.SubOrders {
		if s.TradeState != "SUCCESS" {
			return false, nil
		}
	}
	return true, nil
}

// MchTransfer 发起商家转账到零钱（对齐 epay V3 TransferService::mchTransfer，
// POST /v3/fund-app/mch-transfer/transfer-bills，需商户证书签名）。
// TransferReq.Account 为收款用户 openid；transfer_scene_id 从 Extra["transfer_scene_id"] 取（默认 1000 现金营销）。
func MchTransfer(ctx context.Context, cfg channel.Config, req channel.TransferReq) (channel.TransferResp, error) {
	if cfg.AppID == "" {
		return channel.TransferResp{}, fmt.Errorf("商家转账缺少 appid 配置")
	}
	if req.Account == "" {
		return channel.TransferResp{}, fmt.Errorf("商家转账缺少收款用户 openid")
	}
	remark := req.Remark
	if remark == "" {
		remark = "转账"
	}
	body := map[string]interface{}{
		"appid":          cfg.AppID,
		"out_bill_no":    req.OutBizNo,
		"transfer_scene_id": cfg.ExtraOr("transfer_scene_id", "1000"),
		"openid":         req.Account,
		"transfer_amount": YuanToFen(req.Money),
		"transfer_remark": remark,
	}
	if name := req.Name; name != "" {
		body["user_name"] = name // 金额≥2000 或场景要求时需传实名（此处调用方已加密则放 Extra）
	}
	b, _ := json.Marshal(body)
	respBody, status, err := DoRequest(ctx, cfg, "POST", "/v3/fund-app/mch-transfer/transfer-bills", string(b))
	if err != nil {
		return channel.TransferResp{}, err
	}
	if status < 200 || status >= 300 {
		return channel.TransferResp{}, fmt.Errorf("商家转账返回 %d: %s", status, string(respBody))
	}
	var r struct {
		OutBillNo      string `json:"out_bill_no"`
		TransferBillNo string `json:"transfer_bill_no"`
		State          string `json:"state"`
		FailReason     string `json:"fail_reason"`
		PackageInfo    string `json:"package_info"`
	}
	_ = json.Unmarshal(respBody, &r)
	return channel.TransferResp{
		TransferNo: r.TransferBillNo,
		Status:     transferState(r.State),
		Message:    r.FailReason,
		ProofURL:   r.PackageInfo, // 用户需在微信内点 package_info 领取
	}, nil
}

// MchTransferQuery 查询转账单（对齐 TransferService::queryTransferByOutNo）。
func MchTransferQuery(ctx context.Context, cfg channel.Config, outBizNo string) (channel.TransferResp, error) {
	path := "/v3/fund-app/mch-transfer/transfer-bills/out-bill-no/" + outBizNo
	respBody, status, err := DoRequest(ctx, cfg, "GET", path, "")
	if err != nil {
		return channel.TransferResp{}, err
	}
	if status < 200 || status >= 300 {
		return channel.TransferResp{}, fmt.Errorf("商家转账查询返回 %d: %s", status, string(respBody))
	}
	var r struct {
		TransferBillNo string `json:"transfer_bill_no"`
		State          string `json:"state"`
		FailReason     string `json:"fail_reason"`
	}
	_ = json.Unmarshal(respBody, &r)
	return channel.TransferResp{
		TransferNo: r.TransferBillNo,
		Status:     transferState(r.State),
		Message:    r.FailReason,
	}, nil
}

// transferState 微信转账单状态 → 统一 status（0=处理中 1=成功 2=失败，对齐 epay transfer status）。
func transferState(state string) int8 {
	switch state {
	case "SUCCESS":
		return 1
	case "FAIL", "CANCELLED", "CANCELLING":
		return 2
	default: // ACCEPTED/PROCESSING/WAIT_USER_CONFIRM/TRANSFERING
		return 0
	}
}

// BuildAppParams 生成 APP 支付前端拉起参数（对齐 epay V3 PaymentService::getAppParameters）。
// 签名串 = appid\ntimestamp\nnoncestr\nprepayid\n，用商户私钥做 SHA256withRSA。
// 服务商模式 partnerid 用子商户号（资金进特约商户），appid 仍用 sub_appid（配了则）或 sp_appid。
func BuildAppParams(cfg channel.Config, prepayID string) (map[string]string, error) {
	priv, err := wxpayv3.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("解析商户私钥失败: %w", err)
	}
	nonce, err := wxpayv3.NonceStr(32)
	if err != nil {
		return nil, err
	}
	ts := strconv.FormatInt(wxpayv3.NowUnix(), 10)
	appID := cfg.AppID
	partnerID := cfg.MchID
	if IsPartner(cfg) {
		if sub := SubAppID(cfg); sub != "" {
			appID = sub
		}
		partnerID = SubMchID(cfg)
	}
	message := appID + "\n" + ts + "\n" + nonce + "\n" + prepayID + "\n"
	sign, err := wxpayv3.SignMessage(priv, message)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"appid":     appID,
		"partnerid": partnerID,
		"prepayid":  prepayID,
		"package":   "Sign=WXPay",
		"noncestr":  nonce,
		"timestamp": ts,
		"sign":      sign,
	}, nil
}

// Inputs 返回微信支付 APIv3 各支付方式（Native/JSAPI/H5）共用的密钥表单字段定义。
// 各微信渠道 Configurable.Inputs() 复用它——同源于同一 APIv3 商户凭证。
// 键名与 service.buildChannelConfig 的通用键对齐（appid/mch_id/serial_no/api_v3_key→Key/
// private_key/public_key/notify_url），public_key_id 落 Extra。对齐 epay 微信插件 $info['inputs']。
func Inputs() []channel.FieldInput {
	return []channel.FieldInput{
		{Name: "appid", Label: "AppID", Type: "text", Require: true, Tip: "公众号/应用 appid"},
		{Name: "mch_id", Label: "商户号", Type: "text", Require: true, Tip: "微信支付商户号 mchid"},
		{Name: "serial_no", Label: "证书序列号", Type: "text", Require: true, Tip: "商户 API 证书序列号"},
		{Name: "api_v3_key", Label: "APIv3 密钥", Type: "password", Require: true, Tip: "商户平台设置的 32 字节 APIv3 密钥，用于回调解密"},
		{Name: "private_key", Label: "商户私钥", Type: "textarea", Require: true, Tip: "apiclient_key.pem 内容，用于请求签名"},
		{Name: "public_key", Label: "微信支付公钥", Type: "textarea", Require: false, Tip: "平台公钥/证书公钥，用于回调与应答验签"},
		{Name: "public_key_id", Label: "公钥 ID", Type: "text", Require: false, Tip: "PUB_KEY_ID_xxxx（可选）"},
		{Name: "sub_mchid", Label: "子商户号", Type: "text", Require: false, Tip: "服务商模式填写：特约商户号 sub_mchid（多个用英文逗号分隔随机分配）；直连留空"},
		{Name: "sub_appid", Label: "子商户 AppID", Type: "text", Require: false, Tip: "服务商模式可选：子商户在开放平台/公众号的 appid，配了则 JSAPI 用 sub_openid"},
		{Name: "notify_url", Label: "回调基址", Type: "text", Require: true, Tip: "系统会自动拼接 /系统订单号 作为微信回调地址"},
	}
}
