// Package qixiang 提供「七相聚合支付」渠道（对接外部上游 qixiangpay.cn）。
//
// 协议定性：七相文档明确「本平台兼容易支付接口，无需额外开发」——即彩虹易支付
// V1(MD5) 协议。签名规则逐字节与本站 pkg/sign 同源（ksort→k=v& 拼接→+key→md5 小写，
// 跳过 sign/sign_type/空值）。与本站已有的 epay 渠道差异在「对接目标」：
//   - epay 渠道：对接本站自己的上游 /api/pay/submit，响应包在 code/data 里（code=0 成功）。
//   - qixiang 渠道：对接七相外部网关 mapi.php，响应字段平铺（code=1 成功，payurl/qrcode 平铺）。
// 故独立成包，不复用 epay 渠道逻辑。
//
// 用途：仅用于本平台自身收款（平台侧配置一个七相通道即可，商户下单时由选通道主链
// 路由到它，商户不直接见通道名，天然满足「平台自用」）。
//
// config 字段（通道 config JSON）：
//   appid  = 七相分配的商户号 pid
//   appkey = 七相商户密钥 key（MD5 签名用）
//   appurl = 七相接口网关地址（默认 https://api.payqixiang.cn，末尾不带 /）
//
// 接口坐标（七相开发文档 doc_old.html）：
//   下单：POST {appurl}/mapi.php        （device=jump 返回自适应 payurl）
//   回调：GET  notify_url                （trade_status=TRADE_SUCCESS，应答 success）
//   退款：POST {appurl}/api.php?act=refund
//   查单：GET  {appurl}/api.php?act=order
package qixiang

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/sign"
	"github.com/shopspring/decimal"
)

// 七相默认网关地址（config 未配 appurl 时兜底）。
const defaultGateway = "https://api.payqixiang.cn"

// Channel 七相聚合支付渠道，实现 channel.PaymentChannel + Refunder + Configurable。
type Channel struct{}

func (Channel) Key() string { return "qixiang" }

var httpClient = &http.Client{Timeout: 15 * time.Second}

// upstream 从 config 解出七相三要素（pid/key/gateway）。appurl 缺省用官方网关。
func upstream(cfg channel.Config) (pid, key, gateway string, err error) {
	pid = cfg.AppID
	if pid == "" {
		pid = cfg.ExtraOr("pid", "")
	}
	key = cfg.ExtraOr("appkey", cfg.Key)
	gateway = strings.TrimRight(cfg.ExtraOr("appurl", defaultGateway), "/")
	if pid == "" || key == "" {
		return "", "", "", errors.New("七相渠道缺少 appid/appkey 配置")
	}
	return pid, key, gateway, nil
}

// mapiResp 七相 mapi.php 统一下单响应（JSON 平铺，code=1 成功）。
type mapiResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	TradeNo string `json:"trade_no"`
	PayURL  string `json:"payurl"`
	QRCode  string `json:"qrcode"`
}

// Create 向七相 mapi.php 统一下单：MD5 签名 POST，按 device 返回二维码或跳转 URL。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	pid, key, gateway, err := upstream(cfg)
	if err != nil {
		return channel.CreateResp{}, err
	}

	// 支付方式：七相仅支持 alipay/wxpay，据订单 typename 映射（缺省 alipay）。
	payType := qixiangType(req.Extra["typename"])
	// 设备类型：据下单场景映射七相 device（pc→二维码 / mobile→H5 / wechat→JSAPI / 其它→jump 自适应）。
	device := qixiangDevice(req)

	params := map[string]string{
		"pid":          pid,
		"type":         payType,
		"out_trade_no": req.TradeNo,
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"name":         req.Subject,
		"money":        req.Money.StringFixed(2),
		"clientip":     clientIPOr(req.ClientIP),
		"device":       device,
	}
	if p := req.Extra["param"]; p != "" {
		params["param"] = p
	}
	params["sign"] = sign.MakeMD5(params, key)
	params["sign_type"] = "MD5"

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, gateway+"/mapi.php",
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

	var mr mapiResp
	if err := json.Unmarshal(body, &mr); err != nil {
		return channel.CreateResp{}, errors.New("七相上游返回解析失败: " + string(body))
	}
	if mr.Code != 1 {
		return channel.CreateResp{}, errors.New("七相上游下单失败: " + mr.Msg)
	}

	// 优先二维码链接（pc device），否则支付跳转 URL（jump/mobile/wechat）。
	if mr.QRCode != "" {
		return channel.CreateResp{PayType: channel.PayTypeQRCode, QRCode: mr.QRCode, PayURL: mr.PayURL}, nil
	}
	return channel.CreateResp{PayType: channel.PayTypeRedirect, PayURL: mr.PayURL}, nil
}

// orderResp 七相 api.php?act=order 查单响应（平铺，status=1 已支付）。
type orderResp struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Status int    `json:"status"`
}

// Query 主动查单（api.php?act=order）：status=1 为已支付。用于补偿回调遗漏。
func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	pid, key, gateway, err := upstream(cfg)
	if err != nil {
		return false, err
	}
	q := url.Values{}
	q.Set("act", "order")
	q.Set("pid", pid)
	q.Set("key", key)
	q.Set("out_trade_no", tradeNo)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, gateway+"/api.php?"+q.Encode(), nil)
	if err != nil {
		return false, err
	}
	res, err := httpClient.Do(httpReq)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var or orderResp
	if err := json.Unmarshal(body, &or); err != nil {
		return false, errors.New("七相查单返回解析失败: " + string(body))
	}
	return or.Code == 1 && or.Status == 1, nil
}

// Notify 解析七相异步回调（GET）：重算 MD5 验签 + trade_status 判定。
// 回调参数：pid/trade_no/out_trade_no/type/name/money/trade_status/param/sign/sign_type。
// 文档特别注明：回调若无 name 则不参与签名（我方验签逐字段自适应，无 name 自然不进串，天然对齐）。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	_, key, _, err := upstream(cfg)
	if err != nil {
		return channel.NotifyResult{}, err
	}

	// 剔除保留注入键（_ 前缀），仅用业务参数验签。
	params := map[string]string{}
	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		params[k] = v
	}
	if !sign.VerifyMD5(params, key) {
		return channel.NotifyResult{}, errors.New("七相回调验签失败")
	}

	money, _ := decimal.NewFromString(params["money"])
	return channel.NotifyResult{
		TradeNo:    params["out_trade_no"], // 商户订单号定位本地订单
		ChannelNo:  params["trade_no"],     // 七相系统单号进 ChannelNo
		Money:      money,
		Success:    params["trade_status"] == "TRADE_SUCCESS",
		AckContent: "success",
	}, nil
}

// refundResp 七相 api.php?act=refund 退款响应（code=1 或 0 为成功）。
type refundResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Refund 原路退款（api.php?act=refund，POST）。七相退款用 pid+key 鉴权（非 MD5 签名）。
// 需先在七相商户后台开启退款 API 开关。code=1 或 0 均视为成功（对齐文档）。
func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	pid, key, gateway, err := upstream(cfg)
	if err != nil {
		return channel.RefundResp{}, err
	}
	form := url.Values{}
	form.Set("pid", pid)
	form.Set("key", key)
	form.Set("out_trade_no", req.TradeNo)
	if req.ChannelNo != "" {
		form.Set("trade_no", req.ChannelNo) // 七相系统单号优先
	}
	form.Set("money", req.Money.StringFixed(2))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, gateway+"/api.php?act=refund",
		strings.NewReader(form.Encode()))
	if err != nil {
		return channel.RefundResp{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := httpClient.Do(httpReq)
	if err != nil {
		return channel.RefundResp{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var rr refundResp
	if err := json.Unmarshal(body, &rr); err != nil {
		return channel.RefundResp{}, errors.New("七相退款返回解析失败: " + string(body))
	}
	if rr.Code != 1 && rr.Code != 0 {
		return channel.RefundResp{}, errors.New("七相退款失败: " + rr.Msg)
	}
	return channel.RefundResp{Money: req.Money, Success: true}, nil
}

// qixiangType 把内部支付方式英文名映射为七相 type（仅支持 alipay/wxpay；其它归 alipay）。
func qixiangType(typename string) string {
	if typename == "wxpay" {
		return "wxpay"
	}
	return "alipay"
}

// qixiangDevice 据下单场景映射七相 device 参数（对齐文档设备类型列表）。
//   微信内(wechat) → wechat(JSAPI)；PC → pc(二维码链接)；其它(手机/未知) → jump(自适应)。
func qixiangDevice(req channel.CreateReq) string {
	switch req.Device {
	case "wechat":
		return "wechat"
	case "pc":
		return "pc"
	case "mobile", "alipay", "app":
		return "mobile"
	default:
		return "jump" // 自适应页面，兼容 PC 扫码 / 手机 H5 / 微信内
	}
}

// clientIPOr 保证 clientip 非空（七相必填；文档注明不强求真实，可随意传）。
func clientIPOr(ip string) string {
	if ip == "" {
		return "127.0.0.1"
	}
	return ip
}

// Inputs 声明配置字段（元数据驱动后台密钥表单）。键名与 upstream() 消费一致。
func (Channel) Inputs() []channel.FieldInput {
	return []channel.FieldInput{
		{Name: "appid", Label: "商户 ID", Type: "text", Require: true, Tip: "七相分配的商户号 pid"},
		{Name: "appkey", Label: "商户密钥", Type: "password", Require: true, Tip: "七相商户密钥 key，用于 MD5 签名"},
		{Name: "appurl", Label: "接口网关", Type: "text", Require: false, Tip: "七相接口网关地址，留空默认 " + defaultGateway + "，末尾不带 /"},
	}
}

// Products 声明支持的支付产品形态（七相仅微信/支付宝两方式）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{
		{Code: "alipay", Name: "支付宝"},
		{Code: "wxpay", Name: "微信支付"},
	}
}

func init() {
	channel.Register(Channel{})
}
