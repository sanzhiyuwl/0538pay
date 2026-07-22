// Package wxpayv3 实现微信支付 APIv3 所需的密码学原语：
// 请求签名(Authorization 头)、应答/回调验签、回调报文 AES-256-GCM 解密。
//
// 逐条对齐微信支付 APIv3 官方规范（普通商户）：
//   - 请求签名：SHA256withRSA，签名串 = 方法\nURL路径\n时间戳\n随机串\n请求体\n，
//     用商户 API 证书私钥签名，组装成 WECHATPAY2-SHA256-RSA2048 的 Authorization 头。
//   - 应答/回调验签：签名串 = 时间戳\n随机串\n报文主体\n，用微信支付平台证书/公钥（RSA 公钥）
//     对 Wechatpay-Signature（Base64）做 SHA256 验签。
//   - 回调解密：AEAD_AES_256_GCM，key=APIv3密钥(32字节)，nonce+associated_data 来自回调，
//     ciphertext 为 Base64（末 16 字节为 GCM tag）。
//
// 不引第三方微信 SDK：与 pkg/sign 一致，全部用 Go 标准库手写，便于审计且不增依赖。
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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParsePrivateKey 解析 PKCS#8（微信商户 apiclient_key.pem）或 PKCS#1 的 RSA 私钥。
func ParsePrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemStr)))
	if block == nil {
		return nil, errors.New("私钥 PEM 解析失败：无有效 PEM 块")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("私钥不是 RSA 类型")
		}
		return rk, nil
	}
	// 回退 PKCS#1
	rk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("私钥解析失败: %w", err)
	}
	return rk, nil
}

// ParsePublicKey 解析微信支付公钥（PEM，PKIX 格式）。平台证书场景可先取证书公钥再传入。
func ParsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(pemStr)))
	if block == nil {
		return nil, errors.New("公钥 PEM 解析失败：无有效 PEM 块")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// 尝试按证书解析，取其中公钥
		if cert, e := x509.ParseCertificate(block.Bytes); e == nil {
			if rk, ok := cert.PublicKey.(*rsa.PublicKey); ok {
				return rk, nil
			}
		}
		return nil, fmt.Errorf("公钥解析失败: %w", err)
	}
	rk, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("公钥不是 RSA 类型")
	}
	return rk, nil
}

// signMessage 用私钥对消息做 SHA256withRSA 签名，返回 Base64。
func signMessage(priv *rsa.PrivateKey, message string) (string, error) {
	h := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// SignMessage 用商户私钥对任意消息做 SHA256withRSA 签名并 base64（JSAPI/APP 拉起参数二次签名用）。
// JSAPI paySign 签名串格式：appId\ntimeStamp\nnonceStr\nprefix+prepay_id\n。
func SignMessage(priv *rsa.PrivateKey, message string) (string, error) {
	return signMessage(priv, message)
}

// AuthParams 组装 Authorization 头所需的商户身份 + 请求信息。
type AuthParams struct {
	MchID      string          // 商户号
	SerialNo   string          // 商户 API 证书序列号
	PrivateKey *rsa.PrivateKey // 商户 API 证书私钥
	Method     string          // HTTP 方法，如 POST
	CanonicalURL string        // 含 query 的路径，如 /v3/pay/transactions/native
	Body       string          // 请求体（GET 为空串）
	Timestamp  int64           // Unix 秒
	Nonce      string          // 随机串
}

// BuildAuthorization 构建 APIv3 Authorization 头值。
// 签名串格式：Method\nCanonicalURL\nTimestamp\nNonce\nBody\n（每行以 \n 结尾）。
func BuildAuthorization(p AuthParams) (string, error) {
	message := p.Method + "\n" + p.CanonicalURL + "\n" +
		strconv.FormatInt(p.Timestamp, 10) + "\n" + p.Nonce + "\n" + p.Body + "\n"
	sig, err := signMessage(p.PrivateKey, message)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%d",serial_no="%s"`,
		p.MchID, p.Nonce, sig, p.Timestamp, p.SerialNo,
	), nil
}

// VerifySignature 验证应答/回调签名。
// 验签串格式：Timestamp\nNonce\nBody\n。sigBase64 为 Wechatpay-Signature 头（Base64）。
func VerifySignature(pub *rsa.PublicKey, timestamp, nonce, body, sigBase64 string) error {
	message := timestamp + "\n" + nonce + "\n" + body + "\n"
	sig, err := base64.StdEncoding.DecodeString(sigBase64)
	if err != nil {
		return fmt.Errorf("签名 Base64 解码失败: %w", err)
	}
	h := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig); err != nil {
		return errors.New("应答/回调签名验证不通过")
	}
	return nil
}

// DecryptAESGCM 解密回调报文密文（AEAD_AES_256_GCM）。
// apiV3Key 为商户 APIv3 密钥（32 字节）；ciphertextB64 为 Base64 密文（末 16 字节为 tag）。
func DecryptAESGCM(apiV3Key, nonce, associatedData, ciphertextB64 string) ([]byte, error) {
	if len(apiV3Key) != 32 {
		return nil, errors.New("APIv3 密钥长度必须为 32 字节")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("密文 Base64 解码失败: %w", err)
	}
	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("nonce 长度不合法：期望 %d，实际 %d", gcm.NonceSize(), len(nonce))
	}
	plain, err := gcm.Open(nil, []byte(nonce), ciphertext, []byte(associatedData))
	if err != nil {
		return nil, fmt.Errorf("回调报文解密失败: %w", err)
	}
	return plain, nil
}

// NonceStr 生成指定长度的随机串（大小写字母+数字），用于请求签名。
func NonceStr(n int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// NowUnix 返回当前 Unix 秒（抽出以便单测替换）。
func NowUnix() int64 { return time.Now().Unix() }
