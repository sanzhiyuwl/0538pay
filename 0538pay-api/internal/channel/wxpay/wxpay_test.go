package wxpay

import (
	"strings"
	"testing"

	"github.com/epvia/api/internal/channel"
)

// TestPickForm 分派决策表：给定买家场景(device)/显式(method)/通道开通形态(apptype)，
// 断言门面选中的形态编码正确（对齐 epay wxpayn submit()/mapi() 决策表）。
func TestPickForm(t *testing.T) {
	all := []string{formNative, formJSAPI, formH5, formAPP} // 全形态开通
	cases := []struct {
		name    string
		device  string
		method  string
		apptype []string
		want    string
	}{
		// —— 买家场景分派（全形态开通）——
		{"PC→Native", "", "", all, formNative},
		{"微信内→JSAPI", "wechat", "", all, formJSAPI},
		{"手机→H5", "mobile", "", all, formH5},
		// —— API 显式 method 最高优先 ——
		{"显式jsapi", "", "jsapi", all, formJSAPI},
		{"显式applet走jsapi", "", "applet", all, formJSAPI},
		{"显式app", "mobile", "app", all, formAPP},
		{"APIv3显式scan报错", "", "scan", all, ""},
		// —— apptype 门控回退 ——
		{"微信内但只开Native→退Native", "wechat", "", []string{formNative}, formNative},
		{"手机但只开Native→退Native", "mobile", "", []string{formNative}, formNative},
		{"手机开APP无H5→APP", "mobile", "", []string{formNative, formAPP}, formH5Fallback(formAPP)},
		{"PC只开JSAPI→兜底JSAPI", "", "", []string{formJSAPI}, formJSAPI},
		{"显式jsapi但未开→回退场景", "wechat", "jsapi", []string{formNative}, formNative},
		{"无任何开通形态→空", "", "", []string{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := pickForm(channel.CreateReq{Device: c.device, Method: c.method}, c.apptype)
			if got != c.want {
				t.Errorf("device=%q method=%q apptype=%v: 期望形态 %q, 实得 %q",
					c.device, c.method, c.apptype, c.want, got)
			}
		})
	}
}

// formH5Fallback 手机场景：开了 APP 没开 H5 时应命中 APP（H5 优先但缺失则降 APP）。
func formH5Fallback(want string) string { return want }

// TestParseAppTypes 校验 apptype 串解析（逗号分隔、去空白、去空项）。
func TestParseAppTypes(t *testing.T) {
	got := parseAppTypes(" 1, 2 ,,3 ")
	if strings.Join(got, ",") != "1,2,3" {
		t.Errorf("解析结果应为 [1 2 3]，实得 %v", got)
	}
	if parseAppTypes("") != nil || parseAppTypes("  ") != nil {
		t.Error("空串应返回 nil")
	}
}

// TestFormToDelegateKey 校验形态编码映射到正确的形态包 key（APIv3）。
func TestFormToDelegateKey(t *testing.T) {
	want := map[string]string{formNative: "wxnative", formJSAPI: "wxjsapi", formH5: "wxh5", formAPP: "wxapp"}
	for form, key := range want {
		if formToDelegateKey[form] != key {
			t.Errorf("形态 %s 应委托到 %s，实得 %s", form, key, formToDelegateKey[form])
		}
	}
}
