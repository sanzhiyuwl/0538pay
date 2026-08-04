package wxpayv2

import (
	"strings"
	"testing"

	"github.com/epvia/api/internal/channel"
)

// TestPickForm APIv2 门面分派决策表：在 APIv3 决策基础上多一个付款码(6/scan/auth_code)分支。
func TestPickForm(t *testing.T) {
	all := []string{formNative, formJSAPI, formH5, formAPP, formMicro} // 全形态开通（含付款码）
	cases := []struct {
		name     string
		device   string
		method   string
		authCode string
		apptype  []string
		want     string
	}{
		// —— 买家场景分派（全形态开通）——
		{"PC→Native", "", "", "", all, formNative},
		{"微信内→JSAPI", "wechat", "", "", all, formJSAPI},
		{"手机→H5", "mobile", "", "", all, formH5},
		// —— API 显式 method 最高优先 ——
		{"显式jsapi", "", "jsapi", "", all, formJSAPI},
		{"显式applet走jsapi", "", "applet", "", all, formJSAPI},
		{"显式app", "mobile", "app", "", all, formAPP},
		// —— 付款码分支（APIv2 独有）——
		{"显式scan→付款码", "", "scan", "", all, formMicro},
		{"显式scan但未开付款码→空", "", "scan", "", []string{formNative}, ""},
		{"带auth_code→付款码", "", "", "134567890", all, formMicro},
		{"带auth_code但未开付款码→退场景", "", "", "134567890", []string{formNative}, formNative},
		// —— apptype 门控回退 ——
		{"微信内但只开Native→退Native", "wechat", "", "", []string{formNative}, formNative},
		{"手机开APP无H5→APP", "mobile", "", "", []string{formNative, formAPP}, formAPP},
		{"无任何开通形态→空", "", "", "", []string{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := pickForm(channel.CreateReq{Device: c.device, Method: c.method, AuthCode: c.authCode}, c.apptype)
			if got != c.want {
				t.Errorf("device=%q method=%q authCode=%q apptype=%v: 期望 %q, 实得 %q",
					c.device, c.method, c.authCode, c.apptype, c.want, got)
			}
		})
	}
}

// TestParseAppTypes 校验 apptype 串解析。
func TestParseAppTypes(t *testing.T) {
	got := parseAppTypes(" 1, 2 ,,6 ")
	if strings.Join(got, ",") != "1,2,6" {
		t.Errorf("解析结果应为 [1 2 6]，实得 %v", got)
	}
	if parseAppTypes("") != nil {
		t.Error("空串应返回 nil")
	}
}

// TestFormToDelegateKey 校验形态编码映射到正确的形态包 key（APIv2，含付款码）。
func TestFormToDelegateKey(t *testing.T) {
	want := map[string]string{
		formNative: "wxv2native", formJSAPI: "wxv2jsapi", formH5: "wxv2h5",
		formAPP: "wxv2app", formMicro: "wxv2micro",
	}
	for form, key := range want {
		if formToDelegateKey[form] != key {
			t.Errorf("形态 %s 应委托到 %s，实得 %s", form, key, formToDelegateKey[form])
		}
	}
}

// TestRegistered wxpayv2 门面通过 init() 自注册。
func TestRegistered(t *testing.T) {
	if _, ok := channel.Get("wxpayv2"); !ok {
		t.Fatal("wxpayv2 未注册到 registry")
	}
}
