// Package epayn 提供「二次对接彩虹易支付 V2(RSA)」的免签渠道。
//
// 与 B3 的 epay(V1/MD5) 渠道业务定位相同（作为下游商户级联对接上游易支付站），
// 区别纯在签名协议：epayn 走 RSA 双向签名（对齐 epay plugins/epayn）。
//
// 密钥流向（对齐 epay EpayCore）：
//   下单：本站用「商户私钥(merchant_private_key)」RSA 签名 → 上游用本站在其处登记的商户公钥验。
//   回调/返回：上游用「平台私钥」签 → 本站用「上游平台公钥(platform_public_key)」验。
//
// config 字段（通道 config JSON）：
//   appid    = 本站在上游的商户号 pid
//   appurl   = 上游易支付接口地址（末尾不带 /）
//   platform_public_key   = 上游平台公钥（验上游返回/回调签名，单行 base64 或 PEM）
//   merchant_private_key  = 本站在上游的商户私钥（签下单请求）
//   uptype   = 上游收单支付方式（自测闭环可配 mock）
package epayn

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/sign"
	"github.com/shopspring/decimal"
)

type Channel struct{}

func (Channel) Key() string { return "epayn" }

// Inputs 声明本渠道的配置字段（元数据驱动后台密钥表单，对齐 epay $info['inputs']）。
func (Channel) Inputs() []channel.FieldInput {
	return []channel.FieldInput{
		{Name: "appid", Label: "上游商户号 PID", Type: "text", Require: true, Tip: "本站在上游易支付登记的商户 pid"},
		{Name: "appurl", Label: "上游接口地址", Type: "text", Require: true, Tip: "上游易支付接口根地址，末尾不带 /"},
		{Name: "platform_public_key", Label: "上游平台公钥", Type: "textarea", Require: true, Tip: "验上游返回/回调签名，单行 base64 或 PEM"},
		{Name: "merchant_private_key", Label: "本站商户私钥", Type: "textarea", Require: true, Tip: "签下单请求，单行 base64 或 PEM"},
		{Name: "uptype", Label: "上游支付方式", Type: "text", Require: false, Tip: "上游收单支付方式，自测闭环可配 mock"},
	}
}

// Products 声明本渠道支持的支付产品形态（对齐 epay $info['transtypes']）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{
		{Code: "alipay", Name: "支付宝"},
		{Code: "wxpay", Name: "微信支付"},
		{Code: "qqpay", Name: "QQ钱包"},
	}
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// conf 从 config 解出上游对接四要素。
func conf(cfg channel.Config) (pid, apiURL, platformPub, merchantPriv string, err error) {
	pid = cfg.AppID
	if pid == "" {
		pid = cfg.ExtraOr("pid", "")
	}
	apiURL = strings.TrimRight(cfg.ExtraOr("appurl", ""), "/")
	platformPub = cfg.ExtraOr("platform_public_key", cfg.PublicKey)
	merchantPriv = cfg.ExtraOr("merchant_private_key", cfg.PrivateKey)
	if pid == "" || apiURL == "" || platformPub == "" || merchantPriv == "" {
		return "", "", "", "", errors.New("epayn 渠道缺少 appid/appurl/platform_public_key/merchant_private_key 配置")
	}
	return pid, apiURL, platformPub, merchantPriv, nil
}

type submitResp struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TradeNo string `json:"trade_no"`
		PayType string `json:"pay_type"`
		PayURL  string `json:"pay_url"`
		QRCode  string `json:"qrcode"`
		Money   string `json:"money"`
	} `json:"data"`
}

// Create RSA 签名 POST {appurl}/api/pay/submit，透传收银台信息。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	pid, apiURL, _, merchantPriv, err := conf(cfg)
	if err != nil {
		return channel.CreateResp{}, err
	}
	upType := cfg.ExtraOr("uptype", epaynType(req.Extra["typename"]))

	params := map[string]string{
		"pid":          pid,
		"type":         upType,
		"out_trade_no": req.TradeNo,
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"name":         req.Subject,
		"money":        req.Money.StringFixed(2),
		"clientip":     req.ClientIP,
		"timestamp":    strconv.FormatInt(time.Now().Unix(), 10), // V2 必带时间戳
	}
	// 商户私钥 RSA 签名
	sig, err := sign.MakeRSA(params, merchantPriv)
	if err != nil {
		return channel.CreateResp{}, errors.New("epayn 下单签名失败: " + err.Error())
	}
	params["sign"] = sig
	params["sign_type"] = "RSA"

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+"/api/pay/submit",
		strings.NewReader(form.Encode()))
	if err != nil {
		return channel.CreateResp{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := httpClient.Do(httpReq)
	if err != nil {
		return channel.CreateResp{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var sr submitResp
	if err := json.Unmarshal(body, &sr); err != nil {
		return channel.CreateResp{}, errors.New("epayn 上游返回解析失败: " + string(body))
	}
	if sr.Code != 0 {
		return channel.CreateResp{}, errors.New("epayn 上游下单失败: " + sr.Msg)
	}

	pt := channel.PayTypeQRCode
	if sr.Data.PayType == string(channel.PayTypeRedirect) {
		pt = channel.PayTypeRedirect
	}
	return channel.CreateResp{PayType: pt, PayURL: sr.Data.PayURL, QRCode: sr.Data.QRCode}, nil
}

// Query 主动查单：占位（以回调为准）。
func (Channel) Query(_ context.Context, _ channel.Config, _ string) (bool, error) {
	return false, nil
}

// Notify 解析上游回调：用上游平台公钥 RSA 验签 + trade_status 判定。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	_, _, platformPub, _, err := conf(cfg)
	if err != nil {
		return channel.NotifyResult{}, err
	}
	params := map[string]string{}
	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		params[k] = v
	}
	if !sign.VerifyRSA(params, platformPub) {
		return channel.NotifyResult{}, errors.New("epayn 回调 RSA 验签失败")
	}
	money, _ := decimal.NewFromString(params["money"])
	return channel.NotifyResult{
		TradeNo:    params["out_trade_no"],
		ChannelNo:  params["trade_no"],
		Money:      money,
		Success:    params["trade_status"] == "TRADE_SUCCESS",
		AckContent: "success",
	}, nil
}

func epaynType(typename string) string {
	switch typename {
	case "wxpay", "alipay", "qqpay", "bank", "jdpay":
		return typename
	default:
		return "alipay"
	}
}

func init() {
	channel.Register(Channel{})
}
