package channel

import (
	"context"
	"testing"
)

// groupedChan 是仅用于测试的渠道，声明带 Group/NeedSign 的分组子产品，
// 验证 L-10 扩展的 ProductType 元数据经 Meta() 正确透传（对齐 ltzf select_xxx + alipay 签约门控）。
type groupedChan struct{}

func (groupedChan) Key() string { return "_grouped_test" }
func (groupedChan) Create(context.Context, Config, CreateReq) (CreateResp, error) {
	return CreateResp{}, nil
}
func (groupedChan) Query(context.Context, Config, string) (bool, error) { return false, nil }
func (groupedChan) Notify(context.Context, Config, map[string]string) (NotifyResult, error) {
	return NotifyResult{}, nil
}
func (groupedChan) Inputs() []FieldInput { return nil }
func (groupedChan) Products() []ProductType {
	return []ProductType{
		{Code: "wx_scan", Name: "扫码支付", Group: "wxpay"},
		{Code: "wx_h5", Name: "H5支付", Group: "wxpay"},
		{Code: "ali_scan", Name: "扫码支付", Group: "alipay"},
		{Code: "ali_preauth", Name: "预授权支付", Group: "alipay", NeedSign: true},
	}
}

func TestMetaCarriesGroupAndNeedSign(t *testing.T) {
	Register(groupedChan{})
	m, ok := Meta("_grouped_test")
	if !ok {
		t.Fatal("渠道未注册")
	}
	if !m.Configurable || len(m.Products) != 4 {
		t.Fatalf("元数据异常: configurable=%v products=%d", m.Configurable, len(m.Products))
	}
	// 验证分组与签约门控字段逐项透传。
	var sawWxGroup, sawNeedSign bool
	for _, p := range m.Products {
		if p.Group == "wxpay" {
			sawWxGroup = true
		}
		if p.Code == "ali_preauth" {
			if p.Group != "alipay" || !p.NeedSign {
				t.Errorf("ali_preauth 分组/签约门控丢失: %+v", p)
			}
			sawNeedSign = true
		}
	}
	if !sawWxGroup {
		t.Error("wxpay 分组未透传")
	}
	if !sawNeedSign {
		t.Error("NeedSign 未透传")
	}
}

// TestDescribeFallback 未登记的 key 回退用 key 兜底，不返回空串（保证前端永远有可显示名）。
func TestDescribeFallback(t *testing.T) {
	d := Describe("__not_registered__")
	if d.ShowName != "__not_registered__" || d.Brand != "__not_registered__" {
		t.Fatalf("未登记 key 应回退到 key 兜底, 实际 %+v", d)
	}
}

// TestMetaCarriesDescriptor Meta() 应把展示元数据（ShowName/Brand）透传进 PluginMeta。
func TestMetaCarriesDescriptor(t *testing.T) {
	Register(groupedChan{})
	// _grouped_test 未登记描述，应回退裸 key（不为空）。
	m, _ := Meta("_grouped_test")
	if m.ShowName != "_grouped_test" || m.Brand != "_grouped_test" {
		t.Fatalf("未登记渠道应回退裸 key, 实际 showname=%q brand=%q", m.ShowName, m.Brand)
	}
}
