package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// 微信 JSAPI 收银台网页授权（对齐 epay wxpayn jsapiPay → \WeChatPay\JsApiTool）：
//
// JSAPI 支付要求下单时携带买家 openid，而 openid 只能在微信内置浏览器里通过公众号网页授权
// （snsapi_base）换取。epay 的做法：收银台在微信环境发现无 code → 302 跳授权 → 微信带 code
// 跳回 → 用【通道绑定公众号】的 appid/appsecret 调 sns/oauth2/access_token 换 openid → 再下单。
//
// 我方 SPA 收银台拆成两个后端接口 + 前端编排：
//   1. CashierWxAuthURL：据订单通道 appwxmp 取公众号 appid，拼授权跳转 URL（前端 redirect）。
//   2. CashierWxOpenID：微信带 code 跳回后，用 appid/appsecret 换 openid 返回前端。
//   前端拿到 openid 再调 ChoosePay(openid) 真正 JSAPI 下单。
//
// openid 用的是【通道绑定公众号】(channel.appwxmp → pay_weixin) 的 appid/appsecret，
// 与商户快捷登录的 login_appid 是两套凭证（对齐 epay getWeixin($channel['appwxmp'])）。

const wxCashierHTTPTimeout = 8 * time.Second

// resolveCashierWxApp 据订单号定位其支付通道绑定的公众号（appwxmp），返回 appid/appsecret。
// 无绑定/未注入 weixinRepo/非服务号时返回错误。
func (s *PayService) resolveCashierWxApp(tradeNo string) (appid, secret string, err error) {
	if s.weixins == nil {
		return "", "", payErr("系统未配置微信公众号，无法获取 openid")
	}
	o, e := s.orders.FindByTradeNo(tradeNo)
	if e != nil {
		return "", "", e
	}
	if o == nil {
		return "", "", payErr("订单不存在")
	}
	// 取订单所选通道（含子通道场景，appwxmp 挂主通道）。
	if o.Channel <= 0 || s.channels == nil {
		return "", "", payErr("订单未绑定支付通道，无法获取 openid")
	}
	ch, e := s.channels.FindByID(uint(o.Channel))
	if e != nil || ch == nil {
		return "", "", payErr("支付通道不存在，无法获取 openid")
	}
	if ch.AppWxMp <= 0 {
		return "", "", payErr("支付通道未绑定微信公众号，无法进行 JSAPI 支付")
	}
	wx, e := s.weixins.FindByID(uint(ch.AppWxMp))
	if e != nil || wx == nil {
		return "", "", payErr("支付通道绑定的微信公众号不存在")
	}
	if wx.AppID == "" || wx.AppSecret == "" {
		return "", "", payErr("微信公众号未配置 appid/appsecret")
	}
	return wx.AppID, wx.AppSecret, nil
}

// CashierWxAuthURL 生成微信公众号网页授权跳转 URL（snsapi_base，静默授权）。
// redirectURI 为微信授权后带 code 跳回的收银台地址（前端拼好，需 URL 编码前的原始值）。
func (s *PayService) CashierWxAuthURL(tradeNo, redirectURI string) (string, error) {
	if redirectURI == "" {
		return "", payErr("缺少授权回跳地址")
	}
	appid, _, err := s.resolveCashierWxApp(tradeNo)
	if err != nil {
		return "", err
	}
	q := url.Values{}
	q.Set("appid", appid)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "snsapi_base")
	q.Set("state", tradeNo)
	return "https://open.weixin.qq.com/connect/oauth2/authorize?" + q.Encode() + "#wechat_redirect", nil
}

// CashierWxOpenID 用微信授权 code 换取买家 openid（对齐 epay JsApiTool::GetOpenidFromMp）。
// 据订单通道公众号 appid/appsecret 调 sns/oauth2/access_token。
func (s *PayService) CashierWxOpenID(ctx context.Context, tradeNo, code string) (string, error) {
	if code == "" {
		return "", payErr("缺少微信授权 code")
	}
	appid, secret, err := s.resolveCashierWxApp(tradeNo)
	if err != nil {
		return "", err
	}
	q := url.Values{}
	q.Set("appid", appid)
	q.Set("secret", secret)
	q.Set("code", code)
	q.Set("grant_type", "authorization_code")
	reqURL := "https://api.weixin.qq.com/sns/oauth2/access_token?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: wxCashierHTTPTimeout}).Do(req)
	if err != nil {
		return "", fmt.Errorf("请求微信网页授权失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var r struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("微信授权应答解析失败: %w", err)
	}
	if r.OpenID == "" {
		return "", payErr(fmt.Sprintf("微信换取 openid 失败[%d]: %s", r.ErrCode, r.ErrMsg))
	}
	return r.OpenID, nil
}
