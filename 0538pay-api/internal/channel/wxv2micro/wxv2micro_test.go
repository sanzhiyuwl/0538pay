package wxv2micro

import (
	"context"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/epvia/api/internal/channel"
	"github.com/epvia/api/internal/channel/wxv2base"
	"github.com/epvia/api/pkg/wxpayv2"
	"github.com/shopspring/decimal"
)

const testKey = "01234567890123456789012345678901"

// signedXML 用测试密钥自签一份 <xml> 应答。
func signedXML(m map[string]string) string {
	m["sign"] = wxpayv2.MakeSign(m, testKey)
	return wxpayv2.MapToXML(m)
}

// fakeGateway 起一个假微信网关，按请求路径与调用次数返回预设应答，替换 APIHost。
type fakeGateway struct {
	mu           sync.Mutex
	queryCalls   int
	reverseCalls int
	// queryResponder 据第 n 次查单（从 1 计）返回应答 map。
	queryResponder func(n int) map[string]string
	// microResponse 首次 micropay 的应答（含成功或 USERPAYING 错误码）。
	microResponse map[string]string
}

func (g *fakeGateway) start(t *testing.T) func() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		g.mu.Lock()
		defer g.mu.Unlock()
		switch {
		case strings.Contains(r.URL.Path, "/pay/micropay"):
			_, _ = w.Write([]byte(signedXML(cloneMap(g.microResponse))))
		case strings.Contains(r.URL.Path, "/pay/orderquery"):
			g.queryCalls++
			_, _ = w.Write([]byte(signedXML(g.queryResponder(g.queryCalls))))
		case strings.Contains(r.URL.Path, "/secapi/pay/reverse"):
			g.reverseCalls++
			_, _ = w.Write([]byte(signedXML(map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS"})))
		default:
			http.NotFound(w, r)
		}
	}))
	oldHost := wxv2base.APIHost
	wxv2base.APIHost = srv.URL
	return func() {
		wxv2base.APIHost = oldHost
		srv.Close()
	}
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func testReq() channel.CreateReq {
	return channel.CreateReq{
		TradeNo:  "T20260728",
		Money:    decimal.RequireFromString("1.00"),
		Subject:  "商品",
		ClientIP: "1.2.3.4",
		AuthCode: "134567890123456789",
	}
}

func testCfg() channel.Config {
	return channel.Config{AppID: "wx1", MchID: "160", Key: testKey}
}

// testCfgWithCert 带自签商户证书（reverse/退款走 mTLS 分支需要，httpClient 要能 X509KeyPair 加载）。
func testCfgWithCert(t *testing.T) channel.Config {
	t.Helper()
	certPEM, keyPEM := genSelfSignedCert(t)
	cfg := testCfg()
	cfg.Extra = map[string]string{"cert_pem": certPEM, "key_pem": keyPEM}
	return cfg
}

func genSelfSignedCert(t *testing.T) (certPEM, keyPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1)}
	der, err := x509.CreateCertificate(crand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	return
}

// TestMicroImmediateSuccess micropay 直接返回成功：无需查单，回 paid=1。
func TestMicroImmediateSuccess(t *testing.T) {
	g := &fakeGateway{
		microResponse: map[string]string{
			"return_code": "SUCCESS", "result_code": "SUCCESS",
			"transaction_id": "wx-txn-1", "out_trade_no": "T20260728", "openid": "o1", "total_fee": "100",
		},
	}
	defer g.start(t)()

	resp, err := Channel{}.Create(context.Background(), testCfg(), testReq())
	if err != nil {
		t.Fatalf("直接成功不应报错: %v", err)
	}
	assertPaid(t, resp)
	if g.queryCalls != 0 {
		t.Fatalf("直接成功不应查单, 实际查了 %d 次", g.queryCalls)
	}
}

// TestMicroUserPayingThenSuccess USERPAYING → 轮询查单第2次成功：回 paid=1，未撤单。
func TestMicroUserPayingThenSuccess(t *testing.T) {
	microPollInterval = 5 * time.Millisecond // 加速轮询
	g := &fakeGateway{
		microResponse: map[string]string{
			"return_code": "SUCCESS", "result_code": "FAIL",
			"err_code": "USERPAYING", "err_code_des": "用户支付中",
		},
		queryResponder: func(n int) map[string]string {
			if n >= 2 {
				return map[string]string{
					"return_code": "SUCCESS", "result_code": "SUCCESS", "trade_state": "SUCCESS",
					"transaction_id": "wx-txn-2", "out_trade_no": "T20260728", "openid": "o2", "total_fee": "100",
				}
			}
			return map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "trade_state": "USERPAYING"}
		},
	}
	defer g.start(t)()

	// USERPAYING 分支会先 sleep(2s)，为让测试快速，临时缩短——但 sleep 是硬编码 2s，
	// 用带超时 ctx 的方式无法绕过 2s。故这里直接接受 ~2s 等待（仍在测试可接受范围）。
	resp, err := Channel{}.Create(context.Background(), testCfg(), testReq())
	if err != nil {
		t.Fatalf("轮询后成功不应报错: %v", err)
	}
	assertPaid(t, resp)
	if g.reverseCalls != 0 {
		t.Fatalf("成功不应撤单, 实际撤了 %d 次", g.reverseCalls)
	}
}

// TestMicroTimeoutReverse SYSTEMERROR → 轮询全返回超时状态 → 撤单并报失败。
func TestMicroTimeoutReverse(t *testing.T) {
	microPollInterval = 5 * time.Millisecond
	g := &fakeGateway{
		microResponse: map[string]string{
			"return_code": "SUCCESS", "result_code": "FAIL",
			"err_code": "SYSTEMERROR", "err_code_des": "系统错误",
		},
		queryResponder: func(n int) map[string]string {
			// 一直返回 CLOSED（非 SUCCESS 非 USERPAYING）→ 立即撤单
			return map[string]string{"return_code": "SUCCESS", "result_code": "SUCCESS", "trade_state": "CLOSED"}
		},
	}
	defer g.start(t)()

	// reverse 走 mTLS 分支，需商户证书（否则 httpClient 报错，撤单请求发不出去）。
	_, err := Channel{}.Create(context.Background(), testCfgWithCert(t), testReq())
	if err == nil {
		t.Fatal("超时应报失败")
	}
	if g.reverseCalls != 1 {
		t.Fatalf("超时应撤单 1 次, 实际 %d 次", g.reverseCalls)
	}
}

func assertPaid(t *testing.T, resp channel.CreateResp) {
	t.Helper()
	if resp.PayType != "scan" {
		t.Fatalf("PayType 应为 scan, 实际 %s", resp.PayType)
	}
	var p map[string]string
	if err := json.Unmarshal([]byte(resp.RawHTML), &p); err != nil {
		t.Fatalf("RawHTML 解析失败: %v", err)
	}
	if p["paid"] != "1" {
		t.Fatalf("同步成功应带 paid=1, 实际 %v", p)
	}
	if p["transaction_id"] == "" {
		t.Fatal("应回填 transaction_id")
	}
}
