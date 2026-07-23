package service

import (
	"strings"
	"testing"
)

// TestParseMsgConfig 覆盖商户 msgconfig 解析：数字型/字符串型/空/非法。
func TestParseMsgConfig(t *testing.T) {
	// 数字型（epay 反序列化后是数字）
	m := parseMsgConfig(`{"order":1,"settle":2,"login":3,"order_money":"5.00"}`)
	if got := msgConfInt(m, "order"); got != 1 {
		t.Errorf("order 通道应为 1(微信)，得 %d", got)
	}
	if got := msgConfInt(m, "settle"); got != 2 {
		t.Errorf("settle 通道应为 2(邮件)，得 %d", got)
	}
	if got := msgConfInt(m, "login"); got != 3 {
		t.Errorf("login 通道应为 3(短信)，得 %d", got)
	}
	if got := msgConfFloat(m, "order_money"); got != 5.0 {
		t.Errorf("order_money 阈值应为 5.0，得 %v", got)
	}
	// 字符串型数字（前端存字符串）
	m2 := parseMsgConfig(`{"order":"1"}`)
	if got := msgConfInt(m2, "order"); got != 1 {
		t.Errorf("字符串型 order 应解析为 1，得 %d", got)
	}
	// 空 / 非法 JSON 不 panic，返回空 map（通道 0=关闭）
	if got := msgConfInt(parseMsgConfig(""), "order"); got != 0 {
		t.Errorf("空配置通道应为 0，得 %d", got)
	}
	if got := msgConfInt(parseMsgConfig("not-json"), "order"); got != 0 {
		t.Errorf("非法配置通道应为 0，得 %d", got)
	}
}

// TestTruncate 覆盖模板消息字段截断：中文按字符，订单号按字节。
func TestTruncate(t *testing.T) {
	// truncRune 按字符（中文不截半个字）
	if got := truncRune("一二三四五", 3); got != "一二三" {
		t.Errorf("truncRune 按字符截断失败，得 %q", got)
	}
	if got := truncRune("abc", 5); got != "abc" {
		t.Errorf("truncRune 短于上限应原样返回，得 %q", got)
	}
	// truncByte 按字节
	long := strings.Repeat("a", 40)
	if got := truncByte(long, 32); len(got) != 32 {
		t.Errorf("truncByte 应截到 32 字节，得 %d", len(got))
	}
}

// TestMailTemplateScenes 覆盖各场景邮件模板生成非空且含关键字段。
func TestMailTemplateScenes(t *testing.T) {
	s := &NoticeService{cfg: &ConfigService{cache: map[string]string{"sitename": "0538支付"}}}
	cases := []struct {
		scene  string
		param  map[string]string
		expect string // 正文应包含的关键字
	}{
		{"order", map[string]string{"name": "商品A", "money": "66.00", "trade_no": "T1"}, "商品A"},
		{"settle", map[string]string{"money": "100.00", "realmoney": "99.00"}, "结算"},
		{"login", map[string]string{"user": "u1", "clientip": "1.2.3.4", "time": "2026-07-23"}, "登录"},
		{"regaudit", map[string]string{"uid": "5", "account": "a@b.com"}, "待审核"},
		{"apply", map[string]string{"uid": "5", "realmoney": "50.00", "type": "支付宝"}, "提现"},
		{"domain", map[string]string{"uid": "5", "domain": "x.com"}, "域名"},
		{"balance", map[string]string{"msgmoney": "10", "money": "3"}, "余额不足"},
	}
	for _, c := range cases {
		title, content := s.mailTemplate(c.scene, c.param)
		if title == "" || content == "" {
			t.Errorf("场景 %s 模板不应为空", c.scene)
			continue
		}
		if !strings.Contains(content, c.expect) {
			t.Errorf("场景 %s 正文应含 %q，得 %q", c.scene, c.expect, content)
		}
	}
	// 未知场景返回空（不发信）
	if title, content := s.mailTemplate("unknown", nil); title != "" || content != "" {
		t.Errorf("未知场景应返回空模板")
	}
}

// TestBuildMIME 覆盖 MIME 报文头组装与中文主题 B 编码。
func TestBuildMIME(t *testing.T) {
	msg := string(buildMIME("0538支付", "no-reply@a.com", "to@b.com", "测试主题", "<b>hi</b>"))
	if !strings.Contains(msg, "Content-Type: text/html; charset=UTF-8") {
		t.Error("缺少 HTML Content-Type 头")
	}
	if !strings.Contains(msg, "=?UTF-8?B?") {
		t.Error("中文主题应做 RFC2047 B 编码")
	}
	if !strings.Contains(msg, "To: to@b.com") {
		t.Error("缺少收件人头")
	}
	// 纯 ASCII 主题不编码
	msg2 := string(buildMIME("Site", "a@a.com", "b@b.com", "Hello", "hi"))
	if strings.Contains(msg2, "=?UTF-8?B?SGVsbG8") {
		t.Error("纯 ASCII 主题不应做 B 编码")
	}
}
