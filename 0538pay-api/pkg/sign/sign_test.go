package sign

import "testing"

// 基准数据由 epay php8.2 的 getSignContent+md5 现场计算（见提交说明），
// 这里断言 Go 实现逐字节对齐。密钥 = testkey_uid1_abcdef。
var baseParams = map[string]string{
	"pid":          "1",
	"type":         "alipay",
	"out_trade_no": "TEST20260721001",
	"notify_url":   "http://shop-a.com/notify",
	"return_url":   "http://shop-a.com/return",
	"name":         "VIP",
	"money":        "1.00",
	"sign_type":    "MD5",
	"empty1":       "",    // 应被跳过
	"empty2":       "   ", // trim 后为空，应被跳过
}

const testKey = "testkey_uid1_abcdef"

func TestContent(t *testing.T) {
	want := "money=1.00&name=VIP&notify_url=http://shop-a.com/notify&out_trade_no=TEST20260721001&pid=1&return_url=http://shop-a.com/return&type=alipay"
	if got := Content(baseParams); got != want {
		t.Fatalf("Content 不匹配\n got=%q\nwant=%q", got, want)
	}
}

func TestMakeMD5(t *testing.T) {
	want := "119872168c9c25e06b30ff8d0b580614" // epay php 现场计算
	if got := MakeMD5(baseParams, testKey); got != want {
		t.Fatalf("MakeMD5 不匹配\n got=%s\nwant=%s", got, want)
	}
}

func TestVerifyMD5(t *testing.T) {
	p := map[string]string{}
	for k, v := range baseParams {
		p[k] = v
	}
	p["sign"] = MakeMD5(baseParams, testKey)

	if !VerifyMD5(p, testKey) {
		t.Fatal("VerifyMD5 应通过")
	}
	// 大写签名也应通过
	p["sign"] = "119872168C9C25E06B30FF8D0B580614"
	if !VerifyMD5(p, testKey) {
		t.Fatal("VerifyMD5 大写十六进制应通过")
	}
	// 错误密钥应失败
	if VerifyMD5(p, "wrongkey") {
		t.Fatal("VerifyMD5 错误密钥应失败")
	}
	// 缺 sign 应失败
	delete(p, "sign")
	if VerifyMD5(p, testKey) {
		t.Fatal("VerifyMD5 缺签名应失败")
	}
}
