package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	cfg     *ConfigService
	apiHost string // 微信 APIv3 网关地址；默认正式网关，单测可指向 httptest 服务器
}

func NewSubMerchantService(cfg *ConfigService) *SubMerchantService {
	return &SubMerchantService{cfg: cfg, apiHost: wxpayv3APIHost}
}

// host 返回当前网关地址（未设置则回退正式网关）。
func (s *SubMerchantService) host() string {
	if s.apiHost == "" {
		return wxpayv3APIHost
	}
	return s.apiHost
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
}

// creds 从 ConfigService 读取当前服务商凭证。
func (s *SubMerchantService) creds() PartnerCreds {
	return PartnerCreds{
		SpMchID:     strings.TrimSpace(s.cfg.Str("wx_partner_sp_mchid")),
		SpAppID:     strings.TrimSpace(s.cfg.Str("wx_partner_sp_appid")),
		SerialNo:    strings.TrimSpace(s.cfg.Str("wx_partner_serial_no")),
		// 私钥/公钥支持文件指针 @file:xxx.pem（存 secrets/，不入库不对外），resolveSecret 读回原文；
		// 旧明文数据无前缀原样返回，向后兼容。
		PrivateKey:  resolveSecret(s.cfg.Str("wx_partner_private_key")),
		PublicKey:   resolveSecret(s.cfg.Str("wx_partner_public_key")),
		PublicKeyID: strings.TrimSpace(s.cfg.Str("wx_partner_public_key_id")),
		APIv3Key:    strings.TrimSpace(s.cfg.Str("wx_partner_apiv3_key")),
	}
}

// Configured 判断服务商凭证是否已配齐（未配则进件接口一律拒绝，避免空跑微信网关）。
func (s *SubMerchantService) Configured() bool {
	c := s.creds()
	return c.SpMchID != "" && c.SerialNo != "" && c.PrivateKey != "" && c.PublicKey != ""
}

// PartnerCredView 服务商凭证脱敏视图（供设置页 GET 回填）。
// ★ 敏感字段（私钥/公钥/APIv3 密钥）一律不回原文，只回"是否已配 + 内容指纹"供核对；
//   非机密字段（商户号/序列号/appid/公钥ID）明文回显便于编辑。
type PartnerCredView struct {
	SpMchID       string `json:"sp_mchid"`
	SpAppID       string `json:"sp_appid"`
	SerialNo      string `json:"serial_no"`
	PublicKeyID   string `json:"public_key_id"`
	HasPrivateKey bool   `json:"has_private_key"`
	HasPublicKey  bool   `json:"has_public_key"`
	HasAPIv3Key   bool   `json:"has_apiv3_key"`
	PrivateKeyFp  string `json:"private_key_fp"` // 私钥内容指纹（SHA256 前 12 位），空=未配
	PublicKeyFp   string `json:"public_key_fp"`  // 公钥内容指纹
	APIv3KeyFp    string `json:"apiv3_key_fp"`   // APIv3 密钥指纹
	Configured    bool   `json:"configured"`     // 四项必填是否齐全
}

// CredView 返回脱敏后的服务商凭证视图（敏感字段不回原文）。
func (s *SubMerchantService) CredView() PartnerCredView {
	c := s.creds()
	apiv3 := c.APIv3Key
	return PartnerCredView{
		SpMchID:       c.SpMchID,
		SpAppID:       c.SpAppID,
		SerialNo:      c.SerialNo,
		PublicKeyID:   c.PublicKeyID,
		HasPrivateKey: c.PrivateKey != "",
		HasPublicKey:  c.PublicKey != "",
		HasAPIv3Key:   apiv3 != "",
		PrivateKeyFp:  secretFingerprint(c.PrivateKey),
		PublicKeyFp:   secretFingerprint(c.PublicKey),
		APIv3KeyFp:    secretFingerprint(apiv3),
		Configured:    s.Configured(),
	}
}

// PartnerCredUpdate 服务商凭证保存入参。
// 明文字段直接覆盖；私钥/公钥留空=不改（保留现有），非空=校验合法后写 secrets/ 文件。
// APIv3Key 留空=不改，非空=覆盖（存 config，长度须 32）。
type PartnerCredUpdate struct {
	SpMchID     string
	SpAppID     string
	SerialNo    string
	PublicKeyID string
	PrivateKey  string // 留空不改
	PublicKey   string // 留空不改
	APIv3Key    string // 留空不改
}

// SaveCreds 保存服务商凭证：明文字段入 config，私钥/公钥校验合法后落 secrets/ 文件并把 config 值置为文件指针。
// ★ 私钥/公钥"留空=不改"，避免脱敏回显（前端拿不到原文）导致误清空。
func (s *SubMerchantService) SaveCreds(u PartnerCredUpdate) error {
	kv := map[string]string{
		"wx_partner_sp_mchid":      strings.TrimSpace(u.SpMchID),
		"wx_partner_sp_appid":      strings.TrimSpace(u.SpAppID),
		"wx_partner_serial_no":     strings.TrimSpace(u.SerialNo),
		"wx_partner_public_key_id": strings.TrimSpace(u.PublicKeyID),
	}
	// 私钥：非空才改。校验能解析为 RSA 私钥后写文件。
	if pk := strings.TrimSpace(u.PrivateKey); pk != "" {
		if _, err := wxpayv3.ParsePrivateKey(pk); err != nil {
			return smErr("商户 API 私钥格式不正确（须为 PEM 格式的 RSA 私钥）")
		}
		ref, err := saveSecretFile("wx_partner_private_key.pem", pk)
		if err != nil {
			return smErr("保存私钥文件失败: " + err.Error())
		}
		kv["wx_partner_private_key"] = ref
	}
	// 公钥：非空才改。校验能解析为 RSA 公钥后写文件。
	if pub := strings.TrimSpace(u.PublicKey); pub != "" {
		if _, err := wxpayv3.ParsePublicKey(pub); err != nil {
			return smErr("微信支付公钥格式不正确（须为 PEM 格式的 RSA 公钥/证书）")
		}
		ref, err := saveSecretFile("wx_partner_public_key.pem", pub)
		if err != nil {
			return smErr("保存公钥文件失败: " + err.Error())
		}
		kv["wx_partner_public_key"] = ref
	}
	// APIv3 密钥：非空才改（长度须 32）。仍存 config（非文件，回调解密频繁读）。
	if k := strings.TrimSpace(u.APIv3Key); k != "" {
		if len(k) != 32 {
			return smErr("APIv3 密钥长度必须为 32 位")
		}
		kv["wx_partner_apiv3_key"] = k
	}
	// SaveGroup 仅写入 map 中出现的白名单键（未提供的键保持原值），天然实现"留空不改"。
	return s.cfg.SaveGroup("wx_partner", kv)
}

// EncryptSensitive 用平台证书公钥 RSA-OAEP 加密敏感字段（身份证号/银行账号/手机号等）。
// 进件接口要求敏感字段密文 + 请求头 Wechatpay-Serial 标识加密用的证书。
func (s *SubMerchantService) EncryptSensitive(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return wxpayv3.EncryptOAEP(plain, s.creds().PublicKey)
}

// UploadMedia 上传进件资料图片（营业执照/身份证正反面等），返回微信 media_id。
// 接口：POST /v3/merchant/media/upload（multipart/form-data）。JPG/PNG/BMP ≤2M（调用方已校验）。
func (s *SubMerchantService) UploadMedia(ctx context.Context, filename string, data []byte) (string, error) {
	return s.uploadMultipart(ctx, "/v3/merchant/media/upload", filename, data, "图片")
}

// UploadVideo 上传进件资料视频（部分指定行业进件时微信要求补充），返回微信 media_id。
// 接口：POST /v3/merchant/media/video_upload（multipart/form-data）。
// 支持 avi/wmv/mpeg/mp4/mov/mkv/flv/f4v/m4v/rmvb ≤5M（调用方已校验）。
// ★ 与图片上传同一套 multipart+meta 签名机制，仅接口路径与体积/扩展名限制不同。
func (s *SubMerchantService) UploadVideo(ctx context.Context, filename string, data []byte) (string, error) {
	return s.uploadMultipart(ctx, "/v3/merchant/media/video_upload", filename, data, "视频")
}

// uploadMultipart 微信媒体文件上传通用实现（图片/视频共用）。
// ★ 与普通 APIv3 请求的关键差异：参与签名的 body 是 meta 的 JSON 串（{"filename","sha256"}），
//   而不是整个 multipart 请求体；Content-Type 由 multipart writer 带 boundary 生成。
// path 为接口路径；kind 仅用于错误文案（"图片"/"视频"）。
func (s *SubMerchantService) uploadMultipart(ctx context.Context, path, filename string, data []byte, kind string) (string, error) {
	c := s.creds()
	if c.SpMchID == "" || c.PrivateKey == "" || c.SerialNo == "" {
		return "", smErr("微信服务商凭证未配置，请先在系统设置填写")
	}
	priv, err := wxpayv3.ParsePrivateKey(c.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("解析服务商私钥失败: %w", err)
	}
	// meta：filename + 文件二进制的 sha256（十六进制小写）。
	sum := sha256.Sum256(data)
	meta := fmt.Sprintf(`{"filename":"%s","sha256":"%s"}`, filename, hex.EncodeToString(sum[:]))

	// 组 multipart body：meta（application/json）+ file（文件二进制）。
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	metaHead := textproto.MIMEHeader{}
	metaHead.Set("Content-Disposition", `form-data; name="meta"`)
	metaHead.Set("Content-Type", "application/json")
	metaPart, err := mw.CreatePart(metaHead)
	if err != nil {
		return "", err
	}
	if _, err := metaPart.Write([]byte(meta)); err != nil {
		return "", err
	}
	filePart, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := filePart.Write(data); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	// 签名 body = meta JSON 串（★不是 multipart 整体）。
	nonce, err := wxpayv3.NonceStr(32)
	if err != nil {
		return "", err
	}
	auth, err := wxpayv3.BuildAuthorization(wxpayv3.AuthParams{
		MchID:        c.SpMchID,
		SerialNo:     c.SerialNo,
		PrivateKey:   priv,
		Method:       http.MethodPost,
		CanonicalURL: path,
		Body:         meta,
		Timestamp:    wxpayv3.NowUnix(),
		Nonce:        nonce,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.host()+path, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", mw.FormDataContentType()) // multipart/form-data; boundary=...
	req.Header.Set("Authorization", auth)
	if c.PublicKeyID != "" {
		req.Header.Set("Wechatpay-Serial", c.PublicKeyID)
	}
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("上传%s到微信失败: %w", kind, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", smErr(kind + "上传失败: " + wxErrMsg(respBody))
	}
	var r struct {
		MediaID string `json:"media_id"`
	}
	if err := json.Unmarshal(respBody, &r); err != nil {
		return "", fmt.Errorf("解析上传应答失败: %w", err)
	}
	if r.MediaID == "" {
		return "", smErr(kind + "上传应答无 media_id")
	}
	return r.MediaID, nil
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
	req, err := http.NewRequestWithContext(ctx, method, s.host()+path, reader)
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

// ApplymentAuditDetail 驳回详情单项（申请单被驳回时逐字段返回）。
type ApplymentAuditDetail struct {
	Field        string `json:"field"`         // 字段名
	FieldName    string `json:"field_name"`    // 字段中文名称
	RejectReason string `json:"reject_reason"` // 驳回原因
}

// ApplymentStateResp 查询申请单状态应答（关注状态与 sub_mchid）。
type ApplymentStateResp struct {
	ApplymentID       int64                  `json:"applyment_id"`
	ApplymentState    string                 `json:"applyment_state"` // APPLYMENT_STATE_FINISHED / REJECTED / ...
	ApplymentStateMsg string                 `json:"applyment_state_msg"`
	SubMchID          string                 `json:"sub_mchid"`     // TO_BE_SIGNED/SIGNING/FINISHED 时返回
	SignURL           string                 `json:"sign_url"`      // 超管签约链接
	AuditDetail       []ApplymentAuditDetail `json:"audit_detail"`  // 驳回详情，仅 REJECTED 时返回
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

// —— 子商户管控情况查询（风控第二段：进件成功后按 sub_mchid 查被管控能力及原因）——
//
// 接口：GET /v3/mch-operation-manage/merchant-limitations/sub-mchid/{sub_mchid}
// 前提：只能查已进件成功、且在本服务商名下（存在受理关系）的子商户；无受理关系报
//       INVALID_REQUEST。无管控时应答各字段缺省（非错误）。字段以官方 4012803072 为准。

// MchLimitationRecovery 被管控原因及解脱路径单项。
// ★ recovery_specifications 是列表，一商户可能多条、每条各自关联不同能力；
//   顶层 LimitedFunctions 只是聚合，明细在每条里，别当扁平一维。
type MchLimitationRecovery struct {
	LimitationCaseID         string   `json:"limitation_case_id"`         // 被管控单据号（↔第三段关联主键/幂等键）
	LimitationReasonType     string   `json:"limitation_reason_type"`     // 原因类型（RISK_ABNORMAL 等 10 项枚举）
	LimitationReason         string   `json:"limitation_reason"`          // 被管控原因文本
	LimitationReasonDescribe string   `json:"limitation_reason_describe"` // 原因描述（给商户看的人话）
	RelateLimitations        []string `json:"relate_limitations"`         // 本条关联的被管控能力（枚举 7 项）
	OtherRelateLimitations   string   `json:"other_relate_limitations"`   // 其他关联能力（自由文本）
	RecoverWay               string   `json:"recover_way"`                // 解脱路径（19 项枚举）
	RecoverWayParam          string   `json:"recover_way_param"`          // 解脱路径参数
	RecoverHelpURL           string   `json:"recover_help_url"`           // 解脱帮助链接
	LimitationActionType     string   `json:"limitation_action_type"`     // 处置方式（立即/延迟管控）
	LimitationStartDate      string   `json:"limitation_start_date"`      // 预计管控开始时间（延迟管控，rfc3339）
	LimitationDate           string   `json:"limitation_date"`            // 实际被管控时间（rfc3339）
}

// MchLimitationResp 子商户管控情况应答（官方 4012803072 字段全集）。
type MchLimitationResp struct {
	Mchid                  string                  `json:"mchid"`                    // 被查子商户号
	LimitedFunctions       []string                `json:"limited_functions"`        // 顶层聚合的被管控能力列表（枚举 7 项）
	OtherLimitedFunctions  string                  `json:"other_limited_functions"`  // 枚举外的其他被管控能力（自由文本）
	RecoverySpecifications []MchLimitationRecovery `json:"recovery_specifications"`  // 被管控原因+解脱路径列表
}

// Limited 是否处于任一被管控能力（有枚举命中或其他能力文本非空）。
func (r *MchLimitationResp) Limited() bool {
	return len(r.LimitedFunctions) > 0 || strings.TrimSpace(r.OtherLimitedFunctions) != ""
}

// QuerySubMchLimitation 查询子商户管控情况。
// 无管控时返回的 resp 各字段缺省（resp.Limited()==false）；429 RATELIMIT_EXCEEDED 由调用方限速处理。
func (s *SubMerchantService) QuerySubMchLimitation(ctx context.Context, subMchID string) (*MchLimitationResp, []byte, error) {
	if strings.TrimSpace(subMchID) == "" {
		return nil, nil, smErr("子商户号为空，无法查询管控情况")
	}
	path := "/v3/mch-operation-manage/merchant-limitations/sub-mchid/" + subMchID
	raw, code, err := s.doRequest(ctx, http.MethodGet, path, "")
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("查询子商户管控情况失败: " + wxErrMsg(raw))
	}
	var r MchLimitationResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析子商户管控应答失败: %w", err)
	}
	return &r, raw, nil
}

// —— 结算账户（进件成功后售后：修改/查询结算银行账户）——

// ModifySettlementReq 修改结算账户入参（明文，敏感字段由 service 加密后组装）。
type ModifySettlementReq struct {
	AccountType   string `json:"account_type"`   // ACCOUNT_TYPE_BUSINESS 对公 / ACCOUNT_TYPE_PRIVATE 对私
	AccountBank   string `json:"account_bank"`   // 开户银行
	BankName      string `json:"bank_name"`      // 开户银行全称（含支行），按需
	BankBranchID  string `json:"bank_branch_id"` // 联行号，按需
	AccountNumber string `json:"account_number"` // ★银行账号（加密）
	AccountName   string `json:"account_name"`   // ★开户名称（加密，按需）
}

// ModifySettlement 修改特约商户结算银行账户。
// 接口：POST /v3/apply4sub/sub_merchants/{sub_mchid}/modify-settlement。
// 敏感字段（account_number/account_name）在此加密；返回修改申请单号 application_no（供查改单状态）。
func (s *SubMerchantService) ModifySettlement(ctx context.Context, subMchID string, req ModifySettlementReq) (string, []byte, error) {
	encNumber, err := s.EncryptSensitive(req.AccountNumber)
	if err != nil {
		return "", nil, fmt.Errorf("加密银行账号失败: %w", err)
	}
	body := map[string]any{
		"account_type":   req.AccountType,
		"account_bank":   req.AccountBank,
		"account_number": encNumber,
	}
	if strings.TrimSpace(req.BankName) != "" {
		body["bank_name"] = req.BankName
	}
	if strings.TrimSpace(req.BankBranchID) != "" {
		body["bank_branch_id"] = req.BankBranchID
	}
	if strings.TrimSpace(req.AccountName) != "" {
		encName, e := s.EncryptSensitive(req.AccountName)
		if e != nil {
			return "", nil, fmt.Errorf("加密开户名称失败: %w", e)
		}
		body["account_name"] = encName
	}
	bs, err := json.Marshal(body)
	if err != nil {
		return "", nil, err
	}
	raw, code, err := s.doRequest(ctx, http.MethodPost, "/v3/apply4sub/sub_merchants/"+subMchID+"/modify-settlement", string(bs))
	if err != nil {
		return "", raw, err
	}
	if code < 200 || code >= 300 {
		return "", raw, smErr("修改结算账户失败: " + wxErrMsg(raw))
	}
	var r struct {
		ApplicationNo string `json:"application_no"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", raw, fmt.Errorf("解析修改结算账户应答失败: %w", err)
	}
	return r.ApplicationNo, raw, nil
}

// SettlementResp 查询结算账户应答（掩码账号 + 验证结果，非敏感）。
type SettlementResp struct {
	AccountType      string `json:"account_type"`
	AccountBank      string `json:"account_bank"`
	BankName         string `json:"bank_name"`
	BankBranchID     string `json:"bank_branch_id"`
	AccountNumber    string `json:"account_number"`     // 掩码显示
	VerifyResult     string `json:"verify_result"`      // VERIFY_SUCCESS / VERIFY_FAIL / VERIFYING
	VerifyFailReason string `json:"verify_fail_reason"` // 验证失败原因
}

// QuerySettlement 查询特约商户当前生效的结算账户（掩码账号 + 验证结果）。
// 接口：GET /v3/apply4sub/sub_merchants/{sub_mchid}/settlement。
func (s *SubMerchantService) QuerySettlement(ctx context.Context, subMchID string) (*SettlementResp, []byte, error) {
	raw, code, err := s.doRequest(ctx, http.MethodGet, "/v3/apply4sub/sub_merchants/"+subMchID+"/settlement", "")
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("查询结算账户失败: " + wxErrMsg(raw))
	}
	var r SettlementResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析结算账户应答失败: %w", err)
	}
	return &r, raw, nil
}

// SettlementApplicationResp 查询结算账户修改申请状态应答。
type SettlementApplicationResp struct {
	AccountName      string `json:"account_name"`  // 掩码
	AccountType      string `json:"account_type"`
	AccountBank      string `json:"account_bank"`
	BankName         string `json:"bank_name"`
	BankBranchID     string `json:"bank_branch_id"`
	AccountNumber    string `json:"account_number"`     // 掩码
	VerifyResult     string `json:"verify_result"`      // AUDIT_SUCCESS / AUDITING / AUDIT_FAIL
	VerifyFailReason string `json:"verify_fail_reason"`
	VerifyFinishTime string `json:"verify_finish_time"`
}

// QuerySettlementApplication 查询结算账户修改申请单的审核状态。
// 接口：GET /v3/apply4sub/sub_merchants/{sub_mchid}/application/{application_no}。
func (s *SubMerchantService) QuerySettlementApplication(ctx context.Context, subMchID, applicationNo string) (*SettlementApplicationResp, []byte, error) {
	raw, code, err := s.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/v3/apply4sub/sub_merchants/%s/application/%s", subMchID, applicationNo), "")
	if err != nil {
		return nil, raw, err
	}
	if code < 200 || code >= 300 {
		return nil, raw, smErr("查询结算账户修改申请状态失败: " + wxErrMsg(raw))
	}
	var r SettlementApplicationResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, raw, fmt.Errorf("解析结算账户修改申请状态应答失败: %w", err)
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
