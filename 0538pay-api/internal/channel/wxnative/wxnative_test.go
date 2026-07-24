package wxnative

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/epvia/api/internal/channel"
	"github.com/shopspring/decimal"
)

func TestYuanToFen(t *testing.T) {
	cases := map[string]int64{
		"100":    10000,
		"0.01":   1,
		"88.88":  8888,
		"0.005":  1, // 四舍五入到分（0.5 分进位）
		"12.344": 1234,
	}
	for in, want := range cases {
		got := yuanToFen(decimal.RequireFromString(in))
		if got != want {
			t.Errorf("yuanToFen(%s)=%d, 期望 %d", in, got, want)
		}
	}
}

func TestBuildPrepayBody(t *testing.T) {
	cfg := channel.Config{AppID: "wxappid", MchID: "16000000", NotifyURL: "https://x.com/notify"}
	req := channel.CreateReq{TradeNo: "20260721001", Money: decimal.RequireFromString("1.00"), Subject: "测试商品"}
	body, err := buildPrepayBody(cfg, req)
	if err != nil {
		t.Fatalf("组装下单体失败: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("下单体非合法 JSON: %v", err)
	}
	if m["appid"] != "wxappid" || m["mchid"] != "16000000" {
		t.Fatalf("appid/mchid 不对: %v", m)
	}
	if m["out_trade_no"] != "20260721001" {
		t.Fatalf("out_trade_no 不对: %v", m["out_trade_no"])
	}
	amount := m["amount"].(map[string]any)
	if amount["total"].(float64) != 100 {
		t.Fatalf("金额应为 100 分, 实际 %v", amount["total"])
	}
	if amount["currency"] != "CNY" {
		t.Fatalf("币种应为 CNY")
	}
}

func TestBuildPrepayBodyMissingConfig(t *testing.T) {
	// 缺 appid
	_, err := buildPrepayBody(channel.Config{MchID: "1"}, channel.CreateReq{NotifyURL: "https://x/n"})
	if err == nil {
		t.Fatal("缺 appid 应报错")
	}
	// 缺 notify_url
	_, err = buildPrepayBody(channel.Config{AppID: "a", MchID: "1"}, channel.CreateReq{})
	if err == nil {
		t.Fatal("缺 notify_url 应报错")
	}
}

// makeNotify 构造一份「已验签 + 已加密」的回调（自签名+自加密），供 parseNotify 端到端测。
func makeNotify(t *testing.T, apiV3Key, outTradeNo, tradeState string, totalFen int64) (cfg channel.Config, raw map[string]string) {
	t.Helper()
	// 1. 生成平台密钥对，pub 作为 cfg.PublicKey
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkix, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}))

	// 2. 业务对象 → AES-256-GCM 加密
	resource := notifyResource{OutTradeNo: outTradeNo, TransactionID: "wx-txn-1", TradeState: tradeState}
	resource.Amount.Total = totalFen
	resource.Amount.Currency = "CNY"
	plain, _ := json.Marshal(resource)

	nonce := "abcdef123456"
	ad := "transaction"
	block, _ := aes.NewCipher([]byte(apiV3Key))
	gcm, _ := cipher.NewGCM(block)
	sealed := gcm.Seal(nil, []byte(nonce), plain, []byte(ad))
	cipherB64 := base64.StdEncoding.EncodeToString(sealed)

	// 3. 外层报文
	env := notifyEnvelope{ID: "EV-1", EventType: "TRANSACTION.SUCCESS", ResourceType: "encrypt-resource"}
	env.Resource.Algorithm = "AEAD_AES_256_GCM"
	env.Resource.Ciphertext = cipherB64
	env.Resource.AssociatedData = ad
	env.Resource.Nonce = nonce
	env.Resource.OriginalType = "transaction"
	body, _ := json.Marshal(env)

	// 4. 对 body 按微信验签串格式签名
	ts, sigNonce := "1700000000", "sig-nonce-1"
	message := ts + "\n" + sigNonce + "\n" + string(body) + "\n"
	h := sha256.Sum256([]byte(message))
	sigRaw, _ := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h[:])
	sigB64 := base64.StdEncoding.EncodeToString(sigRaw)

	cfg = channel.Config{Key: apiV3Key, PublicKey: pubPEM}
	raw = map[string]string{
		KeyBody:      string(body),
		KeySignature: sigB64,
		KeyTimestamp: ts,
		KeyNonce:     sigNonce,
	}
	return
}

func TestParseNotifySuccess(t *testing.T) {
	key := "01234567890123456789012345678901"
	cfg, raw := makeNotify(t, key, "20260721001", "SUCCESS", 10000)
	res, err := parseNotify(cfg, raw)
	if err != nil {
		t.Fatalf("解析成功回调失败: %v", err)
	}
	if !res.Success {
		t.Fatal("trade_state=SUCCESS 应判定成功")
	}
	if res.TradeNo != "20260721001" {
		t.Fatalf("out_trade_no 不对: %s", res.TradeNo)
	}
	if !res.Money.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("金额应为 100 元, 实际 %s", res.Money)
	}
	if res.ChannelNo != "wx-txn-1" {
		t.Fatalf("transaction_id 不对: %s", res.ChannelNo)
	}
}

func TestParseNotifyNotPaid(t *testing.T) {
	key := "01234567890123456789012345678901"
	cfg, raw := makeNotify(t, key, "20260721002", "NOTPAY", 10000)
	res, err := parseNotify(cfg, raw)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if res.Success {
		t.Fatal("trade_state=NOTPAY 不应判定成功")
	}
}

func TestParseNotifyBadSignature(t *testing.T) {
	key := "01234567890123456789012345678901"
	cfg, raw := makeNotify(t, key, "20260721003", "SUCCESS", 10000)
	raw[KeyBody] = raw[KeyBody] + " " // 篡改报文，验签应失败
	if _, err := parseNotify(cfg, raw); err == nil {
		t.Fatal("报文篡改后应验签失败")
	}
}

func TestParseNotifyWrongAPIv3Key(t *testing.T) {
	key := "01234567890123456789012345678901"
	cfg, raw := makeNotify(t, key, "20260721004", "SUCCESS", 10000)
	cfg.Key = "ffffffffffffffffffffffffffffffff" // 错误 APIv3 密钥，解密应失败
	if _, err := parseNotify(cfg, raw); err == nil {
		t.Fatal("错误 APIv3 密钥应解密失败")
	}
}

func TestParseNotifyEmptyBody(t *testing.T) {
	if _, err := parseNotify(channel.Config{}, map[string]string{}); err == nil {
		t.Fatal("空报文应报错")
	}
}

func TestParseQueryResp(t *testing.T) {
	paid, err := parseQueryResp([]byte(`{"out_trade_no":"T1","transaction_id":"wx1","trade_state":"SUCCESS"}`))
	if err != nil || !paid {
		t.Fatalf("SUCCESS 应判已支付, paid=%v err=%v", paid, err)
	}
	for _, st := range []string{"NOTPAY", "CLOSED", "USERPAYING"} {
		paid, err := parseQueryResp([]byte(`{"trade_state":"` + st + `"}`))
		if err != nil || paid {
			t.Fatalf("%s 不应判已支付, paid=%v", st, paid)
		}
	}
	if _, err := parseQueryResp([]byte(`not json`)); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

func TestKeyRegistered(t *testing.T) {
	if _, ok := channel.Get("wxnative"); !ok {
		t.Fatal("wxnative 未注册到 registry")
	}
}
