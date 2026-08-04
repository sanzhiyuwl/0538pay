package qixiang

import (
	"context"
	"testing"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/sign"
)

func testCfg() channel.Config {
	return channel.Config{
		AppID: "1003",
		Extra: map[string]string{
			"appkey": "fnv5Xf0BnV5n5bGzFf7V7Fvn9tVtzn9v",
			"appurl": "https://api.payqixiang.cn",
		},
	}
}

// TestNotifySuccess 七相回调：正确 MD5 签名 + TRADE_SUCCESS → 判成功，金额/单号解析正确。
func TestNotifySuccess(t *testing.T) {
	cfg := testCfg()
	key := "fnv5Xf0BnV5n5bGzFf7V7Fvn9tVtzn9v"
	params := map[string]string{
		"pid":          "1003",
		"trade_no":     "20160806151343349021",
		"out_trade_no": "2026073112345678",
		"type":         "alipay",
		"name":         "测试商品",
		"money":        "1.00",
		"trade_status": "TRADE_SUCCESS",
	}
	params["sign"] = sign.MakeMD5(params, key)
	params["sign_type"] = "MD5"

	nr, err := (Channel{}).Notify(context.Background(), cfg, params)
	if err != nil {
		t.Fatalf("验签应通过，却报错: %v", err)
	}
	if !nr.Success {
		t.Error("trade_status=TRADE_SUCCESS 应判成功")
	}
	if nr.Money.StringFixed(2) != "1.00" {
		t.Errorf("金额解析错误: %s", nr.Money.StringFixed(2))
	}
	if nr.ChannelNo != "20160806151343349021" {
		t.Errorf("七相系统单号应进 ChannelNo: %s", nr.ChannelNo)
	}
	if nr.TradeNo != "2026073112345678" {
		t.Errorf("商户订单号应进 TradeNo: %s", nr.TradeNo)
	}
	if nr.AckContent != "success" {
		t.Errorf("应答内容应为 success: %s", nr.AckContent)
	}
}

// TestNotifyNoName 回调无 name 字段时不参与签名（文档规则），验签仍应通过。
func TestNotifyNoName(t *testing.T) {
	cfg := testCfg()
	key := "fnv5Xf0BnV5n5bGzFf7V7Fvn9tVtzn9v"
	params := map[string]string{
		"pid":          "1003",
		"trade_no":     "20160806151343349021",
		"out_trade_no": "2026073112345678",
		"type":         "wxpay",
		"money":        "1.00",
		"trade_status": "TRADE_SUCCESS",
	}
	params["sign"] = sign.MakeMD5(params, key)

	nr, err := (Channel{}).Notify(context.Background(), cfg, params)
	if err != nil {
		t.Fatalf("无 name 回调验签应通过: %v", err)
	}
	if !nr.Success {
		t.Error("应判成功")
	}
}

// TestNotifyTampered 签名后篡改金额 → 验签失败。
func TestNotifyTampered(t *testing.T) {
	cfg := testCfg()
	key := "fnv5Xf0BnV5n5bGzFf7V7Fvn9tVtzn9v"
	params := map[string]string{
		"pid":          "1003",
		"out_trade_no": "2026073112345678",
		"money":        "1.00",
		"trade_status": "TRADE_SUCCESS",
	}
	params["sign"] = sign.MakeMD5(params, key)
	params["money"] = "9999.00" // 签名后篡改

	if _, err := (Channel{}).Notify(context.Background(), cfg, params); err == nil {
		t.Error("篡改金额后验签应失败")
	}
}

// TestNotifyNotPaid 验签通过但状态非成功 → Success=false。
func TestNotifyNotPaid(t *testing.T) {
	cfg := testCfg()
	key := "fnv5Xf0BnV5n5bGzFf7V7Fvn9tVtzn9v"
	params := map[string]string{
		"pid":          "1003",
		"out_trade_no": "2026073112345678",
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

// TestConfigMissing 缺 appid/appkey 时下单应报错。
func TestConfigMissing(t *testing.T) {
	if _, err := (Channel{}).Create(context.Background(), channel.Config{}, channel.CreateReq{TradeNo: "X"}); err == nil {
		t.Error("缺 appid/appkey 应报错")
	}
}

// TestTypeMapping 支付方式映射：仅 wxpay 直通，其它归 alipay。
func TestTypeMapping(t *testing.T) {
	if qixiangType("wxpay") != "wxpay" {
		t.Error("wxpay 应映射为 wxpay")
	}
	if qixiangType("alipay") != "alipay" {
		t.Error("alipay 应映射为 alipay")
	}
	if qixiangType("qqpay") != "alipay" {
		t.Error("七相不支持的方式应归 alipay")
	}
}

// TestDeviceMapping 设备类型映射对齐七相文档设备列表。
func TestDeviceMapping(t *testing.T) {
	cases := map[string]string{
		"wechat": "wechat",
		"pc":     "pc",
		"mobile": "mobile",
		"app":    "mobile",
		"":       "jump",
		"qq":     "jump",
	}
	for in, want := range cases {
		if got := qixiangDevice(channel.CreateReq{Device: in}); got != want {
			t.Errorf("device %q → %q，期望 %q", in, got, want)
		}
	}
}

// TestKey 渠道注册名。
func TestKey(t *testing.T) {
	if (Channel{}).Key() != "qixiang" {
		t.Errorf("渠道 key 应为 qixiang，实际 %s", (Channel{}).Key())
	}
}

// TestCapabilities 声明的可选能力：应实现 Refunder + Configurable。
func TestCapabilities(t *testing.T) {
	var c channel.PaymentChannel = Channel{}
	if _, ok := c.(channel.Refunder); !ok {
		t.Error("七相应实现 Refunder（支持原路退款）")
	}
	if _, ok := c.(channel.Configurable); !ok {
		t.Error("七相应实现 Configurable（元数据驱动后台表单）")
	}
}
