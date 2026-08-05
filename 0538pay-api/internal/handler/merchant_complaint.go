package handler

import (
	"io"
	"strconv"

	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// MerchantComplaintHandler 消费者投诉2.0 商户端（/m）接口——商户自助看/处理【自己名下】的投诉。
// 自研扩展，挂服务商进件线，与 admin 端共用同一 WxComplaintService。数据隔离：
// 列表强制按登录商户 uid 过滤（ListForMerchant），每个详情/动作前置 AssertMerchantOwn，
// 在 admin 越权拦截（被诉子商户须在本平台名下）之上再叠「merchant_id==登录 uid」一层，双保险。
// 不开放：回调地址自管理 / 兜底对账（平台服务商级运维，仅 admin）。
type MerchantComplaintHandler struct {
	svc *service.WxComplaintService
}

func NewMerchantComplaintHandler(svc *service.WxComplaintService) *MerchantComplaintHandler {
	return &MerchantComplaintHandler{svc: svc}
}

// MyList GET /api/merchant/complaints 我的投诉单列表（强制按登录商户隔离）。
func (h *MerchantComplaintHandler) MyList(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pagesize"))
	res, err := h.svc.ListForMerchant(uid, repository.WxComplaintQuery{
		Keyword:  c.Query("keyword"),
		State:    c.Query("state"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// MyDetail GET /api/merchant/complaints/:id 我的投诉单详情（读本地 + 回调时间线）。
func (h *MerchantComplaintHandler) MyDetail(c *gin.Context) {
	uid, id, ok := h.uidAndID(c)
	if !ok {
		return
	}
	res, err := h.svc.DetailForMerchant(id, uid)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// MySync POST /api/merchant/complaints/:id/sync 现查微信详情覆盖本地快照。
func (h *MerchantComplaintHandler) MySync(c *gin.Context) {
	uid, id, ok := h.uidAndID(c)
	if !ok {
		return
	}
	if _, err := h.svc.AssertMerchantOwn(id, uid); err != nil {
		failWC(c, err)
		return
	}
	res, err := h.svc.SyncDetail(c.Request.Context(), id)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, res)
}

// MyHistory GET /api/merchant/complaints/:id/history 协商历史（现查微信）。
func (h *MerchantComplaintHandler) MyHistory(c *gin.Context) {
	uid, id, ok := h.uidAndID(c)
	if !ok {
		return
	}
	if _, err := h.svc.AssertMerchantOwn(id, uid); err != nil {
		failWC(c, err)
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

// MyReply POST /api/merchant/complaints/:id/reply 回复用户。
func (h *MerchantComplaintHandler) MyReply(c *gin.Context) {
	uid, id, ok := h.uidAndID(c)
	if !ok {
		return
	}
	if _, err := h.svc.AssertMerchantOwn(id, uid); err != nil {
		failWC(c, err)
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

// MyComplete POST /api/merchant/complaints/:id/complete 反馈处理完成。
func (h *MerchantComplaintHandler) MyComplete(c *gin.Context) {
	uid, id, ok := h.uidAndID(c)
	if !ok {
		return
	}
	if _, err := h.svc.AssertMerchantOwn(id, uid); err != nil {
		failWC(c, err)
		return
	}
	if err := h.svc.Complete(c.Request.Context(), id); err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// MyUpdateRefund POST /api/merchant/complaints/:id/refund 更新退款审批（APPROVE/REJECT）。
func (h *MerchantComplaintHandler) MyUpdateRefund(c *gin.Context) {
	uid, id, ok := h.uidAndID(c)
	if !ok {
		return
	}
	if _, err := h.svc.AssertMerchantOwn(id, uid); err != nil {
		failWC(c, err)
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

// MyReplyImmediate POST /api/merchant/complaints/:id/immediate 回复需即时服务的投诉单。
func (h *MerchantComplaintHandler) MyReplyImmediate(c *gin.Context) {
	uid, id, ok := h.uidAndID(c)
	if !ok {
		return
	}
	if _, err := h.svc.AssertMerchantOwn(id, uid); err != nil {
		failWC(c, err)
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

// MyUploadImage POST /api/merchant/complaints/upload 上传反馈图片，返回 media_id（供回复引用）。
func (h *MerchantComplaintHandler) MyUploadImage(c *gin.Context) {
	if _, ok := currentUID(c); !ok {
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
	mediaID, err := h.svc.UploadImage(c.Request.Context(), fh.Filename, data)
	if err != nil {
		failWC(c, err)
		return
	}
	resp.OK(c, gin.H{"media_id": mediaID})
}

// uidAndID 取登录商户 uid + 路径 :id，任一无效已写响应并返回 ok=false。
func (h *MerchantComplaintHandler) uidAndID(c *gin.Context) (uint, uint, bool) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return 0, 0, false
	}
	id, ok := wcID(c)
	if !ok {
		return 0, 0, false
	}
	return uid, id, true
}
