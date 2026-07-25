package fuiou2

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/pkg/fuiou"
)

func genKeys(t *testing.T) (priv, pub string) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privDER, _ := x509.MarshalPKCS8PrivateKey(k)
	pubDER, _ := x509.MarshalPKIXPublicKey(&k.PublicKey)
	return base64.StdEncoding.EncodeToString(privDER), base64.StdEncoding.EncodeToString(pubDER)
}

func TestKeyRegistered(t *testing.T) {
	if _, ok := channel.Get("fuiou2"); !ok {
		t.Fatal("fuiou2 未注册到 registry")
	}
}

func TestOrderType(t *testing.T) {
	cases := map[string]string{"alipay": "ALIPAY", "wxpay": "WECHAT", "bank": "UNIONPAY"}
	for in, want := range cases {
		got, err := orderType(in)
		if err != nil || got != want {
			t.Fatalf("orderType(%s)=%s,%v 期望 %s", in, got, err, want)
		}
	}
	if _, err := orderType("qqpay"); err == nil {
		t.Fatal("不支持的支付方式应报错")
	}
}

func TestResolveMissing(t *testing.T) {
	// 缺商户号
	if _, err := resolve(channel.Config{AppID: "ins"}); err == nil {
		t.Fatal("缺商户号/私钥应报错")
	}
}

func TestResolveEnv(t *testing.T) {
	priv, _ := genKeys(t)
	c, err := resolve(channel.Config{
		AppID: "ins", Extra: map[string]string{"appmchid": "m", "appsecret": priv, "appswitch": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.gateway != gatewayTest {
		t.Fatalf("appswitch=1 应用测试网关, 实际 %s", c.gateway)
	}
}

// TestNotifyEndToEnd 自造一份富友回调（GBK+RSA-MD5 签名+双重 urlencode），Notify 应验签通过并解出字段。
func TestNotifyEndToEnd(t *testing.T) {
	priv, pub := genKeys(t)
	cfg := channel.Config{
		AppID: "08M0",
		Extra: map[string]string{"appmchid": "0002", "appsecret": priv, "appkey": pub, "appurl": "PX"},
	}
	// 富友回调字段（模拟平台侧：值 UTF-8→GBK 后签名）。
	data := map[string]string{
		"result_code":    "000000",
		"mchnt_order_no": "PXT20260725", // 前缀 PX + 系统单号 T20260725
		"transaction_id": "fy-txn-9",
		"order_amt":      "10000",
		"user_id":        "buyer-openid",
	}
	gbk := map[string]string{}
	for k, v := range data {
		gbk[k] = fuiou.ToGBK(v)
	}
	sig, err := fuiou.Sign(fuiou.SignContent(gbk), priv)
	if err != nil {
		t.Fatal(err)
	}
	data["sign"] = sig
	// 组 XML（平铺）→ 模拟 handler 已 ParseForm 解一层：raw["req"] = urlencode(xml)。
	xmlBody := "<?xml version=\"1.0\" encoding=\"GBK\" standalone=\"yes\"?><xml>"
	for k, v := range data {
		xmlBody += "<" + k + ">" + v + "</" + k + ">"
	}
	xmlBody += "</xml>"
	raw := map[string]string{"req": url.QueryEscape(xmlBody)}

	res, err := (Channel{}).Notify(context.Background(), cfg, raw)
	if err != nil {
		t.Fatalf("回调解析失败: %v", err)
	}
	if !res.Success {
		t.Fatal("result_code=000000 应判成功")
	}
	if res.TradeNo != "T20260725" {
		t.Fatalf("订单号前缀应被剥离, 实际 %s", res.TradeNo)
	}
	if res.ChannelNo != "fy-txn-9" {
		t.Fatalf("transaction_id 不对: %s", res.ChannelNo)
	}
	if res.AckContent != "1" {
		t.Fatalf("富友回调应答应为 '1', 实际 %s", res.AckContent)
	}
	if !res.Money.Equal(fenToYuan("10000")) {
		t.Fatalf("金额应为 100 元, 实际 %s", res.Money)
	}

	// 篡改签名后应验签失败
	data["order_amt"] = "1"
	bad := "<xml>"
	for k, v := range data {
		bad += "<" + k + ">" + v + "</" + k + ">"
	}
	bad += "</xml>"
	if _, err := (Channel{}).Notify(context.Background(), cfg, map[string]string{"req": url.QueryEscape(bad)}); err == nil {
		t.Fatal("篡改后回调应验签失败")
	}
}
