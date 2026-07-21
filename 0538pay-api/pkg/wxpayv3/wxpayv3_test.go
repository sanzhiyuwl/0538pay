package wxpayv3

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
)

// genKeyPEM 生成一对 RSA 私钥(PKCS#8)/公钥(PKIX) PEM，供测试自给自足（不依赖真实商户凭证）。
func genKeyPEM(t *testing.T) (privPEM, pubPEM string, priv *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 私钥失败: %v", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("序列化私钥失败: %v", err)
	}
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}))
	pkix, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("序列化公钥失败: %v", err)
	}
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}))
	return
}

func TestParsePrivateKeyPKCS8(t *testing.T) {
	privPEM, _, want := genKeyPEM(t)
	got, err := ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatalf("解析私钥失败: %v", err)
	}
	if got.N.Cmp(want.N) != 0 {
		t.Fatal("解析出的私钥与原始不一致")
	}
}

func TestParsePrivateKeyInvalid(t *testing.T) {
	if _, err := ParsePrivateKey("not a pem"); err == nil {
		t.Fatal("非法 PEM 应报错")
	}
}

// TestBuildAuthorization 验证 Authorization 头结构，且其中 signature 能被对应公钥验签通过。
func TestBuildAuthorization(t *testing.T) {
	_, _, priv := genKeyPEM(t)
	p := AuthParams{
		MchID:        "1230000109",
		SerialNo:     "ABC123SERIAL",
		PrivateKey:   priv,
		Method:       "POST",
		CanonicalURL: "/v3/pay/transactions/native",
		Body:         `{"appid":"wx123"}`,
		Timestamp:    1700000000,
		Nonce:        "testnonce123456",
	}
	auth, err := BuildAuthorization(p)
	if err != nil {
		t.Fatalf("构建 Authorization 失败: %v", err)
	}
	for _, want := range []string{
		`WECHATPAY2-SHA256-RSA2048 `,
		`mchid="1230000109"`,
		`nonce_str="testnonce123456"`,
		`serial_no="ABC123SERIAL"`,
		`timestamp="1700000000"`,
		`signature="`,
	} {
		if !strings.Contains(auth, want) {
			t.Fatalf("Authorization 缺少片段 %q，实际: %s", want, auth)
		}
	}

	// 取出 signature，用公钥按同样的签名串验签，确认逐字节对齐。
	sigB64 := extractField(auth, "signature")
	message := p.Method + "\n" + p.CanonicalURL + "\n1700000000\n" + p.Nonce + "\n" + p.Body + "\n"
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		t.Fatalf("signature 解码失败: %v", err)
	}
	h := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, h[:], sig); err != nil {
		t.Fatalf("Authorization 签名验签失败: %v", err)
	}
}

func extractField(auth, field string) string {
	marker := field + `="`
	i := strings.Index(auth, marker)
	if i < 0 {
		return ""
	}
	rest := auth[i+len(marker):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// TestVerifySignature 用私钥按微信验签串格式签名，再用公钥 VerifySignature，round-trip 通过；篡改应失败。
func TestVerifySignature(t *testing.T) {
	_, pubPEM, priv := genKeyPEM(t)
	pub, err := ParsePublicKey(pubPEM)
	if err != nil {
		t.Fatalf("解析公钥失败: %v", err)
	}
	ts, nonce, body := "1700000000", "abcdef123456", `{"code":"SUCCESS"}`
	message := ts + "\n" + nonce + "\n" + body + "\n"
	h := sha256.Sum256([]byte(message))
	raw, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	sigB64 := base64.StdEncoding.EncodeToString(raw)

	if err := VerifySignature(pub, ts, nonce, body, sigB64); err != nil {
		t.Fatalf("正确签名应验签通过: %v", err)
	}
	// 篡改报文
	if err := VerifySignature(pub, ts, nonce, body+"x", sigB64); err == nil {
		t.Fatal("报文篡改后应验签失败")
	}
	// 篡改时间戳
	if err := VerifySignature(pub, "1700000001", nonce, body, sigB64); err == nil {
		t.Fatal("时间戳篡改后应验签失败")
	}
}

// TestDecryptAESGCM 用与微信一致的 AEAD_AES_256_GCM 参数自行加密，再解密，验证 round-trip 与鉴权失败分支。
func TestDecryptAESGCM(t *testing.T) {
	key := "01234567890123456789012345678901" // 32 字节 APIv3 密钥
	nonce := "abcdef123456"                    // GCM 12 字节 nonce
	ad := "transaction"
	plain := `{"out_trade_no":"20260721001","trade_state":"SUCCESS"}`

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		t.Fatalf("建 cipher 失败: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("建 GCM 失败: %v", err)
	}
	sealed := gcm.Seal(nil, []byte(nonce), []byte(plain), []byte(ad))
	cipherB64 := base64.StdEncoding.EncodeToString(sealed)

	got, err := DecryptAESGCM(key, nonce, ad, cipherB64)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if string(got) != plain {
		t.Fatalf("解密结果不一致，期望 %q 实际 %q", plain, string(got))
	}

	// 错误的 APIv3 密钥应解密失败
	if _, err := DecryptAESGCM("00000000000000000000000000000000", nonce, ad, cipherB64); err == nil {
		t.Fatal("错误密钥应解密失败")
	}
	// 密钥长度不对
	if _, err := DecryptAESGCM("shortkey", nonce, ad, cipherB64); err == nil {
		t.Fatal("32 字节以外的密钥应报错")
	}
}

func TestNonceStr(t *testing.T) {
	s, err := NonceStr(32)
	if err != nil {
		t.Fatalf("生成随机串失败: %v", err)
	}
	if len(s) != 32 {
		t.Fatalf("随机串长度应为 32，实际 %d", len(s))
	}
	s2, _ := NonceStr(32)
	if s == s2 {
		t.Fatal("两次随机串不应相同")
	}
}
