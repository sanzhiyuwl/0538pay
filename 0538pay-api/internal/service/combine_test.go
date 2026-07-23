package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func fen(y float64) decimal.Decimal { return decimal.NewFromFloat(y) }

func sum(a []int64) int64 {
	var s int64
	for _, v := range a {
		s += v
	}
	return s
}

func TestCombineDisabled(t *testing.T) {
	// 未开启
	if got := CombineSubMoneys(100000, CombineConfig{Open: false, MinMoney: fen(1), SubMoney: fen(500)}); got != nil {
		t.Errorf("未开启应返回 nil，得 %v", got)
	}
	// minmoney 未配
	if got := CombineSubMoneys(100000, CombineConfig{Open: true, SubMoney: fen(500)}); got != nil {
		t.Errorf("minmoney 未配应返回 nil，得 %v", got)
	}
}

func TestCombineBelowThreshold(t *testing.T) {
	// minmoney=1000 元 → 100000 分；订单 99999 分 < 阈值不拆
	c := CombineConfig{Open: true, MinMoney: fen(1000), SubMoney: fen(500)}
	if got := CombineSubMoneys(99999, c); got != nil {
		t.Errorf("低于阈值应返回 nil，得 %v", got)
	}
}

func TestCombineBasicSplit(t *testing.T) {
	// minmoney=100 元(10000分)，submoney=500 元(50000分)。订单 300 元=30000 分。
	// 30000 ≥ 10000 触发；subnum=3，submoney=10000 ≤ 50000 不再增；mod=0 → [10000,10000,10000]。
	c := CombineConfig{Open: true, MinMoney: fen(100), SubMoney: fen(500)}
	got := CombineSubMoneys(30000, c)
	if len(got) != 3 {
		t.Fatalf("应拆 3 单，得 %d 单 %v", len(got), got)
	}
	if sum(got) != 30000 {
		t.Errorf("子单求和 %d 应等于 30000", sum(got))
	}
}

func TestCombineIncreaseSubnum(t *testing.T) {
	// submoney=100 元(10000分)上限较小 → 需增加子单数。订单 100000 分(1000元)。
	// 100000/3=33333 >10000 → 增到 subnum，直到 100000/subnum ≤ 10000。
	// 100000/10=10000 ≤10000 → subnum=10。
	c := CombineConfig{Open: true, MinMoney: fen(100), SubMoney: fen(100)}
	got := CombineSubMoneys(100000, c)
	if len(got) != 10 {
		t.Fatalf("应拆 10 单，得 %d 单", len(got))
	}
	if sum(got) != 100000 {
		t.Errorf("求和 %d 应等于 100000", sum(got))
	}
}

func TestCombineRemainderSpread(t *testing.T) {
	// 制造余数：money=100 分，minmoney=1 元(100分)，submoney=500 元。
	// 100≥100 触发；subnum=3，submoney=33，mod=1 → [34,33,33]，求和=100。
	c := CombineConfig{Open: true, MinMoney: fen(1), SubMoney: fen(500)}
	got := CombineSubMoneys(100, c)
	if len(got) != 3 || sum(got) != 100 {
		t.Fatalf("拆分错误 %v 求和 %d", got, sum(got))
	}
	if got[0] != 34 || got[1] != 33 || got[2] != 33 {
		t.Errorf("余数平摊错误 %v，应为 [34,33,33]", got)
	}
}

func TestCombineSubnumCap50(t *testing.T) {
	// submoney 极小(1分)逼到 50 单上限。money 很大。
	c := CombineConfig{Open: true, MinMoney: fen(1), SubMoney: fen(0.01)}
	got := CombineSubMoneys(1000000, c)
	if len(got) != 50 {
		t.Fatalf("应封顶 50 单，得 %d", len(got))
	}
	if sum(got) != 1000000 {
		t.Errorf("求和 %d 应等于 1000000（守恒）", sum(got))
	}
}
