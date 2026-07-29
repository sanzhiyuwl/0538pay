package ocr

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// tencentRecognizer 腾讯云 OCR 引擎。
// 接入域名 ocr.tencentcloudapi.com，服务 ocr，API 版本 2018-11-19，
// 签名走 TC3-HMAC-SHA256；营业执照 BizLicenseOCR，身份证 IDCardOCR。
// 图片以 base64（ImageBase64 字段）提交。
type tencentRecognizer struct {
	secretID  string
	secretKey string
	region    string
	host      string
	service   string
	version   string
	client    *http.Client
	// now 便于测试注入固定时间；生产用 time.Now。
	now func() time.Time
}

// NewTencent 创建腾讯云 OCR 识别引擎。region 空则默认 ap-guangzhou。
func NewTencent(secretID, secretKey, region string) Recognizer {
	region = strings.TrimSpace(region)
	if region == "" {
		region = "ap-guangzhou"
	}
	return &tencentRecognizer{
		secretID:  strings.TrimSpace(secretID),
		secretKey: strings.TrimSpace(secretKey),
		region:    region,
		host:      "ocr.tencentcloudapi.com",
		service:   "ocr",
		version:   "2018-11-19",
		client:    &http.Client{Timeout: 30 * time.Second},
		now:       time.Now,
	}
}

// tencentBizLicenseResp 腾讯云 BizLicenseOCR 响应（营业执照）。
type tencentBizLicenseResp struct {
	Response struct {
		RegNum       string `json:"RegNum"`       // 统一社会信用代码
		Name         string `json:"Name"`         // 名称
		Person       string `json:"Person"`       // 法定代表人/经营者
		Address      string `json:"Address"`      // 地址
		Type         string `json:"Type"`         // 类型
		SetDate      string `json:"SetDate"`      // 成立日期
		Period       string `json:"Period"`       // 营业期限
		Business     string `json:"Business"`     // 经营范围
		Error        *tcErr `json:"Error"`
	} `json:"Response"`
}

// tencentIDCardResp 腾讯云 IDCardOCR 响应（身份证）。
// Name/Sex/Nation/Birth/Address/IdNum 为正面；Authority/ValidDate 为反面。
type tencentIDCardResp struct {
	Response struct {
		Name      string `json:"Name"`
		Sex       string `json:"Sex"`
		Nation    string `json:"Nation"`
		Birth     string `json:"Birth"`
		Address   string `json:"Address"`
		IdNum     string `json:"IdNum"`
		Authority string `json:"Authority"` // 签发机关（反面）
		ValidDate string `json:"ValidDate"` // 有效期限，如 2018.09.01-2038.09.01（反面）
		Error     *tcErr `json:"Error"`
	} `json:"Response"`
}

type tcErr struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// tencentFriendlyErr 把腾讯云 OCR 常见错误码转成给用户看的中文引导。
// 未收录的码保留原始 [Code] Message，便于排查。
func tencentFriendlyErr(e *tcErr) error {
	switch e.Code {
	case "FailedOperation.NoBizLicense":
		return fmt.Errorf("这张图片不是营业执照，请上传清晰完整的营业执照原件照片")
	case "FailedOperation.NoIDCard", "FailedOperation.IdCardNotFound":
		return fmt.Errorf("这张图片不是身份证，请上传清晰完整的身份证照片")
	case "FailedOperation.ImageDecodeFailed":
		return fmt.Errorf("图片解码失败，请重新拍摄或更换清晰的 JPG/PNG 图片")
	case "FailedOperation.ImageBlur":
		return fmt.Errorf("图片过于模糊，请上传更清晰的证件照片")
	case "FailedOperation.OcrFailed", "FailedOperation.UnKnowError":
		return fmt.Errorf("识别失败，请换一张更清晰、无反光、完整的证件照片重试")
	case "FailedOperation.ImageSizeTooLarge":
		return fmt.Errorf("图片过大，请压缩后重试")
	case "AuthFailure.SignatureFailure", "AuthFailure.SecretIdNotFound", "AuthFailure.TokenFailure":
		return fmt.Errorf("腾讯云 OCR 密钥无效或签名失败，请检查 SecretId/SecretKey 配置")
	case "UnauthorizedOperation.Unauthorized", "ResourceUnavailable.NotExist", "FailedOperation.OcrServiceUnavailable":
		return fmt.Errorf("腾讯云账号尚未开通对应 OCR 服务，请先在腾讯云开通营业执照/身份证识别")
	case "RequestLimitExceeded", "FailedOperation.RequestLimitExceeded":
		return fmt.Errorf("识别请求过于频繁，请稍后再试")
	}
	return fmt.Errorf("[%s] %s", e.Code, e.Message)
}

func (t *tencentRecognizer) RecognizeBusinessLicense(ctx context.Context, image []byte) (*LicenseResult, error) {
	body, err := t.call(ctx, "BizLicenseOCR", image)
	if err != nil {
		return nil, err
	}
	var r tencentBizLicenseResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("解析营业执照识别结果失败: %w", err)
	}
	if r.Response.Error != nil && r.Response.Error.Code != "" {
		return nil, tencentFriendlyErr(r.Response.Error)
	}
	begin, end := splitAliyunPeriod(r.Response.Period, "", "")
	return &LicenseResult{
		RegNumber:        r.Response.RegNum,
		Name:             r.Response.Name,
		LegalPerson:      r.Response.Person,
		Address:          r.Response.Address,
		CompanyType:      r.Response.Type,
		EstablishDate:    normalizeDate(r.Response.SetDate),
		ValidPeriodBegin: begin,
		ValidPeriodEnd:   end,
		Business:         r.Response.Business,
	}, nil
}

func (t *tencentRecognizer) RecognizeIDCard(ctx context.Context, image []byte, side IDCardSide) (*IDCardResult, error) {
	body, err := t.call(ctx, "IDCardOCR", image)
	if err != nil {
		return nil, err
	}
	var r tencentIDCardResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("解析身份证识别结果失败: %w", err)
	}
	if r.Response.Error != nil && r.Response.Error.Code != "" {
		return nil, tencentFriendlyErr(r.Response.Error)
	}
	if side == IDCardBack {
		begin, end := splitAliyunPeriod(r.Response.ValidDate, "", "")
		return &IDCardResult{
			Authority:        r.Response.Authority,
			ValidPeriodBegin: begin,
			ValidPeriodEnd:   end,
		}, nil
	}
	return &IDCardResult{
		Name:     r.Response.Name,
		IDNumber: r.Response.IdNum,
		Sex:      r.Response.Sex,
		Nation:   r.Response.Nation,
		Birth:    normalizeDate(r.Response.Birth),
		Address:  r.Response.Address,
	}, nil
}

// call 发起一次腾讯云 OCR 调用，返回响应体。action 为接口名，image 为图片字节（内部转 base64）。
func (t *tencentRecognizer) call(ctx context.Context, action string, image []byte) ([]byte, error) {
	if t.secretID == "" || t.secretKey == "" {
		return nil, fmt.Errorf("腾讯云 OCR 未配置 SecretId/SecretKey")
	}
	payload, _ := json.Marshal(map[string]string{
		"ImageBase64": base64.StdEncoding.EncodeToString(image),
	})

	now := t.now().UTC()
	timestamp := now.Unix()
	date := now.Format("2006-01-02")

	// 1. 规范请求串
	canonicalHeaders := fmt.Sprintf("content-type:application/json; charset=utf-8\nhost:%s\n", t.host)
	signedHeaders := "content-type;host"
	hashedPayload := sha256Hex(payload)
	canonicalRequest := strings.Join([]string{
		http.MethodPost, "/", "", canonicalHeaders, signedHeaders, hashedPayload,
	}, "\n")

	// 2. 待签名字符串
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, t.service)
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		fmt.Sprintf("%d", timestamp),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// 3. 计算签名
	secretDate := hmacSHA256([]byte("TC3"+t.secretKey), date)
	secretService := hmacSHA256(secretDate, t.service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	// 4. 组装 Authorization
	authorization := fmt.Sprintf(
		"TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		t.secretID, credentialScope, signedHeaders, signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+t.host, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", t.host)
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", t.version)
	req.Header.Set("X-TC-Region", t.region)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))

	res, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("腾讯云 OCR 请求失败: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("腾讯云 OCR 识别失败(HTTP %d): %s", res.StatusCode, truncate(string(body), 200))
	}
	return body, nil
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
