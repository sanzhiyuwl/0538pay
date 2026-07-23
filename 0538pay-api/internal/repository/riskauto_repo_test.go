package repository

import "testing"

// TestAllUnpaidWithMinSample B1-69：通道自动关停核心算法「最近 failCount 单全未支付」。
// 对齐 epay cron.php:262-268：样本不足不关；样本充足且无 status>0 才关。
func TestAllUnpaidWithMinSample(t *testing.T) {
	cases := []struct {
		name      string
		statuses  []int8
		failCount int
		want      bool
	}{
		{"全未支付且样本充足→关", []int8{0, 0, 0}, 3, true},
		{"有一单成功→不关", []int8{0, 1, 0}, 3, false},
		{"样本不足→不关", []int8{0, 0}, 3, false},
		{"样本为空→不关", []int8{}, 3, false},
		{"failCount=0→不关(未开启)", []int8{0, 0, 0}, 0, false},
		{"退款态status=2也算已支付(>0)→不关", []int8{0, 2, 0}, 3, false},
		{"恰好等于failCount且全未付→关", []int8{0, 0}, 2, true},
	}
	for _, c := range cases {
		if got := AllUnpaidWithMinSample(c.statuses, c.failCount); got != c.want {
			t.Errorf("%s: AllUnpaidWithMinSample(%v,%d)=%v 期望 %v", c.name, c.statuses, c.failCount, got, c.want)
		}
	}
}
