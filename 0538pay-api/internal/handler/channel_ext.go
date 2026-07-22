package handler

import (
	"errors"
	"strconv"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// ===== 通道轮询组 =====

// RollHandler 通道轮询组接口。
type RollHandler struct {
	svc *service.RollService
}

func NewRollHandler(svc *service.RollService) *RollHandler { return &RollHandler{svc: svc} }

func failFromRollErr(c *gin.Context, err error) {
	var re *service.RollError
	if errors.As(err, &re) {
		resp.Fail(c, re.Code, re.Msg)
		return
	}
	resp.Fail(c, 1105, "操作失败: "+err.Error())
}

// List GET /api/admin/rolls
func (h *RollHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		resp.Fail(c, 1105, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// Create POST /api/admin/rolls
func (h *RollHandler) Create(c *gin.Context) {
	var req dto.RollSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		failFromRollErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Update PUT /api/admin/rolls/:id
func (h *RollHandler) Update(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "轮询组 ID 不合法")
		return
	}
	var req dto.RollSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(id, req); err != nil {
		failFromRollErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// SetStatus PUT /api/admin/rolls/:id/status
func (h *RollHandler) SetStatus(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "轮询组 ID 不合法")
		return
	}
	var req dto.RollStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(id, req.Status); err != nil {
		failFromRollErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": req.Status})
}

// Delete DELETE /api/admin/rolls/:id
func (h *RollHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "轮询组 ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromRollErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// ===== 子通道（商户维度）=====

// SubChannelHandler 子通道接口。
type SubChannelHandler struct {
	svc *service.SubChannelService
}

func NewSubChannelHandler(svc *service.SubChannelService) *SubChannelHandler {
	return &SubChannelHandler{svc: svc}
}

func failFromSubErr(c *gin.Context, err error) {
	var se *service.SubChannelError
	if errors.As(err, &se) {
		resp.Fail(c, se.Code, se.Msg)
		return
	}
	resp.Fail(c, 1106, "操作失败: "+err.Error())
}

// uidQuery 解析 query ?uid= 商户号，失败返回 0。
func uidQuery(c *gin.Context) uint {
	n, err := strconv.ParseUint(c.Query("uid"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

// List GET /api/admin/subchannels?uid= 某商户的子通道列表。
func (h *SubChannelHandler) List(c *gin.Context) {
	uid := uidQuery(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	list, err := h.svc.ListByMerchant(uid)
	if err != nil {
		resp.Fail(c, 1106, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// Create POST /api/admin/subchannels?uid= 新增子通道。
func (h *SubChannelHandler) Create(c *gin.Context) {
	uid := uidQuery(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	var req dto.SubChannelSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.Create(uid, req)
	if err != nil {
		failFromSubErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Update PUT /api/admin/subchannels/:id 编辑子通道。
func (h *SubChannelHandler) Update(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "子通道 ID 不合法")
		return
	}
	var req dto.SubChannelSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(id, req); err != nil {
		failFromSubErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// SetStatus PUT /api/admin/subchannels/:id/status 切换子通道开关。
func (h *SubChannelHandler) SetStatus(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "子通道 ID 不合法")
		return
	}
	var req dto.SubChannelStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(id, req.Status); err != nil {
		failFromSubErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": req.Status})
}

// Delete DELETE /api/admin/subchannels/:id 删除子通道。
func (h *SubChannelHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "子通道 ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromSubErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// ===== 支付方式 PayType =====

type PayTypeHandler struct{ svc *service.PayTypeService }

func NewPayTypeHandler(svc *service.PayTypeService) *PayTypeHandler { return &PayTypeHandler{svc: svc} }

func failFromPayTypeErr(c *gin.Context, err error) {
	var e *service.PayTypeError
	if errors.As(err, &e) {
		resp.Fail(c, e.Code, e.Msg)
		return
	}
	resp.Fail(c, 1107, "操作失败: "+err.Error())
}

func (h *PayTypeHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		resp.Fail(c, 1107, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (h *PayTypeHandler) Create(c *gin.Context) {
	var req dto.PayTypeSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		failFromPayTypeErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (h *PayTypeHandler) Update(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "支付方式 ID 不合法")
		return
	}
	var req dto.PayTypeSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(id, req); err != nil {
		failFromPayTypeErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (h *PayTypeHandler) SetStatus(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "支付方式 ID 不合法")
		return
	}
	var req dto.PayTypeStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(id, req.Status); err != nil {
		failFromPayTypeErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": req.Status})
}

func (h *PayTypeHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "支付方式 ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromPayTypeErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// ===== 微信公众号/小程序 Weixin =====

type WeixinHandler struct{ svc *service.WeixinService }

func NewWeixinHandler(svc *service.WeixinService) *WeixinHandler { return &WeixinHandler{svc: svc} }

func failFromWeixinErr(c *gin.Context, err error) {
	var e *service.WeixinError
	if errors.As(err, &e) {
		resp.Fail(c, e.Code, e.Msg)
		return
	}
	resp.Fail(c, 1108, "操作失败: "+err.Error())
}

func (h *WeixinHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		resp.Fail(c, 1108, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (h *WeixinHandler) Create(c *gin.Context) {
	var req dto.WeixinSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		failFromWeixinErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (h *WeixinHandler) Update(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "账号 ID 不合法")
		return
	}
	var req dto.WeixinSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(id, req); err != nil {
		failFromWeixinErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (h *WeixinHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "账号 ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromWeixinErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// ===== 企业微信 Wework =====

type WeworkHandler struct{ svc *service.WeworkService }

func NewWeworkHandler(svc *service.WeworkService) *WeworkHandler { return &WeworkHandler{svc: svc} }

func failFromWeworkErr(c *gin.Context, err error) {
	var e *service.WeworkError
	if errors.As(err, &e) {
		resp.Fail(c, e.Code, e.Msg)
		return
	}
	resp.Fail(c, 1109, "操作失败: "+err.Error())
}

func (h *WeworkHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		resp.Fail(c, 1109, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

func (h *WeworkHandler) Create(c *gin.Context) {
	var req dto.WeworkSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		failFromWeworkErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (h *WeworkHandler) Update(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "账号 ID 不合法")
		return
	}
	var req dto.WeworkSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(id, req); err != nil {
		failFromWeworkErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

func (h *WeworkHandler) SetStatus(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "账号 ID 不合法")
		return
	}
	var req dto.WeworkStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(id, req.Status); err != nil {
		failFromWeworkErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": req.Status})
}

func (h *WeworkHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "账号 ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromWeworkErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}
