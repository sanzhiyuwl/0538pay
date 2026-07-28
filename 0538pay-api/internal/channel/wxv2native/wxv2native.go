// Package wxv2native 实现微信支付 APIv2「Native 扫码支付」渠道（PC 扫码，对齐 epay wxpay qrcode）。
//
// 复用 wxv2base 共享包（公共参数+签名+XML+验应答+回调+退款+mTLS），本包仅声明 trade_type=NATIVE
// 与应答解析（code_url）。直连/服务商由 config sub_mchid 自适应。key = "wxv2native"。
package wxv2native

import (
	"context"
	"fmt"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
)

const pluginKey = "wxv2native"

type Channel struct{}

func (Channel) Key() string { return pluginKey }

// Create Native 下单：trade_type=NATIVE + product_id，取 code_url 渲染二维码（对齐 nativePay）。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	params, err := wxv2base.BaseOrderParams(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	params["trade_type"] = "NATIVE"
	params["product_id"] = "01001"
	result, err := wxv2base.UnifiedOrder(ctx, cfg, params)
	if err != nil {
		return channel.CreateResp{}, err
	}
	codeURL := result["code_url"]
	if codeURL == "" {
		return channel.CreateResp{}, fmt.Errorf("微信 V2 Native 下单未返回 code_url")
	}
	return channel.CreateResp{
		PayType: channel.PayTypeQRCode,
		QRCode:  codeURL,
		PayURL:  codeURL,
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

// Transfer 企业付款（实现 channel.Transferer，对齐 epay wxpay transfer；与 V3 wxnative 对称）。
// Extra["type"]=="bank" 走到银行卡（Extra["bank_code"]/["bank_abbr"] 由主链据卡号解析后传入）；
// 否则到零钱（Account=收款用户 openid）。真实打款待商户证书凭证。
func (Channel) Transfer(ctx context.Context, cfg channel.Config, req channel.TransferReq) (channel.TransferResp, error) {
	if req.Extra != nil && req.Extra["type"] == "bank" {
		bankCode := req.Extra["bank_code"]
		if bankCode == "" {
			bankCode = wxv2base.BankCode(req.Extra["bank_abbr"])
		}
		return wxv2base.TransferToBank(ctx, cfg, req, bankCode)
	}
	return wxv2base.Transfer(ctx, cfg, req)
}

// TransferQuery 查询企业付款状态（对齐 epay wxpay transfer_query）。
func (Channel) TransferQuery(ctx context.Context, cfg channel.Config, outBizNo string) (channel.TransferResp, error) {
	return wxv2base.TransferQuery(ctx, cfg, outBizNo)
}

func (Channel) Inputs() []channel.FieldInput { return wxv2base.Inputs() }

func (Channel) Products() []channel.ProductType {
	return []channel.ProductType{{Code: "native", Name: "Native 扫码支付"}}
}

func init() { channel.Register(Channel{}) }
