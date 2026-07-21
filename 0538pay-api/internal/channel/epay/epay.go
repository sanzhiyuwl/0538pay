// Package epay 提供「二次对接易支付 V1(MD5)」的免签渠道。
//
// 免签特征:不需要任何持牌资质,只要一个上游易支付的接口地址 + 商户号(pid) + 密钥(key),
// 即可作为上游的商户向其下单/收单。因本系统自身就实现了易支付 V1 上游(/api/pay/submit + MD5 验签),
// 该渠道可配置成"对接本站自己",从而无外部依赖地端到端真跑真实渠道下单。
//
// 签名逐字节复用 pkg/sign(与本站上游同源,ksort→k=v 拼接→+key→md5)。
// 通过 init() 自注册到 channel.registry,key = "epay"。
//
// config 字段(通道 config JSON):
//   appid  = 上游分配的商户号 pid
//   appkey = 上游商户密钥 key
//   appurl = 上游易支付接口地址(如 http://127.0.0.1:8080,末尾不带 /)
package epay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/pkg/sign"
	"github.com/shopspring/decimal"
)

// Channel 免签易支付 V1 渠道,实现 channel.PaymentChannel。
type Channel struct{}

func (Channel) Key() string { return "epay" }

var httpClient = &http.Client{Timeout: 15 * time.Second}

// upstream 从 config 解出上游三要素(pid/key/appurl)。
func upstream(cfg channel.Config) (pid, key, apiURL string, err error) {
	pid = cfg.AppID
	if pid == "" {
		pid = cfg.ExtraOr("pid", "")
	}
	key = cfg.ExtraOr("appkey", cfg.Key)
	apiURL = strings.TrimRight(cfg.ExtraOr("appurl", ""), "/")
	if pid == "" || key == "" || apiURL == "" {
		return "", "", "", errors.New("epay 渠道缺少 appid/appkey/appurl 配置")
	}
	return pid, key, apiURL, nil
}

// submitResp 上游 /api/pay/submit 的 JSON 响应(对齐本站 dto.SubmitResp,包在 code/data 里)。
type submitResp struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TradeNo string `json:"trade_no"`
		PayType string `json:"pay_type"`
		PayURL  string `json:"pay_url"`
		QRCode  string `json:"qrcode"`
		Money   string `json:"money"`
	} `json:"data"`
}

// Create 向上游易支付下单:MD5 签名 POST {appurl}/api/pay/submit,透传返回的收银台信息。
func (Channel) Create(ctx context.Context, cfg channel.Config, req channel.CreateReq) (channel.CreateResp, error) {
	pid, key, apiURL, err := upstream(cfg)
	if err != nil {
		return channel.CreateResp{}, err
	}

	// 上游用哪个支付方式(type)收单:优先 config uptype,其次订单 typename,默认 alipay。
	// 自测闭环可配 uptype=mock,让上游落到 mock 终端渠道,避免选到需真实凭证的渠道。
	upType := cfg.ExtraOr("uptype", epayType(req.Extra["typename"]))

	// 组装易支付 V1 下单参数(与本站上游 Submit 期望字段对齐)。
	params := map[string]string{
		"pid":          pid,
		"type":         upType,
		"out_trade_no": req.TradeNo,
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"name":         req.Subject,
		"money":        req.Money.StringFixed(2),
		"clientip":     req.ClientIP,
	}
	params["sign"] = sign.MakeMD5(params, key)
	params["sign_type"] = "MD5"

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+"/api/pay/submit",
		strings.NewReader(form.Encode()))
	if err != nil {
		return channel.CreateResp{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := httpClient.Do(httpReq)
	if err != nil {
		return channel.CreateResp{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var sr submitResp
	if err := json.Unmarshal(body, &sr); err != nil {
		return channel.CreateResp{}, errors.New("epay 上游返回解析失败: " + string(body))
	}
	if sr.Code != 0 {
		return channel.CreateResp{}, errors.New("epay 上游下单失败: " + sr.Msg)
	}

	pt := channel.PayTypeQRCode
	if sr.Data.PayType == string(channel.PayTypeRedirect) {
		pt = channel.PayTypeRedirect
	}
	return channel.CreateResp{
		PayType: pt,
		PayURL:  sr.Data.PayURL,
		QRCode:  sr.Data.QRCode,
	}, nil
}

// Query 主动查单:占位(以回调为准)。后续可对接上游 /api/pay/query。
func (Channel) Query(_ context.Context, _ channel.Config, _ string) (bool, error) {
	return false, nil
}

// Notify 解析上游回调:同一 key 重算 MD5 验签 + trade_status 判定。金额由上层二次校验。
// 上游回调参数(GET/表单):pid/trade_no/out_trade_no/type/name/money/trade_status/sign/sign_type。
func (Channel) Notify(_ context.Context, cfg channel.Config, raw map[string]string) (channel.NotifyResult, error) {
	_, key, _, err := upstream(cfg)
	if err != nil {
		return channel.NotifyResult{}, err
	}

	// 剔除保留注入键(表单型回调用不到),仅用业务参数验签。
	params := map[string]string{}
	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		params[k] = v
	}
	if !sign.VerifyMD5(params, key) {
		return channel.NotifyResult{}, errors.New("epay 回调验签失败")
	}

	money, _ := decimal.NewFromString(params["money"])
	// 回调用商户订单号(out_trade_no)定位本地订单;上游系统单号进 ChannelNo。
	tradeNo := params["out_trade_no"]
	return channel.NotifyResult{
		TradeNo:    tradeNo,
		ChannelNo:  params["trade_no"],
		Money:      money,
		Success:    params["trade_status"] == "TRADE_SUCCESS",
		AckContent: "success",
	}, nil
}

// epayType 把内部支付方式英文名映射为易支付 type(对齐上游期望;缺省 alipay)。
func epayType(typename string) string {
	switch typename {
	case "wxpay", "alipay", "qqpay", "bank", "jdpay":
		return typename
	default:
		return "alipay"
	}
}

func init() {
	channel.Register(Channel{})
}
