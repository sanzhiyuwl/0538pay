package service

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/pkg/money"
	"github.com/0538pay/api/pkg/sign"
)

// notifyClient 商户异步通知用的 HTTP 客户端，带超时防止卡死。
var notifyClient = &http.Client{Timeout: 10 * time.Second}

// buildCallbackParams 构造回商户的回调参数（对齐 epay creat_callback V1/MD5 分支）。
// 返回已含 sign/sign_type 的完整参数表。key 为商户密钥。
func buildCallbackParams(o *model.Order, m *model.Merchant) map[string]string {
	params := map[string]string{
		"pid":          uintToStr(o.UID),
		"trade_no":     o.TradeNo,
		"out_trade_no": o.OutTradeNo,
		"type":         o.TypeName,
		"name":         o.Name,
		"money":        money.String(o.Money),
		"trade_status": "TRADE_SUCCESS",
	}
	if o.APITradeNo != "" {
		params["api_trade_no"] = o.APITradeNo
	}
	if o.Buyer != "" {
		params["buyer"] = o.Buyer
	}
	if o.Param != "" {
		params["param"] = o.Param
	}
	params["sign"] = sign.MakeMD5(params, m.AppKey)
	params["sign_type"] = "MD5"
	return params
}

// appendQuery 把参数拼到 notify_url 上（对齐 epay：有 ? 用 &，否则 ?）。
func appendQuery(rawURL string, params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + q.Encode()
}

// doNotify GET 商户回调地址，返回体含 success(不分大小写) 视为通知成功。
// 对齐 epay do_notify。
func doNotify(ctx context.Context, callbackURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, callbackURL, nil)
	if err != nil {
		return false
	}
	resp, err := notifyClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(body)), "success")
}
