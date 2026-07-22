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

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/pkg/jwtauth"
)

// OAuthService 商户快捷登录（对齐 epay user/connect.php[QQ] + wxlogin.php[微信] + oauth.php[支付宝]）。
//
// 语义对齐 epay：
//   - 不自动注册。第三方授权拿到 openid 后：已绑定商户→直接登录；未绑定→返回 need_bind + 临时 openid 凭据，
//     引导用户输入商户ID/密码后绑定（BindByLogin）。
//   - openid 存 pre_user 的 qq_uid/wx_uid/alipay_uid 字段。
//   - 登录态载体用项目统一 JWT(scope=merchant)，不移植 epay 的 authcode/RC4 cookie。
//
// 各 provider 的 appid/密钥来自 config oauth 分组。真实第三方调用需真实凭证。
type OAuthService struct {
	repo    *repository.MerchantRepo
	cfg     *ConfigService
	jm      *jwtauth.Manager
	authSvc *MerchantAuthService // 复用密码/密钥登录做绑定校验
}

func NewOAuthService(repo *repository.MerchantRepo, cfg *ConfigService, jm *jwtauth.Manager, authSvc *MerchantAuthService) *OAuthService {
	return &OAuthService{repo: repo, cfg: cfg, jm: jm, authSvc: authSvc}
}

const oauthHTTPTimeout = 10 * time.Second

// providerColumn 把 provider 名映射到 pre_user 的 openid 字段列。
func providerColumn(provider string) (string, bool) {
	switch provider {
	case "qq":
		return "qq_uid", true
	case "wx", "weixin":
		return "wx_uid", true
	case "alipay":
		return "alipay_uid", true
	}
	return "", false
}

// AuthorizeURL 生成第三方授权跳转 URL（对齐 epay 各 connect 的 getAuthUrl）。
// redirectURI 为本站回调地址（前端拼好 provider）。state 防 CSRF。
func (s *OAuthService) AuthorizeURL(provider, redirectURI, state string) (string, error) {
	if s.cfg.Int("login_"+normalizeProvider(provider), 0) == 0 {
		return "", maErr("该快捷登录方式未开启")
	}
	switch provider {
	case "qq":
		appid := s.cfg.Str("login_qq_appid")
		if appid == "" {
			return "", maErr("QQ 登录未配置 appid")
		}
		q := url.Values{}
		q.Set("response_type", "code")
		q.Set("client_id", appid)
		q.Set("redirect_uri", redirectURI)
		q.Set("state", state)
		return "https://graph.qq.com/oauth2.0/authorize?" + q.Encode(), nil
	case "wx", "weixin":
		appid := s.cfg.Str("login_appid")
		if appid == "" {
			return "", maErr("微信登录未配置 appid")
		}
		q := url.Values{}
		q.Set("appid", appid)
		q.Set("redirect_uri", redirectURI)
		q.Set("response_type", "code")
		q.Set("scope", "snsapi_base")
		q.Set("state", state)
		return "https://open.weixin.qq.com/connect/oauth2/authorize?" + q.Encode() + "#wechat_redirect", nil
	case "alipay":
		appid := s.cfg.Str("login_appid")
		if appid == "" {
			return "", maErr("支付宝登录未配置 appid")
		}
		q := url.Values{}
		q.Set("app_id", appid)
		q.Set("scope", "auth_base")
		q.Set("redirect_uri", redirectURI)
		q.Set("state", state)
		return "https://openauth.alipay.com/oauth2/publicAppAuthorize.htm?" + q.Encode(), nil
	}
	return "", maErr("不支持的快捷登录方式")
}

func normalizeProvider(p string) string {
	if p == "weixin" {
		return "wx"
	}
	return p
}

// Callback 处理第三方回调：用 code 换 openid → 已绑定则登录，未绑定返回 need_bind。
// redirectURI 需与 AuthorizeURL 时一致（部分平台校验）。
func (s *OAuthService) Callback(ctx context.Context, provider, code, redirectURI, ip string) (*dto.OAuthResult, error) {
	col, ok := providerColumn(provider)
	if !ok {
		return nil, maErr("不支持的快捷登录方式")
	}
	openid, err := s.exchangeOpenID(ctx, provider, code, redirectURI)
	if err != nil {
		return nil, err
	}
	if openid == "" {
		return nil, maErr("未能获取第三方账号标识")
	}
	m, err := s.repo.FindByOAuth(col, openid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		// 未绑定：返回 need_bind + openid（前端引导输入商户ID/密码后调 Bind）。
		return &dto.OAuthResult{NeedBind: true, Provider: provider, OpenID: openid}, nil
	}
	if m.Status == 0 {
		return nil, maErr("账号已被封禁，请联系平台")
	}
	token, err := s.jm.Generate(m.UID, merchantName(m.UID), "merchant", "merchant")
	if err != nil {
		return nil, err
	}
	info := toMerchantInfo(m)
	return &dto.OAuthResult{Token: token, Info: &info}, nil
}

// Bind 未绑定用户输入商户账号+密码后，把 openid 绑定到该商户并登录（对齐 epay login.php?connect=true）。
func (s *OAuthService) Bind(provider, openid string, loginReq dto.MerchantLoginReq, ip string) (*dto.OAuthResult, error) {
	col, ok := providerColumn(provider)
	if !ok {
		return nil, maErr("不支持的快捷登录方式")
	}
	if openid == "" {
		return nil, maErr("绑定凭据已失效，请重新授权")
	}
	// 复用密码/密钥登录校验账号（成功即得到已认证的商户 token + info）。
	loginResp, err := s.authSvc.Login(loginReq, ip)
	if err != nil {
		return nil, err
	}
	// 该 openid 若已被别人绑定则拒绝。
	if exist, _ := s.repo.FindByOAuth(col, openid); exist != nil && exist.UID != loginResp.Info.UID {
		return nil, maErr("该第三方账号已绑定其它商户")
	}
	if err := s.repo.BindOAuth(loginResp.Info.UID, col, openid); err != nil {
		return nil, err
	}
	info := loginResp.Info
	return &dto.OAuthResult{Token: loginResp.Token, Info: &info}, nil
}

// exchangeOpenID 用授权 code 换取第三方唯一标识 openid/user_id（对齐 epay 各 SDK）。
// 真实调用需真实凭证；无凭证时第三方会返回错误，如实上抛。
func (s *OAuthService) exchangeOpenID(ctx context.Context, provider, code, redirectURI string) (string, error) {
	switch provider {
	case "qq":
		return s.qqOpenID(ctx, code, redirectURI)
	case "wx", "weixin":
		return s.wxOpenID(ctx, code)
	case "alipay":
		return s.alipayUserID(ctx, code)
	}
	return "", maErr("不支持的快捷登录方式")
}

// qqOpenID QQ 互联：code→access_token→openid（对齐 epay QC.php）。
func (s *OAuthService) qqOpenID(ctx context.Context, code, redirectURI string) (string, error) {
	appid := s.cfg.Str("login_qq_appid")
	appkey := s.cfg.Str("login_qq_appkey")
	if appid == "" || appkey == "" {
		return "", maErr("QQ 登录未配置 appid/appkey")
	}
	// 1. 换 access_token
	q := url.Values{}
	q.Set("grant_type", "authorization_code")
	q.Set("client_id", appid)
	q.Set("client_secret", appkey)
	q.Set("code", code)
	q.Set("redirect_uri", redirectURI)
	q.Set("fmt", "json")
	tokBody, err := httpGet(ctx, "https://graph.qq.com/oauth2.0/token?"+q.Encode())
	if err != nil {
		return "", err
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.Unmarshal(tokBody, &tok)
	if tok.AccessToken == "" {
		return "", maErr("QQ 换取 access_token 失败: " + string(tokBody))
	}
	// 2. 拿 openid
	meBody, err := httpGet(ctx, "https://graph.qq.com/oauth2.0/me?fmt=json&access_token="+url.QueryEscape(tok.AccessToken))
	if err != nil {
		return "", err
	}
	var me struct {
		OpenID string `json:"openid"`
	}
	_ = json.Unmarshal(meBody, &me)
	return me.OpenID, nil
}

// wxOpenID 微信公众号网页授权：code→openid（snsapi_base，对齐 epay JsApiTool）。
func (s *OAuthService) wxOpenID(ctx context.Context, code string) (string, error) {
	appid := s.cfg.Str("login_appid")
	secret := s.cfg.Str("login_appkey")
	if appid == "" || secret == "" {
		return "", maErr("微信登录未配置 appid/appsecret")
	}
	q := url.Values{}
	q.Set("appid", appid)
	q.Set("secret", secret)
	q.Set("code", code)
	q.Set("grant_type", "authorization_code")
	body, err := httpGet(ctx, "https://api.weixin.qq.com/sns/oauth2/access_token?"+q.Encode())
	if err != nil {
		return "", err
	}
	var r struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(body, &r)
	if r.OpenID == "" {
		return "", maErr("微信换取 openid 失败: " + r.ErrMsg)
	}
	return r.OpenID, nil
}

// alipayUserID 支付宝快捷登录：auth_code→user_id（alipay.system.oauth.token，对齐 epay AlipayOauthService）。
// 支付宝 OAuth 走 openapi 网关 RSA2 签名，与收单同源。此处需支付宝渠道凭证(login_appid + 私钥)。
func (s *OAuthService) alipayUserID(ctx context.Context, code string) (string, error) {
	// 支付宝 OAuth 需 RSA2 签名网关调用（app_private_key 从 oauth 配置或专用支付宝渠道取）。
	// 我方 oauth 分组目前只存 appid/appkey(公钥) 概念，私钥应由支付宝支付渠道复用。
	// 真实凭证到位前，如实返回待凭证提示（代码结构就绪）。
	if s.cfg.Str("login_appid") == "" {
		return "", maErr("支付宝登录未配置 appid")
	}
	return "", maErr("支付宝快捷登录需配置支付宝应用私钥（RSA2 网关签名），待真实凭证接入")
}

// httpGet 简单 GET 拉取 body。
func httpGet(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: oauthHTTPTimeout}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求第三方失败: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// QQ 的 /me 可能返回 callback( {...} ); 剥壳。
	s := strings.TrimSpace(string(b))
	if strings.HasPrefix(s, "callback(") {
		s = strings.TrimSuffix(strings.TrimPrefix(s, "callback("), ");")
		s = strings.TrimSpace(strings.TrimSuffix(s, ")"))
		return []byte(s), nil
	}
	return b, nil
}
