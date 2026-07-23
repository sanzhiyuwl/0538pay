// Package wxh5 实现微信支付 APIv3「H5 支付」渠道（手机浏览器）。
//
// 对齐 epay wxpayn h5Pay：POST /v3/pay/transactions/h5 带 scene_info，应答返回 h5_url，
// 前端跳转该 URL 拉起微信支付。key = "wxh5"。
package wxh5

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/channel/wxbase"
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
	body["scene_info"] = map[string]interface{}{
		"payer_client_ip": clientIP,
		"h5_info":         map[string]string{"type": "Wap"},
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

// Inputs 声明配置字段（复用微信共用字段，元数据驱动后台密钥表单）。
func (Channel) Inputs() []channel.FieldInput { return wxbase.Inputs() }

// Products 声明本渠道支持的支付产品形态（对齐 epay $info['select']）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "h5", Name: "H5 手机浏览器支付"}}
}

func init() { channel.Register(Channel{}) }
