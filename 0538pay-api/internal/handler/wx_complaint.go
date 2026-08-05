package handler

import (
	"errors"
	"io"
	"strconv"

	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// WxComplaintHandler 消费者投诉2.0 后台接口（自研扩展，admin 全量统一台）。
// 一期：列表/详情/协商历史（读）+ 回复/结单/退款审批/即时服务/传图（代处理）+ 回调地址自管理 + 手动刷新/对账。
type WxComplaintHandler struct {
	svc *service.WxComplaintService
}

func NewWxComplaintHandler(svc *service.WxComplaintService) *WxComplaintHandler {
	return &WxComplaintHandler{svc: svc}
}

func failWC(c *gin.Context, err error) {
	var we *service.WxComplaintError
	if errors.As(err, &we) {
		resp.Fail(c, 1131, we.Msg)
		return
	}
	resp.Fail(c, 1131, "操作失败: "+err.Error())
}

func wcID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 400, "投诉单号无效")
		return 0, false
	}
	return uint(id), true
}

// List GET /api/admin/wx-complaints 投诉单列表（全量，可按状态/子商户/关键词筛选）。
func (h *WxComplaintHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pagesize"))
	q := repository.WxComplaintQuery{
		Keyword:          c.Query("keyword"),
		ComplaintedMchID: c.Query("complainted_mchid"),
		State:            c.Query("state"),
		Page:             page,
		PageSize:         pageSize,
	}
	res, err := h.svc.List(q)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// Detail GET /api/admin/wx-complaints/:id 投诉单详情（读本地 + 回调时间线）。
func (h *WxComplaintHandler) Detail(c *gin.Context) {
	id, ok := wcID(c)
	if !ok {
		return
	}
	res, err := h.svc.Detail(id)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// Sync POST /api/admin/wx-complaints/:id/sync 现查微信详情覆盖本地快照。
func (h *WxComplaintHandler) Sync(c *gin.Context) {
	id, ok := wcID(c)
	if !ok {
		return
	}
	res, err := h.svc.SyncDetail(c.Request.Context(), id)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// History GET /api/admin/wx-complaints/:id/history 协商历史（现查微信）。
func (h *WxComplaintHandler) History(c *gin.Context) {
	id, ok := wcID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	res, err := h.svc.History(c.Request.Context(), int(id), limit, offset)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// Reply POST /api/admin/wx-complaints/:id/reply 回复用户。
func (h *WxComplaintHandler) Reply(c *gin.Context) {
	id, ok := wcID(c)
	if !ok {
		return
	}
	var req struct {
		Content  string   `json:"content"`
		Images   []string `json:"images"`
		JumpURL  string   `json:"jump_url"`
		JumpText string   `json:"jump_url_text"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Reply(c.Request.Context(), id, req.Content, req.Images, req.JumpURL, req.JumpText); err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Complete POST /api/admin/wx-complaints/:id/complete 反馈处理完成。
func (h *WxComplaintHandler) Complete(c *gin.Context) {
	id, ok := wcID(c)
	if !ok {
		return
	}
	if err := h.svc.Complete(c.Request.Context(), id); err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// UpdateRefund POST /api/admin/wx-complaints/:id/refund 更新退款审批结果（APPROVE/REJECT）。
func (h *WxComplaintHandler) UpdateRefund(c *gin.Context) {
	id, ok := wcID(c)
	if !ok {
		return
	}
	var req struct {
		Action          string   `json:"action"`
		LaunchRefundDay int      `json:"launch_refund_day"`
		RejectReason    string   `json:"reject_reason"`
		RejectMediaList []string `json:"reject_media_list"`
		Remark          string   `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	err := h.svc.UpdateRefund(c.Request.Context(), id, service.UpdateRefundReq{
		Action:          req.Action,
		LaunchRefundDay: req.LaunchRefundDay,
		RejectReason:    req.RejectReason,
		RejectMediaList: req.RejectMediaList,
		Remark:          req.Remark,
	})
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ReplyImmediate POST /api/admin/wx-complaints/:id/immediate 回复需即时服务的投诉单。
func (h *WxComplaintHandler) ReplyImmediate(c *gin.Context) {
	id, ok := wcID(c)
	if !ok {
		return
	}
	var req struct {
		Content string   `json:"content"`
		Images  []string `json:"images"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.ReplyImmediate(c.Request.Context(), id, req.Content, req.Images); err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// UploadImage POST /api/admin/wx-complaints/upload 上传商户反馈图片，返回 media_id。
func (h *WxComplaintHandler) UploadImage(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 400, "请选择要上传的图片文件")
		return
	}
	f, err := fh.Open()
	if err != nil {
		resp.Fail(c, 400, "读取上传文件失败")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		resp.Fail(c, 400, "读取上传文件失败")
		return
	}
	mediaID, err := h.svc.UploadImage(c.Request.Context(), fh.Filename, data)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"media_id": mediaID})
}

// GetNotifyURL GET /api/admin/wx-complaints/notify-url 查询回调地址（微信侧已注册 + 本地期望）。
func (h *WxComplaintHandler) GetNotifyURL(c *gin.Context) {
	res, err := h.svc.GetNotifyURL(c.Request.Context())
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// SetNotifyURL PUT /api/admin/wx-complaints/notify-url 设置回调地址（创建或更新，幂等）。
func (h *WxComplaintHandler) SetNotifyURL(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	res, err := h.svc.SetNotifyURL(c.Request.Context(), req.URL)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// DeleteNotifyURL DELETE /api/admin/wx-complaints/notify-url 删除微信侧回调地址。
func (h *WxComplaintHandler) DeleteNotifyURL(c *gin.Context) {
	if err := h.svc.DeleteNotifyURL(c.Request.Context()); err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Reconcile POST /api/admin/wx-complaints/reconcile 手动触发轮询兜底对账（begin_date/end_date，跨度≤30天）。
func (h *WxComplaintHandler) Reconcile(c *gin.Context) {
	var req struct {
		BeginDate string `json:"begin_date"`
		EndDate   string `json:"end_date"`
	}
	_ = c.ShouldBindJSON(&req)
	n, err := h.svc.Reconcile(c.Request.Context(), req.BeginDate, req.EndDate)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"synced": n})
}
