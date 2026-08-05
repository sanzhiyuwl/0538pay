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

// buildComplaintNotify 构造一条微信投诉回调（GCM 加密 resource + 信封 + 测试私钥签名），
// 模拟微信真实推送，供 WxComplaintService.verifyAndDecrypt 端到端覆盖验签+解密路径。
func buildComplaintNotify(t *testing.T, priv *rsa.PrivateKey, apiV3Key, notifyID, eventType string, resourceObj any) (body []byte, ts, nonce, sig string) {
	t.Helper()
	plain, err := json.Marshal(resourceObj)
	if err != nil {
		t.Fatalf("序列化 resource 失败: %v", err)
	}
	gcmNonce := "0123456789ab" // 12 字节
	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		t.Fatalf("建 AES cipher 失败: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("建 GCM 失败: %v", err)
	}
	aad := "complaint"
	sealed := gcm.Seal(nil, []byte(gcmNonce), plain, []byte(aad))
	ct := base64.StdEncoding.EncodeToString(sealed)

	env := map[string]any{
		"id":            notifyID,
		"create_time":   "2026-08-04T10:00:00+08:00",
		"event_type":    eventType,
		"resource_type": "encrypt-resource",
		"summary":       "投诉回调测试",
		"resource": map[string]any{
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      ct,
			"original_type":   "complaint",
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

// newComplaintTestService 造一个凭证齐全（含 APIv3 密钥）的投诉服务；repos 置空，
// 只覆盖 verifyAndDecrypt（不落库、不刷新，避免依赖 DB）。
func newComplaintTestService(t *testing.T) (*WxComplaintService, *rsa.PrivateKey, string) {
	t.Helper()
	privPEM, pubPEM := testRSAKeyPair(t)
	apiV3Key := "0123456789abcdef0123456789abcdef" // 32 字节
	cfg := &ConfigService{cache: map[string]string{
		"wx_partner_sp_mchid":    "1900000001",
		"wx_partner_serial_no":   "TESTSERIAL",
		"wx_partner_private_key": privPEM,
		"wx_partner_public_key":  pubPEM,
		"wx_partner_apiv3_key":   apiV3Key,
	}}
	sm := &SubMerchantService{cfg: cfg, apiHost: "https://example.test"}
	svc := NewWxComplaintService(nil, nil, nil, sm, cfg)
	return svc, parseTestPriv(t, privPEM), apiV3Key
}

// TestComplaintVerifyAndDecrypt 端到端覆盖投诉回调：验签 + GCM 解密 + complaint_id/action_type 还原。
func TestComplaintVerifyAndDecrypt(t *testing.T) {
	svc, priv, apiV3Key := newComplaintTestService(t)
	res := complaintNotifyResource{
		ComplaintID: "200201820200101080076610000",
		ActionType:  "CREATE_COMPLAINT",
	}
	body, ts, nonce, sig := buildComplaintNotify(t, priv, apiV3Key, "EV-C-001", "COMPLAINT.CREATE", res)
	env, got, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: sig, Timestamp: ts, Nonce: nonce}, body)
	if err != nil {
		t.Fatalf("验签+解密应成功: %v", err)
	}
	if env.ID != "EV-C-001" || env.EventType != "COMPLAINT.CREATE" {
		t.Fatalf("信封字段错: id=%s event=%s", env.ID, env.EventType)
	}
	if got.ComplaintID != res.ComplaintID || got.ActionType != res.ActionType {
		t.Fatalf("解密字段还原不一致: %+v", got)
	}
	if got.raw == "" {
		t.Fatal("raw 明文应回填供落流水")
	}
}

// TestComplaintVerifyStateChange 覆盖状态变更事件（COMPLAINT.STATE_CHANGE）。
func TestComplaintVerifyStateChange(t *testing.T) {
	svc, priv, apiV3Key := newComplaintTestService(t)
	res := complaintNotifyResource{ComplaintID: "200201820200101080076610001", ActionType: "SELLER_REFUND"}
	body, ts, nonce, sig := buildComplaintNotify(t, priv, apiV3Key, "EV-C-002", "COMPLAINT.STATE_CHANGE", res)
	env, got, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: sig, Timestamp: ts, Nonce: nonce}, body)
	if err != nil {
		t.Fatalf("验签+解密应成功: %v", err)
	}
	if env.EventType != "COMPLAINT.STATE_CHANGE" || got.ActionType != "SELLER_REFUND" {
		t.Fatalf("字段还原不一致: event=%s action=%s", env.EventType, got.ActionType)
	}
}

// TestComplaintRejectTamper 篡改报文必须验签失败被拒。
func TestComplaintRejectTamper(t *testing.T) {
	svc, priv, apiV3Key := newComplaintTestService(t)
	body, ts, nonce, sig := buildComplaintNotify(t, priv, apiV3Key, "EV-C-003", "COMPLAINT.CREATE", complaintNotifyResource{ComplaintID: "X"})
	tampered := append([]byte(nil), body...)
	tampered = append(tampered, ' ')
	if _, _, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: sig, Timestamp: ts, Nonce: nonce}, tampered); err == nil {
		t.Fatal("篡改报文应验签失败")
	}
}

// TestComplaintRejectSignTest 签名探测流量（WECHATPAY/SIGNTEST/）必须被拒。
func TestComplaintRejectSignTest(t *testing.T) {
	svc, priv, apiV3Key := newComplaintTestService(t)
	body, ts, nonce, _ := buildComplaintNotify(t, priv, apiV3Key, "EV-C-004", "COMPLAINT.CREATE", complaintNotifyResource{ComplaintID: "X"})
	probeSig := "WECHATPAY/SIGNTEST/" + base64.StdEncoding.EncodeToString([]byte("probe"))
	if _, _, err := svc.verifyAndDecrypt(NotifyHeaders{Signature: probeSig, Timestamp: ts, Nonce: nonce}, body); err == nil {
		t.Fatal("签名探测流量应被拒绝")
	}
}

// TestMaskPhone 手机号脱敏前3后4。
func TestMaskPhone(t *testing.T) {
	cases := map[string]string{
		"13800138000": "138****8000",
		"12345":       "****",
		"":            "",
	}
	for in, want := range cases {
		if got := maskPhone(in); got != want {
			t.Fatalf("maskPhone(%q)=%q, 期望 %q", in, got, want)
		}
	}
}
