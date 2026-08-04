package channel

import "testing"

// TestDelegateFormPackages 校验 9 个微信形态包已标记 Delegate（退役，仅作门面委托目标，
// 后端 saveChannel 据此拒绝直接建通道，前端下拉据此过滤——单一数据源）。
func TestDelegateFormPackages(t *testing.T) {
	delegated := []string{
		"wxnative", "wxjsapi", "wxh5", "wxapp",
		"wxv2native", "wxv2jsapi", "wxv2h5", "wxv2app", "wxv2micro",
	}
	for _, k := range delegated {
		if !Describe(k).Delegate {
			t.Errorf("形态包 %s 应标记 Delegate=true（已退役并入聚合门面）", k)
		}
	}
	// 聚合门面本身不是委托目标，必须可直接建通道。
	for _, k := range []string{"wxpay", "wxpayv2"} {
		if Describe(k).Delegate {
			t.Errorf("聚合门面 %s 不应标记 Delegate", k)
		}
	}
}
