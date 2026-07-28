package handler

import (
	"strconv"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/middleware"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// AgentHandler 独立代理端 /agent 接口（代理只看/只碰自己名下）。
// 登录签发 scope=agent 的 JWT，token 里 UID 即 agent_id，所有业务接口据此强制隔离——
// 不信任前端传来的 agent_id。写操作按代理已开通的 permissions 逐项门控。
type AgentHandler struct {
	agent  *service.AgentService
	enroll *service.EnrollService
}

func NewAgentHandler(agent *service.AgentService, enroll *service.EnrollService) *AgentHandler {
	return &AgentHandler{agent: agent, enroll: enroll}
}

// currentAgentID 从鉴权上下文取当前代理 id（middleware 注入，不信任前端传参）。
func currentAgentID(c *gin.Context) (uint, bool) {
	v, _ := c.Get(middleware.CtxUID)
	id, ok := v.(uint)
	return id, ok && id != 0
}

// requirePerm 校验当前代理是否拥有某权限点；无则回 403 并阻断。
// ★ 实时查库判断（不读 JWT 快照）：平台改权限后旧 token 里的权限是过期的，
//   实时读库保证平台一开通、代理下次操作即刻生效，无需重登。
func (h *AgentHandler) requirePerm(c *gin.Context, key string) bool {
	id, ok := currentAgentID(c)
	if ok && h.agent.HasPermissionLive(id, key) {
		return true
	}
	resp.Fail(c, 1303, "无该功能权限，请联系平台开通")
	return false
}

func agentIntQuery(c *gin.Context, key string, def int) int {
	if v, err := strconv.Atoi(c.Query(key)); err == nil {
		return v
	}
	return def
}

func agentIDParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// —— 登录 / 会话 ——

// Login POST /api/agent/login 代理端登录（公开）。
func (h *AgentHandler) Login(c *gin.Context) {
	var req struct {
		Account  string `json:"account"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	r, err := h.agent.Login(req.Account, req.Password)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, r)
}

// Profile GET /api/agent/profile 当前代理资料（含权限串，供前端动态渲染菜单）。
func (h *AgentHandler) Profile(c *gin.Context) {
	id, ok := currentAgentID(c)
	if !ok {
		resp.Fail(c, 401, "未登录")
		return
	}
	a, err := h.agent.Profile(id)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, a)
}

// Permissions GET /api/agent/permissions 权限点清单（前端渲染菜单文案用，与平台端同源）。
func (h *AgentHandler) Permissions(c *gin.Context) {
	resp.OK(c, h.agent.Permissions())
}

// —— 名额钱包（quota 权限）——

// Wallet GET /api/agent/quota 自己的名额钱包。
func (h *AgentHandler) Wallet(c *gin.Context) {
	if !h.requirePerm(c, service.PermQuota) {
		return
	}
	id, _ := currentAgentID(c)
	w, err := h.agent.Wallet(id)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, w)
}

// QuotaLogs GET /api/agent/quota-logs 自己的名额流水。
func (h *AgentHandler) QuotaLogs(c *gin.Context) {
	if !h.requirePerm(c, service.PermQuota) {
		return
	}
	id, _ := currentAgentID(c)
	page := agentIntQuery(c, "page", 1)
	pageSize := agentIntQuery(c, "pageSize", 20)
	list, total, err := h.agent.QuotaLogs(&id, page, pageSize)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, page, pageSize)
}

// —— 进件申请（enroll 权限）——

// ListEnrolls GET /api/agent/enrolls 自己名下的进件单（强制按 agent_id 隔离）。
func (h *AgentHandler) ListEnrolls(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	id, _ := currentAgentID(c)
	q := repository.EnrollQuery{
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		AgentID:  &id, // 强制只看自己名下
		Page:     agentIntQuery(c, "page", 1),
		PageSize: agentIntQuery(c, "pageSize", 20),
	}
	list, total, err := h.enroll.ListEnrolls(q)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// GetEnroll GET /api/agent/enrolls/:id 进件单详情（校验归属）。
func (h *AgentHandler) GetEnroll(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	id, _ := currentAgentID(c)
	e, err := h.enroll.GetEnroll(agentIDParam(c), &id)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, e)
}

// CreateEnroll POST /api/agent/enrolls 代理代填建进件单（付费前置，source=2）。
func (h *AgentHandler) CreateEnroll(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	id, _ := currentAgentID(c)
	var req struct {
		MerchantName string `json:"merchant_name"`
		ContactPhone string `json:"contact_phone"`
		Path         int    `json:"path"`
		Plugin       string `json:"plugin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	e, pay, err := h.enroll.CreateEnroll(c.Request.Context(), service.CreateEnrollReq{
		AgentID:      id, // 强制归属自己
		MerchantName: req.MerchantName,
		ContactPhone: req.ContactPhone,
		Path:         req.Path,
		Source:       2, // 代理代填
		Plugin:       req.Plugin,
	})
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, gin.H{"enroll": e, "pay": pay})
}

// SubmitEnroll POST /api/agent/enrolls/:id/submit 提交微信审核（校验归属）。
func (h *AgentHandler) SubmitEnroll(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	id, _ := currentAgentID(c)
	e, err := h.enroll.SubmitToWx(c.Request.Context(), agentIDParam(c), &id)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, e)
}

// SyncEnroll POST /api/agent/enrolls/:id/sync 查微信最新状态（校验归属）。
func (h *AgentHandler) SyncEnroll(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	id, _ := currentAgentID(c)
	e, err := h.enroll.SyncWxState(c.Request.Context(), agentIDParam(c), &id)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, e)
}

// RefundEnroll POST /api/agent/enrolls/:id/refund 代理手动退款（refund 权限 + 强隔离只退自己名下）。
// 四道拦截 + sub_mchid 硬锁定在 service 内校验：单存在/归属自己/状态可退/未开通。
func (h *AgentHandler) RefundEnroll(c *gin.Context) {
	if !h.requirePerm(c, service.PermRefund) {
		return
	}
	id, _ := currentAgentID(c)
	r, err := h.enroll.RefundEnroll(c.Request.Context(), agentIDParam(c), &id)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, r)
}

// GetMaterial GET /api/agent/enrolls/:id/material 回显填料表单（enroll 权限 + 归属自己）。
func (h *AgentHandler) GetMaterial(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	id, _ := currentAgentID(c)
	v, err := h.enroll.GetMaterialView(agentIDParam(c), &id)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, v)
}

// FillMaterial POST /api/agent/enrolls/:id/material 填/改全套资料（enroll 权限 + 归属自己，敏感字段加密）。
func (h *AgentHandler) FillMaterial(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	var req dto.EnrollMaterialReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, _ := currentAgentID(c)
	e, err := h.enroll.FillMaterial(agentIDParam(c), &id, req)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, e)
}

// UploadMedia POST /api/agent/enrolls/:id/media 上传进件资料图片（enroll 权限 + 归属自己），返回 media_id。
func (h *AgentHandler) UploadMedia(c *gin.Context) {
	if !h.requirePerm(c, service.PermEnroll) {
		return
	}
	filename, data, ok := readEnrollMedia(c)
	if !ok {
		return
	}
	id, _ := currentAgentID(c)
	mediaID, err := h.enroll.UploadMaterialMedia(c.Request.Context(), agentIDParam(c), &id, filename, data)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, gin.H{"media_id": mediaID})
}

// —— 邀请链接（invite 权限）——

// ListInvites GET /api/agent/enroll-invites 自己名下的邀请链接。
func (h *AgentHandler) ListInvites(c *gin.Context) {
	if !h.requirePerm(c, service.PermInvite) {
		return
	}
	id, _ := currentAgentID(c)
	page := agentIntQuery(c, "page", 1)
	pageSize := agentIntQuery(c, "pageSize", 20)
	list, total, err := h.enroll.ListInvites(&id, page, pageSize)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, page, pageSize)
}

// CreateInvite POST /api/agent/enroll-invites 生成自己名下的邀请链接。
func (h *AgentHandler) CreateInvite(c *gin.Context) {
	if !h.requirePerm(c, service.PermInvite) {
		return
	}
	id, _ := currentAgentID(c)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	v, err := h.enroll.CreateInvite(id, req.Name)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, v)
}

// SetInviteStatus PUT /api/agent/enroll-invites/:id/status 停启用自己的邀请链接（校验归属）。
func (h *AgentHandler) SetInviteStatus(c *gin.Context) {
	if !h.requirePerm(c, service.PermInvite) {
		return
	}
	id, _ := currentAgentID(c)
	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.enroll.SetInviteStatus(agentIDParam(c), req.Status, &id); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// DeleteInvite DELETE /api/agent/enroll-invites/:id 删除自己的邀请链接（校验归属）。
func (h *AgentHandler) DeleteInvite(c *gin.Context) {
	if !h.requirePerm(c, service.PermInvite) {
		return
	}
	id, _ := currentAgentID(c)
	if err := h.enroll.DeleteInvite(agentIDParam(c), &id); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// —— 佣金结算（settlement 权限）——

// ListSettlements GET /api/agent/enroll-settlements 自己名下的进件结算流水。
func (h *AgentHandler) ListSettlements(c *gin.Context) {
	if !h.requirePerm(c, service.PermSettlement) {
		return
	}
	id, _ := currentAgentID(c)
	page := agentIntQuery(c, "page", 1)
	pageSize := agentIntQuery(c, "pageSize", 20)
	list, total, err := h.enroll.ListSettleLogs(&id, page, pageSize)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, page, pageSize)
}
