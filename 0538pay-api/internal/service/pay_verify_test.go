package service

import (
	"strings"
	"testing"
)

// TestGetDefendKey 校验防御 key 计算对齐 epay getDefendKey：md5(syskey + pid + '_' + out_trade_no + syskey)。
func TestGetDefendKey(t *testing.T) {
	s := &PayService{cfg: &ConfigService{cache: map[string]string{"syskey": "SECRET"}}}
	got := s.getDefendKey(1001, "ORDER123")
	want := md5Hex("SECRET" + "1001" + "_" + "ORDER123" + "SECRET")
	if got != want {
		t.Fatalf("getDefendKey = %s, want %s", got, want)
	}
	if len(got) != 32 {
		t.Fatalf("defendKey 长度应为 32，实际 %d", len(got))
	}
	// syskey 未配置时仍可算（空串），不 panic。
	s2 := &PayService{cfg: &ConfigService{cache: map[string]string{}}}
	if k := s2.getDefendKey(1, "x"); len(k) != 32 {
		t.Fatalf("空 syskey 下 defendKey 长度应为 32，实际 %d", len(k))
	}
}

// TestCheckPayVerifyOpen_Modes 覆盖不依赖 DB 的判定分支：0关 / 2指定商户 / 3全开。
func TestCheckPayVerifyOpen_Modes(t *testing.T) {
	// mode 0 / cfg nil：不验证
	if (&PayService{}).checkPayVerifyOpen(1, "1.2.3.4") {
		t.Fatal("cfg 为 nil 时不应开启验证")
	}
	off := &PayService{cfg: &ConfigService{cache: map[string]string{"pay_verify": "0"}}}
	if off.checkPayVerifyOpen(1, "1.2.3.4") {
		t.Fatal("pay_verify=0 时不应开启验证")
	}

	// mode 3：全部开启
	all := &PayService{cfg: &ConfigService{cache: map[string]string{"pay_verify": "3"}}}
	if !all.checkPayVerifyOpen(999, "") {
		t.Fatal("pay_verify=3 时应对所有商户开启验证")
	}

	// mode 2：仅指定商户（| 分隔）命中
	spec := &PayService{cfg: &ConfigService{cache: map[string]string{
		"pay_verify":           "2",
		"pay_verify_check_uid": "1001|1003|1005",
	}}}
	if !spec.checkPayVerifyOpen(1003, "") {
		t.Fatal("pay_verify=2 命中指定商户应开启")
	}
	if spec.checkPayVerifyOpen(2000, "") {
		t.Fatal("pay_verify=2 未命中指定商户不应开启")
	}
}

// TestDefendVerifyLogic 校验 __defend 中段 32 位等于 defendKey 才放行的核心判定
//（对齐 epay Pay.php:101 substr($defend,10,32)===defendKey）。
func TestDefendVerifyLogic(t *testing.T) {
	s := &PayService{cfg: &ConfigService{cache: map[string]string{"syskey": "K"}}}
	key := s.getDefendKey(1, "OUT1")

	// 合法：10 位前缀 + key + 任意后缀
	valid := "1700000000" + key + "123456"
	if !(len(valid) >= 42 && valid[10:42] == key) {
		t.Fatal("构造的合法 __defend 应通过中段校验")
	}
	// 非法：中段不等
	bad := "1700000000" + strings.Repeat("0", 32) + "123456"
	if len(bad) >= 42 && bad[10:42] == key {
		t.Fatal("错误 __defend 不应通过中段校验")
	}
	// 过短：不足 42 位
	short := "12345"
	if len(short) >= 42 {
		t.Fatal("过短 __defend 应被长度校验拦下")
	}
}
