package sign

import "testing"

// TestRSARoundTrip 生成密钥对 → 私钥签 → 公钥验，全链路应通过。
func TestRSARoundTrip(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}
	params := map[string]string{
		"pid": "1", "type": "alipay", "out_trade_no": "V2TEST001",
		"money": "1.00", "timestamp": "1784600000", "sign_type": "RSA",
	}
	sig, err := MakeRSA(params, priv)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	params["sign"] = sig
	if !VerifyRSA(params, pub) {
		t.Fatal("正确签名验签应通过")
	}
}

// TestRSATamperFails 篡改参数后验签应失败。
func TestRSATamperFails(t *testing.T) {
	priv, pub, _ := GenerateRSAKeyPair()
	params := map[string]string{"pid": "1", "money": "1.00", "sign_type": "RSA"}
	sig, _ := MakeRSA(params, priv)
	params["sign"] = sig
	params["money"] = "9999.00" // 篡改金额
	if VerifyRSA(params, pub) {
		t.Fatal("篡改金额后验签应失败")
	}
}

// TestRSAWrongKeyFails 用另一对密钥的公钥验签应失败。
func TestRSAWrongKeyFails(t *testing.T) {
	priv, _, _ := GenerateRSAKeyPair()
	_, otherPub, _ := GenerateRSAKeyPair()
	params := map[string]string{"pid": "1", "money": "1.00", "sign_type": "RSA"}
	sig, _ := MakeRSA(params, priv)
	params["sign"] = sig
	if VerifyRSA(params, otherPub) {
		t.Fatal("错误公钥验签应失败")
	}
}

// TestRSAContentMatchesV1 RSA 与 MD5 复用同一待签名串（sign_type 不参与签名）。
func TestRSAContentMatchesV1(t *testing.T) {
	params := map[string]string{
		"pid": "1", "money": "1.00", "sign_type": "RSA", "sign": "xxx",
	}
	got := Content(params)
	want := "money=1.00&pid=1"
	if got != want {
		t.Fatalf("待签名串应为 %q，实际 %q", want, got)
	}
}
