// Package wxnative 实现微信支付 APIv3「Native 支付」渠道（PC 扫码）。
//
// 对齐官方规范：
//   - 下单：POST https://api.mch.weixin.qq.com/v3/pay/transactions/native，
//     RSA-SHA256 Authorization 头（见 pkg/wxpayv3），应答含 code_url，前端据此展示二维码。
//   - 回调：验签(Wechatpay-Signature 头 + 平台公钥) → AES-256-GCM 解密 resource.ciphertext
//     → 读 trade_state/out_trade_no/amount。
//
// 通道 config JSON 需含（key 见 ConfigKeys）：
//   appid / mch_id / serial_no / api_v3_key / private_key(PEM) /
//   public_key(平台公钥 PEM) / public_key_id / notify_url(回调基址)。
//
// 通过 init() 自注册到 channel.registry，key = "wxnative"。
// 依赖仅 Go 标准库 + pkg/wxpayv3，不引第三方微信 SDK。
package wxnative

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/channel/wxbase"
	"github.com/0538pay/api/pkg/wxpayv3"
	"github.com/shopspring/decimal"
)

const (
	apiHost     = "https://api.mch.weixin.qq.com"
	prepayPath  = "/v3/pay/transactions/native"
	pluginKey   = "wxnative"
	httpTimeout = 15 * time.Second
)

// 回调 handler 注入到 raw map 的保留键，复用 channel 包的通用常量。
const (
	KeyBody      = channel.RawBody
	KeySignature = channel.RawSignature
	KeyTimestamp = channel.RawTimestamp
	KeyNonce     = channel.RawNonce
	KeySerial    = channel.RawSerial
)

// Channel 微信 Native 渠道实现 channel.PaymentChannel。
type Channel struct{}

func (Channel) Key() string { return pluginKey }

// prepayReq 是 Native 下单请求体（仅取必要字段；完整字段见官方文档）。
type prepayReq struct {
	AppID       string     `json:"appid"`
	MchID       string     `json:"mchid"`
	Description string     `json:"description"`
	OutTradeNo  string     `json:"out_trade_no"`
	NotifyURL   string     `json:"notify_url"`
	Amount      amountInfo `json:"amount"`
}

type amountInfo struct {
	Total    int64  `json:"total"`    // 单位：分
	Currency string `json:"currency"` // CNY
}

type prepayResp struct {
	CodeURL string `json:"code_url"`
}

// yuanToFen 元(decimal) → 分(int64)，四舍五入。
func yuanToFen(yuan decimal.Decimal) int64 {
	return yuan.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

// buildPrepayBody 组装 Native 下单请求体 JSON（纯函数，便于单测）。
// req.TradeNo 作为 out_trade_no（我方系统单号即微信侧商户单号）。
func buildPrepayBody(cfg channel.Config, req channel.CreateReq) ([]byte, error) {
	if cfg.AppID == "" || cfg.MchID == "" {
		return nil, fmt.Errorf("微信通道缺少 appid/mch_id 配置")
	}
	notifyURL := strings.TrimSpace(req.NotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(cfg.NotifyURL)
	}
	if notifyURL == "" {
		return nil, fmt.Errorf("微信通道缺少 notify_url 回调地址")
	}
	desc := req.Subject
	if desc == "" {
		desc = "商品支付"
	}
	body := prepayReq{
		AppID:       cfg.AppID,
		MchID:       cfg.MchID,
		Description: desc,
		OutTradeNo:  req.TradeNo,
		NotifyURL:   notifyURL,
		Amount: amountInfo{
			Total:    yuanToFen(req.Money),
			Currency: "CNY",
		},
	}
	return json.Marshal(body)
}

// Create Native 下单：签名 → POST → 验应答签名 → 取 code_url。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	reqBody, err := buildPrepayBody(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}

	priv, err := wxpayv3.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return channel.CreateResp{}, fmt.Errorf("解析商户私钥失败: %w", err)
	}
	nonce, err := wxpayv3.NonceStr(32)
	if err != nil {
		return channel.CreateResp{}, err
	}
	auth, err := wxpayv3.BuildAuthorization(wxpayv3.AuthParams{
		MchID:        cfg.MchID,
		SerialNo:     cfg.SerialNo,
		PrivateKey:   priv,
		Method:       "POST",
		CanonicalURL: prepayPath,
		Body:         string(reqBody),
		Timestamp:    wxpayv3.NowUnix(),
		Nonce:        nonce,
	})
	if err != nil {
		return channel.CreateResp{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiHost+prepayPath, bytes.NewReader(reqBody))
	if err != nil {
		return channel.CreateResp{}, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", auth)

	client := &http.Client{Timeout: httpTimeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return channel.CreateResp{}, fmt.Errorf("请求微信下单失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return channel.CreateResp{}, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return channel.CreateResp{}, fmt.Errorf("微信下单返回 %d: %s", httpResp.StatusCode, string(respBody))
	}

	// 验应答签名（防伪造应答）：需平台公钥
	if cfg.PublicKey != "" {
		if err := verifyRespSignature(cfg, httpResp.Header, respBody); err != nil {
			return channel.CreateResp{}, err
		}
	}

	var pr prepayResp
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return channel.CreateResp{}, err
	}
	if pr.CodeURL == "" {
		return channel.CreateResp{}, fmt.Errorf("微信下单未返回 code_url")
	}
	return channel.CreateResp{
		PayType: channel.PayTypeQRCode,
		QRCode:  pr.CodeURL, // 前端把该链接渲染成二维码
		PayURL:  pr.CodeURL,
	}, nil
}

// verifyRespSignature 用平台公钥验证应答签名（应答头同回调头字段名）。
func verifyRespSignature(cfg channel.Config, header http.Header, body []byte) error {
	pub, err := wxpayv3.ParsePublicKey(cfg.PublicKey)
	if err != nil {
		return fmt.Errorf("解析平台公钥失败: %w", err)
	}
	return wxpayv3.VerifySignature(
		pub,
		header.Get("Wechatpay-Timestamp"),
		header.Get("Wechatpay-Nonce"),
		string(body),
		header.Get("Wechatpay-Signature"),
	)
}

// queryResp 查单应答（仅取判定所需字段）。
type queryResp struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
}

// parseQueryResp 解析查单应答体，判定是否已支付（trade_state=SUCCESS）。纯函数便于单测。
func parseQueryResp(body []byte) (bool, error) {
	var r queryResp
	if err := json.Unmarshal(body, &r); err != nil {
		return false, fmt.Errorf("查单应答解析失败: %w", err)
	}
	return r.TradeState == "SUCCESS", nil
}

// Query 主动查单：GET /v3/pay/transactions/out-trade-no/{out_trade_no}?mchid=...
// tradeNo 为系统订单号（下单时作为微信侧 out_trade_no）。RSA 签名，GET 请求体为空。
func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	if cfg.MchID == "" {
		return false, fmt.Errorf("微信通道缺少 mch_id 配置")
	}
	priv, err := wxpayv3.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return false, fmt.Errorf("解析商户私钥失败: %w", err)
	}
	path := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(tradeNo) + "?mchid=" + url.QueryEscape(cfg.MchID)

	nonce, err := wxpayv3.NonceStr(32)
	if err != nil {
		return false, err
	}
	auth, err := wxpayv3.BuildAuthorization(wxpayv3.AuthParams{
		MchID:        cfg.MchID,
		SerialNo:     cfg.SerialNo,
		PrivateKey:   priv,
		Method:       "GET",
		CanonicalURL: path,
		Body:         "", // GET 无请求体
		Timestamp:    wxpayv3.NowUnix(),
		Nonce:        nonce,
	})
	if err != nil {
		return false, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", apiHost+path, nil)
	if err != nil {
		return false, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", auth)

	client := &http.Client{Timeout: httpTimeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("请求微信查单失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return false, err
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return false, fmt.Errorf("微信查单返回 %d: %s", httpResp.StatusCode, string(respBody))
	}
	if cfg.PublicKey != "" {
		if err := verifyRespSignature(cfg, httpResp.Header, respBody); err != nil {
			return false, err
		}
	}
	return parseQueryResp(respBody)
}

// notifyEnvelope 回调外层报文。
type notifyEnvelope struct {
	ID           string `json:"id"`
	EventType    string `json:"event_type"`
	ResourceType string `json:"resource_type"`
	Resource     struct {
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
		OriginalType   string `json:"original_type"`
	} `json:"resource"`
}

// notifyResource 解密后的业务对象（仅取校验所需字段）。
type notifyResource struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	Payer         struct {
		OpenID string `json:"openid"`
	} `json:"payer"`
	Amount struct {
		Total    int64  `json:"total"`
		Currency string `json:"currency"`
	} `json:"amount"`
}

// Notify 验签 + 解密回调，产出统一的 NotifyResult。
// raw 由 handler 注入原始 body 与 Wechatpay-* 头（见本包 Key* 常量）。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return parseNotify(cfg, raw)
}

// Refund 原路退款（/v3/refund/domestic/refunds），复用 wxbase。
func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return wxbase.Refund(ctx, cfg, req)
}

// parseNotify 纯函数：从 raw 取报文/头，验签 + 解密，返回结果（便于单测）。
func parseNotify(cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	body := raw[KeyBody]
	if body == "" {
		return channel.NotifyResult{}, fmt.Errorf("回调报文为空")
	}

	// 1. 验签（平台公钥）。配置了公钥才验；未配置则跳过（仅限联调期，生产必须配）。
	if cfg.PublicKey != "" {
		pub, err := wxpayv3.ParsePublicKey(cfg.PublicKey)
		if err != nil {
			return channel.NotifyResult{}, fmt.Errorf("解析平台公钥失败: %w", err)
		}
		if err := wxpayv3.VerifySignature(pub, raw[KeyTimestamp], raw[KeyNonce], body, raw[KeySignature]); err != nil {
			return channel.NotifyResult{}, err
		}
	}

	// 2. 解析外层报文
	var env notifyEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return channel.NotifyResult{}, fmt.Errorf("回调报文解析失败: %w", err)
	}
	if env.Resource.Ciphertext == "" {
		return channel.NotifyResult{}, fmt.Errorf("回调缺少密文")
	}

	// 3. AES-256-GCM 解密业务对象
	plain, err := wxpayv3.DecryptAESGCM(cfg.Key, env.Resource.Nonce, env.Resource.AssociatedData, env.Resource.Ciphertext)
	if err != nil {
		return channel.NotifyResult{}, err
	}
	var res notifyResource
	if err := json.Unmarshal(plain, &res); err != nil {
		return channel.NotifyResult{}, fmt.Errorf("回调业务对象解析失败: %w", err)
	}

	// 4. 组装统一结果。金额分→元。成功条件：事件成功 且 trade_state=SUCCESS。
	success := env.EventType == "TRANSACTION.SUCCESS" && res.TradeState == "SUCCESS"
	money := decimal.NewFromInt(res.Amount.Total).Div(decimal.NewFromInt(100))
	return channel.NotifyResult{
		TradeNo:    res.OutTradeNo,
		ChannelNo:  res.TransactionID,
		Money:      money,
		Success:    success,
		AckContent: "", // V3 回调应答用 HTTP 状态码，空串由 handler 处理
	}, nil
}

// Inputs 声明配置字段（复用微信共用字段，元数据驱动后台密钥表单）。
func (Channel) Inputs() []channel.FieldInput { return wxbase.Inputs() }

// Products 声明本渠道支持的支付产品形态（对齐 epay $info['select']）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "native", Name: "Native 扫码支付"}}
}

func init() {
	channel.Register(Channel{})
}
