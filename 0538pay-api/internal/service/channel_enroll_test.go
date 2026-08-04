package service

import (
	"encoding/json"
	"testing"

	"github.com/epvia/api/pkg/sign"
)

// TestSubMchPlaceholderKey 校验从主通道 config 识别子商户号占位 key（0730 4.3 各渠道字段对照）。
func TestSubMchPlaceholderKey(t *testing.T) {
	// 微信服务商：sub_mchid 为占位符 → 命中。
	if k := subMchPlaceholderKey(`{"appid":"wx1","sub_mchid":"[sub_mchid]"}`); k != "sub_mchid" {
		t.Errorf("微信服务商应命中 sub_mchid，得到 %q", k)
	}
	// 富友：appmchid 为占位符 → 命中。
	if k := subMchPlaceholderKey(`{"appmchid":"[appmchid]","key":"SHARED"}`); k != "appmchid" {
		t.Errorf("富友应命中 appmchid，得到 %q", k)
	}
	// 非占位（固定值）→ 不命中（直连通道，无需进件）。
	if k := subMchPlaceholderKey(`{"sub_mchid":"1600001111"}`); k != "" {
		t.Errorf("固定值不应识别为进件通道，得到 %q", k)
	}
	// 无子商户号字段 → 不命中。
	if k := subMchPlaceholderKey(`{"appid":"[appid]","mch_id":"[mchid]"}`); k != "" {
		t.Errorf("无子商户号占位不应命中，得到 %q", k)
	}
	// 非法 JSON → 空。
	if k := subMchPlaceholderKey("not-json"); k != "" {
		t.Errorf("非法 JSON 应返回空，得到 %q", k)
	}
}

// TestMergeInfoKey 校验把子商户号写进子通道 info（保留其他键，只覆盖目标 key）。
func TestMergeInfoKey(t *testing.T) {
	// 空 info：新建仅含目标 key。
	got := mergeInfoKey("", "sub_mchid", "1600009999")
	var m map[string]string
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("结果非合法 JSON: %v", err)
	}
	if m["sub_mchid"] != "1600009999" {
		t.Errorf("sub_mchid 未写入：%q", m["sub_mchid"])
	}
	// 已有其他占位键：合并保留，只覆盖 sub_mchid。
	got2 := mergeInfoKey(`{"appid":"wxsub123","sub_mchid":"OLD"}`, "sub_mchid", "NEW")
	_ = json.Unmarshal([]byte(got2), &m)
	if m["appid"] != "wxsub123" {
		t.Errorf("其他键应保留：%q", m["appid"])
	}
	if m["sub_mchid"] != "NEW" {
		t.Errorf("sub_mchid 应被覆盖为 NEW：%q", m["sub_mchid"])
	}
}

// TestSubMchPlaceholderValueAndInnerKey 校验「占位符括号内 key ≠ config 键名」时，
// Approve 侧应按括号内 key 写 info，才能被下单占位替换 mergeSubChannelConfig 命中（问题3 修复回归）。
func TestSubMchPlaceholderValueAndInnerKey(t *testing.T) {
	// config 写成 "appmchid":"[fy_mchnt_cd]"：键名=appmchid，但括号内 key=fy_mchnt_cd。
	const cfg = `{"appmchid":"[fy_mchnt_cd]","key":"SHARED"}`
	subKey := subMchPlaceholderKey(cfg)
	if subKey != "appmchid" {
		t.Fatalf("应命中 config 键 appmchid，得到 %q", subKey)
	}
	innerKey := placeholderKey(subMchPlaceholderValue(cfg, subKey))
	if innerKey != "fy_mchnt_cd" {
		t.Fatalf("应解出括号内 key fy_mchnt_cd，得到 %q", innerKey)
	}
	// 按括号内 key 写 info，下单占位替换应命中真实号（钱直清到商户自己的号）。
	info := mergeInfoKey("", innerKey, "0002900F0000001")
	kv := mergeSubChannelConfig(cfg, info)
	if kv["appmchid"] != "0002900F0000001" {
		t.Errorf("占位替换应把 appmchid 换成商户自己的号，得到 %q（若为 [fy_mchnt_cd] 说明按键名写 info 落空）", kv["appmchid"])
	}
	// 括号内==键名（微信服务商常见）时也须成立。
	const cfg2 = `{"sub_mchid":"[sub_mchid]"}`
	ik2 := placeholderKey(subMchPlaceholderValue(cfg2, subMchPlaceholderKey(cfg2)))
	info2 := mergeInfoKey("", ik2, "1600009999")
	if mergeSubChannelConfig(cfg2, info2)["sub_mchid"] != "1600009999" {
		t.Error("微信服务商 sub_mchid 占位替换应命中")
	}
}

// TestChannelEnrollRSARoundtrip 校验敏感字段 RSA 加解密可逆（半自动审核：公钥加密落库→私钥解密报送）。
func TestChannelEnrollRSARoundtrip(t *testing.T) {
	priv, pub, err := sign.GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}
	plain := "440101199001011234" // 模拟身份证号
	cipher, err := sign.EncryptRSA(plain, pub)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if cipher == plain || cipher == "" {
		t.Fatal("密文不应等于明文或为空")
	}
	back, err := sign.DecryptRSA(cipher, priv)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if back != plain {
		t.Errorf("解密还原不一致：want %q got %q", plain, back)
	}
}
