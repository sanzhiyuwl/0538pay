package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/epvia/api/pkg/wxpayv3"
)

// SubMerchantService 特约商户进件域（自研扩展，微信服务商 APIv3 / 合作伙伴 / applyment4sub）。
//
// ★ 边界（对齐 docs-代理进件/01 第二节）：进件是独立 service，不实现 PaymentChannel 契约、
//   不进渠道 registry / 收单主链 / 收银台 / 插件卡片墙。
// ★ 复用 pkg/wxpayv3 的密码学原语（BuildAuthorization/VerifySignature/EncryptOAEP），
//   但自己发请求——因为进件请求必须带 Wechatpay-Serial 头（标识用哪张平台证书加密敏感字段），
//   而 wxbase.DoRequest 不设该头。这样零改动现有支付逻辑，无回归风险。
// ★ 凭证来自系统设置 wx_partner 分组（ConfigService 单例），与"微信服务商模式收单"共用同一份 sp_mchid。
type SubMerchantService struct {
	cfg *ConfigService
}

func NewSubMerchantService(cfg *ConfigService) *SubMerchantService {
	return &SubMerchantService{cfg: cfg}
}

// SubMchError 携带业务提示，handler 统一返回错误码。
type SubMchError struct{ Msg string }

func (e *SubMchError) Error() string { return e.Msg }

func smErr(msg string) *SubMchError { return &SubMchError{Msg: msg} }

// PartnerCreds 微信服务商凭证快照（从 wx_partner 配置分组读取）。
type PartnerCreds struct {
	SpMchID     string // 服务商商户号
	SpAppID     string // 服务商 appid（可选）
	SerialNo    string // 商户证书序列号
	PrivateKey  string // 商户 API 私钥 PEM
	PublicKey   string // 平台证书/微信支付公钥 PEM
	PublicKeyID string // 平台公钥/证书序列号（Wechatpay-Serial 头用）
	APIv3Key    string // APIv3 密钥
	Sandbox     bool   // 是否沙箱
}

// creds 从 ConfigService 读取当前服务商凭证。
func (s *SubMerchantService) creds() PartnerCreds {
	return PartnerCreds{
		SpMchID:     strings.TrimSpace(s.cfg.Str("wx_partner_sp_mchid")),
		SpAppID:     strings.TrimSpace(s.cfg.Str("wx_partner_sp_appid")),
		SerialNo:    strings.TrimSpace(s.cfg.Str("wx_partner_serial_no")),
		PrivateKey:  s.cfg.Str("wx_partner_private_key"),
		PublicKey:   s.cfg.Str("wx_partner_public_key"),
		PublicKeyID: strings.TrimSpace(s.cfg.Str("wx_partner_public_key_id")),
		APIv3Key:    strings.TrimSpace(s.cfg.Str("wx_partner_apiv3_key")),
		Sandbox:     s.cfg.Str("wx_partner_sandbox") != "0",
	}
}

// Configured 判断服务商凭证是否已配齐（未配则进件接口一律拒绝，避免空跑微信网关）。
func (s *SubMerchantService) Configured() bool {
	c := s.creds()
	return c.SpMchID != "" && c.SerialNo != "" && c.PrivateKey != "" && c.PublicKey != ""
}

// EncryptSensitive 用平台证书公钥 RSA-OAEP 加密敏感字段（身份证号/银行账号/手机号等）。
// 进件接口要求敏感字段密文 + 请求头 Wechatpay-Serial 标识加密用的证书。
func (s *SubMerchantService) EncryptSensitive(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return wxpayv3.EncryptOAEP(plain, s.creds().PublicKey)
}

// doRequest 发起带签名的 APIv3 服务商请求，并补 Wechatpay-Serial 头（进件必需）。
// method POST/GET；path 含 query；body 请求体 JSON（GET 传空串）。返回应答体 + HTTP 状态码。
func (s *SubMerchantService) doRequest(ctx context.Context, method, path, body string) ([]byte, int, error) {
	c := s.creds()
	if c.SpMchID == "" || c.PrivateKey == "" || c.SerialNo == "" {
		return nil, 0, smErr("微信服务商凭证未配置，请先在系统设置填写")
	}
	priv, err := wxpayv3.ParsePrivateKey(c.PrivateKey)
	if err != nil {
		return nil, 0, fmt.Errorf("解析服务商私钥失败: %w", err)
	}
	nonce, err := wxpayv3.NonceStr(32)
	if err != nil {
		return nil, 0, err
	}
	auth, err := wxpayv3.BuildAuthorization(wxpayv3.AuthParams{
		MchID:        c.SpMchID, // 服务商用自己的商户号签名
		SerialNo:     c.SerialNo,
		PrivateKey:   priv,
		Method:       method,
		CanonicalURL: path,
		Body:         body,
		Timestamp:    wxpayv3.NowUnix(),
		Nonce:        nonce,
	})
	if err != nil {
		return nil, 0, err
	}
	var reader io.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	}
	// 进件走微信正式网关；沙箱通过测试凭证/小额验证实现（微信进件无独立沙箱域名）。
	req, err := http.NewRequestWithContext(ctx, method, wxpayv3APIHost+path, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", auth)
	// ★ 进件增量：Wechatpay-Serial 头标识加密敏感字段用的平台证书序列号。
	if c.PublicKeyID != "" {
		req.Header.Set("Wechatpay-Serial", c.PublicKeyID)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求微信进件网关失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	// 验应答签名（配了平台公钥且 2xx 才验）。
	if c.PublicKey != "" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		pub, e := wxpayv3.ParsePublicKey(c.PublicKey)
		if e != nil {
			return nil, resp.StatusCode, fmt.Errorf("解析平台公钥失败: %w", e)
		}
		if e := wxpayv3.VerifySignature(pub, resp.Header.Get("Wechatpay-Timestamp"),
			resp.Header.Get("Wechatpay-Nonce"), string(respBody), resp.Header.Get("Wechatpay-Signature")); e != nil {
			return nil, resp.StatusCode, fmt.Errorf("应答验签失败: %w", e)
		}
	}
	return respBody, resp.StatusCode, nil
}

// wxpayv3APIHost 微信支付 APIv3 网关（与 wxbase.APIHost 同址；此处独立常量避免耦合渠道包）。
const wxpayv3APIHost = "https://api.mch.weixin.qq.com"

// SubmitApplymentResp 提交申请单应答（关注 applyment_id）。
type SubmitApplymentResp struct {
	ApplymentID int64 `json:"applyment_id"`
}

// SubmitApplyment 提交特约商户进件申请单。
// 接口：POST /v3/applyment4sub/applyment/（服务商体系）。
// body 为完整进件请求体 JSON（含 business_code + business_info/subject_info/identity_info/
// bank_account_info/contact_info/settlement_info，敏感字段调用方须先经 EncryptSensitive 加密）。
// 返回微信申请单号 applyment_id 与原始应答。
func (s *SubMerchantService) SubmitApplyment(ctx context.Context, body string) (*SubmitApplymentResp, []byte, error) {
	raw, code, err := s.doRequest(ctx, http.MethodPost, "/v3/applyment4sub/applyment/", body)
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("微信进件提交失败: " + wxErrMsg(raw))
	}
	var r SubmitApplymentResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析进件应答失败: %w", err)
	}
	return &r, raw, nil
}

// ApplymentStateResp 查询申请单状态应答（关注状态与 sub_mchid）。
type ApplymentStateResp struct {
	ApplymentID     int64  `json:"applyment_id"`
	ApplymentState  string `json:"applyment_state"`      // APPLYMENT_STATE_FINISHED / REJECTED / ...
	ApplymentStateMsg string `json:"applyment_state_msg"`
	SubMchID        string `json:"sub_mchid"`             // 完成后返回
	SignURL         string `json:"sign_url"`              // 待签约时返回
}

// QueryApplymentByBusinessCode 按业务申请编号查申请单状态。
// 接口：GET /v3/applyment4sub/applyment/business_code/{business_code}。
func (s *SubMerchantService) QueryApplymentByBusinessCode(ctx context.Context, businessCode string) (*ApplymentStateResp, []byte, error) {
	return s.queryApplyment(ctx, "/v3/applyment4sub/applyment/business_code/"+businessCode)
}

// QueryApplymentByID 按微信申请单号查申请单状态。
// 接口：GET /v3/applyment4sub/applyment/applyment_id/{applyment_id}。
func (s *SubMerchantService) QueryApplymentByID(ctx context.Context, applymentID int64) (*ApplymentStateResp, []byte, error) {
	return s.queryApplyment(ctx, fmt.Sprintf("/v3/applyment4sub/applyment/applyment_id/%d", applymentID))
}

func (s *SubMerchantService) queryApplyment(ctx context.Context, path string) (*ApplymentStateResp, []byte, error) {
	raw, code, err := s.doRequest(ctx, http.MethodGet, path, "")
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("微信进件状态查询失败: " + wxErrMsg(raw))
	}
	var r ApplymentStateResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析进件状态应答失败: %w", err)
	}
	return &r, raw, nil
}

// wxErrMsg 从微信错误应答体里提取 message，便于回传给运营定位。
func wxErrMsg(raw []byte) string {
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &e) == nil && e.Message != "" {
		if e.Code != "" {
			return e.Code + ": " + e.Message
		}
		return e.Message
	}
	return string(raw)
}
