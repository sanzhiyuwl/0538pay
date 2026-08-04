// Package wxpayv2 是微信支付 APIv2「聚合门面」渠道：一个通道承载 Native/JSAPI/H5/APP/付款码 全部形态，
// 下单时按买家场景（device）+ 通道勾选的 apptype 集合，自动分派委托到对应形态包
// （wxv2native/wxv2jsapi/wxv2h5/wxv2app/wxv2micro），对齐 epay wxpay_plugin.php 单插件内分派。
//
// 与 APIv3 门面 wxpay 同构，差异：协议为 APIv2、多一个「付款码(scan)」形态（APIv3 不支持）。
// Query/Notify/Refund 与形态无关，复用 wxv2base。
//
// apptype 形态编码对齐 epay $info['select']：1=Native 2=JSAPI 3=H5 5=APP 6=付款码。key = "wxpayv2"。
package wxpayv2

import (
	"context"
	"fmt"
	"strings"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
)

const pluginKey = "wxpayv2"

// 形态编码（对齐 epay wxpay select：1=Native 2=JSAPI 3=H5 5=APP；付款码我方用 6 表意，对齐 pay_type 扩展）。
const (
	formNative = "1"
	formJSAPI  = "2"
	formH5     = "3"
	formAPP    = "5"
	formMicro  = "6" // 付款码（商家扫用户码，同步支付）
)

// formToDelegateKey 形态编码 → 委托的形态包 key（APIv2）。
var formToDelegateKey = map[string]string{
	formNative: "wxv2native",
	formJSAPI:  "wxv2jsapi",
	formH5:     "wxv2h5",
	formAPP:    "wxv2app",
	formMicro:  "wxv2micro",
}

// Channel 微信 APIv2 聚合门面。
type Channel struct{}

func (Channel) Key() string { return pluginKey }

// Create 按买家场景 + apptype 分派到形态包委托下单。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	apptypes := parseAppTypes(req.Extra["apptype"])
	if len(apptypes) == 0 {
		return channel.CreateResp{}, fmt.Errorf("微信通道未开通任何支付形态（apptype 为空），请在通道配置中勾选")
	}
	form := pickForm(req, apptypes)
	if form == "" {
		return channel.CreateResp{}, fmt.Errorf("当前场景无匹配的微信支付形态（已开通：%s）", strings.Join(apptypes, ","))
	}
	delegateKey := formToDelegateKey[form]
	target, ok := channel.Get(delegateKey)
	if !ok {
		return channel.CreateResp{}, fmt.Errorf("微信形态渠道未注册：%s", delegateKey)
	}
	return target.Create(ctx, cfg, req)
}

// pickForm 决策选出命中形态（对齐 epay wxpay submit/mapi 二维分派，含付款码分支）。
func pickForm(req channel.CreateReq, apptypes []string) string {
	has := func(f string) bool {
		for _, a := range apptypes {
			if a == f {
				return true
			}
		}
		return false
	}
	method := strings.ToLower(strings.TrimSpace(req.Method))
	device := strings.ToLower(strings.TrimSpace(req.Device))

	// (1) API 显式指定形态（对齐 epay mapi 的 $method 最高优先）。
	switch method {
	case "jsapi", "applet":
		if has(formJSAPI) {
			return formJSAPI
		}
	case "app":
		if has(formAPP) {
			return formAPP
		}
	case "scan":
		// 付款码：需通道开通且请求带 auth_code（对齐 epay wxpay scanpay）。
		if has(formMicro) {
			return formMicro
		}
		return ""
	}
	// 带 auth_code 视为付款码场景（收银机/门店扫码枪，无显式 method 时）。
	if strings.TrimSpace(req.AuthCode) != "" && has(formMicro) {
		return formMicro
	}

	// (2) 买家场景分派。
	switch {
	case device == "wechat":
		if has(formJSAPI) {
			return formJSAPI
		}
		return firstAvailable(has, formNative)
	case isMobile(device):
		if has(formH5) {
			return formH5
		}
		if has(formAPP) {
			return formAPP
		}
		return firstAvailable(has, formNative)
	default:
		return firstAvailable(has, formNative, formH5, formJSAPI, formAPP)
	}
}

func firstAvailable(has func(string) bool, prefer ...string) string {
	for _, f := range prefer {
		if has(f) {
			return f
		}
	}
	return ""
}

func isMobile(device string) bool {
	switch device {
	case "mobile", "wap", "h5", "alipay", "qq", "app":
		return true
	}
	return false
}

func parseAppTypes(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	out := make([]string, 0, 5)
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// Query 主动查单：复用 wxv2base.QueryPaid。
func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	return wxv2base.QueryPaid(ctx, cfg, tradeNo)
}

// Notify 验签回调：复用 wxv2base.ParseNotify。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return wxv2base.ParseNotify(cfg, raw)
}

// Refund 原路退款：复用 wxv2base.Refund。
func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return wxv2base.Refund(ctx, cfg, req)
}

// Inputs 声明配置字段（复用微信 V2 共用字段）。
func (Channel) Inputs() []channel.FieldInput { return wxv2base.Inputs() }

// Products 声明本聚合渠道支持的全部形态（供后台勾选 apptype）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{
		{Code: formNative, Name: "Native 扫码", Group: "wxpay"},
		{Code: formJSAPI, Name: "JSAPI 公众号/小程序", Group: "wxpay"},
		{Code: formH5, Name: "H5", Group: "wxpay"},
		{Code: formAPP, Name: "APP", Group: "wxpay"},
		{Code: formMicro, Name: "付款码", Group: "wxpay"},
	}
}

func init() {
	channel.Register(Channel{})
}
