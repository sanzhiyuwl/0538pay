package alipayf2f

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/pkg/alipay"
	"github.com/shopspring/decimal"
)

func genKeys(t *testing.T) (privPEM, pubPEM string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}))
	pkix, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}))
	return
}

func TestBuildBizContent(t *testing.T) {
	cfg := channel.Config{AppID: "2021000", Extra: map[string]string{"seller_id": "2088xxxx"}}
	req := channel.CreateReq{TradeNo: "20260721001", Money: decimal.RequireFromString("1.5"), Subject: "测试"}
	s, err := buildBizContent(cfg, req)
	if err != nil {
		t.Fatalf("组装 biz_content 失败: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("biz_content 非合法 JSON: %v", err)
	}
	if m["out_trade_no"] != "20260721001" {
		t.Fatalf("out_trade_no 不对: %s", m["out_trade_no"])
	}
	if m["total_amount"] != "1.50" {
		t.Fatalf("金额应两位小数 1.50, 实际 %s", m["total_amount"])
	}
	if m["seller_id"] != "2088xxxx" {
		t.Fatalf("seller_id 应带入")
	}
}

func TestBuildBizContentMissingAppID(t *testing.T) {
	if _, err := buildBizContent(channel.Config{}, channel.CreateReq{}); err == nil {
		t.Fatal("缺 appid 应报错")
	}
}

func TestBuildSysParams(t *testing.T) {
	cfg := channel.Config{AppID: "2021", NotifyURL: "https://x/n"}
	req := channel.CreateReq{TradeNo: "T1"}
	p := buildSysParams(cfg, req, `{"a":1}`, "2026-07-21 12:00:00")
	for k, want := range map[string]string{
		"app_id": "2021", "method": precreate, "sign_type": "RSA2",
		"version": "1.0", "timestamp": "2026-07-21 12:00:00",
		"biz_content": `{"a":1}`, "notify_url": "https://x/n",
	} {
		if p[k] != want {
			t.Fatalf("系统参数 %s=%q, 期望 %q", k, p[k], want)
		}
	}
}

func TestParsePrecreateRespSuccess(t *testing.T) {
	body := []byte(`{"alipay_trade_precreate_response":{"code":"10000","msg":"Success","out_trade_no":"T1","qr_code":"https://qr.alipay.com/abc"},"sign":"xx"}`)
	qr, err := parsePrecreateResp(body)
	if err != nil {
		t.Fatalf("应解析成功: %v", err)
	}
	if qr != "https://qr.alipay.com/abc" {
		t.Fatalf("qr_code 不对: %s", qr)
	}
}

func TestParsePrecreateRespError(t *testing.T) {
	body := []byte(`{"alipay_trade_precreate_response":{"code":"40004","msg":"Business Failed","sub_code":"ACQ.TRADE_HAS_SUCCESS","sub_msg":"交易已被支付"},"sign":"xx"}`)
	if _, err := parsePrecreateResp(body); err == nil {
		t.Fatal("code!=10000 应报错")
	}
}

// makeNotify 构造一份已 RSA2 签名的表单回调，供 parseNotify 端到端测。
func makeNotify(t *testing.T, outTradeNo, tradeStatus, amount string) (cfg channel.Config, raw map[string]string) {
	t.Helper()
	privPEM, pubPEM := genKeys(t)
	priv, err := alipay.ParsePrivateKey(privPEM)
	if err != nil {
		t.Fatal(err)
	}
	params := map[string]string{
		"out_trade_no": outTradeNo,
		"trade_no":     "2026072122001",
		"total_amount": amount,
		"trade_status": tradeStatus,
		"buyer_id":     "2088buyer",
	}
	sig, err := alipay.SignParams(priv, params)
	if err != nil {
		t.Fatal(err)
	}
	params["sign_type"] = "RSA2"
	params["sign"] = sig
	// 模拟框架注入的原始报文保留键，parseNotify 应忽略它们
	params["_raw_body"] = "ignored"
	return channel.Config{PublicKey: pubPEM}, params
}

func TestParseNotifySuccess(t *testing.T) {
	cfg, raw := makeNotify(t, "20260721001", "TRADE_SUCCESS", "1.00")
	res, err := parseNotify(cfg, raw)
	if err != nil {
		t.Fatalf("解析成功回调失败: %v", err)
	}
	if !res.Success {
		t.Fatal("TRADE_SUCCESS 应判定成功")
	}
	if res.TradeNo != "20260721001" {
		t.Fatalf("out_trade_no 不对: %s", res.TradeNo)
	}
	if !res.Money.Equal(decimal.RequireFromString("1.00")) {
		t.Fatalf("金额不对: %s", res.Money)
	}
	if res.AckContent != "success" {
		t.Fatalf("支付宝应答应为 success")
	}
}

func TestParseNotifyBadSign(t *testing.T) {
	cfg, raw := makeNotify(t, "20260721002", "TRADE_SUCCESS", "1.00")
	raw["total_amount"] = "9999.00" // 篡改金额，验签应失败
	if _, err := parseNotify(cfg, raw); err == nil {
		t.Fatal("篡改后应验签失败")
	}
}

func TestParseNotifyNotSuccess(t *testing.T) {
	cfg, raw := makeNotify(t, "20260721003", "WAIT_BUYER_PAY", "1.00")
	res, err := parseNotify(cfg, raw)
	if err != nil {
		t.Fatalf("验签应通过: %v", err)
	}
	if res.Success {
		t.Fatal("WAIT_BUYER_PAY 不应判定成功")
	}
}

func TestParseQueryResp(t *testing.T) {
	// 已支付
	paid, err := parseQueryResp([]byte(`{"alipay_trade_query_response":{"code":"10000","msg":"Success","out_trade_no":"T1","trade_no":"202607","trade_status":"TRADE_SUCCESS"}}`))
	if err != nil || !paid {
		t.Fatalf("TRADE_SUCCESS 应判已支付, paid=%v err=%v", paid, err)
	}
	// 未支付（等待买家付款）
	paid, err = parseQueryResp([]byte(`{"alipay_trade_query_response":{"code":"10000","trade_status":"WAIT_BUYER_PAY"}}`))
	if err != nil || paid {
		t.Fatalf("WAIT_BUYER_PAY 不应判已支付, paid=%v", paid)
	}
	// 订单不存在 → 未支付，非错误
	paid, err = parseQueryResp([]byte(`{"alipay_trade_query_response":{"code":"40004","sub_code":"ACQ.TRADE_NOT_EXIST","msg":"Business Failed"}}`))
	if err != nil || paid {
		t.Fatalf("TRADE_NOT_EXIST 应判未支付且无错误, paid=%v err=%v", paid, err)
	}
	// 其它业务错误 → 报错
	if _, err := parseQueryResp([]byte(`{"alipay_trade_query_response":{"code":"40002","sub_code":"ISV.INVALID-APP-ID","msg":"Invalid Arguments"}}`)); err == nil {
		t.Fatal("其它错误码应报错")
	}
}

func TestKeyRegistered(t *testing.T) {
	if _, ok := channel.Get("alipayf2f"); !ok {
		t.Fatal("alipayf2f 未注册")
	}
}
