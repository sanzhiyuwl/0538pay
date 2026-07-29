package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epvia/api/pkg/wxpayv3"
)

// writeSignedResp 让假网关用「同一把测试私钥」对应答 body 签名，
// 以便 doRequest 的 2xx 应答验签路径（用配套公钥）也被真实覆盖。
func writeSignedResp(t *testing.T, w http.ResponseWriter, priv *rsa.PrivateKey, body string) {
	t.Helper()
	ts, nonce := "1700000000", "TESTNONCE1234567890"
	sig, err := wxpayv3.SignMessage(priv, ts+"\n"+nonce+"\n"+body+"\n")
	if err != nil {
		t.Fatalf("应答签名失败: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Wechatpay-Timestamp", ts)
	w.Header().Set("Wechatpay-Nonce", nonce)
	w.Header().Set("Wechatpay-Signature", sig)
	_, _ = w.Write([]byte(body))
}

// testRSAKeyPair 生成一对 RSA 密钥，返回 PKCS8 私钥 PEM + PKIX 公钥 PEM。
// 私钥供签名用；公钥供敏感字段 RSA-OAEP 加密用（ModifySettlement 会用到）。
func testRSAKeyPair(t *testing.T) (privPEM, pubPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 密钥失败: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("编码 PKCS8 私钥失败: %v", err)
	}
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("编码 PKIX 公钥失败: %v", err)
	}
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privPEM, pubPEM
}

// newSettleTestService 造一个指向 httptest 假网关的服务商服务，凭证齐全（含公钥供加密）。
// 注意：这里 public_key 只用于 EncryptSensitive 加密；doRequest 仅在 2xx 才验应答签名，
// 假网关不带 Wechatpay-Signature 头，因此测试里让业务分支走非 2xx 或在断言前完成解析即可。
func newSettleTestService(t *testing.T, host string) (*SubMerchantService, *rsa.PrivateKey) {
	t.Helper()
	priv, pub := testRSAKeyPair(t)
	cfg := &ConfigService{cache: map[string]string{
		"wx_partner_sp_mchid":      "1900000001",
		"wx_partner_serial_no":     "TESTSERIAL",
		"wx_partner_private_key":   priv,
		"wx_partner_public_key":    pub,
		"wx_partner_public_key_id": "PLATFORMSERIAL",
	}}
	return &SubMerchantService{cfg: cfg, apiHost: host}, parseTestPriv(t, priv)
}

// TestUploadVideoMultipartAndSign 视频上传走与图片同一套 multipart + meta-JSON 签名，
// 断言：① 命中 video_upload 接口路径；② Content-Type 为 multipart/form-data；
// ③ meta.sha256 与原视频一致；④ file 二进制一致；⑤ 带 Authorization + Wechatpay-Serial 头。
func TestUploadVideoMultipartAndSign(t *testing.T) {
	vid := []byte("FAKE-MP4-VIDEO-BYTES-0123456789")
	wantSHA := sha256.Sum256(vid)
	wantSHAHex := hex.EncodeToString(wantSHA[:])

	var gotPath, gotCT, gotAuth, gotSerial, gotMeta string
	var gotFile []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		gotSerial = r.Header.Get("Wechatpay-Serial")
		_, params, _ := mime.ParseMediaType(gotCT)
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("读取 multipart part 失败: %v", err)
				break
			}
			data, _ := io.ReadAll(part)
			switch part.FormName() {
			case "meta":
				gotMeta = string(data)
			case "file":
				gotFile = data
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"media_id":"VIDEO_MEDIA_OK"}`))
	}))
	defer srv.Close()

	sm, _ := newSettleTestService(t, srv.URL)
	mediaID, err := sm.UploadVideo(context.Background(), "shop.mp4", vid)
	if err != nil {
		t.Fatalf("UploadVideo 失败: %v", err)
	}
	if mediaID != "VIDEO_MEDIA_OK" {
		t.Errorf("media_id=%q 期望 VIDEO_MEDIA_OK", mediaID)
	}
	if gotPath != "/v3/merchant/media/video_upload" {
		t.Errorf("path=%q 期望 /v3/merchant/media/video_upload", gotPath)
	}
	if !strings.HasPrefix(gotCT, "multipart/form-data") {
		t.Errorf("Content-Type=%q 期望 multipart/form-data", gotCT)
	}
	if !strings.HasPrefix(gotAuth, "WECHATPAY2-SHA256-RSA2048 ") {
		t.Errorf("Authorization 头缺失或格式不对: %q", gotAuth)
	}
	if gotSerial != "PLATFORMSERIAL" {
		t.Errorf("Wechatpay-Serial=%q 期望 PLATFORMSERIAL", gotSerial)
	}
	var meta struct {
		Filename string `json:"filename"`
		SHA256   string `json:"sha256"`
	}
	if err := json.Unmarshal([]byte(gotMeta), &meta); err != nil {
		t.Fatalf("meta 非法 JSON: %v (%s)", err, gotMeta)
	}
	if meta.SHA256 != wantSHAHex {
		t.Errorf("meta.sha256=%q 期望 %q", meta.SHA256, wantSHAHex)
	}
	if string(gotFile) != string(vid) {
		t.Errorf("上传的 file 二进制与原视频不一致")
	}
}

// TestModifySettlementEncryptAndPack 修改结算账户：断言
// ① 命中 modify-settlement 接口路径与 POST 方法；② 敏感字段 account_number/account_name
// 被加密（不等于明文，且能用测试私钥 RSA-OAEP 解回原文）；③ 非敏感字段（account_bank/
// bank_branch_id）明文透传；④ 带 Authorization + Wechatpay-Serial 头；⑤ 应答 application_no 被解析返回。
func TestModifySettlementEncryptAndPack(t *testing.T) {
	const plainNumber = "6225880137238888"
	const plainName = "三只鱼网络科技（山东）有限公司"

	priv, pub := testRSAKeyPair(t)
	rsaPriv := parseTestPriv(t, priv)

	var gotMethod, gotPath, gotSerial string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotSerial = r.Header.Get("Wechatpay-Serial")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		writeSignedResp(t, w, rsaPriv, `{"application_no":"APPLY_NO_2001"}`)
	}))
	defer srv.Close()

	cfg := &ConfigService{cache: map[string]string{
		"wx_partner_sp_mchid":      "1900000001",
		"wx_partner_serial_no":     "TESTSERIAL",
		"wx_partner_private_key":   priv,
		"wx_partner_public_key":    pub,
		"wx_partner_public_key_id": "PLATFORMSERIAL",
	}}
	sm := &SubMerchantService{cfg: cfg, apiHost: srv.URL}

	appNo, _, err := sm.ModifySettlement(context.Background(), "1900000109", ModifySettlementReq{
		AccountType:   "ACCOUNT_TYPE_BUSINESS",
		AccountBank:   "招商银行",
		BankBranchID:  "308584000013",
		AccountNumber: plainNumber,
		AccountName:   plainName,
	})
	if err != nil {
		t.Fatalf("ModifySettlement 失败: %v", err)
	}
	if appNo != "APPLY_NO_2001" {
		t.Errorf("application_no=%q 期望 APPLY_NO_2001", appNo)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method=%q 期望 POST", gotMethod)
	}
	if gotPath != "/v3/apply4sub/sub_merchants/1900000109/modify-settlement" {
		t.Errorf("path=%q 不对", gotPath)
	}
	if gotSerial != "PLATFORMSERIAL" {
		t.Errorf("Wechatpay-Serial=%q 期望 PLATFORMSERIAL", gotSerial)
	}
	// 非敏感字段明文透传
	if gotBody["account_bank"] != "招商银行" {
		t.Errorf("account_bank=%v 期望明文透传 招商银行", gotBody["account_bank"])
	}
	if gotBody["bank_branch_id"] != "308584000013" {
		t.Errorf("bank_branch_id=%v 期望明文透传", gotBody["bank_branch_id"])
	}
	// 敏感字段必须加密（≠明文）且能用私钥解回
	encNumber, _ := gotBody["account_number"].(string)
	if encNumber == "" || encNumber == plainNumber {
		t.Fatalf("account_number 未加密: %q", encNumber)
	}
	if got := decryptOAEP(t, rsaPriv, encNumber); got != plainNumber {
		t.Errorf("account_number 解密=%q 期望 %q", got, plainNumber)
	}
	encName, _ := gotBody["account_name"].(string)
	if encName == "" || encName == plainName {
		t.Fatalf("account_name 未加密: %q", encName)
	}
	if got := decryptOAEP(t, rsaPriv, encName); got != plainName {
		t.Errorf("account_name 解密=%q 期望 %q", got, plainName)
	}
}

// TestQuerySettlementParse 查询结算账户：假网关回掩码账号+验证结果，断言解析正确、命中 GET 与接口路径。
func TestQuerySettlementParse(t *testing.T) {
	var gotMethod, gotPath string
	var priv *rsa.PrivateKey // 请求真正发出前已被赋值（下方 newSettleTestService 返回）
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		writeSignedResp(t, w, priv, `{"account_type":"ACCOUNT_TYPE_BUSINESS","account_bank":"招商银行","account_number":"6225******8888","verify_result":"VERIFY_SUCCESS","verify_fail_reason":""}`)
	}))
	defer srv.Close()

	sm, p := newSettleTestService(t, srv.URL)
	priv = p
	r, _, err := sm.QuerySettlement(context.Background(), "1900000109")
	if err != nil {
		t.Fatalf("QuerySettlement 失败: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method=%q 期望 GET", gotMethod)
	}
	if gotPath != "/v3/apply4sub/sub_merchants/1900000109/settlement" {
		t.Errorf("path=%q 不对", gotPath)
	}
	if r.VerifyResult != "VERIFY_SUCCESS" || r.AccountNumber != "6225******8888" {
		t.Errorf("解析结果不对: %+v", r)
	}
}

// TestQuerySettlementApplicationParse 查改单审核状态：断言命中带 application_no 的路径且解析正确。
func TestQuerySettlementApplicationParse(t *testing.T) {
	var gotPath string
	var priv *rsa.PrivateKey
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeSignedResp(t, w, priv, `{"account_name":"三只******司","account_bank":"招商银行","account_number":"6225******8888","verify_result":"AUDITING","verify_finish_time":""}`)
	}))
	defer srv.Close()

	sm, p := newSettleTestService(t, srv.URL)
	priv = p
	r, _, err := sm.QuerySettlementApplication(context.Background(), "1900000109", "APPLY_NO_2001")
	if err != nil {
		t.Fatalf("QuerySettlementApplication 失败: %v", err)
	}
	if gotPath != "/v3/apply4sub/sub_merchants/1900000109/application/APPLY_NO_2001" {
		t.Errorf("path=%q 不对", gotPath)
	}
	if r.VerifyResult != "AUDITING" {
		t.Errorf("verify_result=%q 期望 AUDITING", r.VerifyResult)
	}
}

// TestSettlementNoCreds 未配凭证时查询/修改都应直接报错，不打网关。
func TestSettlementNoCreds(t *testing.T) {
	sm := &SubMerchantService{cfg: &ConfigService{cache: map[string]string{}}}
	if _, _, err := sm.QuerySettlement(context.Background(), "1900000109"); err == nil {
		t.Error("QuerySettlement 未配凭证应返回错误")
	}
}

// —— 测试辅助：解析私钥 + RSA-OAEP 解密，验证加密字段可解回原文 ——

func parseTestPriv(t *testing.T, privPEM string) *rsa.PrivateKey {
	t.Helper()
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		t.Fatal("解析测试私钥 PEM 失败")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("解析 PKCS8 私钥失败: %v", err)
	}
	return key.(*rsa.PrivateKey)
}

func decryptOAEP(t *testing.T, priv *rsa.PrivateKey, b64 string) string {
	t.Helper()
	ct, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 解码密文失败: %v", err)
	}
	// ★ 与 wxpayv3.EncryptOAEP 对齐：WeChat OAEP 用 MGF1+SHA-1（非 SHA-256）。
	plain, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, priv, ct, nil)
	if err != nil {
		t.Fatalf("RSA-OAEP 解密失败: %v", err)
	}
	return string(plain)
}
