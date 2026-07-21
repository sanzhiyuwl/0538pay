// RSA(V2) 签名与验签，对齐 epay includes/lib/Payment.php 的 makeSign/verifySign 的 RSA 分支：
//   - 待签名串复用 V1 的 Content()（ksort→跳空/跳 sign,sign_type→k=v& 拼接，不 URL 编码）。
//   - 算法 SHA256WithRSA（openssl_sign OPENSSL_ALGO_SHA256），签名结果 base64。
//   - 密钥 2048 位 RSA，PKCS#8 头（PUBLIC KEY / PRIVATE KEY），库里存"去头尾单行 base64"。
//
// 密钥流向（双向，不要搞反）：
//   下单：商户私钥签 → 平台用商户公钥(pre_user.publickey)验。
//   回调/返回：平台私钥(conf.private_key)签 → 商户用平台公钥(conf.public_key)验。
package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// stripPEM 去掉 PEM 头尾与所有空白，得到纯 base64 单行（对齐 epay pemToBase64 的存储格式）。
func stripPEM(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "-----") {
			continue
		}
		b.WriteString(line)
	}
	// 若本身就是单行裸 base64（无 PEM 头），上面循环也能正确保留
	out := b.String()
	if out == "" {
		// 单行无换行的裸串
		r := strings.NewReplacer(" ", "", "\t", "", "\r", "", "\n", "")
		return r.Replace(s)
	}
	return out
}

// ParseRSAPrivateKey 解析平台/商户私钥（epay 单行 base64 或标准 PEM，PKCS#8 优先，兼容 PKCS#1）。
func ParseRSAPrivateKey(key string) (*rsa.PrivateKey, error) {
	der, err := base64.StdEncoding.DecodeString(stripPEM(key))
	if err != nil {
		return nil, errors.New("RSA 私钥非合法 base64")
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, errors.New("RSA 私钥不是 RSA 类型")
	}
	rk, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("RSA 私钥解析失败: %w", err)
	}
	return rk, nil
}

// ParseRSAPublicKey 解析平台/商户公钥（epay 单行 base64 或标准 PEM，PKIX/PKCS#8 头）。
func ParseRSAPublicKey(key string) (*rsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(stripPEM(key))
	if err != nil {
		return nil, errors.New("RSA 公钥非合法 base64")
	}
	if pub, err := x509.ParsePKIXPublicKey(der); err == nil {
		if rk, ok := pub.(*rsa.PublicKey); ok {
			return rk, nil
		}
		return nil, errors.New("RSA 公钥不是 RSA 类型")
	}
	// 兼容 PKCS#1 公钥
	if rk, err := x509.ParsePKCS1PublicKey(der); err == nil {
		return rk, nil
	}
	return nil, errors.New("RSA 公钥解析失败")
}

// MakeRSA 用私钥对待签名串做 SHA256WithRSA 签名，返回 base64（对齐 epay makeSign RSA 分支）。
// privKey 为 epay 单行 base64 或 PEM 格式。
func MakeRSA(params map[string]string, privKey string) (string, error) {
	priv, err := ParseRSAPrivateKey(privKey)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(Content(params)))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifyRSA 用公钥校验 params["sign"]（对齐 epay verifySign RSA 分支）。
// pubKey 为 epay 单行 base64 或 PEM 格式。
func VerifyRSA(params map[string]string, pubKey string) bool {
	got := params["sign"]
	if got == "" {
		return false
	}
	pub, err := ParseRSAPublicKey(pubKey)
	if err != nil {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		return false
	}
	h := sha256.Sum256([]byte(Content(params)))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], raw) == nil
}

// GenerateRSAKeyPair 生成 2048 位 RSA 密钥对，返回 epay 格式的单行 base64（PKCS#8 私钥 / PKIX 公钥）。
// 对齐 epay generate_key_pair + pemToBase64：库里存去头尾单行 base64。
func GenerateRSAKeyPair() (privB64, pubB64 string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(privDER),
		base64.StdEncoding.EncodeToString(pubDER), nil
}
