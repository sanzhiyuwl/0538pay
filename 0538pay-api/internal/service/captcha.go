package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// CaptchaService 自研图形验证码（代替 epay 短信/邮箱 OTP + 极验，无外部依赖）。
// 后端生成 4 位数字码 + token，码存内存(带 5 分钟过期)，前端拿 SVG 展示、提交时带 token+输入值校验。
// 一次性：校验成功即删除，防重放。
type CaptchaService struct {
	mu    sync.Mutex
	store map[string]captchaEntry
}

type captchaEntry struct {
	code   string
	expire time.Time
}

func NewCaptchaService() *CaptchaService {
	s := &CaptchaService{store: map[string]captchaEntry{}}
	return s
}

// captchaTTL 验证码有效期。
const captchaTTL = 5 * time.Minute

// Generate 生成一个验证码，返回 (token, svg)。token 交前端随表单回传。
func (s *CaptchaService) Generate() (string, string, error) {
	code, err := randDigits(4)
	if err != nil {
		return "", "", err
	}
	token, err := randomHex(16)
	if err != nil {
		return "", "", err
	}
	s.mu.Lock()
	s.gcLocked()
	s.store[token] = captchaEntry{code: code, expire: time.Now().Add(captchaTTL)}
	s.mu.Unlock()
	return token, renderCaptchaSVG(code), nil
}

// Verify 校验 token+code。成功即删除(一次性)。大小写无关(纯数字无影响)。
func (s *CaptchaService) Verify(token, code string) bool {
	token = strings.TrimSpace(token)
	code = strings.TrimSpace(code)
	if token == "" || code == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.store[token]
	if !ok {
		return false
	}
	delete(s.store, token) // 一次性，无论对错都删，避免暴力重试
	if time.Now().After(e.expire) {
		return false
	}
	return strings.EqualFold(e.code, code)
}

// gcLocked 清理过期项（调用方须持锁）。store 很小，线性扫描即可。
func (s *CaptchaService) gcLocked() {
	now := time.Now()
	for k, e := range s.store {
		if now.After(e.expire) {
			delete(s.store, k)
		}
	}
}

// randDigits 生成 n 位数字字符串（加密随机）。
func randDigits(n int) (string, error) {
	var b strings.Builder
	for i := 0; i < n; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + d.Int64()))
	}
	return b.String(), nil
}

// renderCaptchaSVG 把 4 位码画成简单 SVG（每位一个字符 + 轻微错位/颜色，避免纯文本）。
// 纯自研，无第三方图形库依赖；前端 <img src> 或 v-html 直接渲染。
func renderCaptchaSVG(code string) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="40" viewBox="0 0 100 40">`)
	b.WriteString(`<rect width="100" height="40" fill="#f3f4f6"/>`)
	colors := []string{"#2563eb", "#dc2626", "#059669", "#7c3aed"}
	for i, ch := range code {
		x := 15 + i*22
		y := 28 + (i%2)*(-3) + (i%3)*2 // 上下轻微错位
		rot := (i%2)*8 - 4             // 轻微旋转
		fmt.Fprintf(&b,
			`<text x="%d" y="%d" font-size="24" font-family="monospace" font-weight="bold" fill="%s" transform="rotate(%d %d %d)">%c</text>`,
			x, y, colors[i%len(colors)], rot, x, y, ch)
	}
	// 两条干扰线
	b.WriteString(`<line x1="5" y1="12" x2="95" y2="30" stroke="#9ca3af" stroke-width="1"/>`)
	b.WriteString(`<line x1="10" y1="32" x2="90" y2="8" stroke="#d1d5db" stroke-width="1"/>`)
	b.WriteString(`</svg>`)
	return b.String()
}
