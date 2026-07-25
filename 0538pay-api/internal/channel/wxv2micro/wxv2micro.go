// Package wxv2micro 实现微信支付 APIv2「付款码支付」渠道（商家扫用户付款码，对齐 epay wxpay scanpay）。
//
// 付款码支付是同步流程：商户上送用户 auth_code，微信同步返回支付结果（无异步回调）。
// 复用 wxv2base.MicroPay 完成 /pay/micropay 调用；SUCCESS 时把 transaction_id/openid 放 RawHTML 透传，
// PayType=scan 交由上层据同步结果直接入账（对齐 epay scanpay 内 processNotify）。
// USERPAYING/SYSTEMERROR（用户支付中/系统错误）由上层轮询查单确认，见 Query。key = "wxv2micro"。
package wxv2micro

import (
	"context"
	"encoding/json"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
)

type Channel struct{}

func (Channel) Key() string { return "wxv2micro" }

// Create 上送付款码，同步取支付结果。SUCCESS 时 RawHTML 携带交易号/买家标识供上层入账。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	result, err := wxv2base.MicroPay(ctx, cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	// Execute 已保证 result_code=SUCCESS，否则返 error（含 USERPAYING/SYSTEMERROR，上层据错误码轮询查单）。
	payload, _ := json.Marshal(map[string]string{
		"transaction_id": result["transaction_id"],
		"out_trade_no":   result["out_trade_no"],
		"openid":         result["openid"],
		"total_fee":      result["total_fee"],
	})
	return channel.CreateResp{
		PayType: "scan", // V2 契约原生值，mapi 层透传
		RawHTML: string(payload),
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
	return []channel.ProductType{{Code: "scan", Name: "付款码支付"}}
}

func init() { channel.Register(Channel{}) }
