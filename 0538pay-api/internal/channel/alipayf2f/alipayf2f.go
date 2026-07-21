// Package alipayf2f 实现支付宝「当面付-扫码」渠道（alipay.trade.precreate → 二维码）。
//
// 对齐 epay alipay 插件 qrcode() + cccyun/alipay-sdk：
//   - 下单：POST openapi.alipay.com/gateway.do，系统参数 + biz_content(JSON)，RSA2 签名，
//     应答 alipay_trade_precreate_response.qr_code 即二维码内容。
//   - 回调：表单 POST，RSA2 验签(支付宝公钥) → trade_status=TRADE_SUCCESS + 金额校验。
//
// 通道 config JSON 需含（对齐前端 pluginConfigFields.alipayf2f）：
//   appid(应用APPID) / private_key(应用私钥) / public_key(支付宝公钥) /
//   notify_url(回调基址) / seller_id(卖家ID，可选)。
//
// 通过 init() 自注册，key = "alipayf2f"。依赖仅 Go 标准库 + pkg/alipay。
package alipayf2f

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/pkg/alipay"
	"github.com/shopspring/decimal"
)

const (
	gatewayURL  = "https://openapi.alipay.com/gateway.do"
	precreate   = "alipay.trade.precreate"
	pluginKey   = "alipayf2f"
	httpTimeout = 15 * time.Second
	timeLayout  = "2006-01-02 15:04:05"
)

// Channel 支付宝当面付扫码渠道实现 channel.PaymentChannel。
type Channel struct{}

func (Channel) Key() string { return pluginKey }

// buildSysParams 组装支付宝系统参数（含 biz_content），未签名。纯函数便于单测。
// ts 由调用方传入（单测可固定）。
func buildSysParams(cfg channel.Config, req channel.CreateReq, bizContent string, ts string) map[string]string {
	p := map[string]string{
		"app_id":      cfg.AppID,
		"method":      precreate,
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   ts,
		"version":     "1.0",
		"biz_content": bizContent,
	}
	notifyURL := strings.TrimSpace(req.NotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(cfg.NotifyURL)
	}
	if notifyURL != "" {
		p["notify_url"] = notifyURL
	}
	return p
}

// buildBizContent 组装当面付下单业务参数 JSON。金额保留两位小数字符串（对齐支付宝 total_amount）。
func buildBizContent(cfg channel.Config, req channel.CreateReq) (string, error) {
	if cfg.AppID == "" {
		return "", fmt.Errorf("支付宝通道缺少 appid 配置")
	}
	subject := req.Subject
	if subject == "" {
		subject = "商品支付"
	}
	biz := map[string]string{
		"out_trade_no": req.TradeNo,
		"total_amount": req.Money.StringFixed(2),
		"subject":      subject,
	}
	if sellerID := cfg.ExtraOr("seller_id", ""); sellerID != "" {
		biz["seller_id"] = sellerID
	}
	b, err := json.Marshal(biz)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Create 当面付下单：签名 → POST → 验应答签名 → 取 qr_code。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	bizContent, err := buildBizContent(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	priv, err := alipay.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return channel.CreateResp{}, fmt.Errorf("解析应用私钥失败: %w", err)
	}

	params := buildSysParams(cfg, req, bizContent, time.Now().Format(timeLayout))
	sign, err := alipay.SignParams(priv, params)
	if err != nil {
		return channel.CreateResp{}, err
	}
	params["sign"] = sign

	// application/x-www-form-urlencoded 提交
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", gatewayURL, strings.NewReader(form.Encode()))
	if err != nil {
		return channel.CreateResp{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	client := &http.Client{Timeout: httpTimeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return channel.CreateResp{}, fmt.Errorf("请求支付宝下单失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return channel.CreateResp{}, err
	}

	qrCode, err := parsePrecreateResp(respBody)
	if err != nil {
		return channel.CreateResp{}, err
	}
	return channel.CreateResp{
		PayType: channel.PayTypeQRCode,
		QRCode:  qrCode,
		PayURL:  qrCode,
	}, nil
}

// precreateEnvelope 当面付下单应答外层结构。
type precreateEnvelope struct {
	Response struct {
		Code       string `json:"code"`
		Msg        string `json:"msg"`
		SubCode    string `json:"sub_code"`
		SubMsg     string `json:"sub_msg"`
		OutTradeNo string `json:"out_trade_no"`
		QRCode     string `json:"qr_code"`
	} `json:"alipay_trade_precreate_response"`
	Sign string `json:"sign"`
}

// parsePrecreateResp 解析下单应答，取 qr_code；code!=10000 视为失败。
// 注：应答验签需支付宝公钥对 alipay_trade_precreate_response 原文验签，
// 这里为结构先行只做业务码校验；真凭证联调阶段再补严格应答验签。
func parsePrecreateResp(body []byte) (string, error) {
	var env precreateEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return "", fmt.Errorf("支付宝应答解析失败: %w", err)
	}
	r := env.Response
	if r.Code != "10000" {
		return "", fmt.Errorf("支付宝下单失败[%s]: %s %s", r.Code, r.Msg, r.SubMsg)
	}
	if r.QRCode == "" {
		return "", fmt.Errorf("支付宝下单未返回 qr_code")
	}
	return r.QRCode, nil
}

// queryEnvelope 查单应答外层结构。
type queryEnvelope struct {
	Response struct {
		Code       string `json:"code"`
		Msg        string `json:"msg"`
		SubCode    string `json:"sub_code"`
		SubMsg     string `json:"sub_msg"`
		OutTradeNo string `json:"out_trade_no"`
		TradeNo    string `json:"trade_no"`
		TradeState string `json:"trade_status"`
	} `json:"alipay_trade_query_response"`
	Sign string `json:"sign"`
}

// parseQueryResp 解析查单应答，判定是否已支付。纯函数便于单测。
// code=10000 且 trade_status ∈ {TRADE_SUCCESS,TRADE_FINISHED} 视为已支付。
// ACQ.TRADE_NOT_EXIST（订单未创建/未支付）视为未支付而非错误。
func parseQueryResp(body []byte) (bool, error) {
	var env queryEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return false, fmt.Errorf("查单应答解析失败: %w", err)
	}
	r := env.Response
	if r.Code == "10000" {
		return r.TradeState == "TRADE_SUCCESS" || r.TradeState == "TRADE_FINISHED", nil
	}
	if r.SubCode == "ACQ.TRADE_NOT_EXIST" {
		return false, nil // 订单尚未支付/不存在，非错误
	}
	return false, fmt.Errorf("支付宝查单失败[%s]: %s %s", r.Code, r.Msg, r.SubMsg)
}

// Query 主动查单：alipay.trade.query。tradeNo 为系统订单号（下单时的 out_trade_no）。
func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	if cfg.AppID == "" {
		return false, fmt.Errorf("支付宝通道缺少 appid 配置")
	}
	priv, err := alipay.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return false, fmt.Errorf("解析应用私钥失败: %w", err)
	}
	bizContent, err := json.Marshal(map[string]string{"out_trade_no": tradeNo})
	if err != nil {
		return false, err
	}
	params := map[string]string{
		"app_id":      cfg.AppID,
		"method":      "alipay.trade.query",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format(timeLayout),
		"version":     "1.0",
		"biz_content": string(bizContent),
	}
	sign, err := alipay.SignParams(priv, params)
	if err != nil {
		return false, err
	}
	params["sign"] = sign

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", gatewayURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	client := &http.Client{Timeout: httpTimeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("请求支付宝查单失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return false, err
	}
	return parseQueryResp(respBody)
}

// Notify 表单型回调：RSA2 验签 + trade_status 判定。raw 为回调表单参数（含 sign）。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return parseNotify(cfg, raw)
}

// parseNotify 纯函数：验签(支付宝公钥) → 读 trade_status/out_trade_no/total_amount/trade_no。
func parseNotify(cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	// 剔除框架注入的原始报文保留键，只留支付宝真实表单字段参与验签。
	params := map[string]string{}
	for k, v := range raw {
		if strings.HasPrefix(k, "_raw_") {
			continue
		}
		params[k] = v
	}
	if params["out_trade_no"] == "" {
		return channel.NotifyResult{}, fmt.Errorf("回调缺少 out_trade_no")
	}

	// 验签（配置了公钥才验；未配置仅限联调，生产必须配）。
	if cfg.PublicKey != "" {
		pub, err := alipay.ParsePublicKey(cfg.PublicKey)
		if err != nil {
			return channel.NotifyResult{}, fmt.Errorf("解析支付宝公钥失败: %w", err)
		}
		if !alipay.Verify(pub, params) {
			return channel.NotifyResult{}, fmt.Errorf("支付宝回调验签失败")
		}
	}

	status := params["trade_status"]
	success := status == "TRADE_SUCCESS" || status == "TRADE_FINISHED"
	money, _ := decimal.NewFromString(params["total_amount"])
	return channel.NotifyResult{
		TradeNo:    params["out_trade_no"],
		ChannelNo:  params["trade_no"],
		Money:      money,
		Success:    success,
		AckContent: "success", // 支付宝异步回调要求返回纯文本 success
	}, nil
}

func init() {
	channel.Register(Channel{})
}
