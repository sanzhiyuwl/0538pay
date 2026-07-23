package chsign

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"
)

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// xunhupay：ksort → a=1&b=2 → md5(串 + KEY)，小写。
func TestXunhupayStyle(t *testing.T) {
	s := MD5Signer{Key: "KEY123", Mode: KeyAppend, SignKey: "hash"}
	p := map[string]string{"b": "2", "a": "1", "hash": "ignore", "empty": ""}
	want := md5hex("a=1&b=2" + "KEY123")
	if got := s.Sign(p); got != want {
		t.Errorf("xunhupay sign=%s want %s", got, want)
	}
	// 验签：把算出的塞回 hash 字段应通过。
	p["hash"] = want
	if !s.Verify(p) {
		t.Error("xunhupay verify 应通过")
	}
}

// swiftpass2 / ltzf：ksort → a=1&b=2 → md5(串 + &key=KEY)，大写。
func TestSwiftpassStyle(t *testing.T) {
	s := MD5Signer{Key: "SECRET", Mode: KeyAmpEq, Upper: true}
	p := map[string]string{"mch_id": "10", "out_trade_no": "T1", "sign": "x"}
	want := strings.ToUpper(md5hex("mch_id=10&out_trade_no=T1" + "&key=SECRET"))
	if got := s.Sign(p); got != want {
		t.Errorf("swiftpass sign=%s want %s", got, want)
	}
	if want != strings.ToUpper(want) {
		t.Error("应为大写")
	}
}

// ltzf 白名单：只签 OnlyKeys 指定字段，其余不参与。
func TestOnlyKeys(t *testing.T) {
	s := MD5Signer{Key: "K", Mode: KeyEqPrefix, Upper: true,
		OnlyKeys: []string{"mch_id", "out_trade_no", "total_fee"}}
	p := map[string]string{
		"mch_id": "10", "out_trade_no": "T1", "total_fee": "100",
		"extra": "should_skip", "body": "商品",
	}
	// 只签白名单三字段（ksort：mch_id/out_trade_no/total_fee）。
	want := strings.ToUpper(md5hex("mch_id=10&out_trade_no=T1&total_fee=100" + "&key=K"))
	if got := s.Sign(p); got != want {
		t.Errorf("onlykeys sign=%s want %s", got, want)
	}
}

func TestVerifyBadSign(t *testing.T) {
	s := MD5Signer{Key: "K"}
	p := map[string]string{"a": "1", "sign": "deadbeef"}
	if s.Verify(p) {
		t.Error("错误签名不应通过")
	}
	if s.Verify(map[string]string{"a": "1"}) {
		t.Error("缺 sign 不应通过")
	}
}

// 空值与签名字段本身都不参与签名。
func TestSkipEmptyAndSignField(t *testing.T) {
	s := MD5Signer{Key: "K", Mode: KeyAppend}
	p := map[string]string{"a": "1", "b": "", "c": "  ", "sign": "whatever"}
	if got, want := s.Content(p), "a=1"; got != want {
		t.Errorf("content=%q want %q", got, want)
	}
}
