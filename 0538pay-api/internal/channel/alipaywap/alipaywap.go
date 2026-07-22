// Package alipaywap 实现支付宝「手机网站支付」渠道（alipay.trade.wap.pay）。
//
// 对齐 epay alipay 插件 wapPay：手机浏览器 H5 跳转支付宝。product_code=QUICK_WAP_WAY。
// 与 alipaypage 同源，仅 method/product_code/PayType 不同。回调/查单/退款复用 alipaybase。key = "alipaywap"。
package alipaywap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/channel/alipaybase"
)

type Channel struct{}

func (Channel) Key() string { return "alipaywap" }

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
		"product_code": "QUICK_WAP_WAY",
	}
	bizJSON, _ := json.Marshal(biz)
	notify := req.NotifyURL
	if notify == "" {
		notify = cfg.NotifyURL
	}
	params, err := alipaybase.SignedParams(cfg, alipaybase.BuildSysParams(cfg, "alipay.trade.wap.pay", string(bizJSON), notify, req.ReturnURL))
	if err != nil {
		return channel.CreateResp{}, err
	}
	return channel.CreateResp{
		PayType: channel.PayTypeWap,
		PayURL:  alipaybase.BuildRedirectURL(params),
	}, nil
}

func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	return alipaybase.QueryPaid(ctx, cfg, tradeNo)
}

func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return alipaybase.ParseNotify(cfg, raw)
}

func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return alipaybase.Refund(ctx, cfg, req)
}

func init() { channel.Register(Channel{}) }
