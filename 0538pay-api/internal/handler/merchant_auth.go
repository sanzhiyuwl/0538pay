package handler

import (
	"errors"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/middleware"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// MerchantAuthHandler 商户端登录 / 当前商户信息 / 自助流程（注册/完善资料/找回密码）。
type MerchantAuthHandler struct {
	svc     *service.MerchantAuthService
	reg     *service.MerchantRegService // 自助流程（可空，SetRegService 注入）
	captcha *service.CaptchaService     // 图形验证码（可空）
	oauth   *service.OAuthService       // 快捷登录（可空，SetOAuthService 注入）
	sms     *service.SmsService         // 短信 OTP（可空）
	mail    *service.MailService        // 邮箱 OTP（可空）
	geetest *service.GeetestService     // 极验（可空）
	cfg     *service.ConfigService      // 读注册方式开关（可空）
}

// SetOAuthService 注入快捷登录服务（QQ/微信/支付宝 OAuth）。
func (h *MerchantAuthHandler) SetOAuthService(o *service.OAuthService) { h.oauth = o }

// SetSmsGeetest 注入短信 OTP + 极验服务。
func (h *MerchantAuthHandler) SetSmsGeetest(sms *service.SmsService, gt *service.GeetestService) {
	h.sms = sms
	h.geetest = gt
}

// SendSms POST /api/merchant/sms {scene,phone} 发送短信验证码（公开，频控在 service）。
func (h *MerchantAuthHandler) SendSms(c *gin.Context) {
	if h.sms == nil {
		resp.Fail(c, 1101, "短信服务未启用")
		return
	}
	var req struct {
		Scene string `json:"scene"`
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	scene := req.Scene
	if scene == "" {
		scene = "reg"
	}
	if err := h.sms.Send(c.Request.Context(), scene, req.Phone, c.ClientIP()); err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, gin.H{"sent": true})
}

// RegMethods GET /api/merchant/reg/methods 返回开放的注册方式与验证码模式（公开，注册页读）。
// phone/email：手机/邮箱注册是否开放（都关=不可注册）；otp：是否启用真实短信/邮箱验证码。
func (h *MerchantAuthHandler) RegMethods(c *gin.Context) {
	if h.cfg == nil {
		resp.OK(c, gin.H{"open": true, "invite_only": false, "phone": true, "email": true, "otp": false})
		return
	}
	// reg_open：0关闭注册 / 1开放 / 2仅邀请。关闭时整体不可注册（前端出暂停页）。
	regOpen := h.cfg.Int("reg_open", 1)
	resp.OK(c, gin.H{
		"open":        regOpen != 0,
		"invite_only": regOpen == 2,
		"phone":       h.cfg.Int("reg_phone", 1) == 1,
		"email":       h.cfg.Int("reg_email", 1) == 1,
		"otp":         h.cfg.Int("reg_otp", 0) == 1,
	})
}

// Legal GET /api/merchant/legal 服务协议 / 隐私政策正文（公开，登录/注册页弹窗读取）。
// 后台「使用说明」页可编辑；留空则前端用内置默认文案。epay 为硬编码静态页，我方做成可编辑（超出 epay）。
func (h *MerchantAuthHandler) Legal(c *gin.Context) {
	if h.cfg == nil {
		resp.OK(c, gin.H{"agreement": "", "privacy": "", "sitename": ""})
		return
	}
	resp.OK(c, gin.H{
		"agreement": h.cfg.Str("agreement_content"),
		"privacy":   h.cfg.Str("privacy_content"),
		"sitename":  h.cfg.Str("sitename"),
	})
}

// SendEmailCode POST /api/merchant/email-code {scene,email} 发送邮箱验证码（公开，频控在 service）。
func (h *MerchantAuthHandler) SendEmailCode(c *gin.Context) {
	if h.mail == nil {
		resp.Fail(c, 1101, "邮箱验证码服务未启用")
		return
	}
	var req struct {
		Scene string `json:"scene"`
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	scene := req.Scene
	if scene == "" {
		scene = "reg"
	}
	if err := h.mail.SendCode(c.Request.Context(), scene, req.Email, c.ClientIP()); err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, gin.H{"sent": true})
}

// GeetestInit GET /api/merchant/geetest 极验初始化参数（公开）。
func (h *MerchantAuthHandler) GeetestInit(c *gin.Context) {
	if h.geetest == nil || !h.geetest.Enabled() {
		resp.OK(c, gin.H{"enabled": false})
		return
	}
	resp.OK(c, gin.H{"enabled": true, "params": h.geetest.InitParams()})
}

// OAuthMethods GET /api/merchant/oauth/methods 返回开启的快捷登录方式（公开，供登录页决定显示哪些入口）。
// 快捷登录服务未启用或全部关闭时，返回全 false，登录页则不显示"其他登录方式"。
func (h *MerchantAuthHandler) OAuthMethods(c *gin.Context) {
	if h.oauth == nil {
		resp.OK(c, gin.H{"qq": false, "wx": false, "alipay": false})
		return
	}
	resp.OK(c, h.oauth.EnabledMethods())
}

// OAuthURL GET /api/merchant/oauth/:provider/url?redirect=&state= 生成授权跳转 URL（公开）。
func (h *MerchantAuthHandler) OAuthURL(c *gin.Context) {
	if h.oauth == nil {
		resp.Fail(c, 1101, "快捷登录未启用")
		return
	}
	u, err := h.oauth.AuthorizeURL(c.Param("provider"), c.Query("redirect"), c.Query("state"))
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, gin.H{"url": u})
}

// OAuthCallback POST /api/merchant/oauth/:provider/callback {code,redirect} 回调换 openid → 登录或 need_bind（公开）。
func (h *MerchantAuthHandler) OAuthCallback(c *gin.Context) {
	if h.oauth == nil {
		resp.Fail(c, 1101, "快捷登录未启用")
		return
	}
	var req struct {
		Code     string `json:"code" binding:"required"`
		Redirect string `json:"redirect"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.oauth.Callback(c.Request.Context(), c.Param("provider"), req.Code, req.Redirect, c.ClientIP())
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, out)
}

// OAuthBind POST /api/merchant/oauth/bind 未绑定用户输入账号密码绑定 openid 并登录（公开）。
func (h *MerchantAuthHandler) OAuthBind(c *gin.Context) {
	if h.oauth == nil {
		resp.Fail(c, 1101, "快捷登录未启用")
		return
	}
	var req dto.OAuthBindReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.oauth.Bind(req.Provider, req.OpenID,
		dto.MerchantLoginReq{Type: req.Type, Account: req.Account, Password: req.Password}, c.ClientIP())
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, out)
}

// OAuthUnbind POST /api/merchant/oauth/unbind 解绑当前商户的第三方账号（登录态，对齐 epay editinfo 解绑）。
func (h *MerchantAuthHandler) OAuthUnbind(c *gin.Context) {
	if h.oauth == nil {
		resp.Fail(c, 1101, "快捷登录未启用")
		return
	}
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "未登录")
		return
	}
	var req struct {
		Provider string `json:"provider" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.oauth.Unbind(uid, req.Provider); err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, gin.H{"provider": req.Provider})
}

func NewMerchantAuthHandler(svc *service.MerchantAuthService) *MerchantAuthHandler {
	return &MerchantAuthHandler{svc: svc}
}

// SetRegMethodDeps 注入邮箱 OTP 服务 + 配置读取（注册方式开关 / 邮箱验证码）。
func (h *MerchantAuthHandler) SetRegMethodDeps(mail *service.MailService, cfg *service.ConfigService) {
	h.mail = mail
	h.cfg = cfg
}

func failFromMerchantAuthErr(c *gin.Context, err error) {
	var ae *service.MerchantAuthError
	if errors.As(err, &ae) {
		resp.Fail(c, 1101, ae.Msg)
		return
	}
	resp.Fail(c, 1101, "登录失败: "+err.Error())
}

// Login POST /api/merchant/login
func (h *MerchantAuthHandler) Login(c *gin.Context) {
	var req dto.MerchantLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.svc.Login(req, c.ClientIP())
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, out)
}

// SetRegService 注入自助流程服务（注册/完善资料/找回密码）与图形验证码服务。
func (h *MerchantAuthHandler) SetRegService(reg *service.MerchantRegService, captcha *service.CaptchaService) {
	h.reg = reg
	h.captcha = captcha
}

// Captcha GET /api/merchant/captcha 下发图形验证码（公开）。
func (h *MerchantAuthHandler) Captcha(c *gin.Context) {
	token, svg, err := h.captcha.Generate()
	if err != nil {
		resp.Fail(c, 1101, "验证码生成失败")
		return
	}
	resp.OK(c, dto.CaptchaResp{Token: token, SVG: svg})
}

// Register POST /api/merchant/register 商户注册（公开）。
func (h *MerchantAuthHandler) Register(c *gin.Context) {
	var req dto.MerchantRegReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.reg.Register(req)
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, out)
}

// FindPwd POST /api/merchant/findpwd 找回密码（公开）。
func (h *MerchantAuthHandler) FindPwd(c *gin.Context) {
	var req dto.MerchantFindPwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.reg.FindPwd(req); err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, gin.H{"msg": "密码已重置，请用新密码登录"})
}

// Complete POST /api/merchant/complete 完善资料（需登录）。
func (h *MerchantAuthHandler) Complete(c *gin.Context) {
	uid, _ := c.Get(middleware.CtxUID)
	id, ok := uid.(uint)
	if !ok || id == 0 {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.MerchantCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.reg.Complete(id, req); err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": id})
}

// Info GET /api/merchant/info 当前登录商户信息
func (h *MerchantAuthHandler) Info(c *gin.Context) {
	uid, _ := c.Get(middleware.CtxUID)
	id, ok := uid.(uint)
	if !ok || id == 0 {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	info, err := h.svc.CurrentInfo(id)
	if err != nil {
		failFromMerchantAuthErr(c, err)
		return
	}
	resp.OK(c, info)
}
