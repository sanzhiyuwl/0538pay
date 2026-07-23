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
