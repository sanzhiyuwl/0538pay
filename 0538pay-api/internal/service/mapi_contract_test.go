package service

import "testing"

// TestCashierFallbackURL 校验 B1-18 收银台回落地址拼接（对齐 epay siteurl.'pay/submit/TRADE_NO/'）。
func TestCashierFallbackURL(t *testing.T) {
	cases := []struct {
		site, trade, want string
	}{
		{"https://pay.example.com", "20260723001", "https://pay.example.com/pay/submit/20260723001/"},
		{"https://pay.example.com/", "20260723001", "https://pay.example.com/pay/submit/20260723001/"}, // 去重尾斜杠
		{"", "20260723001", "/pay/submit/20260723001/"},                                                // 站点空回落相对路径
	}
	for _, c := range cases {
		if got := cashierFallbackURL(c.site, c.trade); got != c.want {
			t.Errorf("cashierFallbackURL(%q,%q)=%q 期望 %q", c.site, c.trade, got, c.want)
		}
	}
}
