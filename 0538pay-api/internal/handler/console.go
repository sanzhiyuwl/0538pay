package handler

import (
	"errors"
	"strconv"

	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// ConsoleHandler 代理控制台（平台运营视角，管所有代理）接口。
// 挂在 /api/console/* 下，走 admin token 鉴权（前端 admin 与 console 共用 admin_token）。
type ConsoleHandler struct {
	agent  *service.AgentService
	enroll *service.EnrollService
}

func NewConsoleHandler(agent *service.AgentService, enroll *service.EnrollService) *ConsoleHandler {
	return &ConsoleHandler{agent: agent, enroll: enroll}
}

func failConsole(c *gin.Context, err error) {
	var ae *service.AgentError
	if errors.As(err, &ae) {
		resp.Fail(c, 1301, ae.Msg)
		return
	}
	var ee *service.EnrollError
	if errors.As(err, &ee) {
		resp.Fail(c, 1302, ee.Msg)
		return
	}
	resp.Fail(c, 1300, "操作失败: "+err.Error())
}

func consoleIDParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

func consoleIntQuery(c *gin.Context, key string, def int) int {
	if v, err := strconv.Atoi(c.Query(key)); err == nil {
		return v
	}
	return def
}

// —— 权限点清单 ——

// Permissions GET /api/console/agent-permissions 返回权限点清单供平台勾选。
func (h *ConsoleHandler) Permissions(c *gin.Context) {
	resp.OK(c, h.agent.Permissions())
}

// —— 代理管理 ——

// agentSaveReq 代理创建/编辑入参。
type agentSaveReq struct {
	Name        string   `json:"name"`
	Account     string   `json:"account"`
	Password    string   `json:"password"`
	Contact     string   `json:"contact"`
	Remark      string   `json:"remark"`
	Status      *int8    `json:"status"`
	Permissions []string `json:"permissions"`
}

// ListAgents GET /api/console/agents 代理列表
func (h *ConsoleHandler) ListAgents(c *gin.Context) {
	keyword := c.Query("keyword")
	var status *int8
	if v := c.Query("status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			s := int8(n)
			status = &s
		}
	}
	page := consoleIntQuery(c, "page", 1)
	pageSize := consoleIntQuery(c, "pageSize", 20)
	list, total, err := h.agent.List(keyword, status, page, pageSize)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, page, pageSize)
}

// GetAgent GET /api/console/agents/:id 代理详情
func (h *ConsoleHandler) GetAgent(c *gin.Context) {
	a, err := h.agent.Get(consoleIDParam(c))
	if err != nil {
		resp.Fail(c, 1301, "代理不存在")
		return
	}
	resp.OK(c, a)
}

// CreateAgent POST /api/console/agents 新建代理
func (h *ConsoleHandler) CreateAgent(c *gin.Context) {
	var req agentSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	a, err := h.agent.Create(req.Name, req.Account, req.Password, req.Contact, req.Remark, req.Permissions)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, gin.H{"id": a.ID})
}

// UpdateAgent PUT /api/console/agents/:id 编辑代理
func (h *ConsoleHandler) UpdateAgent(c *gin.Context) {
	id := consoleIDParam(c)
	if id == 0 {
		resp.Fail(c, 400, "参数错误")
		return
	}
	var req agentSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.agent.Update(id, req.Name, req.Contact, req.Remark, req.Status, req.Permissions, req.Password); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// SetAgentStatus PUT /api/console/agents/:id/status 启停代理
func (h *ConsoleHandler) SetAgentStatus(c *gin.Context) {
	id := consoleIDParam(c)
	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.agent.SetStatus(id, req.Status); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// DeleteAgent DELETE /api/console/agents/:id 删除代理
func (h *ConsoleHandler) DeleteAgent(c *gin.Context) {
	if err := h.agent.Delete(consoleIDParam(c)); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// —— 名额管理 ——

// AgentWallet GET /api/console/agents/:id/quota 代理名额钱包
func (h *ConsoleHandler) AgentWallet(c *gin.Context) {
	w, err := h.agent.Wallet(consoleIDParam(c))
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, w)
}

// AdjustQuota POST /api/console/agents/:id/quota 平台侧调整代理名额（售卖/纠错）
func (h *ConsoleHandler) AdjustQuota(c *gin.Context) {
	id := consoleIDParam(c)
	var req struct {
		Change int    `json:"change"`
		Amount string `json:"amount"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	amount, _ := decimal.NewFromString(req.Amount)
	if err := h.agent.AdjustQuota(id, req.Change, amount, req.Remark); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// QuotaLogs GET /api/console/quota-logs 名额流水（可按 agent_id 过滤）
func (h *ConsoleHandler) QuotaLogs(c *gin.Context) {
	var agentID *uint
	if v := c.Query("agent_id"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			id := uint(n)
			agentID = &id
		}
	}
	page := consoleIntQuery(c, "page", 1)
	pageSize := consoleIntQuery(c, "pageSize", 20)
	list, total, err := h.agent.QuotaLogs(agentID, page, pageSize)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, page, pageSize)
}

// —— 进件申请 ——

// ListEnrolls GET /api/console/enrolls 进件单列表（平台看全部）
func (h *ConsoleHandler) ListEnrolls(c *gin.Context) {
	q := repository.EnrollQuery{
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		Page:     consoleIntQuery(c, "page", 1),
		PageSize: consoleIntQuery(c, "pageSize", 20),
	}
	if v := c.Query("agent_id"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			id := uint(n)
			q.AgentID = &id
		}
	}
	if v := c.Query("source"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			q.Source = &n
		}
	}
	list, total, err := h.enroll.ListEnrolls(q)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// GetEnroll GET /api/console/enrolls/:id 进件单详情
func (h *ConsoleHandler) GetEnroll(c *gin.Context) {
	e, err := h.enroll.GetEnroll(consoleIDParam(c), nil)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, e)
}

// CreateEnroll POST /api/console/enrolls 平台代填建进件单（付费前置：建单即下开户费收款）
func (h *ConsoleHandler) CreateEnroll(c *gin.Context) {
	var req struct {
		AgentID      uint   `json:"agent_id"`
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
		AgentID:      req.AgentID,
		MerchantName: req.MerchantName,
		ContactPhone: req.ContactPhone,
		Path:         req.Path,
		Source:       1, // 平台代填
		Plugin:       req.Plugin,
	})
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, gin.H{"enroll": e, "pay": pay})
}

// SubmitEnroll POST /api/console/enrolls/:id/submit 平台把已支付待完善的进件单提交微信审核
func (h *ConsoleHandler) SubmitEnroll(c *gin.Context) {
	e, err := h.enroll.SubmitToWx(c.Request.Context(), consoleIDParam(c), nil)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, e)
}

// SyncEnroll POST /api/console/enrolls/:id/sync 主动拉取微信申请单最新状态并推进本地状态机
func (h *ConsoleHandler) SyncEnroll(c *gin.Context) {
	e, err := h.enroll.SyncWxState(c.Request.Context(), consoleIDParam(c), nil)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, e)
}

// RefundEnroll POST /api/console/enrolls/:id/refund 平台兜底退款（任意代理名下单，agentID=nil）。
// 四道拦截 + sub_mchid 硬锁定在 service 内校验；平台端也不破硬锁例外。
func (h *ConsoleHandler) RefundEnroll(c *gin.Context) {
	r, err := h.enroll.RefundEnroll(c.Request.Context(), consoleIDParam(c), nil)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, r)
}

// —— 邀请链接 ——

// ListInvites GET /api/console/enroll-invites 邀请链接列表（平台看全部）
func (h *ConsoleHandler) ListInvites(c *gin.Context) {
	var agentID *uint
	if v := c.Query("agent_id"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			id := uint(n)
			agentID = &id
		}
	}
	page := consoleIntQuery(c, "page", 1)
	pageSize := consoleIntQuery(c, "pageSize", 20)
	list, total, err := h.enroll.ListInvites(agentID, page, pageSize)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, page, pageSize)
}

// CreateInvite POST /api/console/enroll-invites 平台生成邀请链接（可指定归属代理）
func (h *ConsoleHandler) CreateInvite(c *gin.Context) {
	var req struct {
		AgentID uint   `json:"agent_id"`
		Name    string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	v, err := h.enroll.CreateInvite(req.AgentID, req.Name)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, v)
}

// SetInviteStatus PUT /api/console/enroll-invites/:id/status 停启用邀请链接
func (h *ConsoleHandler) SetInviteStatus(c *gin.Context) {
	id := consoleIDParam(c)
	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.enroll.SetInviteStatus(id, req.Status, nil); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// DeleteInvite DELETE /api/console/enroll-invites/:id 删除邀请链接
func (h *ConsoleHandler) DeleteInvite(c *gin.Context) {
	if err := h.enroll.DeleteInvite(consoleIDParam(c), nil); err != nil {
		failConsole(c, err)
		return
	}
	resp.OK(c, nil)
}

// —— 佣金结算 ——

// ListSettlements GET /api/console/enroll-settlements 进件结算流水（平台看全部）
func (h *ConsoleHandler) ListSettlements(c *gin.Context) {
	var agentID *uint
	if v := c.Query("agent_id"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			id := uint(n)
			agentID = &id
		}
	}
	page := consoleIntQuery(c, "page", 1)
	pageSize := consoleIntQuery(c, "pageSize", 20)
	list, total, err := h.enroll.ListSettleLogs(agentID, page, pageSize)
	if err != nil {
		failConsole(c, err)
		return
	}
	resp.Page(c, list, total, page, pageSize)
}
