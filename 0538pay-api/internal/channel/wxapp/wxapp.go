// Package wxapp 实现微信支付 APIv3「APP 支付」渠道（对齐 epay V3 PaymentService appPay）。
//
// 复用 wxbase：POST /v3/pay/transactions/app 拿 prepay_id，再用商户私钥对
// appid\ntimestamp\nnoncestr\nprepayid\n 做 RSA 签名，组 getAppParameters（App SDK 拉起）。
// 直连/服务商由 config sub_mchid 自适应。key = "wxapp"。
package wxapp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxbase"
)

type Channel struct{}

func (Channel) Key() string { return "wxapp" }

func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	body, err := wxbase.BasePrepayBody(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	respBody, err := wxbase.Prepay(ctx, cfg, "app", body)
	if err != nil {
		return channel.CreateResp{}, err
	}
	var pr struct {
		PrepayID string `json:"prepay_id"`
	}
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return channel.CreateResp{}, err
	}
	if pr.PrepayID == "" {
		return channel.CreateResp{}, fmt.Errorf("微信 APP 下单未返回 prepay_id")
	}
	appParams, err := wxbase.BuildAppParams(cfg, pr.PrepayID)
	if err != nil {
		return channel.CreateResp{}, err
	}
	// App 拉起参数以 JSON 串放 RawHTML 透传给客户端（App SDK 解析后调起微信）。
	infoJSON, _ := json.Marshal(appParams)
	return channel.CreateResp{
		PayType: channel.PayTypeWap,
		RawHTML: string(infoJSON),
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

func (Channel) Inputs() []channel.FieldInput { return wxbase.Inputs() }

func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "app", Name: "APP 支付"}}
}

func init() { channel.Register(Channel{}) }
