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
	cfg *ConfigService
}

func NewMailService(cfg *ConfigService) *MailService { return &MailService{cfg: cfg} }

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
