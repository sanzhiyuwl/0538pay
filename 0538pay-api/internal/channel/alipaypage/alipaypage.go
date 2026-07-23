// Package alipaypage 实现支付宝「电脑网站支付」渠道（alipay.trade.page.pay）。
//
// 对齐 epay alipay 插件 pagePay：PC 端跳转支付宝收银台。product_code=FAST_INSTANT_TRADE_PAY。
// 下单不请求网关，而是把已签名的系统参数拼成 GET 跳转 URL（PayTypeRedirect），浏览器打开即到收银台。
// 回调/查单/退款复用 alipaybase（与当面付同源）。key = "alipaypage"。
package alipaypage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/channel/alipaybase"
)

type Channel struct{}

func (Channel) Key() string { return "alipaypage" }

func (Channel) Create(_ context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	if cfg.AppID == "" {
		return channel.CreateResp{}, fmt.Errorf("支付宝通道缺少 appid 配置")
	}
	subject := req.Subject
	if subject == "" {
		subject = "商品支付"
	}
	biz := map[string]string{
		"out_trade_no": req.TradeNo,
		"total_amount": req.Money.StringFixed(2),
		"subject":      subject,
		"product_code": "FAST_INSTANT_TRADE_PAY",
	}
	bizJSON, _ := json.Marshal(biz)
	notify := req.NotifyURL
	if notify == "" {
		notify = cfg.NotifyURL
	}
	params, err := alipaybase.SignedParams(cfg, alipaybase.BuildSysParams(cfg, "alipay.trade.page.pay", string(bizJSON), notify, req.ReturnURL))
	if err != nil {
		return channel.CreateResp{}, err
	}
	// PC 网页支付：跳转到网关（GET），支付宝渲染收银台。
	return channel.CreateResp{
		PayType: channel.PayTypeRedirect,
		PayURL:  alipaybase.BuildRedirectURL(params),
	}, nil
}

func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	return alipaybase.QueryPaid(ctx, cfg, tradeNo)
}

func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return alipaybase.ParseNotify(cfg, raw)
}

// Refund 原路退款（alipay.trade.refund）。
func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return alipaybase.Refund(ctx, cfg, req)
}

// Inputs 声明配置字段（复用支付宝共用字段，元数据驱动后台密钥表单）。
func (Channel) Inputs() []channel.FieldInput { return alipaybase.Inputs() }

// Products 声明本渠道支持的支付产品形态（对齐 epay $info['select']）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "page", Name: "电脑网站支付"}}
}

func init() { channel.Register(Channel{}) }
