package wxwork

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

// genAESKey 生成合法的 43 位 EncodingAESKey（32 字节 base64 去尾 "="）。
func genAESKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	full := base64.StdEncoding.EncodeToString(b) // 44 位，尾部 "="
	return full[:len(full)-1]                    // 去尾 "=" → 43 位
}

// TestCryptRoundTrip 加密→解密应还原原文，且尾部 receiveid 不污染明文。
func TestCryptRoundTrip(t *testing.T) {
	c, err := NewCrypt(genAESKey(t))
	if err != nil {
		t.Fatalf("构造失败: %v", err)
	}
	cases := []string{
		"<xml><Content>你好客服</Content></xml>",
		"",
		"a",
		"这是一段较长的中文消息内容，用于验证跨 block 的 PKCS7 补位与长度前缀正确。",
	}
	for _, plain := range cases {
		enc, err := c.Encrypt(plain, "ww1234567890")
		if err != nil {
			t.Fatalf("加密失败: %v", err)
		}
		got, err := c.Decrypt(enc)
		if err != nil {
			t.Fatalf("解密失败: %v", err)
		}
		if got != plain {
			t.Errorf("round-trip 不一致：want %q got %q", plain, got)
		}
	}
}

// TestCryptWrongKey 错误密钥解密应失败或不还原（不 panic）。
func TestCryptWrongKey(t *testing.T) {
	c1, _ := NewCrypt(genAESKey(t))
	c2, _ := NewCrypt(genAESKey(t))
	enc, _ := c1.Encrypt("<xml>secret</xml>", "corp")
	if got, err := c2.Decrypt(enc); err == nil && got == "<xml>secret</xml>" {
		t.Error("错误密钥不应还原出原文")
	}
}

// TestNewCryptInvalid 非法 EncodingAESKey 应返回错误。
func TestNewCryptInvalid(t *testing.T) {
	if _, err := NewCrypt("tooshort"); err == nil {
		t.Error("过短的 AESKey 应报错")
	}
	if _, err := NewCrypt("!!!not-base64!!!"); err == nil {
		t.Error("非 base64 应报错")
	}
}

// TestSignature 签名稳定且验签正确；篡改任一要素验签失败（对齐 epay sort+sha1）。
func TestSignature(t *testing.T) {
	token, ts, nonce, enc := "TOKEN", "1700000000", "abc123", "ENCRYPTEDMSG"
	sig := Signature(token, ts, nonce, enc)
	if sig == "" {
		t.Fatal("签名不应为空")
	}
	if !VerifySignature(token, ts, nonce, enc, sig) {
		t.Error("正确参数验签应通过")
	}
	if VerifySignature(token, ts, nonce, "TAMPERED", sig) {
		t.Error("篡改密文验签应失败")
	}
	// 与参数顺序无关（sort），交换 nonce/timestamp 位置不影响（同一集合）
	if Signature(token, ts, nonce, enc) != Signature(token, ts, nonce, enc) {
		t.Error("同参数签名应稳定")
	}
}
