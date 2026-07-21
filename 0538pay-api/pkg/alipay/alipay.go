// Package alipay 实现支付宝开放平台 API 所需的 RSA2 签名与验签。
//
// 逐字节对齐 epay 内置的 cccyun/alipay-sdk（AopClient::getSignContent + generateSign + verify）：
//   - 待签串：对全部参数 ksort（键升序）→ 跳过空值/sign/以@开头的值 → 拼 `k=v&k=v...`（无前导 &）。
//   - 签名：SHA256withRSA（RSA2），私钥 PKCS#1 或 PKCS#8，输出 Base64。
//   - 验签：对回调/应答参数同法构造待签串，用支付宝公钥做 SHA256 验签。
//
// 只做 RSA2（SHA256withRSA），不做已淘汰的 RSA(SHA1)。纯 Go 标准库，不引第三方 SDK。
package alipay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// isEmpty 对齐 SDK isEmpty()：nil 或 trim 后为空串。
func isEmpty(v string) bool { return strings.TrimSpace(v) == "" }

// SignContent 生成待签名字符串（不含 sign）。
// 对齐 AopClient::getSignContent：ksort → 跳过空值/sign/@开头 → `&k=v` 拼接后去首个 &。
func SignContent(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		v := params[k]
		if isEmpty(v) || strings.HasPrefix(v, "@") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	return b.String()
}

// ParsePrivateKey 解析应用私钥（PKCS#8 或 PKCS#1，PEM 或裸 Base64）。
func ParsePrivateKey(key string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(key)))
	var der []byte
	if block != nil {
		der = block.Bytes
	} else {
		// 裸 Base64（支付宝后台直接给的私钥串，无 PEM 头）
		b, err := base64.StdEncoding.DecodeString(stripSpace(key))
		if err != nil {
			return nil, errors.New("应用私钥解析失败：既非 PEM 也非合法 Base64")
		}
		der = b
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("应用私钥不是 RSA 类型")
		}
		return rk, nil
	}
	rk, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf("应用私钥解析失败: %w", err)
	}
	return rk, nil
}

// ParsePublicKey 解析支付宝公钥（PKIX，PEM 或裸 Base64）。
func ParsePublicKey(key string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(key)))
	var der []byte
	if block != nil {
		der = block.Bytes
	} else {
		b, err := base64.StdEncoding.DecodeString(stripSpace(key))
		if err != nil {
			return nil, errors.New("支付宝公钥解析失败：既非 PEM 也非合法 Base64")
		}
		der = b
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		if cert, e := x509.ParseCertificate(der); e == nil {
			if rk, ok := cert.PublicKey.(*rsa.PublicKey); ok {
				return rk, nil
			}
		}
		return nil, fmt.Errorf("支付宝公钥解析失败: %w", err)
	}
	rk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("支付宝公钥不是 RSA 类型")
	}
	return rk, nil
}

// stripSpace 去掉换行/空格，便于解析裸 Base64 密钥。
func stripSpace(s string) string {
	r := strings.NewReplacer("\n", "", "\r", "", " ", "", "\t", "")
	return r.Replace(strings.TrimSpace(s))
}

// Sign 用应用私钥对待签串做 RSA2(SHA256withRSA) 签名，返回 Base64。
func Sign(priv *rsa.PrivateKey, content string) (string, error) {
	h := sha256.Sum256([]byte(content))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// SignParams 对参数集签名并返回 sign 值（内部走 SignContent + Sign）。
func SignParams(priv *rsa.PrivateKey, params map[string]string) (string, error) {
	return Sign(priv, SignContent(params))
}

// Verify 用支付宝公钥验签（对回调/应答参数）。params 含 sign/sign_type，
// 验签时剔除 sign（sign_type 也不参与，对齐 SDK verify）。
func Verify(pub *rsa.PublicKey, params map[string]string) bool {
	sig := params["sign"]
	if sig == "" {
		return false
	}
	// 复制并剔除 sign / sign_type
	filtered := make(map[string]string, len(params))
	for k, v := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		filtered[k] = v
	}
	raw, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	h := sha256.Sum256([]byte(SignContent(filtered)))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], raw) == nil
}
