package fuiou2

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/epvia/api/pkg/fuiou"
	"github.com/shopspring/decimal"
)

const httpTimeout = 15 * time.Second

// execute 发起富友 API 请求：注入公共参数 → GBK+签名+XML+双重 urlencode → POST → urldecode → 解析 → 验签。
// 对齐 PayService::submit。
func (c fuiouConf) execute(ctx context.Context, path string, params map[string]string) (map[string]string, error) {
	// 公共参数（对齐 submit() public_params）。random_str 用 fuiou.NonceStr。
	full := map[string]string{
		"version":    version,
		"ins_cd":     c.insCd,
		"mchnt_cd":   c.mchntCd,
		"term_id":    termID,
		"random_str": randomStr(),
	}
	for k, v := range params {
		full[k] = v
	}
	body, err := fuiou.BuildRequestBody(full, c.privKey)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.gateway+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: httpTimeout}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("富友请求失败: %w", err)
	}
	defer resp.Body.Close()
	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// 应答为 urlencode 后的 XML（对齐 $response = urldecode($response)）。
	decoded, err := url.QueryUnescape(string(rawResp))
	if err != nil {
		decoded = string(rawResp) // 解码失败按原文尝试解析
	}
	result, err := fuiou.ParseResponseXML(decoded)
	if err != nil {
		return nil, fmt.Errorf("富友应答解析失败: %w (原文 %s)", err, string(rawResp))
	}
	if result["result_code"] != "000000" {
		if msg := result["result_msg"]; msg != "" {
			return nil, fmt.Errorf("富友返回失败: %s", msg)
		}
		return nil, fmt.Errorf("富友返回失败: result_code=%s", result["result_code"])
	}
	// 应答验签（对齐 verifyResponse）。
	if !fuiou.VerifyMap(result, c.pubKey) {
		return nil, fmt.Errorf("富友应答验签失败")
	}
	return result, nil
}

// fenStr 元→分字符串（对齐 epay strval($money*100)）。
func fenStr(yuan decimal.Decimal) string {
	return yuan.Mul(decimal.NewFromInt(100)).Round(0).String()
}

// fenToYuan 分字符串→元。
func fenToYuan(fen string) decimal.Decimal {
	d, err := decimal.NewFromString(strings.TrimSpace(fen))
	if err != nil {
		return decimal.Zero
	}
	return d.Div(decimal.NewFromInt(100))
}

// nowYmdHis 当前时间 YmdHis（对齐 date('YmdHis')）。
func nowYmdHis() string { return time.Now().Format("20060102150405") }

// randomStr 生成请求随机串（富友 random_str，取 32 位）。
func randomStr() string {
	s, err := fuiou.NonceStr(32)
	if err != nil {
		return "0000000000000000"
	}
	return s
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
