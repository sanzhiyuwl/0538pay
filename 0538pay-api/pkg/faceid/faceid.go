// Package faceid 提供腾讯云慧眼（FaceID）实名核身能力，用于商户「微信扫码实名认证」。
//
// 与 pkg/ocr 的腾讯云 OCR 是两个独立产品：OCR 只识别证件文字，FaceID 才做人脸活体+权威库核身。
// 域名 faceid.tencentcloudapi.com，服务 faceid，API 版本 2018-03-01，签名走 TC3-HMAC-SHA256。
//
// 核身是两步异步流程（1:1 对齐 epay includes/lib/QcloudFaceid.php）：
//  1. GetRealNameAuthToken(姓名, 身份证号, 回调URL) → 拿到 AuthToken + RedirectURL(二维码链接)；
//     用户微信扫 RedirectURL 做人脸活体，完成后腾讯云带 AuthToken 回跳 回调URL。
//  2. GetRealNameAuthResult(AuthToken) → ResultType==0 即核身通过。
//
// 真实核身需在腾讯云开通「慧眼 FaceID 实名核身」服务并配置 SecretId/SecretKey（与 OCR 相互独立）。
package faceid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client 腾讯云 FaceID 客户端。
type Client struct {
	secretID  string
	secretKey string
	region    string
	host      string
	service   string
	version   string
	http      *http.Client
	// now 便于测试注入固定时间；生产用 time.Now。
	now func() time.Time
}

// New 创建腾讯云 FaceID 客户端。region 空则默认 ap-guangzhou（对齐 epay）。
func New(secretID, secretKey, region string) *Client {
	region = strings.TrimSpace(region)
	if region == "" {
		region = "ap-guangzhou"
	}
	return &Client{
		secretID:  strings.TrimSpace(secretID),
		secretKey: strings.TrimSpace(secretKey),
		region:    region,
		host:      "faceid.tencentcloudapi.com",
		service:   "faceid",
		version:   "2018-03-01",
		http:      &http.Client{Timeout: 15 * time.Second},
		now:       time.Now,
	}
}

// AuthTokenResult GetRealNameAuthToken 返回：AuthToken 用于回调查结果，RedirectURL 是给用户扫的二维码链接。
type AuthTokenResult struct {
	AuthToken   string
	RedirectURL string
}

// AuthResult GetRealNameAuthResult 返回的核身结果。
// ResultType: 0=通过 / -1=姓名与身份证号不一致 / -2=姓名与微信实名不一致 / -3=微信号未实名。
type AuthResult struct {
	ResultType  string
	Description string
}

// tcErr 腾讯云统一错误体。
type tcErr struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// GetRealNameAuthToken 发起实名核身，返回 AuthToken + 扫码链接（对齐 epay QcloudFaceid::GetRealNameAuthToken）。
func (c *Client) GetRealNameAuthToken(ctx context.Context, name, idCard, callbackURL string) (*AuthTokenResult, error) {
	body, err := c.call(ctx, "GetRealNameAuthToken", map[string]string{
		"Name":        name,
		"IDCard":      idCard,
		"CallbackURL": callbackURL,
	})
	if err != nil {
		return nil, err
	}
	var r struct {
		Response struct {
			AuthToken   string `json:"AuthToken"`
			RedirectURL string `json:"RedirectURL"`
			Error       *tcErr `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("解析核身发起结果失败: %w", err)
	}
	if r.Response.Error != nil && r.Response.Error.Code != "" {
		return nil, friendlyErr(r.Response.Error)
	}
	if r.Response.AuthToken == "" {
		return nil, fmt.Errorf("腾讯云未返回 AuthToken，请稍后重试")
	}
	return &AuthTokenResult{AuthToken: r.Response.AuthToken, RedirectURL: r.Response.RedirectURL}, nil
}

// GetRealNameAuthResult 查询核身结果（对齐 epay QcloudFaceid::GetRealNameAuthResult）。
func (c *Client) GetRealNameAuthResult(ctx context.Context, authToken string) (*AuthResult, error) {
	body, err := c.call(ctx, "GetRealNameAuthResult", map[string]string{"AuthToken": authToken})
	if err != nil {
		return nil, err
	}
	var r struct {
		Response struct {
			ResultType  string `json:"ResultType"`
			Description string `json:"Description"`
			Error       *tcErr `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("解析核身查询结果失败: %w", err)
	}
	if r.Response.Error != nil && r.Response.Error.Code != "" {
		return nil, friendlyErr(r.Response.Error)
	}
	if r.Response.ResultType == "" {
		return nil, fmt.Errorf("腾讯云未返回核身结果 ResultType")
	}
	return &AuthResult{ResultType: r.Response.ResultType, Description: r.Response.Description}, nil
}

// ResultMessage 把 ResultType 转成给用户看的中文（对齐 epay alipaycertok.php 微信扫码分支）。
func ResultMessage(resultType string) string {
	switch resultType {
	case "0":
		return "实名认证通过"
	case "-1":
		return "实名认证未通过：姓名和身份证号不一致"
	case "-2":
		return "实名认证未通过：姓名和微信实名姓名不一致"
	case "-3":
		return "实名认证未通过：微信号未实名"
	default:
		return "实名认证未通过（ResultType=" + resultType + "）"
	}
}

// friendlyErr 把腾讯云 FaceID 常见错误码转成中文引导。
func friendlyErr(e *tcErr) error {
	switch e.Code {
	case "AuthFailure.SignatureFailure", "AuthFailure.SecretIdNotFound", "AuthFailure.TokenFailure":
		return fmt.Errorf("腾讯云实名核身密钥无效或签名失败，请检查 SecretId/SecretKey 配置")
	case "UnauthorizedOperation.Unauthorized", "ResourceUnavailable.NotExist", "FailedOperation.NotOpen":
		return fmt.Errorf("腾讯云账号尚未开通「慧眼 FaceID 实名核身」服务，请先在腾讯云开通")
	case "LimitExceeded.Finance", "FailedOperation.QpsLimitExceeded", "RequestLimitExceeded":
		return fmt.Errorf("实名核身请求过于频繁或账户余额不足，请稍后再试")
	case "InvalidParameter.IDCard", "InvalidParameterValue.IDCardError":
		return fmt.Errorf("身份证号格式不正确")
	}
	return fmt.Errorf("[%s] %s", e.Code, e.Message)
}
