// Package wxpayv2 实现微信支付 APIv2 所需的原语：MD5/HMAC-SHA256 签名、
// <xml> 报文与 map 的互转、随机串、退款回调密文解密（AES-256-ECB）。
//
// 逐条对齐微信支付 APIv2 官方规范与 epay 内置 cccyun/wechatpay-sdk（BaseService）：
//   - 签名：参数按 key 字典序（ksort）拼 k=v&，跳过 sign 键与空值，末尾拼 &key=APIv2密钥，
//     再按 sign_type 取 MD5 或 HMAC-SHA256，结果转大写。
//   - 报文：<xml> 下每个字段数值裸写、非数值用 <![CDATA[..]]> 包裹；解析忽略大小写与 CDATA。
//   - 退款回调解密：req_info 为 Base64，key=md5(APIv2密钥)（小写 32 位），AES-256-ECB 解密。
//
// 不引第三方微信 SDK：与 pkg/wxpayv3、pkg/sign 一致，全部用 Go 标准库手写，便于审计。
package wxpayv2

import (
	"crypto/aes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// SignType 签名算法。
const (
	SignTypeMD5    = "MD5"
	SignTypeHMAC   = "HMAC-SHA256"
)

// MakeSign 生成 APIv2 签名（对齐 epay BaseService::makeSign）。
// 规则：字典序拼 k=v&（跳过 sign 键与空值/数组）→ 末尾 &key=apiKey →
// 按 data["sign_type"]=="HMAC-SHA256" 走 HMAC-SHA256，否则 MD5 → 转大写。
func MakeSign(data map[string]string, apiKey string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v := data[k]
		if k == "sign" || v == "" {
			continue
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(v)
		b.WriteByte('&')
	}
	signStr := strings.TrimRight(b.String(), "&") + "&key=" + apiKey
	var sum string
	if data["sign_type"] == SignTypeHMAC {
		h := hmac.New(sha256.New, []byte(apiKey))
		h.Write([]byte(signStr))
		sum = hex.EncodeToString(h.Sum(nil))
	} else {
		s := md5.Sum([]byte(signStr))
		sum = hex.EncodeToString(s[:])
	}
	return strings.ToUpper(sum)
}

// CheckSign 校验报文签名（对齐 epay BaseService::checkSign）：用同规则重算与 data["sign"] 比较。
func CheckSign(data map[string]string, apiKey string) bool {
	sig, ok := data["sign"]
	if !ok || sig == "" {
		return false
	}
	return MakeSign(data, apiKey) == sig
}

// MapToXML 把 map 转 <xml> 报文（对齐 epay BaseService::array2Xml）。
// 数值型裸写，其余用 CDATA 包裹；为可预测输出按 key 字典序排列。
func MapToXML(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("<xml>")
	for _, k := range keys {
		v := data[k]
		if isNumeric(v) {
			fmt.Fprintf(&b, "<%s>%s</%s>", k, v, k)
		} else {
			fmt.Fprintf(&b, "<%s><![CDATA[%s]]></%s>", k, v, k)
		}
	}
	b.WriteString("</xml>")
	return b.String()
}

// XMLToMap 解析 <xml> 报文为 map（对齐 epay BaseService::xml2array，自动去 CDATA）。
func XMLToMap(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("XML 报文为空")
	}
	dec := xml.NewDecoder(strings.NewReader(raw))
	out := map[string]string{}
	var depth int
	var curKey string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 { // <xml> 下第一层字段
				curKey = t.Name.Local
			}
		case xml.EndElement:
			depth--
			if depth == 1 {
				curKey = ""
			}
		case xml.CharData:
			if depth == 2 && curKey != "" {
				out[curKey] += string(t)
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("XML 报文解析为空（非合法微信报文）")
	}
	return out, nil
}

// NonceStr 生成不长于 32 位的随机串（小写字母+数字，对齐 epay getNonceStr 字符集）。
func NonceStr(n int) (string, error) {
	if n <= 0 || n > 32 {
		n = 32
	}
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// DecryptRefundNotify 解密退款回调 req_info（对齐 epay refundNotify）：
// Base64 解 req_info → key=md5(apiKey) 小写 32 位 → AES-256-ECB 解密去 PKCS7 填充。
func DecryptRefundNotify(reqInfoB64, apiKey string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(reqInfoB64)
	if err != nil {
		return nil, fmt.Errorf("req_info Base64 解码失败: %w", err)
	}
	sum := md5.Sum([]byte(apiKey))
	key := []byte(hex.EncodeToString(sum[:])) // md5 十六进制小写，32 字节
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(ciphertext) == 0 || len(ciphertext)%bs != 0 {
		return nil, errors.New("退款回调密文长度不合法")
	}
	plain := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += bs {
		block.Decrypt(plain[i:i+bs], ciphertext[i:i+bs])
	}
	return pkcs7Unpad(plain, bs)
}

// isNumeric 判断字符串是否为纯十进制数字（用于 array2Xml 决定是否 CDATA 包裹）。
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// pkcs7Unpad 去 PKCS7 填充。
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	n := len(data)
	if n == 0 {
		return nil, errors.New("空数据")
	}
	pad := int(data[n-1])
	if pad <= 0 || pad > blockSize || pad > n {
		return nil, errors.New("PKCS7 填充非法")
	}
	return data[:n-pad], nil
}
