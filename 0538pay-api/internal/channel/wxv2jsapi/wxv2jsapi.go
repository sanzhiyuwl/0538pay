// Package wxv2jsapi 实现微信支付 APIv2「JSAPI/公众号支付」渠道（对齐 epay wxpay jspay）。
//
// 复用 wxv2base（公共参数+签名+XML+验应答+回调+退款），本包声明 trade_type=JSAPI 拿 prepay_id，
// 再用 APIv2 密钥对 getJsApiParameters 做 MD5 二次签名（paySign），组前端 wx.chooseWXPay 拉起参数。
// openid 由收银台 OAuth 换得（req.SubOpenID）。直连/服务商由 config sub_mchid 自适应。key = "wxv2jsapi"。
package wxv2jsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
	"github.com/epvia/api/pkg/wxpayv2"
	"github.com/epvia/api/pkg/wxpayv3"
)

type Channel struct{}

func (Channel) Key() string { return "wxv2jsapi" }

func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	openid := req.SubOpenID
	if openid == "" && req.Extra != nil {
		openid = req.Extra["openid"]
	}
	if openid == "" {
		return channel.CreateResp{}, fmt.Errorf("JSAPI 支付缺少用户 openid")
	}
	params, err := wxv2base.BaseOrderParams(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	params["trade_type"] = "JSAPI"
	params["openid"] = openid
	result, err := wxv2base.UnifiedOrder(ctx, cfg, params)
	if err != nil {
		return channel.CreateResp{}, err
	}
	prepayID := result["prepay_id"]
	if prepayID == "" {
		return channel.CreateResp{}, fmt.Errorf("微信 V2 JSAPI 下单未返回 prepay_id")
	}
	jsParams, err := buildJsApiParameters(cfg, prepayID)
	if err != nil {
		return channel.CreateResp{}, err
	}
	infoJSON, _ := json.Marshal(jsParams)
	return channel.CreateResp{
		PayType: channel.PayTypeWap,
		RawHTML: string(infoJSON), // 前端解析后调 WeixinJSBridge/wx.chooseWXPay 拉起
	}, nil
}

// buildJsApiParameters 组前端拉起参数并 MD5 二次签名 paySign（对齐 PaymentService::getJsApiParameters）。
// 注意 V2 JSAPI 拉起用 appId（大写 I）作为公众号 appid，其余 timeStamp/nonceStr/package/signType 同 V3 结构但签名走 MD5。
func buildJsApiParameters(cfg channel.Config, prepayID string) (map[string]string, error) {
	nonce, err := wxpayv2.NonceStr(32)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"appId":     cfg.AppID,
		"timeStamp": strconv.FormatInt(wxpayv3.NowUnix(), 10),
		"nonceStr":  nonce,
		"package":   "prepay_id=" + prepayID,
		"signType":  wxpayv2.SignTypeMD5,
	}
	params["paySign"] = wxpayv2.MakeSign(params, cfg.Key)
	return params, nil
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
	return []channel.ProductType{{Code: "jsapi", Name: "JSAPI/公众号支付"}}
}

func init() { channel.Register(Channel{}) }
