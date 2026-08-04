// Package wxpay 是微信支付 APIv3「聚合门面」渠道：一个通道承载 Native/JSAPI/H5/APP 全部形态，
// 下单时按买家场景（device）+ 通道勾选的 apptype 集合，自动分派委托到对应形态包
// （wxnative/wxjsapi/wxh5/wxapp），对齐 epay wxpayn_plugin.php 的 submit()/mapi() 单插件内分派。
//
// 设计（批次一·门面模式，不动旧形态包）：
//   - Create：读 req.Device（微信内 wechat / 手机 mobile / PC）+ req.Extra["apptype"]（通道开通的形态集，
//     逗号分隔，对齐 epay pre_channel.apptype）+ req.Method（API 显式指定），按决策表选形态并委托。
//   - Query/Notify/Refund/Transfer：与形态无关，直接复用 wxbase（微信回调不分形态，天然兼容）。
//
// apptype 形态编码对齐 epay $info['select']：1=Native 2=JSAPI 3=H5 5=APP。key = "wxpay"。
package wxpay

import (
	"context"
	"fmt"
	"strings"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxbase"
)

const pluginKey = "wxpay"

// 形态编码（对齐 epay wxpayn select：1=Native 2=JSAPI 3=H5 5=APP）。
const (
	formNative = "1"
	formJSAPI  = "2"
	formH5     = "3"
	formAPP    = "5"
)

// formToDelegateKey 形态编码 → 委托的形态包 key（APIv3）。
var formToDelegateKey = map[string]string{
	formNative: "wxnative",
	formJSAPI:  "wxjsapi",
	formH5:     "wxh5",
	formAPP:    "wxapp",
}

// Channel 微信 APIv3 聚合门面，实现 channel.PaymentChannel + Refunder/Transferer/Configurable。
type Channel struct{}

func (Channel) Key() string { return pluginKey }

// Create 按买家场景 + apptype 分派到形态包委托下单（对齐 epay wxpayn submit/mapi 决策表）。
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

// pickForm 决策：选出本次下单命中的形态编码（对齐 epay wxpayn submit()/mapi() 二维分派）。
// 优先级：API 显式 method → 买家场景 device（微信内→JSAPI / 手机→H5→APP / PC→Native），均受 apptype 集合门控。
// 命中不了显式偏好时，回退到 apptype 里"该场景可承载"的形态；仍无则空串（由 Create 报错）。
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
		// APIv3 不支持付款码（对齐 epay wxpayn mapi 'scan' 直接报错）。
		return ""
	}

	// (2) 买家场景分派（对齐 epay submit/mapi 的 checkwechat/checkmobile 分支）。
	switch {
	case device == "wechat":
		// 微信内：优先 JSAPI（公众号/小程序内拉起），否则退回 Native 扫码（长按识别）。
		if has(formJSAPI) {
			return formJSAPI
		}
		return firstAvailable(has, formNative)
	case isMobile(device):
		// 手机浏览器：优先 H5，其次 APP，再退 Native 扫码。
		if has(formH5) {
			return formH5
		}
		if has(formAPP) {
			return formAPP
		}
		return firstAvailable(has, formNative)
	default:
		// PC：Native 扫码优先，否则回退任一已开通形态（兜底）。
		return firstAvailable(has, formNative, formH5, formJSAPI, formAPP)
	}
}

// firstAvailable 按给定优先序返回首个已开通的形态；都没有返回空串。
func firstAvailable(has func(string) bool, prefer ...string) string {
	for _, f := range prefer {
		if has(f) {
			return f
		}
	}
	return ""
}

// isMobile 判断是否手机（非微信内）场景（对齐 epay checkmobile 归类）。
func isMobile(device string) bool {
	switch device {
	case "mobile", "wap", "h5", "alipay", "qq", "app":
		return true
	}
	return false
}

// parseAppTypes 解析通道 apptype 串（逗号分隔）为形态编码切片，去空白/去空项。
func parseAppTypes(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	out := make([]string, 0, 4)
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// Query 主动查单：与形态无关，复用 wxbase.QueryPaid（直连/服务商路径自适应）。
func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	return wxbase.QueryPaid(ctx, cfg, tradeNo)
}

// Notify 验签 + 解密回调：微信回调不分形态，复用 wxbase.ParseNotify。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return wxbase.ParseNotify(cfg, raw)
}

// Refund 原路退款：复用 wxbase.Refund（服务商模式带 sub_mchid）。
func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return wxbase.Refund(ctx, cfg, req)
}

// Transfer 商家转账到零钱：复用 wxbase.MchTransfer。
func (Channel) Transfer(ctx context.Context, cfg channel.Config, req channel.TransferReq) (channel.TransferResp, error) {
	return wxbase.MchTransfer(ctx, cfg, req)
}

// TransferQuery 查询转账单状态。
func (Channel) TransferQuery(ctx context.Context, cfg channel.Config, outBizNo string) (channel.TransferResp, error) {
	return wxbase.MchTransferQuery(ctx, cfg, outBizNo)
}

// Inputs 声明配置字段（复用微信共用字段）。
func (Channel) Inputs() []channel.FieldInput { return wxbase.Inputs() }

// Products 声明本聚合渠道支持的全部形态（对齐 epay $info['select']，供后台勾选 apptype）。
// Code 用 epay 形态编码（1/2/3/5），与 pay_channel.apptype 存储一致。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{
		{Code: formNative, Name: "Native 扫码", Group: "wxpay"},
		{Code: formJSAPI, Name: "JSAPI 公众号/小程序", Group: "wxpay"},
		{Code: formH5, Name: "H5", Group: "wxpay"},
		{Code: formAPP, Name: "APP", Group: "wxpay"},
	}
}

func init() {
	channel.Register(Channel{})
}
