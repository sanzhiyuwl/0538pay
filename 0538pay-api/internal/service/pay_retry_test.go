package service

import (
	"testing"
	"time"
)

// TestNotifyBackoff 校验退避档位逐一对齐 epay：1min/2min/16min/36min/1h，超过放弃(0)。
func TestNotifyBackoff(t *testing.T) {
	cases := map[int]time.Duration{
		1: 1 * time.Minute,
		2: 2 * time.Minute,
		3: 16 * time.Minute,
		4: 36 * time.Minute,
		5: 1 * time.Hour,
		6: 0, // 超过上限 → 放弃
		0: 0,
	}
	for n, want := range cases {
		if got := notifyBackoff(n); got != want {
			t.Errorf("notifyBackoff(%d)=%v, 期望 %v", n, got, want)
		}
	}
}

// TestMaxNotifyRetry 保证上限常量与退避表一致（第 5 次仍有间隔，第 6 次放弃）。
func TestMaxNotifyRetry(t *testing.T) {
	if notifyBackoff(maxNotifyRetry) == 0 {
		t.Fatal("第 maxNotifyRetry 次应仍有退避间隔")
	}
	if notifyBackoff(maxNotifyRetry+1) != 0 {
		t.Fatal("超过 maxNotifyRetry 应放弃(返回 0)")
	}
}
