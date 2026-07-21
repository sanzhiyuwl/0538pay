package alipay

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func genKeys(t *testing.T) (privPEM, pubPEM string, priv *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}))
	pkix, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}))
	return
}

// TestSignContent 对齐 AopClient::getSignContent：ksort、跳过空值/sign、&k=v 拼接。
func TestSignContent(t *testing.T) {
	params := map[string]string{
		"app_id":    "2021000000",
		"method":    "alipay.trade.precreate",
		"charset":   "UTF-8",
		"sign_type": "RSA2",
		"empty":     "",   // 空值应跳过
		"sign":      "xxx", // sign 应跳过
	}
	got := SignContent(params)
	want := "app_id=2021000000&charset=UTF-8&method=alipay.trade.precreate&sign_type=RSA2"
	if got != want {
		t.Fatalf("待签串不对\n got=%q\nwant=%q", got, want)
	}
}

func TestSignContentSkipsAtPrefix(t *testing.T) {
	params := map[string]string{"a": "1", "file": "@/path/x"}
	if got := SignContent(params); got != "a=1" {
		t.Fatalf("@ 开头的值应跳过, got=%q", got)
	}
}

func TestParsePrivateKeyPKCS8(t *testing.T) {
	privPEM, _, want := genKeys(t)
	got, err := ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatalf("解析私钥失败: %v", err)
	}
	if got.N.Cmp(want.N) != 0 {
		t.Fatal("私钥解析结果与原始不一致")
	}
}

func TestParsePrivateKeyInvalid(t *testing.T) {
	if _, err := ParsePrivateKey("!!!not-a-key!!!"); err == nil {
		t.Fatal("非法私钥应报错")
	}
}

// TestSignVerifyRoundTrip 私钥签名 → 公钥验签通过；篡改参数验签失败。
func TestSignVerifyRoundTrip(t *testing.T) {
	privPEM, pubPEM, _ := genKeys(t)
	priv, err := ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := ParsePublicKey(pubPEM)
	if err != nil {
		t.Fatal(err)
	}

	// 支付宝回调签名不含 sign_type（Verify 会剔除 sign_type），故签名内容也不含它。
	params := map[string]string{
		"out_trade_no": "20260721001",
		"trade_no":     "2026072122001",
		"total_amount": "1.00",
		"trade_status": "TRADE_SUCCESS",
	}
	sig, err := SignParams(priv, params)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	params["sign_type"] = "RSA2"
	params["sign"] = sig

	if !Verify(pub, params) {
		t.Fatal("正确签名应验签通过")
	}

	// 篡改金额
	params["total_amount"] = "9999.00"
	if Verify(pub, params) {
		t.Fatal("金额篡改后应验签失败")
	}
}

func TestVerifyNoSign(t *testing.T) {
	_, pubPEM, _ := genKeys(t)
	pub, _ := ParsePublicKey(pubPEM)
	if Verify(pub, map[string]string{"a": "1"}) {
		t.Fatal("无 sign 应验签失败")
	}
}

// TestVerifyIgnoresSignType 验签剔除 sign_type：签名时不含 sign_type 也能通过（对齐 SDK verify 剔除 sign_type）。
func TestVerifyIgnoresSignType(t *testing.T) {
	privPEM, pubPEM, _ := genKeys(t)
	priv, _ := ParsePrivateKey(privPEM)
	pub, _ := ParsePublicKey(pubPEM)

	// 商户对不含 sign_type 的参数签名
	params := map[string]string{"out_trade_no": "X1", "total_amount": "1.00"}
	sig, _ := SignParams(priv, params)

	// 回调里带上 sign_type + sign，Verify 应剔除 sign_type 后验签通过
	params["sign_type"] = "RSA2"
	params["sign"] = sig
	if !Verify(pub, params) {
		t.Fatal("Verify 应剔除 sign_type 后验签通过")
	}
}
