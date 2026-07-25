// Package wxnative 实现微信支付 APIv3「Native 支付」渠道（PC 扫码）。
//
// 全部下单/查单/回调/退款复用 wxbase 共享包（签名+验应答+回调验签解密+服务商模式自适应），
// 本包仅声明 Native 特有的下单路径("native")与应答解析(code_url)。
//   - 下单：直连 /v3/pay/transactions/native，服务商 /v3/pay/partner/transactions/native
//     （wxbase 据 config sub_mchid 判定），应答含 code_url，前端据此展示二维码。
//   - 回调/退款/查单：wxbase.ParseNotify / Refund / QueryPaid。
//
// 通过 init() 自注册到 channel.registry，key = "wxnative"。
package wxnative

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxbase"
)

const pluginKey = "wxnative"

// Channel 微信 Native 渠道实现 channel.PaymentChannel。
type Channel struct{}

func (Channel) Key() string { return pluginKey }

type prepayResp struct {
	CodeURL string `json:"code_url"`
}

// Create Native 下单：复用 wxbase 统一下单（签名+验应答+服务商模式），取 code_url。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	body, err := wxbase.BasePrepayBody(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	respBody, err := wxbase.Prepay(ctx, cfg, "native", body)
	if err != nil {
		return channel.CreateResp{}, err
	}
	var pr prepayResp
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return channel.CreateResp{}, err
	}
	if pr.CodeURL == "" {
		return channel.CreateResp{}, fmt.Errorf("微信下单未返回 code_url")
	}
	return channel.CreateResp{
		PayType: channel.PayTypeQRCode,
		QRCode:  pr.CodeURL, // 前端把该链接渲染成二维码
		PayURL:  pr.CodeURL,
	}, nil
}

// Query 主动查单：复用 wxbase.QueryPaid（直连/服务商路径自适应）。
func (Channel) Query(ctx context.Context, cfg channel.Config, tradeNo string) (bool, error) {
	return wxbase.QueryPaid(ctx, cfg, tradeNo)
}

// Notify 验签 + 解密回调：复用 wxbase.ParseNotify。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	return wxbase.ParseNotify(cfg, raw)
}

// Refund 原路退款：复用 wxbase.Refund（服务商模式带 sub_mchid）。
func (Channel) Refund(ctx context.Context, cfg channel.Config, req channel.RefundReq) (channel.RefundResp, error) {
	return wxbase.Refund(ctx, cfg, req)
}

// Transfer 商家转账到零钱（实现 channel.Transferer，复用 wxbase.MchTransfer）。
// Account 为收款用户 openid，代付主链据此把资金打到用户微信零钱（对齐 epay wxpay transfer）。
func (Channel) Transfer(ctx context.Context, cfg channel.Config, req channel.TransferReq) (channel.TransferResp, error) {
	return wxbase.MchTransfer(ctx, cfg, req)
}

// TransferQuery 查询转账单状态（对齐 epay transfer_query）。
func (Channel) TransferQuery(ctx context.Context, cfg channel.Config, outBizNo string) (channel.TransferResp, error) {
	return wxbase.MchTransferQuery(ctx, cfg, outBizNo)
}

// Inputs 声明配置字段（复用微信共用字段，元数据驱动后台密钥表单）。
func (Channel) Inputs() []channel.FieldInput { return wxbase.Inputs() }

// Products 声明本渠道支持的支付产品形态（对齐 epay $info['select']）。
func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "native", Name: "Native 扫码支付"}}
}

func init() {
	channel.Register(Channel{})
}
