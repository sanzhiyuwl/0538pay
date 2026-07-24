package alipaybase

import (
	"encoding/json"
	"testing"

	"github.com/epvia/api/internal/channel"
	"github.com/shopspring/decimal"
)

// TestBuildSysParams 系统参数含固定字段与 method/biz_content。
func TestBuildSysParams(t *testing.T) {
	cfg := channel.Config{AppID: "2021000000000000"}
	p := BuildSysParams(cfg, "alipay.trade.page.pay", `{"out_trade_no":"T1"}`, "https://x/notify", "https://x/return")
	if p["app_id"] != "2021000000000000" || p["method"] != "alipay.trade.page.pay" {
		t.Fatalf("app_id/method 不对: %+v", p)
	}
	if p["sign_type"] != "RSA2" || p["charset"] != "utf-8" || p["version"] != "1.0" {
		t.Fatalf("固定系统参数不对: %+v", p)
	}
	if p["notify_url"] != "https://x/notify" || p["return_url"] != "https://x/return" {
		t.Fatalf("notify/return 未带上: %+v", p)
	}
}

// TestBuildSysParamsSkipEmptyURL 空 notify/return 不带该键。
func TestBuildSysParamsSkipEmptyURL(t *testing.T) {
	p := BuildSysParams(channel.Config{AppID: "a"}, "alipay.trade.query", "{}", "", "")
	if _, ok := p["notify_url"]; ok {
		t.Fatal("空 notify_url 不应出现")
	}
	if _, ok := p["return_url"]; ok {
		t.Fatal("空 return_url 不应出现")
	}
}

// TestParseRespSuccess code=10000 返回原始 raw。
func TestParseRespSuccess(t *testing.T) {
	body := []byte(`{"alipay_trade_refund_response":{"code":"10000","msg":"Success","trade_no":"TN1","refund_fee":"9.90"},"sign":"x"}`)
	raw, err := ParseResp(body, "alipay_trade_refund_response")
	if err != nil {
		t.Fatalf("应成功: %v", err)
	}
	var r struct {
		RefundFee string `json:"refund_fee"`
	}
	_ = json.Unmarshal(raw, &r)
	if r.RefundFee != "9.90" {
		t.Fatalf("refund_fee 解析错: %s", r.RefundFee)
	}
}

// TestParseRespFail code!=10000 报错。
func TestParseRespFail(t *testing.T) {
	body := []byte(`{"alipay_trade_refund_response":{"code":"40004","msg":"Business Failed","sub_code":"ACQ.TRADE_NOT_EXIST"}}`)
	if _, err := ParseResp(body, "alipay_trade_refund_response"); err == nil {
		t.Fatal("code=40004 应报错")
	}
}

// TestParseNotifySuccess trade_status=TRADE_SUCCESS 判成功（未配公钥跳过验签）。
func TestParseNotifySuccess(t *testing.T) {
	raw := map[string]string{
		"out_trade_no": "T100",
		"trade_no":     "2024xxx",
		"trade_status": "TRADE_SUCCESS",
		"total_amount": "12.34",
		"_raw_body":    "should-be-ignored",
	}
	res, err := ParseNotify(channel.Config{}, raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !res.Success || res.TradeNo != "T100" || res.ChannelNo != "2024xxx" {
		t.Fatalf("结果不对: %+v", res)
	}
	if !res.Money.Equal(decimal.RequireFromString("12.34")) {
		t.Fatalf("金额不对: %s", res.Money)
	}
	if res.AckContent != "success" {
		t.Fatalf("应答应为 success: %s", res.AckContent)
	}
}
