package service

import "testing"

// TestDeviceVisible 校验 B1-63 支付方式设备过滤：device=0 两端可见，1 仅PC，2 仅移动。
func TestDeviceVisible(t *testing.T) {
	cases := []struct {
		device int8
		mobile bool
		want   bool
	}{
		{0, false, true}, // 通用-PC 可见
		{0, true, true},  // 通用-移动 可见
		{1, false, true}, // 仅PC-PC 可见
		{1, true, false}, // 仅PC-移动 隐藏
		{2, false, false}, // 仅移动-PC 隐藏
		{2, true, true},  // 仅移动-移动 可见
	}
	for _, c := range cases {
		if got := deviceVisible(c.device, c.mobile); got != c.want {
			t.Errorf("deviceVisible(device=%d,mobile=%v)=%v 期望 %v", c.device, c.mobile, got, c.want)
		}
	}
}

// TestIsMobileDevice 校验 UA/device 标识的移动端判定（B1-63 设备来源）。
func TestIsMobileDevice(t *testing.T) {
	for _, d := range []string{"mobile", "wechat", "alipay", "qq", "wap", "jsapi", "h5"} {
		if !isMobileDevice(d) {
			t.Errorf("isMobileDevice(%q) 应为 true", d)
		}
	}
	for _, d := range []string{"", "pc", "web", "unknown"} {
		if isMobileDevice(d) {
			t.Errorf("isMobileDevice(%q) 应为 false", d)
		}
	}
}
