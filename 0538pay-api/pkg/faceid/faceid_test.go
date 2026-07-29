package faceid

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fixedTime 固定注入时间，令 TC3 签名可复现。
func fixedTime() time.Time { return time.Date(2026, 7, 29, 3, 30, 0, 0, time.UTC) }

// TestTC3SignStable 校验 TC3-HMAC-SHA256 签名稳定且格式对齐 epay QcloudFaceid。
func TestTC3SignStable(t *testing.T) {
	var gotAuth, gotAction string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAction = r.Header.Get("X-TC-Action")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Write([]byte(`{"Response":{"AuthToken":"tok123","RedirectURL":"https://faceid.qq.com/scan?tok=123"}}`))
	}))
	defer srv.Close()

	c := New("AKIDxxxx", "secretkey", "")
	c.now = fixedTime
	c.host = strings.TrimPrefix(srv.URL, "http://") // 指向 mock 服务器
	// 直接用 httptest 的 http.Client（本地 http，非 https）。
	c.http = srv.Client()

	// 注：call 内部固定拼 https://host/，改走 srv 需覆盖 host 且 srv 为 http，故用 transport 重定向。
	c.http.Transport = rewriteToHTTP{srv.URL}

	r, err := c.GetRealNameAuthToken(context.Background(), "张三", "110101199003071234", "https://cb.example.com/x")
	if err != nil {
		t.Fatalf("GetRealNameAuthToken err: %v", err)
	}
	if r.AuthToken != "tok123" || r.RedirectURL == "" {
		t.Fatalf("unexpected result: %+v", r)
	}
	if gotAction != "GetRealNameAuthToken" {
		t.Errorf("X-TC-Action=%q", gotAction)
	}
	if !strings.HasPrefix(gotAuth, "TC3-HMAC-SHA256 Credential=AKIDxxxx/2026-07-29/faceid/tc3_request") {
		t.Errorf("Authorization 前缀不对: %q", gotAuth)
	}
	if !strings.Contains(gotAuth, "SignedHeaders=content-type;host") {
		t.Errorf("SignedHeaders 缺失: %q", gotAuth)
	}
	if gotBody["Name"] != "张三" || gotBody["IDCard"] != "110101199003071234" {
		t.Errorf("请求体字段不对: %+v", gotBody)
	}
}

// TestResultMessage 校验 ResultType 中文映射对齐 epay。
func TestResultMessage(t *testing.T) {
	cases := map[string]string{
		"0":  "实名认证通过",
		"-1": "实名认证未通过：姓名和身份证号不一致",
		"-2": "实名认证未通过：姓名和微信实名姓名不一致",
		"-3": "实名认证未通过：微信号未实名",
	}
	for in, want := range cases {
		if got := ResultMessage(in); got != want {
			t.Errorf("ResultMessage(%q)=%q want %q", in, got, want)
		}
	}
}

// rewriteToHTTP 把 call 内固定的 https://host/ 重写到 mock 服务器地址。
type rewriteToHTTP struct{ target string }

func (rt rewriteToHTTP) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := req.URL.Parse(rt.target + "/")
	req.URL = u
	return http.DefaultTransport.RoundTrip(req)
}
