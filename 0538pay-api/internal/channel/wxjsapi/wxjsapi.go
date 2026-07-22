// Package wxjsapi 实现微信支付 APIv3「JSAPI/小程序支付」渠道。
//
// 对齐 epay wxpayn jsapiPay：POST /v3/pay/transactions/jsapi 拿 prepay_id，
// 再用商户私钥对 appId\ntimeStamp\nnonceStr\nprepay_id={id}\n 二次 RSA 签名，
// 组 getJsApiParameters（前端 wx.chooseWXPay / wx.requestPayment 拉起）。
// openid 由下单请求的 Extra["openid"] 传入（收银台 OAuth 拿到）。key = "wxjsapi"。
package wxjsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/channel/wxbase"
	"github.com/0538pay/api/pkg/wxpayv3"
)

type Channel struct{}

func (Channel) Key() string { return "wxjsapi" }

func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	openid := ""
	if req.Extra != nil {
		openid = req.Extra["openid"]
	}
	if openid == "" {
		return channel.CreateResp{}, fmt.Errorf("JSAPI 支付缺少用户 openid")
	}
	body, err := wxbase.BasePrepayBody(cfg, req)
	if err != nil {
		return channel.CreateResp{}, err
	}
	body["payer"] = map[string]string{"openid": openid}

	respBody, err := wxbase.Prepay(ctx, cfg, "jsapi", body)
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
		return channel.CreateResp{}, fmt.Errorf("微信 JSAPI 下单未返回 prepay_id")
	}
	// 组装前端拉起参数（paySign 为 prepay_id 的二次 RSA 签名）。
	jsParams, err := buildJsApiParams(cfg, pr.PrepayID)
	if err != nil {
		return channel.CreateResp{}, err
	}
	// pay_info 以 JSON 串放 RawHTML 字段透传给前端（前端解析后调 wx.requestPayment）。
	infoJSON, _ := json.Marshal(jsParams)
	return channel.CreateResp{
		PayType: channel.PayTypeWap,
		RawHTML: string(infoJSON),
	}, nil
}

// buildJsApiParams 生成前端拉起参数：appId/timeStamp/nonceStr/package/signType/paySign。
func buildJsApiParams(cfg channel.Config, prepayID string) (map[string]string, error) {
	priv, err := wxpayv3.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("解析商户私钥失败: %w", err)
	}
	ts := strconv.FormatInt(wxpayv3.NowUnix(), 10)
	nonce, err := wxpayv3.NonceStr(32)
	if err != nil {
		return nil, err
	}
	pkg := "prepay_id=" + prepayID
	// 签名串：appId\ntimeStamp\nnonceStr\npackage\n
	message := cfg.AppID + "\n" + ts + "\n" + nonce + "\n" + pkg + "\n"
	paySign, err := wxpayv3.SignMessage(priv, message)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"appId":     cfg.AppID,
		"timeStamp": ts,
		"nonceStr":  nonce,
		"package":   pkg,
		"signType":  "RSA",
		"paySign":   paySign,
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

func init() { channel.Register(Channel{}) }
