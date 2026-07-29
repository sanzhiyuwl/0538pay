package ocr

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// aliyunRecognizer 阿里云统一 OCR 引擎（对齐 epay includes/lib/AliyunRecognize.php）。
// 接入域名 ocr-api.cn-hangzhou.aliyuncs.com，API 版本 2021-07-07，RPC 风格 + HMAC-SHA1 签名，
// 图片以 application/octet-stream 作为请求体直传，识别结果在响应 Data（JSON 字符串）中。
type aliyunRecognizer struct {
	accessKeyID     string
	accessKeySecret string
	endpoint        string
	version         string
	client          *http.Client
	// nonceSeq 用于生成 SignatureNonce（进程内自增，避免依赖随机源）。
	nonceSeq func() string
}

// NewAliyun 创建阿里云 OCR 识别引擎。
func NewAliyun(accessKeyID, accessKeySecret string) Recognizer {
	var seq int64
	return &aliyunRecognizer{
		accessKeyID:     strings.TrimSpace(accessKeyID),
		accessKeySecret: strings.TrimSpace(accessKeySecret),
		endpoint:        "ocr-api.cn-hangzhou.aliyuncs.com",
		version:         "2021-07-07",
		client:          &http.Client{Timeout: 30 * time.Second},
		nonceSeq: func() string {
			seq++
			return fmt.Sprintf("%d%d", time.Now().UnixNano(), seq)
		},
	}
}

// aliyunResp 阿里云统一 OCR 的外层响应。成功时 Data 为识别结果 JSON 字符串；
// 失败时带 Code / Message（RequestId 恒有）。
type aliyunResp struct {
	RequestID string `json:"RequestId"`
	Data      string `json:"Data"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
}

// aliyunLicenseData 营业执照识别 Data 内层结构（阿里云 2021-07-07 RecognizeBusinessLicense）。
type aliyunLicenseData struct {
	Data struct {
		CompanyName      string `json:"companyName"`
		CreditCode       string `json:"creditCode"`
		LegalPerson      string `json:"legalPerson"`
		Address          string `json:"address"`
		Type             string `json:"type"`
		EstablishDate    string `json:"establishDate"`
		ValidFromDate    string `json:"validFromDate"`
		ValidPeriod      string `json:"validPeriod"`
		ValidToDate      string `json:"validToDate"`
		BusinessScope    string `json:"businessScope"`
		RegistrationCode string `json:"registrationCode"`
	} `json:"data"`
}

// aliyunIDCardData 身份证识别 Data 内层结构（阿里云 2021-07-07 RecognizeIdcard）。
// 人像面在 face.data，国徽面在 back.data。
type aliyunIDCardData struct {
	Data struct {
		Face struct {
			Data struct {
				Name        string `json:"name"`
				Sex         string `json:"sex"`
				Nationality string `json:"nationality"`
				BirthDate   string `json:"birthDate"`
				Address     string `json:"address"`
				IDNumber    string `json:"idNumber"`
			} `json:"data"`
		} `json:"face"`
		Back struct {
			Data struct {
				IssueAuthority string `json:"issueAuthority"`
				ValidPeriod    string `json:"validPeriod"`
				StartDate      string `json:"startDate"`
				EndDate        string `json:"endDate"`
			} `json:"data"`
		} `json:"back"`
	} `json:"data"`
}

func (a *aliyunRecognizer) RecognizeBusinessLicense(ctx context.Context, image []byte) (*LicenseResult, error) {
	raw, err := a.request(ctx, "RecognizeBusinessLicense", image)
	if err != nil {
		return nil, err
	}
	var d aliyunLicenseData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return nil, fmt.Errorf("解析营业执照识别结果失败: %w", err)
	}
	reg := d.Data.CreditCode
	if reg == "" {
		reg = d.Data.RegistrationCode
	}
	begin, end := splitAliyunPeriod(d.Data.ValidPeriod, d.Data.ValidFromDate, d.Data.ValidToDate)
	return &LicenseResult{
		RegNumber:        reg,
		Name:             d.Data.CompanyName,
		LegalPerson:      d.Data.LegalPerson,
		Address:          d.Data.Address,
		CompanyType:      d.Data.Type,
		EstablishDate:    normalizeDate(d.Data.EstablishDate),
		ValidPeriodBegin: begin,
		ValidPeriodEnd:   end,
		Business:         d.Data.BusinessScope,
	}, nil
}

func (a *aliyunRecognizer) RecognizeIDCard(ctx context.Context, image []byte, side IDCardSide) (*IDCardResult, error) {
	raw, err := a.request(ctx, "RecognizeIdcard", image)
	if err != nil {
		return nil, err
	}
	var d aliyunIDCardData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return nil, fmt.Errorf("解析身份证识别结果失败: %w", err)
	}
	if side == IDCardBack {
		begin, end := splitAliyunPeriod(d.Data.Back.Data.ValidPeriod, d.Data.Back.Data.StartDate, d.Data.Back.Data.EndDate)
		return &IDCardResult{
			Authority:        d.Data.Back.Data.IssueAuthority,
			ValidPeriodBegin: begin,
			ValidPeriodEnd:   end,
		}, nil
	}
	f := d.Data.Face.Data
	return &IDCardResult{
		Name:     f.Name,
		IDNumber: f.IDNumber,
		Sex:      f.Sex,
		Nation:   f.Nationality,
		Birth:    normalizeDate(f.BirthDate),
		Address:  f.Address,
	}, nil
}

// request 发起一次阿里云 OCR 请求：签名参数拼进 query，图片作为 octet-stream 请求体。
// 返回响应 Data（识别结果 JSON 字符串）。
func (a *aliyunRecognizer) request(ctx context.Context, action string, image []byte) (string, error) {
	if a.accessKeyID == "" || a.accessKeySecret == "" {
		return "", fmt.Errorf("阿里云 OCR 未配置 AccessKeyId/AccessKeySecret")
	}
	params := map[string]string{
		"Action":           action,
		"Format":           "JSON",
		"Version":          a.version,
		"AccessKeyId":      a.accessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   a.nonceSeq(),
	}
	params["Signature"] = a.sign(params, "POST")

	reqURL := "https://" + a.endpoint + "/?" + buildQuery(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(image))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	res, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("阿里云 OCR 请求失败: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var r aliyunResp
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("阿里云 OCR 返回无法解析: %s", truncate(string(body), 200))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := r.Message
		// 去掉阿里云返回里的调试串（对齐 epay 处理）
		if i := strings.Index(msg, " server string to sign is:"); i > 0 {
			msg = msg[:i]
		}
		if r.Code != "" {
			return "", fmt.Errorf("[%s] %s", r.Code, msg)
		}
		return "", fmt.Errorf("阿里云 OCR 识别失败(HTTP %d)", res.StatusCode)
	}
	if r.Data == "" {
		return "", fmt.Errorf("阿里云 OCR 未返回识别结果")
	}
	return r.Data, nil
}

// sign 计算阿里云 RPC 风格 HMAC-SHA1 签名（对齐 epay aliyunSignature）。
func (a *aliyunRecognizer) sign(params map[string]string, method string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString("&")
		sb.WriteString(percentEncode(k))
		sb.WriteString("=")
		sb.WriteString(percentEncode(params[k]))
	}
	stringToSign := method + "&%2F&" + percentEncode(strings.TrimPrefix(sb.String(), "&"))
	mac := hmac.New(sha1.New, []byte(a.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// percentEncode 阿里云要求的 URL 编码：+→%20, *→%2A, %7E→~（对齐 epay percentEncode）。
func percentEncode(s string) string {
	e := url.QueryEscape(s)
	e = strings.ReplaceAll(e, "+", "%20")
	e = strings.ReplaceAll(e, "*", "%2A")
	e = strings.ReplaceAll(e, "%7E", "~")
	return e
}

// buildQuery 按 key 排序拼 query，值用阿里云编码规则（保证与签名一致）。
func buildQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, percentEncode(k)+"="+percentEncode(params[k]))
	}
	return strings.Join(parts, "&")
}
