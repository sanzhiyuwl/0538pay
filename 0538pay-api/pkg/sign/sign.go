// Package sign 实现易支付 V1 协议的 MD5 签名与验签。
//
// 逐字节对齐 epay includes/lib/Payment.php 的 getSignContent + makeSign + verifySign：
//  1. 按参数名升序排序（ASCII/字典序）。
//  2. 跳过：值为空（null 或 trim 后为空串）、数组、以及 sign / sign_type 两个键。
//  3. 拼成 k=v&k=v...（去掉末尾 &）。
//  4. MD5：md5(signStr + 商户密钥)，返回 32 位小写十六进制。
//
// 只做 MD5（V1）；RSA（V2）留待 V2 协议阶段再实现。
package sign

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
)

// isEmpty 对齐 epay isEmpty()：null 或 trim 后为空串。
// Go 侧参数都是 string，故仅判断 trim 后是否为空。
func isEmpty(v string) bool {
	return strings.TrimSpace(v) == ""
}

// Content 生成待签名字符串（不含密钥）。params 为原始请求参数。
func Content(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" || k == "sign_type" {
			continue
		}
		// 内部注入键（下划线前缀，如 _ip）不参与验签——非商户上送字段。
		if strings.HasPrefix(k, "_") {
			continue
		}
		if isEmpty(params[k]) {
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

// MakeMD5 生成 MD5 签名：md5(签名串 + 密钥)，小写十六进制。
func MakeMD5(params map[string]string, key string) string {
	sum := md5.Sum([]byte(Content(params) + key))
	return hex.EncodeToString(sum[:])
}

// VerifyMD5 校验 MD5 签名。epay 用 === 严格比较（md5 输出小写），
// 这里用 EqualFold 放宽大小写，容忍部分客户端上送大写十六进制。
func VerifyMD5(params map[string]string, key string) bool {
	got := params["sign"]
	if got == "" {
		return false
	}
	return strings.EqualFold(got, MakeMD5(params, key))
}
