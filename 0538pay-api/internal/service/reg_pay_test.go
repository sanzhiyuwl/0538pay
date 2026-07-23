package service

import (
	"encoding/json"
	"testing"
)

// TestRegPayInfoRoundTrip 校验 B1-51 付费注册信息在订单 param 里的序列化/反序列化闭环
// （对齐 epay CACHE reg_<trade_no> 承载 verifytype/email/phone/pwd/upid/invitecodeid）。
func TestRegPayInfoRoundTrip(t *testing.T) {
	in := regPayInfo{Email: "a@b.com", Phone: "", Pwd: "secret6", UpID: 12, InviteID: 3}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out regPayInfo
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Email != in.Email || out.Pwd != in.Pwd || out.UpID != in.UpID || out.InviteID != in.InviteID {
		t.Errorf("往返丢字段：%+v", out)
	}
}

// TestFinalizeRegPayGuards 校验 FinalizeRegPay 对空 param / 非注册 JSON / 缺账号的守卫（均静默跳过不建号）。
func TestFinalizeRegPayGuards(t *testing.T) {
	s := &MerchantRegService{} // 无依赖：以下分支均在建号前返回，不触碰 repo
	if err := s.FinalizeRegPay(""); err != nil {
		t.Errorf("空 param 应静默跳过：%v", err)
	}
	if err := s.FinalizeRegPay("not-json"); err != nil {
		t.Errorf("非法 JSON 应静默跳过：%v", err)
	}
	if err := s.FinalizeRegPay(`{"upid":1}`); err != nil {
		t.Errorf("缺 email/phone 应静默跳过：%v", err)
	}
}

// TestCashierSelectURL 校验 B1-04 空 type 收银台聚合选方式页地址拼接。
func TestCashierSelectURL(t *testing.T) {
	if got := cashierSelectURL("https://p.x", "20260101120000"); got != "https://p.x/pay/cashier/20260101120000" {
		t.Errorf("带 siteurl 拼接错误：%q", got)
	}
	if got := cashierSelectURL("https://p.x/", "T1"); got != "https://p.x/pay/cashier/T1" {
		t.Errorf("尾斜杠未去：%q", got)
	}
	if got := cashierSelectURL("", "T1"); got != "/pay/cashier/T1" {
		t.Errorf("空 siteurl 应回相对路径：%q", got)
	}
}
