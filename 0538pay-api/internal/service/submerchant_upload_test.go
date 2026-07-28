package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
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
)

// testPrivateKeyPEM 生成一把 2048 位 RSA 私钥的 PKCS8 PEM（供签名用；进件上传不验应答签名）。
func testPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成 RSA 私钥失败: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("编码 PKCS8 失败: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// TestUploadMediaMultipartAndSign 验证图片上传的 multipart 组装 + meta-JSON 签名 body + Wechatpay-Serial 头，
// 用 httptest 假网关接收请求并断言：① Content-Type 为 multipart/form-data；② meta 字段含正确 sha256；
// ③ file 字段二进制与原图一致；④ 带 Authorization + Wechatpay-Serial 头。返回 media_id 能被解析回填。
// ★ 这条覆盖真实凭证不可达时无法端到端验证的「打包 + 签名」核心路径。
func TestUploadMediaMultipartAndSign(t *testing.T) {
	img := []byte("\x89PNG\r\n\x1a\nHELLO-FAKE-IMAGE-BYTES")
	wantSHA := sha256.Sum256(img)
	wantSHAHex := hex.EncodeToString(wantSHA[:])

	var gotCT, gotAuth, gotSerial, gotMeta string
	var gotFile []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		gotSerial = r.Header.Get("Wechatpay-Serial")
		// 解析 multipart
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
		_, _ = w.Write([]byte(`{"media_id":"MEDIA_ID_OK_123"}`))
	}))
	defer srv.Close()

	cfg := &ConfigService{cache: map[string]string{
		"wx_partner_sp_mchid":      "1900000001",
		"wx_partner_serial_no":     "TESTSERIAL",
		"wx_partner_private_key":   testPrivateKeyPEM(t),
		"wx_partner_public_key_id": "PLATFORMSERIAL",
	}}
	sm := &SubMerchantService{cfg: cfg, apiHost: srv.URL}

	mediaID, err := sm.UploadMedia(context.Background(), "license.png", img)
	if err != nil {
		t.Fatalf("UploadMedia 失败: %v", err)
	}
	if mediaID != "MEDIA_ID_OK_123" {
		t.Errorf("media_id=%q 期望 MEDIA_ID_OK_123", mediaID)
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
	// meta 应含正确 sha256
	var meta struct {
		Filename string `json:"filename"`
		SHA256   string `json:"sha256"`
	}
	if err := json.Unmarshal([]byte(gotMeta), &meta); err != nil {
		t.Fatalf("meta 非法 JSON: %v (%s)", err, gotMeta)
	}
	if meta.Filename != "license.png" {
		t.Errorf("meta.filename=%q 期望 license.png", meta.Filename)
	}
	if meta.SHA256 != wantSHAHex {
		t.Errorf("meta.sha256=%q 期望 %q", meta.SHA256, wantSHAHex)
	}
	// file 二进制应与原图逐字节一致
	if string(gotFile) != string(img) {
		t.Errorf("上传的 file 二进制与原图不一致")
	}
}

// TestUploadMediaNoCreds 未配凭证时应直接报错，不打网关。
func TestUploadMediaNoCreds(t *testing.T) {
	sm := &SubMerchantService{cfg: &ConfigService{cache: map[string]string{}}}
	if _, err := sm.UploadMedia(context.Background(), "a.png", []byte("x")); err == nil {
		t.Error("未配凭证应返回错误")
	}
}
