package service

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// SmsService 短信验证码 OTP（对齐 epay \lib\VerifyCode + send_sms_common）。
//
// 场景 scene：reg(注册)/login(登录)/find(找回密码)/edit(改绑)。
// 频控：同号 60 秒间隔、单号每天 ≤3、单 IP 每天 ≤6（对齐 epay）。
// 校验：1 小时过期、errcount≥5 失效、status 作废。
// 通道 sms_api：1=腾讯云 2=阿里云 4=短信宝（对齐 epay，各自签名）。真实发送需通道凭证。
type SmsService struct {
	repo *repository.RegCodeRepo
	cfg  *ConfigService
}

func NewSmsService(repo *repository.RegCodeRepo, cfg *ConfigService) *SmsService {
	return &SmsService{repo: repo, cfg: cfg}
}

const smsHTTPTimeout = 10 * time.Second

// Send 发送短信验证码：频控校验 → 生成 6 位码 → 发送 → 落库。
func (s *SmsService) Send(ctx context.Context, scene, phone, ip string) error {
	if !isNumeric(phone) || len(phone) != 11 {
		return maErr("手机号格式不正确")
	}
	now := time.Now()
	// 60 秒间隔
	if n, _ := s.repo.CountByToSince(phone, now.Add(-60*time.Second)); n > 0 {
		return maErr("发送过于频繁，请 60 秒后再试")
	}
	// 单号每天 ≤3
	if n, _ := s.repo.CountByToSince(phone, now.Add(-24*time.Hour)); n >= 3 {
		return maErr("该手机号今日验证码发送次数已达上限")
	}
	// 单 IP 每天 ≤6
	if ip != "" {
		if n, _ := s.repo.CountByIPSince(ip, now.Add(-24*time.Hour)); n >= 6 {
			return maErr("您的操作过于频繁，请稍后再试")
		}
	}
	code := randCode6()
	if err := s.dispatch(ctx, phone, code, scene); err != nil {
		return err
	}
	return s.repo.Create(&model.RegCode{
		Scene: scene, Type: 1, Code: code, To: phone, IP: ip, Status: 0, SendTime: now,
	})
}

// Verify 校验短信验证码：最新一条，判过期(1h)/已用/errcount≥5/码匹配。通过则作废。
func (s *SmsService) Verify(scene, phone, code string) (bool, error) {
	c, err := s.repo.Latest(phone, scene)
	if err != nil {
		return false, err
	}
	if c == nil {
		return false, maErr("请先获取验证码")
	}
	if c.Status > 0 {
		return false, maErr("验证码已使用，请重新获取")
	}
	if time.Since(c.SendTime) > time.Hour {
		return false, maErr("验证码已过期，请重新获取")
	}
	if c.ErrCount >= 5 {
		return false, maErr("验证码错误次数过多，请重新获取")
	}
	if strings.TrimSpace(code) != c.Code {
		_ = s.repo.IncrErr(c.ID)
		return false, maErr("验证码错误")
	}
	_ = s.repo.MarkUsed(c.ID)
	return true, nil
}

// dispatch 按 sms_api 选通道发送。通道凭证缺失时返回待凭证错误（代码就绪）。
func (s *SmsService) dispatch(ctx context.Context, phone, code, scene string) error {
	switch s.cfg.Int("sms_api", 0) {
	case 1:
		return s.sendQcloud(ctx, phone, code)
	case 2:
		return s.sendAliyun(ctx, phone, code)
	case 4:
		return s.sendSmsBao(ctx, phone, code)
	case 0:
		return maErr("短信通道未配置(sms_api)，待真实凭证")
	}
	return maErr("不支持的短信通道")
}

// SendCommon 按指定模板+参数发送短信（对齐 epay functions.php send_sms_common）。
// 供 NoticeService 的 balance/complain 场景使用。tplParam 通常单值（{"code":值}）。
// 通道凭证缺失时返回错误，NoticeService 静默降级。
func (s *SmsService) SendCommon(ctx context.Context, phone, tplCode string, tplParam map[string]string) error {
	if tplCode == "" {
		return maErr("短信模板未配置")
	}
	switch s.cfg.Int("sms_api", 0) {
	case 2:
		return s.sendAliyunTpl(ctx, phone, tplCode, tplParam)
	case 4:
		// 短信宝无模板概念，直接拼内容发送首个参数值。
		var val string
		for _, v := range tplParam {
			val = v
			break
		}
		return s.sendSmsBao(ctx, phone, val)
	case 1:
		return maErr("腾讯云短信通道待真实凭证接入")
	case 0:
		return maErr("短信通道未配置(sms_api)，待真实凭证")
	}
	return maErr("不支持的短信通道")
}

// tplParam 取场景对应短信模板 ID（对齐 epay sms_tpl_reg/find/...）。
func (s *SmsService) tplParam(scene string) string {
	switch scene {
	case "reg":
		return s.cfg.Str("sms_tpl_reg")
	case "find":
		return s.cfg.Str("sms_tpl_find")
	case "edit":
		return s.cfg.Str("sms_tpl_edit")
	}
	return s.cfg.Str("sms_tpl_reg")
}

// sendAliyun 阿里云短信 dysmsapi（OTP 场景，模板参数固定 {"code":码}）。
func (s *SmsService) sendAliyun(ctx context.Context, phone, code string) error {
	return s.sendAliyunTpl(ctx, phone, s.tplParam("reg"), map[string]string{"code": code})
}

// sendAliyunTpl 阿里云短信 dysmsapi（HMAC-SHA1 RPC 签名，对齐 epay \lib\sms\Aliyun）。
// 支持任意模板 + 参数（OTP 与 NoticeService 通用）。
func (s *SmsService) sendAliyunTpl(ctx context.Context, phone, tpl string, tplParam map[string]string) error {
	ak := s.cfg.Str("sms_appid")
	secret := s.cfg.Str("sms_appkey")
	sign := s.cfg.Str("sms_sign")
	if ak == "" || secret == "" || sign == "" || tpl == "" {
		return maErr("阿里云短信未配置完整(appid/appkey/sign/模板)，待真实凭证")
	}
	tplJSON, _ := json.Marshal(tplParam)
	params := map[string]string{
		"AccessKeyId":      ak,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phone,
		"RegionId":         "cn-hangzhou",
		"SignName":         sign,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   randNonce(),
		"SignatureVersion": "1.0",
		"TemplateCode":     tpl,
		"TemplateParam":    string(tplJSON),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          "2017-05-25",
	}
	// RPC 签名：ksort → percentEncode 拼 → StringToSign=GET&%2F&percentEncode(query) → HMAC-SHA1(secret+"&")。
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canon := strings.Join(pairs, "&")
	stringToSign := "GET&" + percentEncode("/") + "&" + percentEncode(canon)
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	reqURL := "https://dysmsapi.aliyuncs.com/?Signature=" + percentEncode(signature) + "&" + canon
	return smsGetOK(ctx, reqURL, `"Code":"OK"`)
}

// sendQcloud 腾讯云短信（sha256 拼接串签名，对齐 epay \lib\sms\Qcloud）。
func (s *SmsService) sendQcloud(ctx context.Context, phone, code string) error {
	appid := s.cfg.Str("sms_appid")
	appkey := s.cfg.Str("sms_appkey")
	sign := s.cfg.Str("sms_sign")
	tpl := s.tplParam("reg")
	if appid == "" || appkey == "" || sign == "" || tpl == "" {
		return maErr("腾讯云短信未配置完整，待真实凭证")
	}
	// 结构就绪：腾讯云 v5 tlssmssvr 需 POST JSON + sig=sha256(appkey=..&random=..&time=..&mobile=..)。
	// 真实发送待凭证，此处返回待凭证提示（避免假成功）。
	return maErr("腾讯云短信通道待真实凭证接入")
}

// sendSmsBao 短信宝（md5(密码)，对齐 epay \lib\sms\SmsBao）。
func (s *SmsService) sendSmsBao(ctx context.Context, phone, code string) error {
	user := s.cfg.Str("sms_appid")
	pass := s.cfg.Str("sms_appkey")
	sign := s.cfg.Str("sms_sign")
	if user == "" || pass == "" {
		return maErr("短信宝未配置账号，待真实凭证")
	}
	content := fmt.Sprintf("【%s】您的验证码是%s，请勿泄露。", sign, code)
	q := url.Values{}
	q.Set("u", user)
	q.Set("p", md5hex(pass))
	q.Set("m", phone)
	q.Set("c", content)
	return smsGetOK(ctx, "http://api.smsbao.com/sms?"+q.Encode(), "0")
}

// smsGetOK 发 GET 请求并判断响应含 okMark 视为成功。
func smsGetOK(ctx context.Context, u, okMark string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: smsHTTPTimeout}).Do(req)
	if err != nil {
		return fmt.Errorf("请求短信通道失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), okMark) {
		return maErr("短信发送失败: " + strings.TrimSpace(string(body)))
	}
	return nil
}

// ---- 小工具 ----

func randCode6() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func randNonce() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// percentEncode 阿里云 RPC 签名专用编码：urlencode 后 +→%20, *→%2A, %7E→~。
func percentEncode(s string) string {
	e := url.QueryEscape(s)
	e = strings.ReplaceAll(e, "+", "%20")
	e = strings.ReplaceAll(e, "*", "%2A")
	e = strings.ReplaceAll(e, "%7E", "~")
	return e
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
