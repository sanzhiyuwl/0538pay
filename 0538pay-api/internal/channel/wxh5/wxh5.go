// Package wxh5 实现微信支付 APIv3「H5 支付」渠道（手机浏览器）。
//
// 对齐 epay wxpayn h5Pay：POST /v3/pay/transactions/h5 带 scene_info，应答返回 h5_url，
// 前端跳转该 URL 拉起微信支付。key = "wxh5"。
package wxh5

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxbase"
)

type Channel struct{}

func (Channel) Key() string { return "wxh5" }

func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	body, err := wxbase.BasePrepayBody(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	clientIP := req.ClientIP
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	// h5_info 补 app_name/app_url（对齐 epay wxpayn h5Pay scene_info：微信要求 H5 场景报备应用名与网址，
	// 部分风控严格的商户号缺省会被拒单）。app_url 取回跳地址的站点根，app_name 取站点名。
	h5Info := map[string]string{"type": "Wap"}
	if appURL := siteRoot(req.ReturnURL); appURL != "" {
		h5Info["app_url"] = appURL
	}
	if req.SiteName != "" {
		h5Info["app_name"] = req.SiteName
	}
	body["scene_info"] = map[string]interface{}{
		"payer_client_ip": clientIP,
		"h5_info":         h5Info,
	}
	respBody, err := wxbase.Prepay(ctx, cfg, "h5", body)
	if err != nil {
		return channel.CreateResp{}, err
	}
	var pr struct {
		H5URL string `json:"h5_url"`
	}
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return channel.CreateResp{}, err
	}
	if pr.H5URL == "" {
		return channel.CreateResp{}, fmt.Errorf("微信 H5 下单未返回 h5_url")
	}
	return channel.CreateResp{
		PayType: channel.PayTypeWap,
		PayURL:  pr.H5URL,
	}, nil
}

func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	return wxbase.QueryPaid(ctx, cfg, tradeNo)
}

func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return wxbase.ParseNotify(cfg, raw)
}

func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return wxbase.Refund(ctx, cfg, req)
}

// siteRoot 从完整 URL 取站点根（scheme://host/），用作 scene_info.h5_info.app_url。解析失败或空返回空串。
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

// Inputs 声明配置字段（复用微信共用字段，元数据驱动后台密钥表单）。
func (Channel) Inputs() []channel.FieldInput { return wxbase.Inputs() }

// Products 声明本渠道支持的支付产品形态（对齐 epay $info['select']）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "h5", Name: "H5 手机浏览器支付"}}
}

func init() { channel.Register(Channel{}) }
