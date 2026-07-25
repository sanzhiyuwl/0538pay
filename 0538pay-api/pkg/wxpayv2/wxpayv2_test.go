package wxpayv2

import (
	"crypto/aes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

// TestMakeSignMD5 对齐 epay BaseService::makeSign：字典序拼接、跳过空值与 sign、末尾 &key=、MD5 大写。
// 手工核算：a=1&b=2&key=KEY → md5 → 大写。
func TestMakeSignMD5(t *testing.T) {
	data := map[string]string{"b": "2", "a": "1", "sign": "OLD", "empty": ""}
	got := MakeSign(data, "KEY")
	// 期望签名串 = "a=1&b=2&key=KEY"
	sum := md5.Sum([]byte("a=1&b=2&key=KEY"))
	want := hex.EncodeToString(sum[:])
	// MakeSign 返回大写
	if got != upper(want) {
		t.Fatalf("MD5 签名不对: got=%s want=%s", got, upper(want))
	}
}

// TestMakeSignSkipsSignAndEmpty 确认 sign 键、空值不参与签名。
func TestMakeSignSkipsSignAndEmpty(t *testing.T) {
	base := MakeSign(map[string]string{"a": "1"}, "K")
	withNoise := MakeSign(map[string]string{"a": "1", "sign": "XXX", "z": ""}, "K")
	if base != withNoise {
		t.Fatalf("sign/空值应被跳过, base=%s noise=%s", base, withNoise)
	}
}

// TestMakeSignHMAC HMAC-SHA256 分支（sign_type=HMAC-SHA256）。
func TestMakeSignHMAC(t *testing.T) {
	data := map[string]string{"a": "1", "sign_type": SignTypeHMAC}
	got := MakeSign(data, "KEY")
	if len(got) != 64 { // HMAC-SHA256 → 32 字节 → 64 位 hex
		t.Fatalf("HMAC-SHA256 签名应为 64 位 hex, 实际 %d 位: %s", len(got), got)
	}
	if got != upper(got) {
		t.Fatal("签名应为大写")
	}
}

func TestCheckSign(t *testing.T) {
	data := map[string]string{"a": "1", "b": "2"}
	data["sign"] = MakeSign(data, "KEY")
	if !CheckSign(data, "KEY") {
		t.Fatal("自签自验应通过")
	}
	data["a"] = "999" // 篡改
	if CheckSign(data, "KEY") {
		t.Fatal("篡改后验签应失败")
	}
	if CheckSign(map[string]string{"a": "1"}, "KEY") {
		t.Fatal("无 sign 键应验签失败")
	}
}

// TestXMLRoundTrip MapToXML → XMLToMap 应无损（数值与含特殊字符文本都覆盖）。
func TestXMLRoundTrip(t *testing.T) {
	in := map[string]string{
		"appid":        "wx123",
		"total_fee":    "100",
		"out_trade_no": "T-2026&<x>",
		"code_url":     "weixin://wxpay/bizpayurl?pr=abc",
	}
	xmlStr := MapToXML(in)
	out, err := XMLToMap(xmlStr)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	for k, v := range in {
		if out[k] != v {
			t.Fatalf("字段 %s 不一致: in=%q out=%q\nxml=%s", k, v, out[k], xmlStr)
		}
	}
}

// TestMapToXMLNumericBare 数值裸写、文本 CDATA 包裹（对齐 epay array2Xml）。
func TestMapToXMLNumericBare(t *testing.T) {
	xmlStr := MapToXML(map[string]string{"total_fee": "100", "body": "商品"})
	if !contains(xmlStr, "<total_fee>100</total_fee>") {
		t.Fatalf("数值应裸写: %s", xmlStr)
	}
	if !contains(xmlStr, "<body><![CDATA[商品]]></body>") {
		t.Fatalf("文本应 CDATA 包裹: %s", xmlStr)
	}
}

func TestXMLToMapEmpty(t *testing.T) {
	if _, err := XMLToMap(""); err == nil {
		t.Fatal("空报文应报错")
	}
	if _, err := XMLToMap("not xml at all"); err == nil {
		t.Fatal("非法报文应报错")
	}
}

func TestNonceStr(t *testing.T) {
	s, err := NonceStr(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 32 {
		t.Fatalf("长度应为 32, 实际 %d", len(s))
	}
	s2, _ := NonceStr(32)
	if s == s2 {
		t.Fatal("两次随机串不应相同")
	}
	// 超长回落 32
	long, _ := NonceStr(999)
	if len(long) != 32 {
		t.Fatalf("超长应回落 32, 实际 %d", len(long))
	}
}

// TestDecryptRefundNotify 自加密（AES-256-ECB + md5(key)）再解密，端到端对齐 epay refundNotify。
func TestDecryptRefundNotify(t *testing.T) {
	apiKey := "01234567890123456789012345678901"
	plain := []byte(`<root><out_trade_no>T1</out_trade_no><refund_status>SUCCESS</refund_status></root>`)

	sum := md5.Sum([]byte(apiKey))
	key := []byte(hex.EncodeToString(sum[:]))
	block, _ := aes.NewCipher(key)
	bs := block.BlockSize()
	// PKCS7 填充
	pad := bs - len(plain)%bs
	padded := append(append([]byte{}, plain...), bytesRepeat(byte(pad), pad)...)
	enc := make([]byte, len(padded))
	for i := 0; i < len(padded); i += bs {
		block.Encrypt(enc[i:i+bs], padded[i:i+bs])
	}
	reqInfo := base64.StdEncoding.EncodeToString(enc)

	got, err := DecryptRefundNotify(reqInfo, apiKey)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if string(got) != string(plain) {
		t.Fatalf("解密结果不一致: got=%s want=%s", got, plain)
	}

	// 错误密钥应解密失败或填充非法
	if _, err := DecryptRefundNotify(reqInfo, "ffffffffffffffffffffffffffffffff"); err == nil {
		t.Fatal("错误密钥应解密失败")
	}
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
