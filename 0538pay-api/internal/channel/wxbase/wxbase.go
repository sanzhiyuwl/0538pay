// Package wxbase 提供微信支付 APIv3 各支付方式（Native/JSAPI/H5/APP/小程序/退款）共用的
// 请求签名、网关调用、应答验签、回调验签+解密、退款逻辑。
//
// 所有 V3 方式同源：同一 api.mch.weixin.qq.com、同一 Authorization 签名(pkg/wxpayv3)、
// 同一回调验签+AES-GCM 解密。各方式仅下单 path/请求体/应答不同。
package wxbase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/pkg/wxpayv3"
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
	path := "/v3/pay/transactions/" + payMethod
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
	return map[string]interface{}{
		"appid":        cfg.AppID,
		"mchid":        cfg.MchID,
		"description":  desc,
		"out_trade_no": req.TradeNo,
		"notify_url":   notify,
		"amount":       AmountInfo{Total: YuanToFen(req.Money), Currency: "CNY"},
	}, nil
}

// QueryPaid 主动查单，判 trade_state=SUCCESS。各方式通用。
func QueryPaid(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	if cfg.MchID == "" {
		return false, fmt.Errorf("微信通道缺少 mch_id 配置")
	}
	path := "/v3/pay/transactions/out-trade-no/" + tradeNo + "?mchid=" + cfg.MchID
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
	if cfg.PublicKey != "" {
		pub, err := wxpayv3.ParsePublicKey(cfg.PublicKey)
		if err != nil {
			return channel.NotifyResult{}, fmt.Errorf("解析平台公钥失败: %w", err)
		}
		if err := wxpayv3.VerifySignature(pub, raw[KeyTimestamp], raw[KeyNonce], body, raw[KeySignature]); err != nil {
			return channel.NotifyResult{}, err
		}
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
		{Name: "notify_url", Label: "回调基址", Type: "text", Require: true, Tip: "系统会自动拼接 /系统订单号 作为微信回调地址"},
	}
}
