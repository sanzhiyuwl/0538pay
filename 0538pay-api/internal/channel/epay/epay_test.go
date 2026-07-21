package epay

import (
	"context"
	"testing"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/pkg/sign"
)

func testCfg() channel.Config {
	return channel.Config{
		AppID: "1",
		Extra: map[string]string{
			"appkey": "testkey_uid1_abcdef",
			"appurl": "http://127.0.0.1:8080",
		},
	}
}

// TestNotifySuccess 上游回调:正确 MD5 签名 + TRADE_SUCCESS → 判成功,金额解析正确。
func TestNotifySuccess(t *testing.T) {
	cfg := testCfg()
	key := "testkey_uid1_abcdef"
	params := map[string]string{
		"pid":          "1",
		"trade_no":     "UP20260721001",
		"out_trade_no": "2026072112345678",
		"type":         "alipay",
		"name":         "测试商品",
		"money":        "1.00",
		"trade_status": "TRADE_SUCCESS",
	}
	params["sign"] = sign.MakeMD5(params, key)
	params["sign_type"] = "MD5"

	nr, err := (Channel{}).Notify(context.Background(), cfg, params)
	if err != nil {
		t.Fatalf("验签应通过,却报错: %v", err)
	}
	if !nr.Success {
		t.Error("trade_status=TRADE_SUCCESS 应判成功")
	}
	if nr.Money.StringFixed(2) != "1.00" {
		t.Errorf("金额解析错误: %s", nr.Money.StringFixed(2))
	}
	if nr.ChannelNo != "UP20260721001" {
		t.Errorf("上游系统单号应进 ChannelNo: %s", nr.ChannelNo)
	}
}

// TestNotifyTampered 篡改金额后签名失配 → 验签失败。
func TestNotifyTampered(t *testing.T) {
	cfg := testCfg()
	key := "testkey_uid1_abcdef"
	params := map[string]string{
		"pid":          "1",
		"out_trade_no": "2026072112345678",
		"money":        "1.00",
		"trade_status": "TRADE_SUCCESS",
	}
	params["sign"] = sign.MakeMD5(params, key)
	// 签名后篡改金额
	params["money"] = "9999.00"

	_, err := (Channel{}).Notify(context.Background(), cfg, params)
	if err == nil {
		t.Error("篡改金额后验签应失败")
	}
}

// TestNotifyNotPaid 验签通过但 trade_status 非成功 → Success=false。
func TestNotifyNotPaid(t *testing.T) {
	cfg := testCfg()
	key := "testkey_uid1_abcdef"
	params := map[string]string{
		"pid":          "1",
		"out_trade_no": "2026072112345678",
		"money":        "1.00",
		"trade_status": "WAIT_BUYER_PAY",
	}
	params["sign"] = sign.MakeMD5(params, key)

	nr, err := (Channel{}).Notify(context.Background(), cfg, params)
	if err != nil {
		t.Fatalf("验签应通过: %v", err)
	}
	if nr.Success {
		t.Error("trade_status 非 TRADE_SUCCESS 不应判成功")
	}
}

// TestUpstreamConfigMissing 缺配置时下单/回调应报错。
func TestUpstreamConfigMissing(t *testing.T) {
	_, err := (Channel{}).Create(context.Background(), channel.Config{}, channel.CreateReq{TradeNo: "X"})
	if err == nil {
		t.Error("缺 appid/appkey/appurl 应报错")
	}
}

// TestKey 渠道 key 注册名。
func TestKey(t *testing.T) {
	if (Channel{}).Key() != "epay" {
		t.Errorf("渠道 key 应为 epay, 实际 %s", (Channel{}).Key())
	}
}
