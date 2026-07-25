// Package fuiou 实现富友支付（fuiou2 合作方版）所需的协议原语：
// 待签名串构造、RSA(MD5 摘要) 签名/验签、GBK 编解码、请求 XML 组装、请求体双重 urlencode。
//
// 逐条对齐 epay plugins/fuiou2/inc/PayService.class.php：
//   - 公共参数：version=1.0 / ins_cd(机构号) / mchnt_cd(商户号) / term_id=88888888 / random_str。
//   - 待签名串：ksort → 跳过 sign 与以 "reserved" 开头的键 → 数组值置空 → k=v& 拼接去尾 &。
//   - 签名：openssl_sign(OPENSSL_ALGO_MD5) = RSA-PKCS1v15 over MD5(data)，结果 base64；商户私钥签、富友公钥验。
//   - 编码：请求参数值先 UTF-8→GBK 再签名与拼 XML；应答/回调为 GBK，验签在 GBK 字节上进行。
//   - 报文：<?xml ... encoding="GBK"?><xml><k>v</k>...</xml>，POST body = "req=" + urlencode(urlencode(xml))。
//
// 不引第三方富友 SDK：与 pkg/sign、pkg/wxpayv2 一致，标准库 + golang.org/x/text 手写，便于审计。
package fuiou

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/epvia/api/pkg/sign"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// SignContent 构造待签名串（对齐 PayService::getSignContent）：
// ksort → 跳过 sign 键与 "reserved" 前缀键 → k=v& 拼接 → 去尾部 &。
func SignContent(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		if k == "sign" || strings.HasPrefix(k, "reserved") {
			continue
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
		b.WriteByte('&')
	}
	return strings.TrimRight(b.String(), "&")
}

// Sign 商户私钥对待签名串做 RSA(MD5) 签名并 base64（对齐 rsaPrivateSign / OPENSSL_ALGO_MD5）。
// privKey 支持 epay 单行 base64 或标准 PEM（PKCS#1/PKCS#8）。
func Sign(content, privKey string) (string, error) {
	priv, err := sign.ParseRSAPrivateKey(privKey)
	if err != nil {
		return "", fmt.Errorf("富友商户私钥解析失败: %w", err)
	}
	h := md5.Sum([]byte(content))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.MD5, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// Verify 富友平台公钥验签（对齐 rsaPubilcSign / OPENSSL_ALGO_MD5）。
// content 为待签名串，sigB64 为 params["sign"]，pubKey 支持单行 base64 或 PEM。
func Verify(content, sigB64, pubKey string) bool {
	if sigB64 == "" {
		return false
	}
	pub, err := sign.ParseRSAPublicKey(pubKey)
	if err != nil {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return false
	}
	h := md5.Sum([]byte(content))
	return rsa.VerifyPKCS1v15(pub, crypto.MD5, h[:], raw) == nil
}

// NonceStr 生成指定长度的随机串（小写字母+数字），用于富友 random_str。
func NonceStr(n int) (string, error) {
	if n <= 0 {
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

// ToGBK 把 UTF-8 字符串转 GBK 字节串（对齐 mb_convert_encoding($v,'GBK','UTF-8')）。转换失败回退原串。
func ToGBK(s string) string {
	if s == "" {
		return s
	}
	out, err := simplifiedchinese.GBK.NewEncoder().String(s)
	if err != nil {
		return s
	}
	return out
}

// FromGBK 把 GBK 字节串转 UTF-8（解析应答/回调里的中文）。转换失败回退原串。
func FromGBK(s string) string {
	if s == "" {
		return s
	}
	out, err := simplifiedchinese.GBK.NewDecoder().String(s)
	if err != nil {
		return s
	}
	return out
}

// BuildRequestBody 组装富友请求体：值 UTF-8→GBK → 签名 → XML(GBK 声明) → "req="+双重 urlencode。
// 返回可直接作为 application/x-www-form-urlencoded body 提交的字节串。
func BuildRequestBody(params map[string]string, privKey string) (string, error) {
	// 1. 值转 GBK（对齐 submit() 里 foreach mb_convert_encoding）
	gbk := make(map[string]string, len(params))
	for k, v := range params {
		gbk[k] = ToGBK(v)
	}
	// 2. 待签名串 + 签名
	sig, err := Sign(SignContent(gbk), privKey)
	if err != nil {
		return "", err
	}
	gbk["sign"] = sig
	// 3. 组 XML
	xmlBody := "<?xml version=\"1.0\" encoding=\"GBK\" standalone=\"yes\"?><xml>" + toXML(gbk) + "</xml>"
	// 4. req= + 双重 urlencode（对齐 'req='.urlencode(urlencode($xml))）
	return "req=" + urlEncode(urlEncode(xmlBody)), nil
}

// toXML 组装平铺 <k>v</k>（对齐 PayService::toXml，值已是 GBK）。按 key 字典序输出以可预测。
func toXML(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString("<" + k + ">")
		b.WriteString(data[k])
		b.WriteString("</" + k + ">")
	}
	return b.String()
}

// urlEncode 对齐 PHP urlencode（空格→+，RFC1738）。
func urlEncode(s string) string {
	const upperhex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.':
			b.WriteByte(c)
		case c == ' ':
			b.WriteByte('+')
		default:
			b.WriteByte('%')
			b.WriteByte(upperhex[c>>4])
			b.WriteByte(upperhex[c&0x0f])
		}
	}
	return b.String()
}

// ParseResponseXML 解析富友 XML 应答/回调（GBK）为 map[string]string（值转回 UTF-8，对齐 PHP simplexml 行为）。
// 富友应答 body 是 urldecode 后的 <xml>...</xml>；回调 req 亦为 urldecode 后的同结构 XML。
func ParseResponseXML(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("富友应答为空")
	}
	// 富友报文声明 GBK：先整体 GBK→UTF-8（ASCII 无损），再去 XML 声明按平铺解析。
	decoded := FromGBK(raw)
	out, err := parseFlatXML(decoded)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, errors.New("富友应答解析为空")
	}
	return out, nil
}

// VerifyMap 验签一份已解析为 UTF-8 的富友报文（应答或回调）。
// 对齐 verifyResponse：把各值 UTF-8→GBK 还原成签名时的字节，再构造待签名串用平台公钥验
// （ASCII 值不受 GBK 还原影响，故 notify 与 response 统一走此路径）。
func VerifyMap(params map[string]string, pubKey string) bool {
	gbk := make(map[string]string, len(params))
	for k, v := range params {
		gbk[k] = ToGBK(v)
	}
	return Verify(SignContent(gbk), gbk["sign"], pubKey)
}

// parseFlatXML 解析平铺 <xml><k>v</k>...</xml>（值为 UTF-8）。忽略 <?xml?> 声明与顶层 xml 包裹。
func parseFlatXML(raw string) (map[string]string, error) {
	dec := xml.NewDecoder(strings.NewReader(raw))
	// 报文已在 FromGBK 阶段转为 UTF-8，但声明仍写 encoding="GBK"；用透传 CharsetReader 避免解码器报未知字符集。
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
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
			if depth == 2 {
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
	return out, nil
}
