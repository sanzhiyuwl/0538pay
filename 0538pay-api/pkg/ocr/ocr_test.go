package ocr

import "testing"

func TestNormalizeDate(t *testing.T) {
	cases := map[string]string{
		"2020年01月01日": "2020-01-01",
		"2020.1.1":      "2020-01-01",
		"2020/01/01":    "2020-01-01",
		"2020-1-1":      "2020-01-01",
		"长期":            "长期",
		"":              "",
	}
	for in, want := range cases {
		if got := normalizeDate(in); got != want {
			t.Errorf("normalizeDate(%q)=%q want %q", in, got, want)
		}
	}
}

func TestSplitPeriod(t *testing.T) {
	// 拆开的 from/to 优先
	if b, e := splitAliyunPeriod("", "2020.01.01", "长期"); b != "2020-01-01" || e != "长期" {
		t.Errorf("from/to split got %q %q", b, e)
	}
	// 合并串按中文分隔
	if b, e := splitAliyunPeriod("2020-01-01至2040-01-01", "", ""); b != "2020-01-01" || e != "2040-01-01" {
		t.Errorf("merged split got %q %q", b, e)
	}
	// 腾讯点分隔有效期
	if b, e := splitAliyunPeriod("2018.09.01-2038.09.01", "", ""); b != "2018-09-01" {
		t.Errorf("tencent validdate got %q %q", b, e)
	}
}

func TestPercentEncode(t *testing.T) {
	// 阿里云编码规则：空格→%20，~ 不编码
	if got := percentEncode("a b~c"); got != "a%20b~c" {
		t.Errorf("percentEncode got %q", got)
	}
}

func TestAliyunSignStable(t *testing.T) {
	a := &aliyunRecognizer{accessKeySecret: "secret"}
	params := map[string]string{"Action": "RecognizeIdcard", "Format": "JSON"}
	s1 := a.sign(params, "POST")
	s2 := a.sign(params, "POST")
	if s1 == "" || s1 != s2 {
		t.Errorf("sign unstable: %q %q", s1, s2)
	}
}
