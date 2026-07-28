package service

import (
	"encoding/json"
	"testing"
)

// TestInjectBusinessCode business_code 缺失时补入、已存在时以我方编号覆盖，且不破坏其余字段。
func TestInjectBusinessCode(t *testing.T) {
	// 缺失时补入
	out, err := injectBusinessCode(`{"subject_info":{"subject_type":"SUBJECT_TYPE_INDIVIDUAL"}}`, "EN20260728001")
	if err != nil {
		t.Fatalf("注入 business_code 失败: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("结果非法 JSON: %v", err)
	}
	if m["business_code"] != "EN20260728001" {
		t.Errorf("business_code=%v 期望 EN20260728001", m["business_code"])
	}
	if _, ok := m["subject_info"]; !ok {
		t.Error("原有 subject_info 字段丢失")
	}

	// 已存在时以我方编号覆盖（防客户端伪造）
	out2, err := injectBusinessCode(`{"business_code":"HACK"}`, "EN20260728002")
	if err != nil {
		t.Fatalf("覆盖 business_code 失败: %v", err)
	}
	json.Unmarshal([]byte(out2), &m)
	if m["business_code"] != "EN20260728002" {
		t.Errorf("business_code=%v 期望被覆盖为 EN20260728002", m["business_code"])
	}
}

// TestInjectBusinessCodeBadJSON 非法 JSON 资料应返回错误，不静默吞。
func TestInjectBusinessCodeBadJSON(t *testing.T) {
	if _, err := injectBusinessCode(`not-json`, "EN1"); err == nil {
		t.Error("非法 JSON 应返回错误")
	}
}

// TestDecimalToApplyment 微信数字 applyment_id 转字符串落库。
func TestDecimalToApplyment(t *testing.T) {
	if got := decimalToApplyment(2000002124775691); got != "2000002124775691" {
		t.Errorf("decimalToApplyment=%s 期望 2000002124775691", got)
	}
}

// TestGenEnrollNo 进件单号格式：TY 前缀 + 8 位数字。
func TestGenEnrollNo(t *testing.T) {
	a := genEnrollNo()
	if len(a) != 2+8 { // TY + 8 位数字
		t.Errorf("进件单号长度=%d 期望 10，值=%s", len(a), a)
	}
	if a[:2] != "TY" {
		t.Errorf("进件单号前缀=%s 期望 TY", a[:2])
	}
	for _, c := range a[2:] {
		if c < '0' || c > '9' {
			t.Errorf("进件单号后缀应为纯数字，值=%s", a)
			break
		}
	}
}

// TestFinalizeEnrollPayParam FinalizeEnrollPay 对非进件订单 param 应静默跳过、不误伤。
// （进件单存在性/状态翻转依赖 DB，这里只覆盖 param 解析的守卫分支。）
func TestFinalizeEnrollPayParam(t *testing.T) {
	s := &EnrollService{} // repo=nil：只要不进到查库分支即安全
	// 空 param → nil
	if err := s.FinalizeEnrollPay(""); err != nil {
		t.Errorf("空 param 应跳过，得到 err=%v", err)
	}
	// 非 JSON → nil（非进件订单）
	if err := s.FinalizeEnrollPay("not-json"); err != nil {
		t.Errorf("非 JSON param 应跳过，得到 err=%v", err)
	}
	// JSON 但无 enroll_no → nil（其它 tid 的订单 param）
	if err := s.FinalizeEnrollPay(`{"uid":"1002"}`); err != nil {
		t.Errorf("无 enroll_no 应跳过，得到 err=%v", err)
	}
}
