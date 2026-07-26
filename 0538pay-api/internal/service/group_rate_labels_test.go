package service

import "testing"

// TestGroupRateLabels 校验会员套餐「可用支付通道及费率」标签解析（对齐 epay groupbuy.php display_info）：
//   - epay map 格式 {"支付方式ID":{channel,rate}}；channel=="0"/"" 跳过（未开通）；
//   - 其余（-1随机 / -2子通道 / 正整数固定/轮询）显示「支付方式名(费率%)」；
//   - 组级 rate 存「到账比例」，商户端展示手续费率 = 100-rate（对齐 epay `round(100-rate)`）；
//   - rate 为空或非法 → 展示为空串（随通道默认费率）；
//   - 支付方式名录缺失 → 回退「支付方式#ID」；输出按ID升序稳定。
func TestGroupRateLabels(t *testing.T) {
	names := map[int]string{1: "支付宝", 2: "微信支付", 3: "QQ钱包", 4: "云闪付"}
	// rate 为到账比例：微信 99.8 → 手续费 0.2%。
	info := `{"1":{"type":"channel","channel":"12","rate":""},"2":{"type":"channel","channel":"13","rate":"99.8"},"3":{"type":"","channel":"-1","rate":""},"4":{"type":"","channel":"0","rate":""}}`

	items := groupRateLabels(info, names)
	// channel=0 的「云闪付」应被跳过 → 3 项。
	if len(items) != 3 {
		t.Fatalf("期望 3 项（云闪付 channel=0 跳过），实际 %d：%+v", len(items), items)
	}
	// 升序：支付宝(空费率) / 微信支付(到账99.8 → 手续费0.2) / QQ钱包(空)。
	want := []struct {
		label, rate string
	}{
		{"支付宝", ""},
		{"微信支付", "0.2"},
		{"QQ钱包", ""},
	}
	for i, w := range want {
		if items[i].Label != w.label || items[i].Rate != w.rate {
			t.Errorf("第 %d 项期望 {%s,%q}，实际 {%s,%q}", i, w.label, w.rate, items[i].Label, items[i].Rate)
		}
	}
}

// TestGroupRateLabelsFallback 空 info / 名录缺失 / 旧数组格式 的兜底行为。
func TestGroupRateLabelsFallback(t *testing.T) {
	// 空 info → 空切片（非 nil，前端 v-for 安全）。
	if got := groupRateLabels("", nil); got == nil || len(got) != 0 {
		t.Errorf("空 info 应返回空切片，实际 %+v", got)
	}
	// 全 channel=0 → 空。
	if got := groupRateLabels(`{"1":{"channel":"0","rate":""}}`, map[int]string{1: "支付宝"}); len(got) != 0 {
		t.Errorf("全未开通应返回空，实际 %+v", got)
	}
	// 名录缺失 → 回退「支付方式#ID」；rate=99.5(到账) → 手续费 0.5%。
	got := groupRateLabels(`{"5":{"channel":"-1","rate":"99.5"}}`, map[int]string{})
	if len(got) != 1 || got[0].Label != "支付方式#5" || got[0].Rate != "0.5" {
		t.Errorf("名录缺失应回退支付方式#5，实际 %+v", got)
	}
	// 数组格式兼容：rate=99.62(到账) → 手续费 0.38%。
	got = groupRateLabels(`[{"label":"微信支付","rate":"99.62"}]`, nil)
	if len(got) != 1 || got[0].Label != "微信支付" || got[0].Rate != "0.38" {
		t.Errorf("数组格式应换算 100-rate，实际 %+v", got)
	}
}
