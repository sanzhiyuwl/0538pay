package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

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
