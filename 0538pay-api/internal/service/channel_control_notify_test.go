package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/epvia/api/pkg/wxpayv3"
)

// buildControlNotify 构造一条微信风控回调（GCM 加密 resource + 信封 + 用测试私钥签名）。
// 返回 body 与验签头，模拟微信真实推送，供 verifyAndDecrypt 端到端覆盖。
func buildControlNotify(t *testing.T, priv *rsa.PrivateKey, apiV3Key, notifyID, eventType, originalType, aad string, resourceObj any) (body []byte, ts, nonce, sig string) {
	t.Helper()
	plain, err := json.Marshal(resourceObj)
	if err != nil {
		t.Fatalf("序列化 resource 失败: %v", err)
	}
	// AEAD_AES_256_GCM 加密（与 wxpayv3.DecryptAESGCM 配套：Go GCM 密文末 16 字节即 tag）。
	gcmNonce := "0123456789ab" // 12 字节
	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		t.Fatalf("建 AES cipher 失败: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("建 GCM 失败: %v", err)
	}
	sealed := gcm.Seal(nil, []byte(gcmNonce), plain, []byte(aad))
	ct := base64.StdEncoding.EncodeToString(sealed)

	env := map[string]any{
		"id":            notifyID,
		"create_time":   "2026-08-04T10:00:00+08:00",
		"event_type":    eventType,
		"resource_type": "encrypt-resource",
		"summary":       "风控回调测试",
		"resource": map[string]any{
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      ct,
			"original_type":   originalType,
			"associated_data": aad,
			"nonce":           gcmNonce,
		},
	}
	body, err = json.Marshal(env)
	if err != nil {
		t.Fatalf("序列化信封失败: %v", err)
	}
	ts, nonce = "1700000000", "TESTNONCE1234567890"
	sig, err = wxpayv3.SignMessage(priv, ts+"\n"+nonce+"\n"+string(body)+"\n")
	if err != nil {
		t.Fatalf("回调签名失败: %v", err)
	}
	return body, ts, nonce, sig
}

// newNotifyTestService 造一个凭证齐全（含 APIv3 密钥）的 notify 服务；repos/control 置空，
// 只覆盖 verifyAndDecrypt（不落库、不刷新，避免依赖 DB）。
func newNotifyTestService(t *testing.T) (*ChannelControlNotifyService, *rsa.PrivateKey, string) {
	t.Helper()
	privPEM, pubPEM := testRSAKeyPair(t)
	apiV3Key := "0123456789abcdef0123456789abcdef" // 32 字节
	cfg := &ConfigService{cache: map[string]string{
		"wx_partner_sp_mchid":     "1900000001",
		"wx_partner_serial_no":    "TESTSERIAL",
		"wx_partner_private_key":  privPEM,
		"wx_partner_public_key":   pubPEM,
		"wx_partner_apiv3_key":    apiV3Key,
	}}
	sm := &SubMerchantService{cfg: cfg, apiHost: "https://example.test"}
	svc := NewChannelControlNotifyService(nil, nil, sm, nil)
	return svc, parseTestPriv(t, privPEM), apiV3Key
}

// TestVerifyAndDecryptViolation 端到端覆盖 (A) 商户平台处置通知：验签 + GCM 解密 + 字段还原。
func TestVerifyAndDecryptViolation(t *testing.T) {
	svc, priv, apiV3Key := newNotifyTestService(t)
	res := violationResource{
		SubMchID:          "1900001234",
		CompanyName:       "示例商户有限公司",
		RecordID:          "REC-20260804-001",
		PunishPlan:        "暂停收款",
		PunishTime:        "2026-08-04T10:00:00+08:00",
		PunishDescription: "涉嫌违规交易",
		RiskType:          "RISK_TYPE_FRAUD",
		RiskDescription:   "存在欺诈风险",
	}
	body, ts, nonce, sig := buildControlNotify(t, priv, apiV3Key, "EV-001", "VIOLATION.PUNISH", "violation", "", res)
	env, plain, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: sig, Timestamp: ts, Nonce: nonce}, body)
	if err != nil {
		t.Fatalf("验签+解密应成功: %v", err)
	}
	if env.ID != "EV-001" || env.EventType != "VIOLATION.PUNISH" {
		t.Fatalf("信封字段错: id=%s event=%s", env.ID, env.EventType)
	}
	var got violationResource
	if err := json.Unmarshal(plain, &got); err != nil {
		t.Fatalf("解密明文解析失败: %v", err)
	}
	if got.SubMchID != res.SubMchID || got.RecordID != res.RecordID || got.RiskType != res.RiskType {
		t.Fatalf("解密字段还原不一致: %+v", got)
	}
}

// TestVerifyAndDecryptMerchantNotify 端到端覆盖 (B) 合作伙伴订阅·管控流水（AAD=merchant_notify）。
func TestVerifyAndDecryptMerchantNotify(t *testing.T) {
	svc, priv, apiV3Key := newNotifyTestService(t)
	res := merchantNotifyResource{}
	res.MessageContent.MerchantCode = "1900005678"
	res.MessageContent.MerchantCompanyName = "示例商户二号"
	res.MessageContent.BusinessTime = "2026-08-04T11:00:00+08:00"
	res.MessageContent.BusinessCode = "CASE-88888"
	res.MessageContent.BusinessState = "PUNISHMENT"
	body, ts, nonce, sig := buildControlNotify(t, priv, apiV3Key, "d1dbc742-uuid", "MERCHANT_NOTIFY.NOTIFY", "merchant_notify", "merchant_notify", res)
	env, plain, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: sig, Timestamp: ts, Nonce: nonce}, body)
	if err != nil {
		t.Fatalf("验签+解密应成功: %v", err)
	}
	if env.EventType != "MERCHANT_NOTIFY.NOTIFY" {
		t.Fatalf("event_type 错: %s", env.EventType)
	}
	var got merchantNotifyResource
	if err := json.Unmarshal(plain, &got); err != nil {
		t.Fatalf("解密明文解析失败: %v", err)
	}
	if got.MessageContent.MerchantCode != "1900005678" || got.MessageContent.BusinessCode != "CASE-88888" || got.MessageContent.BusinessState != "PUNISHMENT" {
		t.Fatalf("解密字段还原不一致: %+v", got.MessageContent)
	}
}

// TestVerifyAndDecryptRejectTamper 篡改签名/报文必须验签失败被拒。
func TestVerifyAndDecryptRejectTamper(t *testing.T) {
	svc, priv, apiV3Key := newNotifyTestService(t)
	body, ts, nonce, sig := buildControlNotify(t, priv, apiV3Key, "EV-002", "VIOLATION.PUNISH", "violation", "", violationResource{SubMchID: "1900001234"})
	// 篡改 body（多加一个字节）→ 验签必然失败。
	tampered := append([]byte(nil), body...)
	tampered = append(tampered, ' ')
	if _, _, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: sig, Timestamp: ts, Nonce: nonce}, tampered); err == nil {
		t.Fatal("篡改报文应验签失败")
	}
}

// TestVerifyAndDecryptRejectSignTest 签名探测流量（WECHATPAY/SIGNTEST/）必须被拒。
func TestVerifyAndDecryptRejectSignTest(t *testing.T) {
	svc, priv, apiV3Key := newNotifyTestService(t)
	body, ts, nonce, _ := buildControlNotify(t, priv, apiV3Key, "EV-003", "VIOLATION.PUNISH", "violation", "", violationResource{SubMchID: "1900001234"})
	probeSig := "WECHATPAY/SIGNTEST/" + base64.StdEncoding.EncodeToString([]byte("probe"))
	if _, _, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: probeSig, Timestamp: ts, Nonce: nonce}, body); err == nil {
		t.Fatal("签名探测流量应被拒绝")
	}
}
