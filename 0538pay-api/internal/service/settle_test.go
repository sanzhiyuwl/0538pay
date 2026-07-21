package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

// TestCalcFee 校验结算手续费算法逐项对齐 epay：
// fee = round(money*rate/100, 2)，clamp[fee_min, fee_max]，realmoney = money - fee。
// 依赖默认配置 rate=0.5% / fee_min=0.1 / fee_max=20。
func TestCalcFee(t *testing.T) {
	cases := []struct {
		money    string
		wantFee  string
		wantReal string
		desc     string
	}{
		{"100.00", "0.50", "99.50", "常规：100*0.5%=0.5"},
		{"10.00", "0.10", "9.90", "封底：10*0.5%=0.05<0.1，取0.1"},
		{"5000.00", "20.00", "4980.00", "封顶：5000*0.5%=25>20，取20"},
		{"200.00", "1.00", "199.00", "常规：200*0.5%=1.0"},
		{"30.00", "0.15", "29.85", "门槛额：30*0.5%=0.15"},
	}
	for _, c := range cases {
		money := decimal.RequireFromString(c.money)
		fee, real := calcFee(money)
		if fee.StringFixed(2) != c.wantFee {
			t.Errorf("%s: calcFee(%s) fee=%s, 期望 %s", c.desc, c.money, fee.StringFixed(2), c.wantFee)
		}
		if real.StringFixed(2) != c.wantReal {
			t.Errorf("%s: calcFee(%s) real=%s, 期望 %s", c.desc, c.money, real.StringFixed(2), c.wantReal)
		}
	}
}

// TestCalcFeeExtremeSmall 极小额保护：手续费不应超过结算金额本身。
func TestCalcFeeExtremeSmall(t *testing.T) {
	// 0.05 元：0.5% 后不足封底 0.1，但 0.1>0.05，手续费被压到不超过本金
	fee, real := calcFee(decimal.RequireFromString("0.05"))
	if fee.GreaterThan(decimal.RequireFromString("0.05")) {
		t.Errorf("极小额手续费 %s 超过了本金 0.05", fee.StringFixed(2))
	}
	if real.IsNegative() {
		t.Errorf("实际到账不应为负：%s", real.StringFixed(2))
	}
}
