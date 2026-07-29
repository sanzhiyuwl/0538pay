package service

import "testing"

// TestJudgePhone3Result 校验三要素返回判定兼容多家云市场服务商格式。
func TestJudgePhone3Result(t *testing.T) {
	cases := []struct {
		name string
		body string
		pass bool // 期望通过
		ok   bool // 期望无 error（业务不一致也算“无 error 但 pass=false”？此处 err!=nil 表示未通过/失败）
	}{
		// 神度：data.result 字符串 "0"=一致
		{"神度一致", `{"code":200,"success":true,"msg":"成功","data":{"result":"0","desc":"一致"}}`, true, true},
		{"神度不一致", `{"code":200,"success":true,"msg":"成功","data":{"result":"1","desc":"不一致"}}`, false, false},
		{"神度查无记录", `{"code":200,"success":true,"msg":"成功","data":{"result":"2","desc":"无记录"}}`, false, false},
		// 另一家：data.result 数字 0=一致
		{"数字result一致", `{"code":200,"data":{"result":0,"desc":"一致"}}`, true, true},
		// epay 老家：无 data.result，顶层 code==200 即通过
		{"顶层code通过", `{"code":200,"msg":"验证通过"}`, true, true},
		// 顶层失败：透出 msg
		{"顶层失败", `{"code":40001,"msg":"参数错误"}`, false, false},
		// success 为真但无 result
		{"success为真", `{"code":0,"success":true,"data":{}}`, true, true},
	}
	for _, c := range cases {
		pass, err := judgePhone3Result([]byte(c.body))
		if pass != c.pass {
			t.Errorf("%s: pass=%v want %v (err=%v)", c.name, pass, c.pass, err)
		}
		if (err == nil) != c.ok {
			t.Errorf("%s: err=%v, want ok=%v", c.name, err, c.ok)
		}
	}
}
