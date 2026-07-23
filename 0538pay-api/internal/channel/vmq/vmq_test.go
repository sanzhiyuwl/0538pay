package vmq

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/0538pay/api/internal/channel"
	"github.com/shopspring/decimal"
)

func cfg() channel.Config {
	return channel.Config{
		AppID: "10001",
		Extra: map[string]string{"appurl": "http://127.0.0.1:8081", "appkey": "SECRETKEY"},
	}
}

func md5s(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestVmqType(t *testing.T) {
	cases := map[string]string{"wxpay": "1", "alipay": "2", "bank": "3", "qqpay": "4", "": "2", "unknown": "2"}
	for in, want := range cases {
		if got := vmqType(in); got != want {
			t.Errorf("vmqType(%q)=%q want %q", in, got, want)
		}
	}
}

func TestCreateHTML(t *testing.T) {
	c := Channel{}
	req := channel.CreateReq{
		TradeNo:   "20260723A1",
		Money:     decimal.RequireFromString("12.34"),
		NotifyURL: "http://site/pay/notify/20260723A1",
		ReturnURL: "http://site/return",
		Extra:     map[string]string{"typename": "wxpay"},
	}
	resp, err := c.Create(context.Background(), cfg(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.PayType != channel.PayTypeHTML {
		t.Fatalf("PayType=%q want html", resp.PayType)
	}
	// 签名 = md5(payId + type + price + appkey)。wxpay→type=1，price=12.34。
	wantSign := md5s("20260723A1" + "1" + "12.34" + "SECRETKEY")
	if !strings.Contains(resp.RawHTML, wantSign) {
		t.Errorf("RawHTML 缺正确签名 %s\n%s", wantSign, resp.RawHTML)
	}
	// 表单 action 指向 {appurl}/createOrder（appurl 自动补斜杠）。
	if !strings.Contains(resp.RawHTML, "http://127.0.0.1:8081/createOrder") {
		t.Errorf("RawHTML action 不对:\n%s", resp.RawHTML)
	}
	if !strings.Contains(resp.RawHTML, `name="payId" value="20260723A1"`) {
		t.Errorf("RawHTML 缺 payId 字段:\n%s", resp.RawHTML)
	}
}

func TestNotifyOK(t *testing.T) {
	c := Channel{}
	// sign = md5(payId + type + price + reallyPrice + appkey)。
	raw := map[string]string{
		"payId": "20260723A1", "type": "1", "price": "12.34", "reallyPrice": "12.34",
	}
	raw["sign"] = md5s(raw["payId"] + raw["type"] + raw["price"] + raw["reallyPrice"] + "SECRETKEY")
	res, err := c.Notify(context.Background(), cfg(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.TradeNo != "20260723A1" || res.AckContent != "success" {
		t.Errorf("notify 结果异常: %+v", res)
	}
	if !res.Money.Equal(decimal.RequireFromString("12.34")) {
		t.Errorf("Money=%s want 12.34", res.Money)
	}
}

func TestNotifyBadSign(t *testing.T) {
	c := Channel{}
	raw := map[string]string{"payId": "x", "type": "1", "price": "1.00", "reallyPrice": "1.00", "sign": "deadbeef"}
	if _, err := c.Notify(context.Background(), cfg(), raw); err == nil {
		t.Error("篡改签名应验签失败")
	}
}

func TestConfMissing(t *testing.T) {
	if _, err := (Channel{}).Create(context.Background(), channel.Config{}, channel.CreateReq{}); err == nil {
		t.Error("缺配置应报错")
	}
}
