package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CertVerifyService 实名认证第三方核验（对齐 epay user/ajax2 certificate 的 cert_open 分派）。
//
// cert_open：1=支付宝身份认证(人脸) / 2=手机号三要素 / 3=支付宝实名比对 / 4=腾讯云实名 / 5=阿里云金融级实人。
// 各方式请求构造/签名 1:1 对齐 epay，真实核验需对应第三方凭证（APPCODE/SecretKey/渠道私钥）。
// 手机三要素(方式2,阿里云市场 APPCODE)为同步核验，凭证到位即可端到端；其余为异步/需渠道私钥，凭证到位后接回调。
type CertVerifyService struct {
	cfg *ConfigService
}

func NewCertVerifyService(cfg *ConfigService) *CertVerifyService {
	return &CertVerifyService{cfg: cfg}
}

const certHTTPTimeout = 10 * time.Second

// Verify 按 cert_open 方式核验姓名+证件号(+手机号)。返回 (通过, error)。
// 无法同步判定（人脸类异步）时返回明确的待凭证/待回调错误。
func (s *CertVerifyService) Verify(ctx context.Context, name, idNo, phone string) (bool, error) {
	switch s.cfg.Int("cert_open", 0) {
	case 0:
		return false, maErr("平台未开启实名认证")
	case 2:
		return s.verifyPhone3(ctx, name, idNo, phone)
	case 1, 3:
		return false, maErr("支付宝实名认证需配置支付宝应用私钥(RSA2 网关签名)并接入异步回调，待真实凭证")
	case 4:
		return false, maErr("腾讯云实名认证需配置 SecretId/SecretKey 并接入扫码回调，待真实凭证")
	case 5:
		return false, maErr("阿里云金融级实人认证需配置 AccessKey 并接入异步回调，待真实凭证")
	}
	return false, maErr("未知的实名认证方式")
}

// verifyPhone3 手机号三要素核验（对齐 epay check_cert，阿里云市场 API + APPCODE）。
// GET phone3.market.alicloudapi.com/phonethree?idcard&phone&realname，Header Authorization: APPCODE {cert_appcode}。
// 同步返回 code==200 即通过。需真实 APPCODE 才能真调。
func (s *CertVerifyService) verifyPhone3(ctx context.Context, name, idNo, phone string) (bool, error) {
	appcode := strings.TrimSpace(s.cfg.Str("cert_appcode"))
	if appcode == "" {
		return false, maErr("手机三要素认证未配置 APPCODE(cert_appcode)，待真实凭证")
	}
	if phone == "" {
		return false, maErr("请先绑定手机号再做三要素认证")
	}
	q := url.Values{}
	q.Set("idcard", idNo)
	q.Set("phone", phone)
	q.Set("realname", name)
	reqURL := "http://phone3.market.alicloudapi.com/phonethree?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "APPCODE "+appcode)
	resp, err := (&http.Client{Timeout: certHTTPTimeout}).Do(req)
	if err != nil {
		return false, fmt.Errorf("请求三要素核验失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return false, maErr("三要素核验未通过或 APPCODE 无效")
	}
	// 阿里云市场返回体判 code==200/结果字段（各服务略有差异，取通用 code）。
	var r struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &r)
	return r.Code == 200, nil
}
