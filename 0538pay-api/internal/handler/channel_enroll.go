package handler

import (
	"errors"
	"io"
	"strconv"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/middleware"
	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// ChannelEnrollHandler 服务商通道商户进件接口（epay 精仿线，pay_channel_enroll）。
// 一套 service 同时服务商户端 /m（自助建单/填料/提交/查看，强制按登录 uid 隔离）
// 与后台 /admin（审核列表/通过/驳回/解密报送，看全部）。
type ChannelEnrollHandler struct {
	svc *service.ChannelEnrollService
}

func NewChannelEnrollHandler(svc *service.ChannelEnrollService) *ChannelEnrollHandler {
	return &ChannelEnrollHandler{svc: svc}
}

func failCE(c *gin.Context, err error) {
	var ce *service.ChannelEnrollError
	if errors.As(err, &ce) {
		resp.Fail(c, 1120, ce.Msg)
		return
	}
	resp.Fail(c, 1120, "操作失败: "+err.Error())
}

// adminName 从鉴权上下文取当前后台操作人用户名（审核留痕用）。
func adminName(c *gin.Context) string {
	v, _ := c.Get(middleware.CtxName)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// —— 商户端 /m（强制按登录 uid 隔离）——

// MyChannels GET /api/merchant/channel-enrolls/channels 可进件的服务商通道选项。
func (h *ChannelEnrollHandler) MyChannels(c *gin.Context) {
	if _, ok := currentUID(c); !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	list, err := h.svc.EnrollableChannels()
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// MyList GET /api/merchant/channel-enrolls 我的进件单列表。
func (h *ChannelEnrollHandler) MyList(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pagesize"))
	list, total, err := h.svc.List(repository.ChannelEnrollQuery{
		UID:       uid,
		ChannelID: atoiDefault(c.Query("channel_id"), 0),
		Status:    c.Query("status"),
		Keyword:   c.Query("keyword"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list, "total": total})
}

// MyGet GET /api/merchant/channel-enrolls/:id 我的进件单详情。
func (h *ChannelEnrollHandler) MyGet(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	detail, err := h.svc.Get(idParam(c), uid)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, detail)
}

// MyCreate POST /api/merchant/channel-enrolls 建单（选服务商主通道）。
func (h *ChannelEnrollHandler) MyCreate(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.ChannelEnrollCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	e, err := h.svc.Create(uid, req)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"id": e.ID, "enroll_no": e.EnrollNo, "status": e.Status})
}

// MyFillMaterial POST /api/merchant/channel-enrolls/:id/material 填/改进件资料。
func (h *ChannelEnrollHandler) MyFillMaterial(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.ChannelEnrollMaterialReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	e, err := h.svc.FillMaterial(idParam(c), uid, req)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"id": e.ID, "status": e.Status})
}

// MySubmit POST /api/merchant/channel-enrolls/:id/submit 提交微信进件（全自动 applyment4sub）。
func (h *ChannelEnrollHandler) MySubmit(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	e, err := h.svc.SubmitToWx(c.Request.Context(), idParam(c), uid)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"id": e.ID, "status": e.Status, "wx_state": e.WxState, "wx_applyment_id": e.WxApplymentID})
}

// MySync POST /api/merchant/channel-enrolls/:id/sync 主动拉取微信进件状态（审核中→开通/驳回）。
func (h *ChannelEnrollHandler) MySync(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	e, err := h.svc.SyncWxState(c.Request.Context(), idParam(c), uid)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{
		"id": e.ID, "status": e.Status, "wx_state": e.WxState,
		"wx_state_text": service.ChannelEnrollWxStateText(e.WxState),
		"sub_mchid": e.SubMchID, "subchannel_id": e.SubChannelID,
		"sign_url": e.SignURL, "reject_reason": e.RejectReason,
	})
}

// MyUploadMedia POST /api/merchant/channel-enrolls/:id/media 上传进件资料图片，换微信 media_id。
func (h *ChannelEnrollHandler) MyUploadMedia(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
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
	mediaID, err := h.svc.UploadMedia(c.Request.Context(), idParam(c), uid, fh.Filename, data)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"media_id": mediaID})
}

// MyUploadVideo POST /api/merchant/channel-enrolls/:id/video 上传进件资料视频，换微信 media_id。
func (h *ChannelEnrollHandler) MyUploadVideo(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 400, "请选择要上传的视频文件")
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
	mediaID, err := h.svc.UploadVideo(c.Request.Context(), idParam(c), uid, fh.Filename, data)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"media_id": mediaID})
}

// MyToggle POST /api/merchant/channel-enrolls/:id/toggle 开关自己已开通渠道（支付开关）。
func (h *ChannelEnrollHandler) MyToggle(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req struct {
		Enable bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	e, err := h.svc.ToggleSubChannel(idParam(c), uid, req.Enable)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"id": e.ID, "subchannel_status": func() int8 {
		if req.Enable {
			return 1
		}
		return 0
	}()})
}

// MyDelete DELETE /api/merchant/channel-enrolls/:id 删除自己的进件单（仅草稿/被驳回单，提交前放弃）。
func (h *ChannelEnrollHandler) MyDelete(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	if err := h.svc.Delete(idParam(c), uid); err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"deleted": true})
}

// —— 后台 /admin（看全部，审核）——

// List GET /api/admin/channel-enrolls 进件单列表（后台看全部）。
func (h *ChannelEnrollHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pagesize"))
	list, total, err := h.svc.List(repository.ChannelEnrollQuery{
		UID:       atoiDefaultUint(c.Query("uid")),
		ChannelID: atoiDefault(c.Query("channel_id"), 0),
		Status:    c.Query("status"),
		WxState:   c.Query("wx_state"),
		Keyword:   c.Query("keyword"),
		Sort:      c.Query("sort"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list, "total": total})
}

// Get GET /api/admin/channel-enrolls/:id 进件单详情（后台，脱敏回显）。
func (h *ChannelEnrollHandler) Get(c *gin.Context) {
	detail, err := h.svc.Get(idParam(c), 0)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, detail)
}

// Approve POST /api/admin/channel-enrolls/:id/approve 审核通过（填子商户号）。
func (h *ChannelEnrollHandler) Approve(c *gin.Context) {
	var req dto.ChannelEnrollApproveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	e, err := h.svc.Approve(idParam(c), adminName(c), req)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"id": e.ID, "status": e.Status, "sub_mchid": e.SubMchID, "subchannel_id": e.SubChannelID})
}

// Reject POST /api/admin/channel-enrolls/:id/reject 审核驳回（填原因）。
func (h *ChannelEnrollHandler) Reject(c *gin.Context) {
	var req dto.ChannelEnrollRejectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	e, err := h.svc.Reject(idParam(c), adminName(c), req)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{"id": e.ID, "status": e.Status})
}

// Sync POST /api/admin/channel-enrolls/:id/sync 后台主动拉取微信进件状态（同商户端逻辑，看全部）。
func (h *ChannelEnrollHandler) Sync(c *gin.Context) {
	e, err := h.svc.SyncWxState(c.Request.Context(), idParam(c), 0)
	if err != nil {
		failCE(c, err)
		return
	}
	resp.OK(c, gin.H{
		"id": e.ID, "status": e.Status, "wx_state": e.WxState,
		"wx_state_text": service.ChannelEnrollWxStateText(e.WxState),
		"sub_mchid": e.SubMchID, "subchannel_id": e.SubChannelID,
		"sign_url": e.SignURL, "reject_reason": e.RejectReason,
	})
}

// atoiDefault 解析字符串为 int，失败返回 def。
func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// atoiDefaultUint 解析字符串为 uint，失败返回 0。
func atoiDefaultUint(s string) uint {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}
