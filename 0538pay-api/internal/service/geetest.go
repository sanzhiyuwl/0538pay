package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GeetestService 极验行为验证（对齐 epay \lib\GeetestLib）。
//
// 支持 V4（captcha_version=1，主推）与 V3（其它）。凭证 captcha_id/captcha_key 来自 config reg 分组。
// V4：前端 initGeetest4({captchaId}) 完成滑块 → 提交 captcha_output 等 → 服务端二次校验(gcaptcha4 validate)。
// 无凭证时校验直接放行降级由调用方决定（与图形验证码二选一）。
type GeetestService struct {
	cfg *ConfigService
}

func NewGeetestService(cfg *ConfigService) *GeetestService {
	return &GeetestService{cfg: cfg}
}

const geetestHTTPTimeout = 8 * time.Second

// Enabled 是否启用极验（配置了 captcha_id 才算启用）。
func (s *GeetestService) Enabled() bool {
	return s.cfg.Str("captcha_id") != ""
}

// InitParams 返回前端初始化所需参数（V4 只需 gt + version）。
func (s *GeetestService) InitParams() map[string]interface{} {
	v4 := s.cfg.Str("captcha_version") == "1"
	out := map[string]interface{}{
		"gt":      s.cfg.Str("captcha_id"),
		"version": map[bool]int{true: 1, false: 0}[v4],
	}
	return out
}

// GeetestV4Params V4 前端提交的二次校验参数。
type GeetestV4Params struct {
	LotNumber     string `json:"lot_number"`
	CaptchaOutput string `json:"captcha_output"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
}

// VerifyV4 极验 V4 服务端二次校验（对齐 epay gt4_validate）。
// sign_token = HMAC-SHA256(lot_number, captcha_key)，GET gcaptcha4.geetest.com/validate，判 result=success。
func (s *GeetestService) VerifyV4(ctx context.Context, p GeetestV4Params) (bool, error) {
	captchaID := s.cfg.Str("captcha_id")
	captchaKey := s.cfg.Str("captcha_key")
	if captchaID == "" || captchaKey == "" {
		return false, maErr("极验未配置 captcha_id/captcha_key，待真实凭证")
	}
	mac := hmac.New(sha256.New, []byte(captchaKey))
	mac.Write([]byte(p.LotNumber))
	signToken := hex.EncodeToString(mac.Sum(nil))

	form := url.Values{}
	form.Set("lot_number", p.LotNumber)
	form.Set("captcha_output", p.CaptchaOutput)
	form.Set("pass_token", p.PassToken)
	form.Set("gen_time", p.GenTime)
	form.Set("sign_token", signToken)
	form.Set("captcha_id", captchaID)

	reqURL := "http://gcaptcha4.geetest.com/validate?captcha_id=" + url.QueryEscape(captchaID)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, nil)
	if err != nil {
		return false, err
	}
	req.URL.RawQuery = form.Encode()
	resp, err := (&http.Client{Timeout: geetestHTTPTimeout}).Do(req)
	if err != nil {
		return false, maErr("极验校验请求失败: " + err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	_ = json.Unmarshal(body, &r)
	// 极验规范：请求异常(status!=success)时按"宕机放行"策略处理，避免误伤正常用户。
	if r.Status != "success" {
		return true, nil
	}
	return r.Result == "success", nil
}
