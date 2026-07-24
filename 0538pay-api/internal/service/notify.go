package service

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/pkg/money"
	"github.com/epvia/api/pkg/sign"
)

// notifyClient 商户异步通知用的 HTTP 客户端，带超时防止卡死。
var notifyClient = &http.Client{Timeout: 10 * time.Second}

// buildCallbackParams 构造回商户的回调参数（A-1，1:1 对齐 epay creat_callback）。
// 按订单 version 分派签名：
//   - version==1（V2）：平台私钥 RSA 签(sign_type=RSA) + timestamp，商户用平台公钥验。
//     缺平台私钥时降级 MD5（保证仍能通知，日志留痕由调用方处理）。
//   - version==0（V1）：商户 key MD5 签(sign_type=MD5)。
// notifyordername==1 时商品名强制为 'product'（对齐 epay functions.php:477/485）。
func (s *PayService) buildCallbackParams(o *model.Order, m *model.Merchant) map[string]string {
	name := o.Name
	if s.cfg != nil && s.cfg.Bool("notifyordername") {
		name = "product"
	}
	params := map[string]string{
		"pid":          uintToStr(o.UID),
		"trade_no":     o.TradeNo,
		"out_trade_no": o.OutTradeNo,
		"type":         o.TypeName,
		"name":         name,
		"money":        money.Float(o.Money), // (float)money 形态去尾零，对齐 epay creat_callback（G-5）
		"trade_status": "TRADE_SUCCESS",
	}
	// param 两版本回调通用（epay creat_callback:476/484 两分支都加）。
	if o.Param != "" {
		params["param"] = o.Param
	}

	// V2：平台私钥 RSA + timestamp（epay creat_callback version==1 分支）。
	// 关键：api_trade_no/buyer 只在 RSA(version==1) 分支参与回调与签名；MD5 分支两者都不带，
	// 否则 MD5 签名字段集与 epay 不一致致商户验签失败（B1-48）。
	if o.Version == 1 && s.cfg != nil {
		if priv := s.cfg.PlatformPrivateKey(); priv != "" {
			// api_trade_no：优先 bill_trade_no，退回 api_trade_no（对齐 epay creat_callback:473-474）。
			if apiTradeNo := firstNonEmpty(o.BillTradeNo, o.APITradeNo); apiTradeNo != "" {
				params["api_trade_no"] = apiTradeNo
			}
			if o.Buyer != "" {
				params["buyer"] = o.Buyer
			}
			params["timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
			params["sign_type"] = "RSA"
			if sig, err := sign.MakeRSA(params, priv); err == nil {
				params["sign"] = sig
				return params
			}
			// 签名失败降级 MD5：删掉 RSA 专属字段(timestamp)与 RSA 专属回调字段(api_trade_no/buyer)，
			// 回到 epay MD5 分支的字段集。
			delete(params, "timestamp")
			delete(params, "api_trade_no")
			delete(params, "buyer")
		}
	}

	// V1（或 V2 降级）：商户 key MD5。字段集只含 pid/trade_no/out_trade_no/type/name/money/trade_status/param。
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
