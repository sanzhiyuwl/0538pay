package service

import "testing"

// TestResolveInternalChannel 覆盖内部订单收款通道解析（七相平台统一内部收款，自研扩展）。
//   - 支付方式(alipay/wxpay) 或空：走配置 internal_pay_plugin（默认 qixiang），payMethod=该方式（空兜底 alipay）。
//   - 真实通道 key（mock 等）：通道即其本身，payMethod 同值。
func TestResolveInternalChannel(t *testing.T) {
	cases := []struct {
		name        string
		cfg         map[string]string
		in          string
		wantChannel string
		wantMethod  string
	}{
		{"空默认七相支付宝", map[string]string{}, "", "qixiang", "alipay"},
		{"微信走七相", map[string]string{}, "wxpay", "qixiang", "wxpay"},
		{"支付宝走七相", map[string]string{}, "alipay", "qixiang", "alipay"},
		{"配置改内部通道", map[string]string{"internal_pay_plugin": "epay"}, "wxpay", "epay", "wxpay"},
		{"真实通道key直用", map[string]string{}, "mock", "mock", "mock"},
		{"带空格支付方式", map[string]string{}, "  wxpay ", "qixiang", "wxpay"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &PayService{cfg: &ConfigService{cache: c.cfg}}
			gotCh, gotM := s.resolveInternalChannel(c.in)
			if gotCh != c.wantChannel || gotM != c.wantMethod {
				t.Fatalf("resolveInternalChannel(%q)=(%q,%q), want (%q,%q)", c.in, gotCh, gotM, c.wantChannel, c.wantMethod)
			}
		})
	}
}

// TestResolveInternalChannelNilCfg cfg 为 nil 时兜底七相，不 panic。
func TestResolveInternalChannelNilCfg(t *testing.T) {
	s := &PayService{}
	gotCh, gotM := s.resolveInternalChannel("wxpay")
	if gotCh != "qixiang" || gotM != "wxpay" {
		t.Fatalf("nil cfg fallback=(%q,%q), want (qixiang,wxpay)", gotCh, gotM)
	}
}
