// Package chsign 提供「渠道对接」通用签名工具，区别于 pkg/sign（易支付协议专用）。
//
// 聚合类第三方渠道（ltzf 蓝兔 / swiftpass2 威富通 / xunhupay 虎皮椒 / vmq 等）多用 MD5 签名，
// 算法共性是「ksort 参数 → 拼 k=v&… → 接密钥 → md5」，但在几个细节上各家不同：
//   - 尾部密钥形态：key=KEY（ltzf 带前缀）/ &key=KEY（swiftpass2）/ 直接拼 KEY（xunhupay/微信风格）
//   - 结果大小写：大写（ltzf/swiftpass2）/ 小写（xunhupay/vmq）
//   - 跳过的签名字段名：sign / hash 等
//   - 空值是否参与：多数跳过空串
//   - 是否只签白名单字段：ltzf 下单只签指定字段子集
//
// 本包用 MD5Signer + 选项表达这些差异，接聚合类渠道时按其文档配置即可，不必每个渠道重写签名。
// 逐字节对齐 epay 各插件 inc/*Client 的 make_sign/generate_hash/makeSign。
package chsign

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
)

// KeyMode 尾部密钥的拼接形态。
type KeyMode int

const (
	// KeyAppend 直接把密钥拼在签名串尾部：md5(a=1&b=2 + KEY)。
	// 对齐 xunhupay generate_hash、微信风格 vmq（vmq 另有值直拼变体，见 channel/vmq）。
	KeyAppend KeyMode = iota
	// KeyEqPrefix 以 key= 前缀拼接：md5(a=1&b=2& + "key=" + KEY)。对齐 ltzf make_sign。
	KeyEqPrefix
	// KeyAmpEq 以 &key= 前缀拼接：md5(a=1&b=2 + "&key=" + KEY)。对齐 swiftpass2 makeSign。
	KeyAmpEq
)

// MD5Signer 描述一个聚合类渠道的 MD5 签名规则。零值即「ksort + k=v& + 直接拼 KEY + 小写」。
type MD5Signer struct {
	Key      string   // 商户通讯密钥
	Mode     KeyMode  // 尾部密钥形态
	Upper    bool     // 结果是否转大写（false=小写）
	SignKey  string   // 待剔除的签名字段名（默认 "sign"，如 xunhupay 用 "hash"）
	OnlyKeys []string // 非空时只签这些字段（白名单，对齐 ltzf $sign_param）；为空则签全部非空字段
}

// signField 返回实际用于剔除的签名字段名。
func (s MD5Signer) signField() string {
	if s.SignKey != "" {
		return s.SignKey
	}
	return "sign"
}

// Content 生成待签名字符串（不含尾部密钥）：ksort → 跳过空值/签名字段（/非白名单）→ k=v& 拼接（去尾 &）。
func (s MD5Signer) Content(params map[string]string) string {
	var allow map[string]struct{}
	if len(s.OnlyKeys) > 0 {
		allow = make(map[string]struct{}, len(s.OnlyKeys))
		for _, k := range s.OnlyKeys {
			allow[k] = struct{}{}
		}
	}
	sf := s.signField()

	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == sf {
			continue
		}
		if strings.TrimSpace(v) == "" { // 跳过空值（对齐各家 $v!==''）
			continue
		}
		if allow != nil {
			if _, ok := allow[k]; !ok {
				continue
			}
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

// Sign 生成签名：按 Mode 拼接尾部密钥后 md5，按 Upper 决定大小写。
func (s MD5Signer) Sign(params map[string]string) string {
	content := s.Content(params)
	var raw string
	switch s.Mode {
	case KeyEqPrefix, KeyAmpEq:
		// ltzf 源码是「每字段带 &」再拼 key=KEY，即 a=1&b=2&key=KEY；
		// swiftpass2 是「去尾 &」再拼 &key=KEY，即 a=1&b=2&key=KEY。
		// 两者最终串一致，故统一为 Content(去尾&) + "&key=" + KEY。
		raw = content + "&key=" + s.Key
	default: // KeyAppend
		raw = content + s.Key
	}
	sum := md5.Sum([]byte(raw))
	out := hex.EncodeToString(sum[:])
	if s.Upper {
		return strings.ToUpper(out)
	}
	return out
}

// Verify 校验 params[signField] 是否匹配（大小写不敏感，容忍上送大写/小写）。
func (s MD5Signer) Verify(params map[string]string) bool {
	got := params[s.signField()]
	if got == "" {
		return false
	}
	return strings.EqualFold(got, s.Sign(params))
}
