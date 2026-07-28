package handler

import (
	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// EnrollPublicHandler 客户自助进件公开页（/enroll/:code）后端接口。
// 全部免登录，靠邀请 code 定位归属代理；图形验证码 + 限流（路由层挂 RateLimit）防刷。
// 建单走 source=3 客户自助，路径固定商户自付，付费前置拉起收银台。
type EnrollPublicHandler struct {
	enroll  *service.EnrollService
	agent   *service.AgentService
	captcha *service.CaptchaService
}

func NewEnrollPublicHandler(enroll *service.EnrollService, agent *service.AgentService, captcha *service.CaptchaService) *EnrollPublicHandler {
	return &EnrollPublicHandler{enroll: enroll, agent: agent, captcha: captcha}
}

// agentName 按 id 取代理名（0=平台）；查不到回退占位，供公开页展示"由 XX 提供服务"。
func (h *EnrollPublicHandler) agentName(id uint) string {
	if id == 0 {
		return "平台"
	}
	a, err := h.agent.Get(id)
	if err != nil || a == nil {
		return "服务商"
	}
	return a.Name
}

// Info GET /api/enroll/:code 公开页落地：校验邀请可用 + 打点打开数，返回展示信息。
func (h *EnrollPublicHandler) Info(c *gin.Context) {
	info, err := h.enroll.ResolveInvite(c.Param("code"), h.agentName)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, info)
}

// Captcha GET /api/enroll/captcha 下发图形验证码（公开）。
func (h *EnrollPublicHandler) Captcha(c *gin.Context) {
	token, svg, err := h.captcha.Generate()
	if err != nil {
		resp.Fail(c, 1302, "验证码生成失败")
		return
	}
	resp.OK(c, dto.CaptchaResp{Token: token, SVG: svg})
}

// Submit POST /api/enroll/:code 客户自助建单：校验验证码 → source=3 建单 → 返回收银台信息。
func (h *EnrollPublicHandler) Submit(c *gin.Context) {
	var req struct {
		MerchantName  string `json:"merchant_name"`
		ContactPhone  string `json:"contact_phone"`
		Plugin        string `json:"plugin"`
		CaptchaToken  string `json:"captcha_token"`
		CaptchaCode   string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	// 图形验证码校验（一次性，防脚本批量提交）。
	if !h.captcha.Verify(req.CaptchaToken, req.CaptchaCode) {
		resp.Fail(c, 1302, "图形验证码错误或已过期")
		return
	}
	e, pay, err := h.enroll.PublicCreateEnroll(c.Request.Context(), service.PublicCreateEnrollReq{
		Code:         c.Param("code"),
		MerchantName: req.MerchantName,
		ContactPhone: req.ContactPhone,
		Plugin:       req.Plugin,
	})
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, gin.H{"enroll": e, "pay": pay})
}
