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
	"errors"
	"time"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
)

type Channel struct{}

func (Channel) Key() string { return "wxv2micro" }

// microRetryMax 付款码轮询查单次数上限（对齐 epay scanpay while($retry<6)）。
const microRetryMax = 6

// microPollInterval 每次轮询查单间隔（对齐 epay sleep(3)）。
var microPollInterval = 3 * time.Second

// Create 上送付款码，同步取支付结果（对齐 epay wxpay scanpay 全流程）。
// SUCCESS 时 RawHTML 携带交易号/买家标识 + paid=1，供上层据同步结果直接入账（付款码无异步回调）。
// 用户支付中(USERPAYING)/系统错误(SYSTEMERROR)时轮询查单最多 6 次；仍未成功则 reverse 撤单避免掉单。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	result, err := wxv2base.MicroPay(ctx, cfg, req)
	if err != nil {
		// 据业务错误码决定是否进入轮询（对齐 epay catch WeChatPayException getErrCode）。
		var biz *wxv2base.BizError
		if errors.As(err, &biz) && (biz.ErrCode == "USERPAYING" || biz.ErrCode == "SYSTEMERROR") {
			return pollAfterMicroPay(ctx, cfg, req, biz.ErrCode == "USERPAYING")
		}
		return channel.CreateResp{}, err
	}
	return scanResp(result), nil
}

// pollAfterMicroPay 付款码下单返回 USERPAYING/SYSTEMERROR 后的轮询+撤单容错（对齐 epay scanpay 内 while 循环）。
// USERPAYING 先等 2s 再轮询；每次间隔 3s 查单，trade_state=SUCCESS 则入账，非 USERPAYING 视为超时/取消，
// 6 次仍未成功则 reverse 撤单并报失败。
func pollAfterMicroPay(ctx context.Context, cfg channel.Config, req channel.CreateReq, userPaying bool) (channel.CreateResp, error) {
	if userPaying {
		if !sleepCtx(ctx, 2*time.Second) {
			return channel.CreateResp{}, ctx.Err()
		}
	}
	for retry := 0; retry < microRetryMax; retry++ {
		if !sleepCtx(ctx, microPollInterval) {
			return channel.CreateResp{}, ctx.Err()
		}
		result, err := wxv2base.QueryOrder(ctx, cfg, req.TradeNo)
		if err != nil {
			return channel.CreateResp{}, err
		}
		switch result["trade_state"] {
		case "SUCCESS":
			return scanResp(result), nil
		case "USERPAYING":
			// 用户仍在支付中，继续轮询
			continue
		default:
			// 超时/取消/关闭等：撤单后报失败（对齐 epay '订单超时或用户取消支付'）。
			_ = wxv2base.Reverse(ctx, cfg, req.TradeNo)
			return channel.CreateResp{}, errors.New("微信支付失败：订单超时或用户取消支付")
		}
	}
	// 轮询用尽仍未成功：撤单避免掉单（对齐 epay reverse + '订单已超时'）。
	_ = wxv2base.Reverse(ctx, cfg, req.TradeNo)
	return channel.CreateResp{}, errors.New("微信支付失败：订单已超时")
}

// scanResp 把付款码同步支付结果封装为渠道响应。paid=1 标记同步已支付，供上层直接入账。
func scanResp(result map[string]string) channel.CreateResp {
	payload, _ := json.Marshal(map[string]string{
		"transaction_id": result["transaction_id"],
		"out_trade_no":   result["out_trade_no"],
		"openid":         result["openid"],
		"total_fee":      result["total_fee"],
		"paid":           "1", // 同步支付成功标记，dispatch 据此触发入账
	})
	return channel.CreateResp{
		PayType: "scan", // V2 契约原生值，mapi 层透传
		RawHTML: string(payload),
	}
}

// sleepCtx 可被 ctx 取消的 sleep，返回 true 表示正常等待完成，false 表示 ctx 已取消。
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
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
