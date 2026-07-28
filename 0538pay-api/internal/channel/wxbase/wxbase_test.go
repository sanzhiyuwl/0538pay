package wxbase

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

// TestYuanToFen 元→分四舍五入（0.5 分进位）。
func TestYuanToFen(t *testing.T) {
	cases := map[string]int64{
		"100":    10000,
		"0.01":   1,
		"88.88":  8888,
		"0.005":  1,
		"12.344": 1234,
	}
	for in, want := range cases {
		if got := YuanToFen(decimal.RequireFromString(in)); got != want {
			t.Errorf("YuanToFen(%s)=%d, 期望 %d", in, got, want)
		}
	}
}

// TestBasePrepayBodyDirect 直连模式下单体：appid/mchid/out_trade_no/金额/币种。
func TestBasePrepayBodyDirect(t *testing.T) {
	cfg := channel.Config{AppID: "wxappid", MchID: "16000000", NotifyURL: "https://x.com/notify"}
	req := channel.CreateReq{TradeNo: "20260721001", Money: decimal.RequireFromString("1.00"), Subject: "测试商品"}
	body, err := BasePrepayBody(cfg, req)
	if err != nil {
		t.Fatalf("组装下单体失败: %v", err)
	}
	if body["appid"] != "wxappid" || body["mchid"] != "16000000" {
		t.Fatalf("appid/mchid 不对: %v", body)
	}
	if body["out_trade_no"] != "20260721001" {
		t.Fatalf("out_trade_no 不对: %v", body["out_trade_no"])
	}
	amount := body["amount"].(AmountInfo)
	if amount.Total != 100 {
		t.Fatalf("金额应为 100 分, 实际 %v", amount.Total)
	}
	if amount.Currency != "CNY" {
		t.Fatalf("币种应为 CNY")
	}
	// 直连不应出现服务商字段
	if _, ok := body["sp_appid"]; ok {
		t.Fatal("直连模式不应带 sp_appid")
	}
	if _, ok := body["sub_mchid"]; ok {
		t.Fatal("直连模式不应带 sub_mchid")
	}
}

// TestBasePrepayBodyPartner 服务商模式：Extra 带 sub_mchid → 下单体用 sp_appid/sp_mchid/sub_mchid。
func TestBasePrepayBodyPartner(t *testing.T) {
	cfg := channel.Config{
		AppID:     "spappid",
		MchID:     "sp16000000",
		NotifyURL: "https://x.com/notify",
		Extra:     map[string]string{"sub_mchid": "sub20000001", "sub_appid": "subappid"},
	}
	req := channel.CreateReq{TradeNo: "P1", Money: decimal.RequireFromString("1.00"), Subject: "商品"}
	body, err := BasePrepayBody(cfg, req)
	if err != nil {
		t.Fatalf("组装服务商下单体失败: %v", err)
	}
	if body["sp_appid"] != "spappid" || body["sp_mchid"] != "sp16000000" {
		t.Fatalf("sp_appid/sp_mchid 不对: %v", body)
	}
	if body["sub_mchid"] != "sub20000001" {
		t.Fatalf("sub_mchid 不对: %v", body["sub_mchid"])
	}
	if body["sub_appid"] != "subappid" {
		t.Fatalf("sub_appid 不对: %v", body["sub_appid"])
	}
	// 服务商模式不应出现直连的 appid/mchid
	if _, ok := body["appid"]; ok {
		t.Fatal("服务商模式不应带直连 appid")
	}
	if _, ok := body["mchid"]; ok {
		t.Fatal("服务商模式不应带直连 mchid")
	}
	if !IsPartner(cfg) {
		t.Fatal("配了 sub_mchid 应判定为服务商模式")
	}
	if SubAppID(cfg) != "subappid" {
		t.Fatalf("SubAppID 不对: %s", SubAppID(cfg))
	}
}

func TestBasePrepayBodyMissingConfig(t *testing.T) {
	// 缺 notify_url（下单必需的回调地址）应报错
	if _, err := BasePrepayBody(channel.Config{AppID: "a", MchID: "1"}, channel.CreateReq{}); err == nil {
		t.Fatal("缺 notify_url 应报错")
	}
}

// TestBasePrepayBodyProfitAndScene 校验分账标记与客户端 IP 拼接（对齐 epay wxpayn jsapiPay）。
func TestBasePrepayBodyProfitAndScene(t *testing.T) {
	cfg := channel.Config{AppID: "wxappid", MchID: "16000000", NotifyURL: "https://x.com/notify"}

	// 1) 命中分账 + 带客户端 IP
	body, err := BasePrepayBody(cfg, channel.CreateReq{
		TradeNo:       "T1",
		Money:         decimal.RequireFromString("1.00"),
		Subject:       "商品",
		ProfitSharing: true,
		ClientIP:      "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("组装下单体失败: %v", err)
	}
	settle, ok := body["settle_info"].(map[string]interface{})
	if !ok || settle["profit_sharing"] != true {
		t.Fatalf("分账单应带 settle_info.profit_sharing=true, 实际 %v", body["settle_info"])
	}
	scene, ok := body["scene_info"].(map[string]interface{})
	if !ok || scene["payer_client_ip"] != "1.2.3.4" {
		t.Fatalf("应带 scene_info.payer_client_ip, 实际 %v", body["scene_info"])
	}

	// 2) 无分账 + 无 IP：两字段都不应出现（避免传空值被微信拒）
	body2, err := BasePrepayBody(cfg, channel.CreateReq{TradeNo: "T2", Money: decimal.RequireFromString("2.00")})
	if err != nil {
		t.Fatalf("组装下单体失败: %v", err)
	}
	if _, exist := body2["settle_info"]; exist {
		t.Fatal("非分账单不应带 settle_info")
	}
	if _, exist := body2["scene_info"]; exist {
		t.Fatal("无客户端 IP 时不应带 scene_info")
	}
}

// TestBuildAppParamsV3 V3 APP 拉起参数：字段齐全 + paySign 用商户私钥 RSA 签名可被公钥验证。
// 签名串格式 appid\ntimestamp\nnoncestr\nprepayid\n（对齐 epay V3 getAppParameters）。
func TestBuildAppParamsV3(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}))

	cfg := channel.Config{AppID: "wxappid", MchID: "160", PrivateKey: privPEM}
	p, err := BuildAppParams(cfg, "prepay_xyz")
	if err != nil {
		t.Fatalf("组 APP 参数失败: %v", err)
	}
	if p["appid"] != "wxappid" || p["partnerid"] != "160" || p["prepayid"] != "prepay_xyz" || p["package"] != "Sign=WXPay" {
		t.Fatalf("字段不对: %v", p)
	}
	// 验证 sign：重建签名串，用公钥验 RSA-SHA256
	message := p["appid"] + "\n" + p["timestamp"] + "\n" + p["noncestr"] + "\n" + p["prepayid"] + "\n"
	sig, err := base64.StdEncoding.DecodeString(p["sign"])
	if err != nil {
		t.Fatalf("sign 非法 base64: %v", err)
	}
	h := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, h[:], sig); err != nil {
		t.Fatalf("APP paySign RSA 验签失败: %v", err)
	}
}

// TestTransferState 微信转账单状态 → 统一 status 映射。
func TestTransferState(t *testing.T) {
	cases := map[string]int8{
		"SUCCESS":          1,
		"FAIL":             2,
		"CANCELLED":        2,
		"PROCESSING":       0,
		"WAIT_USER_CONFIRM": 0,
		"":                 0,
	}
	for state, want := range cases {
		if got := transferState(state); got != want {
			t.Errorf("transferState(%q)=%d, 期望 %d", state, got, want)
		}
	}
}

// makeNotify 构造一份「已验签 + 已加密」的回调（自签名+自加密），供 ParseNotify 端到端测。
func makeNotify(t *testing.T, apiV3Key, outTradeNo, tradeState string, totalFen int64) (cfg channel.Config, raw map[string]string) {
	t.Helper()
	// 1. 生成平台密钥对，pub 作为 cfg.PublicKey
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkix, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pkix}))

	// 2. 业务对象 → AES-256-GCM 加密（结构对齐微信回调 resource 明文）
	plain, _ := json.Marshal(map[string]interface{}{
		"out_trade_no":   outTradeNo,
		"transaction_id": "wx-txn-1",
		"trade_state":    tradeState,
		"amount":         map[string]interface{}{"total": totalFen, "currency": "CNY"},
	})

	nonce := "abcdef123456"
	ad := "transaction"
	block, _ := aes.NewCipher([]byte(apiV3Key))
	gcm, _ := cipher.NewGCM(block)
	sealed := gcm.Seal(nil, []byte(nonce), plain, []byte(ad))
	cipherB64 := base64.StdEncoding.EncodeToString(sealed)

	// 3. 外层报文（含验签无关的富字段，JSON 解码时被忽略）
	body, _ := json.Marshal(map[string]interface{}{
		"id":            "EV-1",
		"event_type":    "TRANSACTION.SUCCESS",
		"resource_type": "encrypt-resource",
		"resource": map[string]interface{}{
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      cipherB64,
			"associated_data": ad,
			"nonce":           nonce,
			"original_type":   "transaction",
		},
	})

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
	res, err := ParseNotify(cfg, raw)
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
	res, err := ParseNotify(cfg, raw)
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
	if _, err := ParseNotify(cfg, raw); err == nil {
		t.Fatal("报文篡改后应验签失败")
	}
}

func TestParseNotifyWrongAPIv3Key(t *testing.T) {
	key := "01234567890123456789012345678901"
	cfg, raw := makeNotify(t, key, "20260721004", "SUCCESS", 10000)
	cfg.Key = "ffffffffffffffffffffffffffffffff" // 错误 APIv3 密钥，解密应失败
	if _, err := ParseNotify(cfg, raw); err == nil {
		t.Fatal("错误 APIv3 密钥应解密失败")
	}
}

func TestParseNotifyEmptyBody(t *testing.T) {
	if _, err := ParseNotify(channel.Config{}, map[string]string{}); err == nil {
		t.Fatal("空报文应报错")
	}
}

// TestParseNotifyMissingPublicKey 验签强制化：未配置平台公钥时必须拒绝回调（不得放行）。
// 对齐微信支付官方签名探测流量规范：无法验签的回调一律视为不可信。
func TestParseNotifyMissingPublicKey(t *testing.T) {
	key := "01234567890123456789012345678901"
	cfg, raw := makeNotify(t, key, "20260721005", "SUCCESS", 10000)
	cfg.PublicKey = "" // 抹掉公钥，模拟漏配
	if _, err := ParseNotify(cfg, raw); err == nil {
		t.Fatal("未配置平台公钥应拒绝回调，而非放行")
	}
}

// TestParseNotifySerialCheck 校验平台证书序列号：配了期望序列号且回调头不匹配时拒绝，匹配/未配则放行。
func TestParseNotifySerialCheck(t *testing.T) {
	key := "01234567890123456789012345678901"

	// 未配期望序列号 → 头部有值也不阻断
	cfg, raw := makeNotify(t, key, "20260728100", "SUCCESS", 10000)
	raw[KeySerial] = "WHATEVER"
	if _, err := ParseNotify(cfg, raw); err != nil {
		t.Fatalf("未配期望序列号不应阻断: %v", err)
	}

	// 配了期望序列号，头部一致（大小写无关）→ 放行
	cfg, raw = makeNotify(t, key, "20260728101", "SUCCESS", 10000)
	cfg.Extra = map[string]string{"platform_serial": "ABC123"}
	raw[KeySerial] = "abc123"
	if _, err := ParseNotify(cfg, raw); err != nil {
		t.Fatalf("序列号一致应放行: %v", err)
	}

	// 配了期望序列号，头部不一致 → 拒绝
	cfg, raw = makeNotify(t, key, "20260728102", "SUCCESS", 10000)
	cfg.Extra = map[string]string{"platform_serial": "ABC123"}
	raw[KeySerial] = "XYZ999"
	if _, err := ParseNotify(cfg, raw); err == nil {
		t.Fatal("序列号不匹配应拒绝回调")
	}
}
