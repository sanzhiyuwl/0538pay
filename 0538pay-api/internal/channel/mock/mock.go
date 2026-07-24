// Package mock 提供一个自建的模拟支付渠道，用于阶段 A 跑通收单全链路，
// 不依赖任何第三方资质。行为：下单即返回一个「模拟收银台」二维码链接，
// 该链接指向本系统的模拟支付页，扫码/点击后可手动触发回调。
//
// 通过 init() 自注册到 channel.registry，key = "mock"。
package mock

import (
	"context"

	"github.com/epvia/api/internal/channel"
	"github.com/shopspring/decimal"
)

// Channel 模拟渠道实现 channel.PaymentChannel。
type Channel struct{}

func (Channel) Key() string { return "mock" }

// Create 模拟下单：返回二维码类型，PayURL 指向本系统模拟收银台。
// 真实渠道会调用第三方下单 API，这里直接构造一个可跳转的支付中间页地址。
func (Channel) Create(_ context.Context, _ channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	// 模拟收银台：前端/中间页据 trade_no 展示二维码，扫码后触发 mock 回调。
	payURL := "/pay/mock/cashier/" + req.TradeNo
	return channel.CreateResp{
		PayType: channel.PayTypeQRCode,
		PayURL:  payURL,
		QRCode:  payURL, // 模拟：二维码内容就是收银台地址
	}, nil
}

// Query 模拟查单：mock 渠道不主动查，恒返回未支付（真实支付以回调为准）。
func (Channel) Query(_ context.Context, _ channel.Config, _ string) (bool, error) {
	return false, nil
}

// Notify 解析模拟回调。约定回调参数：trade_no + money + trade_status(TRADE_SUCCESS)。
func (Channel) Notify(_ context.Context, _ channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	m, _ := decimal.NewFromString(raw["money"]) // 解析失败按 0，交由上层金额二次校验拦截
	return channel.NotifyResult{
		TradeNo:    raw["trade_no"],
		ChannelNo:  raw["channel_no"],
		Money:      m,
		Success:    raw["trade_status"] == "TRADE_SUCCESS",
		AckContent: "success",
	}, nil
}

func init() {
	channel.Register(Channel{})
}
