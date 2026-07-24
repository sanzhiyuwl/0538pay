// Package vmq 提供「V免签」聚合收款渠道（github.com/szvone/vmqphp）。
//
// V免签是最简聚合类渠道：无需持牌资质，自建一个 vmq 监控端（手机挂支付宝/微信收款码），
// 通过「接口地址 + 商户ID + 通讯密钥」即可下单/收单。签名为拼接式 MD5（非 ksort k=v），
// 与 pkg/sign（易支付协议 MD5）不同，故本渠道自带签名逻辑。
//
// 通过 init() 自注册到 channel.registry，key = "vmq"。
//
// config 字段（通道 config JSON）：
//
//	appurl = vmq 接口地址（必须 http:// 或 https:// 开头，末尾带 /，如 http://127.0.0.1:8081/）
//	appid  = 商户 ID（mchId；vmq 单商户模式随便填）
//	appkey = 通讯密钥（appkey，用于拼接式 MD5 签名）
//
// 对齐 epay plugins/vmq/vmq_plugin.php：
//   - 下单 sign = md5(payId + type + price + appkey)，POST {appurl}createOrder，返回 HTML 表单自动提交。
//   - 回调 sign = md5(payId + type + price + reallyPrice + appkey)，GET 参数，校验后回 "success"。
package vmq

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/epvia/api/internal/channel"
	"github.com/shopspring/decimal"
)

// Channel V免签渠道，实现 channel.PaymentChannel。
type Channel struct{}

func (Channel) Key() string { return "vmq" }

// conf 从 Config 解出 vmq 三要素（appurl/appid/appkey）。
func conf(cfg channel.Config) (apiURL, mchID, key string, err error) {
	apiURL = cfg.ExtraOr("appurl", "")
	if apiURL != "" && !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	mchID = cfg.AppID
	if mchID == "" {
		mchID = cfg.ExtraOr("appid", "")
	}
	key = cfg.ExtraOr("appkey", cfg.Key)
	if apiURL == "" || key == "" {
		return "", "", "", errors.New("vmq 渠道缺少 appurl/appkey 配置")
	}
	return apiURL, mchID, key, nil
}

// vmqType 把内部支付方式英文名映射为 vmq type（微信=1 支付宝=2 银联=3 QQ=4；对齐 vmq_plugin.php）。
func vmqType(typename string) string {
	switch typename {
	case "wxpay":
		return "1"
	case "qqpay":
		return "4"
	case "bank":
		return "3"
	default: // alipay 及缺省
		return "2"
	}
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// Create 向 vmq 监控端下单：拼接式 MD5 签名，返回自动提交的 HTML 表单（对齐 vmq_plugin.php::submit）。
func (Channel) Create(_ context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	apiURL, mchID, key, err := conf(cfg)
	if err != nil {
		return channel.CreateResp{}, err
	}
	payType := vmqType(req.Extra["typename"])
	price := req.Money.StringFixed(2)

	// data 顺序固定，与 vmq_plugin.php 一致。sign = md5(payId + type + price + appkey)。
	data := []struct{ k, v string }{
		{"mchId", mchID},
		{"payId", req.TradeNo},
		{"type", payType},
		{"price", price},
		{"isHtml", "1"},
		{"notifyUrl", req.NotifyURL},
		{"returnUrl", req.ReturnURL},
		{"sign", md5hex(req.TradeNo + payType + price + key)},
	}

	// 组装自动提交表单，POST 到 {appurl}createOrder（对齐 vmq_plugin.php 的 html 分支）。
	var b strings.Builder
	b.WriteString(`<form action="` + htmlEscape(apiURL+"createOrder") + `" method="post" id="dopay">`)
	for _, kv := range data {
		b.WriteString(`<input type="hidden" name="` + htmlEscape(kv.k) + `" value="` + htmlEscape(kv.v) + `" />`)
	}
	b.WriteString(`<input type="submit" value="正在跳转" /></form>`)
	b.WriteString(`<script>document.getElementById("dopay").submit();</script>`)

	return channel.CreateResp{PayType: channel.PayTypeHTML, RawHTML: b.String()}, nil
}

// Query 主动查单：vmq 无标准查单接口，以回调为准（占位返回未支付）。
func (Channel) Query(_ context.Context, _ channel.Config, _ string) (bool, error) {
	return false, nil
}

// Notify 解析 vmq 异步回调：sign = md5(payId + type + price + reallyPrice + appkey)，
// 校验后回 "success"（对齐 vmq_plugin.php::notify）。回调为 GET 参数。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	_, _, key, err := conf(cfg)
	if err != nil {
		return channel.NotifyResult{}, err
	}
	payID := raw["payId"]
	sign := raw["sign"]
	if payID == "" || sign == "" {
		return channel.NotifyResult{}, errors.New("vmq 回调参数不完整")
	}
	typ := raw["type"]
	price := raw["price"]
	reallyPrice := raw["reallyPrice"]
	if md5hex(payID+typ+price+reallyPrice+key) != sign {
		return channel.NotifyResult{}, errors.New("vmq 回调验签失败")
	}
	// vmq 回调金额以订单原始金额 price 为准；实付 reallyPrice 供上层核对。
	money, _ := decimal.NewFromString(price)
	return channel.NotifyResult{
		TradeNo:    payID,
		Money:      money,
		Success:    true, // vmq 只在支付成功时回调
		AckContent: "success",
	}, nil
}

// htmlEscape 转义 HTML 属性值中的特殊字符，防止渠道地址/参数破坏表单结构。
func htmlEscape(s string) string {
	r := strings.NewReplacer(`&`, "&amp;", `"`, "&quot;", `<`, "&lt;", `>`, "&gt;", `'`, "&#39;")
	return r.Replace(s)
}

// Inputs 声明配置字段（元数据驱动后台密钥表单，对齐 vmq_plugin.php $info['inputs']）。
func (Channel) Inputs() []channel.FieldInput {
	return []channel.FieldInput{
		{Name: "appurl", Label: "接口地址", Type: "text", Require: true, Tip: "V免签接口地址，必须 http:// 或 https:// 开头，末尾带 /"},
		{Name: "appid", Label: "商户 ID", Type: "text", Require: false, Tip: "vmq mchId，单商户模式随便填"},
		{Name: "appkey", Label: "通讯密钥", Type: "password", Require: true, Tip: "V免签后台的通讯密钥，用于 MD5 签名"},
	}
}

// Products 声明支持的支付产品形态（对齐 vmq_plugin.php types: alipay/qqpay/wxpay）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{
		{Code: "alipay", Name: "支付宝"},
		{Code: "wxpay", Name: "微信支付"},
		{Code: "qqpay", Name: "QQ钱包"},
	}
}

func init() {
	channel.Register(Channel{})
}
