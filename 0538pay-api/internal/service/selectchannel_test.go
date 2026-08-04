package service

import (
	"testing"

	"github.com/epvia/api/internal/model"
	"github.com/shopspring/decimal"
)

// TestChannelAllowsMerchant 校验商户白名单过滤（自研扩展）：空=放行；非空按 uid 命中；uid=0 有名单时拒绝。
func TestChannelAllowsMerchant(t *testing.T) {
	cases := []struct {
		wl   string
		uid  uint
		want bool
	}{
		{"", 1000, true},        // 空名单：不限制
		{"", 0, true},           // 空名单 + 无上下文：仍放行
		{"1000", 1000, true},    // 命中
		{"1000", 1001, false},   // 未命中
		{"1000,1002", 1002, true}, // 多商户命中
		{" 1000 , 1002 ", 1000, true}, // 带空格命中
		{"1000", 0, false},      // 有名单但无商户上下文：拒绝
		{"1000", 999, false},    // 未命中
	}
	for _, c := range cases {
		got := channelAllowsMerchant(&model.Channel{MerchantWhitelist: c.wl}, c.uid)
		if got != c.want {
			t.Errorf("channelAllowsMerchant(wl=%q,uid=%d)=%v，期望%v", c.wl, c.uid, got, c.want)
		}
	}
}

// TestNormalizeMerchantWhitelist 校验白名单规整：去空段/去空格/去重/校验正整数。
func TestNormalizeMerchantWhitelist(t *testing.T) {
	ok := map[string]string{
		"":                  "",
		"1000":              "1000",
		" 1000 , 1002 ":     "1000,1002",
		"1000,,1002,":       "1000,1002",
		"1000,1000,1002":    "1000,1002", // 去重
	}
	for in, want := range ok {
		got, err := normalizeMerchantWhitelist(in)
		if err != nil {
			t.Errorf("normalizeMerchantWhitelist(%q) 意外报错: %v", in, err)
		}
		if got != want {
			t.Errorf("normalizeMerchantWhitelist(%q)=%q，期望%q", in, got, want)
		}
	}
	for _, bad := range []string{"abc", "1000,x", "-5", "0"} {
		if _, err := normalizeMerchantWhitelist(bad); err == nil {
			t.Errorf("normalizeMerchantWhitelist(%q) 应报错", bad)
		}
	}
}

// TestParseGroupInfo 校验用户组 info 解析：typeid 键 + 分配对象。
func TestParseGroupInfo(t *testing.T) {
	m := parseGroupInfo(`{"1":{"type":"","channel":"-1","rate":""},"2":{"type":"roll","channel":"101","rate":"1.5"}}`)
	if m == nil {
		t.Fatal("应解析出分配图")
	}
	if m[1].Channel != "-1" {
		t.Errorf("type1 channel=%q 期望 -1", m[1].Channel)
	}
	if m[2].Type != "roll" || m[2].Channel != "101" || m[2].Rate != "1.5" {
		t.Errorf("type2 解析错误：%+v", m[2])
	}
	// 空串/非法 → nil
	if parseGroupInfo("") != nil || parseGroupInfo("not-json") != nil || parseGroupInfo("[]") != nil {
		t.Error("空/非法/非对象 应返回 nil")
	}
	// 旧格式（费率数组）应视为无分配（走无组随机）
	if parseGroupInfo(`[{"label":"支付宝","rate":"1"}]`) != nil {
		t.Error("旧费率数组格式应返回 nil")
	}
}

// TestParseRollInfo 校验轮询组 info 串解析（两种格式）。
func TestParseRollInfo(t *testing.T) {
	weighted := parseRollInfo("12:3,15:7")
	if len(weighted) != 2 || weighted[0].ChannelID != 12 || weighted[0].Weight != 3 || weighted[1].Weight != 7 {
		t.Errorf("权重格式解析错误：%+v", weighted)
	}
	plain := parseRollInfo("12,15,18")
	if len(plain) != 3 || plain[2].ChannelID != 18 || plain[2].Weight != 0 {
		t.Errorf("顺序格式解析错误：%+v", plain)
	}
	// 空段/非法段跳过
	got := parseRollInfo("12,,x,20:5")
	if len(got) != 2 || got[0].ChannelID != 12 || got[1].ChannelID != 20 {
		t.Errorf("容错解析错误：%+v", got)
	}
	if parseRollInfo("") != nil {
		t.Error("空串应返回 nil")
	}
}

// TestBuildRollInfo 校验反向拼串（保存轮询组通道用）。
func TestBuildRollInfo(t *testing.T) {
	members := []rollMember{{ChannelID: 12, Weight: 3}, {ChannelID: 15, Weight: 0}}
	if got := buildRollInfo(1, members); got != "12:3,15:1" { // 权重<=0 补 1
		t.Errorf("权重串=%q 期望 12:3,15:1", got)
	}
	if got := buildRollInfo(0, members); got != "12,15" {
		t.Errorf("顺序串=%q 期望 12,15", got)
	}
}

// TestRandomWeight 校验加权随机命中区间（用确定随机源逐点验证）。
func TestRandomWeight(t *testing.T) {
	members := []rollMember{{ChannelID: 12, Weight: 3}, {ChannelID: 15, Weight: 7}} // 和=10
	sel := &ChannelSelector{}
	// r = randIntn(10)+1 ∈ [1,10]；r<=3 命中12，否则命中15。
	cases := map[int]int{0: 12, 2: 12, 3: 15, 9: 15} // randIntn 返回值 → 期望通道
	for rv, want := range cases {
		sel.randIntn = func(n int) int { return rv }
		if got := sel.randomWeight(members); got != want {
			t.Errorf("randIntn=%d → 通道%d，期望%d", rv, got, want)
		}
	}
	// 权重和<=0 返回 0
	sel.randIntn = func(n int) int { return 0 }
	if got := sel.randomWeight([]rollMember{{ChannelID: 1, Weight: 0}}); got != 0 {
		t.Errorf("零权重和应返回0，得 %d", got)
	}
}

// TestParseRateOverride 校验组级费率覆盖解析。
func TestParseRateOverride(t *testing.T) {
	if d, ok := parseRateOverride("1.5"); !ok || !d.Equal(decimal.RequireFromString("1.5")) {
		t.Errorf("1.5 解析错误：%v %v", d, ok)
	}
	if _, ok := parseRateOverride(""); ok {
		t.Error("空串应无覆盖")
	}
	if _, ok := parseRateOverride("abc"); ok {
		t.Error("非法值应无覆盖")
	}
	if _, ok := parseRateOverride("-1"); ok {
		t.Error("负值应无覆盖")
	}
}
