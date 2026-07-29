package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/epvia/api/internal/model"
	"github.com/epvia/api/internal/repository"
)

// MailService 邮件发送三通道（对齐 epay includes/functions.php send_mail + lib/mail/*）。
//
// 通道由 config mail_cloud 决定：
//   - 0：SMTP（PHPMailer 等价，net/smtp + STARTTLS/SSL 按端口判定）
//   - 1：Sendcloud（apiv2/mail/send 表单 POST）
//   - 2：阿里云邮件推送（dm.aliyuncs.com SingleSendMail，HMAC-SHA1 RPC 签名）
//
// 与 send_mail 语义一致：配置不全返回错误（不假成功），发送成功返回 nil。
// 承载邮箱 OTP 与 NoticeService 邮件场景（K-1 邮件通道）。
type MailService struct {
	cfg      *ConfigService
	codeRepo *repository.RegCodeRepo // 邮箱验证码 OTP（可空；SetCodeRepo 注入，复用 pre_regcode 表 type=2）
}

func NewMailService(cfg *ConfigService) *MailService { return &MailService{cfg: cfg} }

// SetCodeRepo 注入验证码仓储，开启邮箱验证码 OTP 能力（与短信 OTP 共用 pre_regcode 表，type=2 区分）。
func (s *MailService) SetCodeRepo(r *repository.RegCodeRepo) { s.codeRepo = r }

// SendCode 发送邮箱验证码 OTP（对齐 epay VerifyCode::send_code type=0 邮箱分支）。
// 频控：同邮箱 60 秒间隔、单邮箱每天 ≤7、单 IP 每天 ≤11（对齐 epay 邮箱风控常量）。
// 生成 6 位码 → 发 HTML 邮件 → 落 pre_regcode(type=2)。校验走 VerifyCode。
func (s *MailService) SendCode(ctx context.Context, scene, email, ip string) error {
	if s.codeRepo == nil {
		return maErr("邮箱验证码服务未启用")
	}
	email = strings.TrimSpace(email)
	if !emailRe.MatchString(email) {
		return maErr("邮箱格式不正确")
	}
	now := time.Now()
	if n, _ := s.codeRepo.CountByToSince(email, now.Add(-60*time.Second)); n > 0 {
		return maErr("发送过于频繁，请 60 秒后再试")
	}
	if n, _ := s.codeRepo.CountByToSince(email, now.Add(-24*time.Hour)); n >= 7 {
		return maErr("该邮箱今日验证码发送次数已达上限")
	}
	if ip != "" {
		if n, _ := s.codeRepo.CountByIPSince(ip, now.Add(-24*time.Hour)); n >= 11 {
			return maErr("您的操作过于频繁，请稍后再试")
		}
	}
	code := randCode6()
	site := s.cfg.Str("sitename")
	if site == "" {
		site = "Epvia Neo"
	}
	body := fmt.Sprintf("您的验证码是 <b>%s</b>，5 分钟内有效，请勿泄露。<br/><br/>来自：%s", code, site)
	if err := s.Send(ctx, email, "注册验证码 - "+site, body); err != nil {
		return err
	}
	return s.codeRepo.Create(&model.RegCode{
		Scene: scene, Type: 2, Code: code, To: email, IP: ip, Status: 0, SendTime: now,
	})
}

// VerifyCode 校验邮箱验证码：最新一条，判过期(1h)/已用/errcount≥5/码匹配。通过则作废。
// 语义与 SmsService.Verify 完全一致（同表同规则，仅接收方为邮箱）。
func (s *MailService) VerifyCode(scene, email, code string) (bool, error) {
	if s.codeRepo == nil {
		return false, maErr("邮箱验证码服务未启用")
	}
	c, err := s.codeRepo.Latest(strings.TrimSpace(email), scene)
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
		_ = s.codeRepo.IncrErr(c.ID)
		return false, maErr("验证码错误")
	}
	_ = s.codeRepo.MarkUsed(c.ID)
	return true, nil
}

const mailHTTPTimeout = 10 * time.Second

// Configured 判断当前所选邮件通道是否已配置完整（供 OTP/通知前置判断，避免无谓发送）。
func (s *MailService) Configured() bool {
	switch s.cfg.Int("mail_cloud", 0) {
	case 1, 2:
		return s.cfg.Str("mail_apiuser") != "" && s.cfg.Str("mail_apikey") != ""
	default:
		return s.cfg.Str("mail_name") != "" && s.cfg.Str("mail_smtp") != "" &&
			s.cfg.Str("mail_pwd") != "" && s.cfg.Str("mail_port") != ""
	}
}

// Send 发送 HTML 邮件（对齐 epay send_mail($to, $sub, $msg)）。
func (s *MailService) Send(ctx context.Context, to, subject, htmlBody string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return maErr("收件人为空")
	}
	switch s.cfg.Int("mail_cloud", 0) {
	case 1:
		return s.sendSendcloud(ctx, to, subject, htmlBody)
	case 2:
		return s.sendAliyun(ctx, to, subject, htmlBody)
	default:
		return s.sendSMTP(to, subject, htmlBody)
	}
}

// sendSMTP 走 SMTP（对齐 PHPMailer 分支）：587=STARTTLS，≥465=隐式 SSL，其它=明文。
func (s *MailService) sendSMTP(to, subject, htmlBody string) error {
	host := s.cfg.Str("mail_smtp")
	user := s.cfg.Str("mail_name") // 登录账号 = 发件地址
	pwd := s.cfg.Str("mail_pwd")
	port := s.cfg.Int("mail_port", 0)
	if host == "" || user == "" || pwd == "" || port == 0 {
		return maErr("SMTP 邮件未配置完整(mail_smtp/mail_name/mail_pwd/mail_port)")
	}
	fromName := s.cfg.Str("sitename")
	if fromName == "" {
		fromName = "Epvia Neo"
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	auth := smtp.PlainAuth("", user, pwd, host)
	msg := buildMIME(fromName, user, to, subject, htmlBody)

	switch {
	case port >= 465 && port != 587:
		// 隐式 SSL：先建 TLS 连接再走 SMTP。
		return smtpOverTLS(addr, host, auth, user, to, msg)
	default:
		// 587=STARTTLS，其它端口 net/smtp 明文（STARTTLS 若服务端支持会自动协商）。
		return smtp.SendMail(addr, auth, user, []string{to}, msg)
	}
}

// smtpOverTLS 在隐式 TLS（465）上手动建立 SMTP 会话。
func smtpOverTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: mailHTTPTimeout}, "tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("连接 SMTP(SSL) 失败: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("建立 SMTP 会话失败: %w", err)
	}
	defer c.Close()
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// buildMIME 组装 HTML 邮件报文头 + 正文（UTF-8，对齐 PHPMailer isHTML）。
func buildMIME(fromName, fromAddr, to, subject, htmlBody string) []byte {
	var b strings.Builder
	b.WriteString("From: " + mimeWord(fromName) + " <" + fromAddr + ">\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + mimeWord(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n")
	b.WriteString("\r\n")
	// base64 正文按 76 字符折行（RFC 2045）。
	enc := base64.StdEncoding.EncodeToString([]byte(htmlBody))
	for i := 0; i < len(enc); i += 76 {
		end := i + 76
		if end > len(enc) {
			end = len(enc)
		}
		b.WriteString(enc[i:end] + "\r\n")
	}
	return []byte(b.String())
}

// mimeWord 对含非 ASCII 的头部字段做 RFC 2047 B 编码（中文主题/发件人名）。
func mimeWord(s string) string {
	for _, r := range s {
		if r > 127 {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
		}
	}
	return s
}

// sendSendcloud Sendcloud apiv2 邮件（对齐 epay lib/mail/Sendcloud）。
func (s *MailService) sendSendcloud(ctx context.Context, to, subject, htmlBody string) error {
	apiUser := s.cfg.Str("mail_apiuser")
	apiKey := s.cfg.Str("mail_apikey")
	if apiUser == "" || apiKey == "" {
		return maErr("Sendcloud 邮件未配置(mail_apiuser/mail_apikey)")
	}
	from := s.cfg.Str("mail_name2")
	fromName := s.cfg.Str("sitename")
	form := url.Values{}
	form.Set("apiUser", apiUser)
	form.Set("apiKey", apiKey)
	form.Set("from", from)
	form.Set("fromName", fromName)
	form.Set("to", to)
	form.Set("subject", subject)
	form.Set("html", htmlBody)
	return mailPostContains(ctx, "http://api.sendcloud.net/apiv2/mail/send", form, `"statusCode":200`)
}

// sendAliyun 阿里云邮件推送 SingleSendMail（对齐 epay lib/mail/Aliyun，HMAC-SHA1 RPC 签名）。
func (s *MailService) sendAliyun(ctx context.Context, to, subject, htmlBody string) error {
	ak := s.cfg.Str("mail_apiuser")
	secret := s.cfg.Str("mail_apikey")
	from := s.cfg.Str("mail_name2")
	fromName := s.cfg.Str("sitename")
	if ak == "" || secret == "" || from == "" {
		return maErr("阿里云邮件未配置(mail_apiuser/mail_apikey/mail_name2)")
	}
	params := map[string]string{
		"Action":           "SingleSendMail",
		"AccountName":      from,
		"ReplyToAddress":   "false",
		"AddressType":      "1",
		"ToAddress":        to,
		"FromAlias":        fromName,
		"Subject":          subject,
		"HtmlBody":         htmlBody,
		"Format":           "JSON",
		"Version":          "2015-11-23",
		"AccessKeyId":      ak,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   randNonce(),
	}
	// RPC 签名：ksort → &percentEncode(k)=percentEncode(v) 拼接（去首 &）→
	// StringToSign = POST&%2F&percentEncode(canon) → HMAC-SHA1(secret+"&") → base64。
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
	stringToSign := "POST&" + percentEncode("/") + "&" + percentEncode(canon)
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	form.Set("Signature", signature)
	return mailPostContains(ctx, "https://dm.aliyuncs.com/", form, "")
}

// mailPostContains POST 表单，2xx 且（okMark 为空或响应含 okMark）视为成功。
func mailPostContains(ctx context.Context, endpoint string, form url.Values, okMark string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := (&http.Client{Timeout: mailHTTPTimeout}).Do(req)
	if err != nil {
		return fmt.Errorf("请求邮件通道失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return maErr("邮件发送失败(HTTP " + resp.Status + "): " + strings.TrimSpace(string(body)))
	}
	if okMark != "" && !strings.Contains(string(body), okMark) {
		return maErr("邮件发送失败: " + strings.TrimSpace(string(body)))
	}
	return nil
}
