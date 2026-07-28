// Package wxv2h5 实现微信支付 APIv2「H5 支付」渠道（手机浏览器，对齐 epay wxpay h5）。
//
// 复用 wxv2base，声明 trade_type=MWEB + scene_info.h5_info，取 mweb_url 跳转拉起。
// 直连/服务商由 config sub_mchid 自适应。key = "wxv2h5"。
package wxv2h5

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
)

type Channel struct{}

func (Channel) Key() string { return "wxv2h5" }

func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	params, err := wxv2base.BaseOrderParams(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	params["trade_type"] = "MWEB"
	// scene_info 为 JSON 串（对齐 epay h5：json_encode(['h5_info'=>['type'=>'Wap','wap_url'=>siteurl,'wap_name'=>sitename]])）。
	// wap_url 取回跳地址的站点根（部分风控严格的商户号要求非空，否则下单被拒）；wap_name 取站点名。
	h5Info := map[string]string{"type": "Wap"}
	if wapURL := siteRoot(req.ReturnURL); wapURL != "" {
		h5Info["wap_url"] = wapURL
	}
	if name := req.SiteName; name != "" {
		h5Info["wap_name"] = name
	}
	sceneInfo, _ := json.Marshal(map[string]interface{}{"h5_info": h5Info})
	params["scene_info"] = string(sceneInfo)
	result, err := wxv2base.UnifiedOrder(ctx, cfg, params)
	if err != nil {
		return channel.CreateResp{}, err
	}
	mwebURL := result["mweb_url"]
	if mwebURL == "" {
		return channel.CreateResp{}, fmt.Errorf("微信 V2 H5 下单未返回 mweb_url")
	}
	if req.ReturnURL != "" {
		mwebURL += "&redirect_url=" + url.QueryEscape(req.ReturnURL)
	}
	return channel.CreateResp{
		PayType: channel.PayTypeWap,
		PayURL:  mwebURL,
	}, nil
}

func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	return wxv2base.QueryPaid(ctx, cfg, tradeNo)
}

func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return wxv2base.ParseNotify(cfg, raw)
}

func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return wxv2base.Refund(ctx, cfg, req)
}

// siteRoot 从完整 URL 取站点根（scheme://host/），用作 scene_info.wap_url。解析失败或空返回空串。
func siteRoot(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/"
}

func (Channel) Inputs() []channel.FieldInput { return wxv2base.Inputs() }

func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "h5", Name: "H5 手机浏览器支付"}}
}

func init() { channel.Register(Channel{}) }
