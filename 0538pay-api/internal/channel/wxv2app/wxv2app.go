// Package wxv2app 实现微信支付 APIv2「APP 支付」渠道（对齐 epay wxpay appPay）。
//
// 复用 wxv2base：trade_type=APP 统一下单拿 prepay_id，再组 getAppParameters（MD5 二次签 sign），
// 前端 App SDK 拉起。直连/服务商由 config sub_mchid 自适应。key = "wxv2app"。
package wxv2app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
)

type Channel struct{}

func (Channel) Key() string { return "wxv2app" }

func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	params, err := wxv2base.BaseOrderParams(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	params["trade_type"] = "APP"
	result, err := wxv2base.UnifiedOrder(ctx, cfg, params)
	if err != nil {
		return channel.CreateResp{}, err
	}
	prepayID := result["prepay_id"]
	if prepayID == "" {
		return channel.CreateResp{}, fmt.Errorf("微信 V2 APP 下单未返回 prepay_id")
	}
	appParams, err := wxv2base.BuildAppParams(cfg, prepayID)
	if err != nil {
		return channel.CreateResp{}, err
	}
	infoJSON, _ := json.Marshal(appParams)
	return channel.CreateResp{
		PayType: channel.PayTypeWap,
		RawHTML: string(infoJSON),
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

func (Channel) Inputs() []channel.FieldInput { return wxv2base.Inputs() }

func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "app", Name: "APP 支付"}}
}

func init() { channel.Register(Channel{}) }
