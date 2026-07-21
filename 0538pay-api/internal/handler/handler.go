package handler

import (
	"errors"
	"strconv"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// AuthHandler 登录相关接口。
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login POST /api/admin/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.svc.Login(req)
	if err != nil {
		resp.Fail(c, 1001, err.Error())
		return
	}
	resp.OK(c, out)
}

// MerchantHandler 商户相关接口。
type MerchantHandler struct {
	svc *service.MerchantService
}

func NewMerchantHandler(svc *service.MerchantService) *MerchantHandler {
	return &MerchantHandler{svc: svc}
}

// List GET /api/admin/merchants
func (h *MerchantHandler) List(c *gin.Context) {
	var q dto.MerchantQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1003, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// ChannelHandler 支付通道相关接口。
type ChannelHandler struct {
	svc *service.ChannelService
}

func NewChannelHandler(svc *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{svc: svc}
}

// List GET /api/admin/channels
func (h *ChannelHandler) List(c *gin.Context) {
	var q dto.ChannelQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1004, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// channelIDParam 解析路径 :id，失败返回 0。
func channelIDParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// failFromChannelErr 把 service.ChannelError 的业务码透传，其它错误按通用失败处理。
func failFromChannelErr(c *gin.Context, err error) {
	var ce *service.ChannelError
	if errors.As(err, &ce) {
		resp.Fail(c, ce.Code, ce.Msg)
		return
	}
	resp.Fail(c, 1104, "操作失败: "+err.Error())
}

// Create POST /api/admin/channels
func (h *ChannelHandler) Create(c *gin.Context) {
	var req dto.ChannelSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	id, err := h.svc.Create(req)
	if err != nil {
		failFromChannelErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Update PUT /api/admin/channels/:id
func (h *ChannelHandler) Update(c *gin.Context) {
	id := channelIDParam(c)
	if id == 0 {
		resp.Fail(c, 400, "通道 ID 不合法")
		return
	}
	var req dto.ChannelSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(id, req); err != nil {
		failFromChannelErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Delete DELETE /api/admin/channels/:id
func (h *ChannelHandler) Delete(c *gin.Context) {
	id := channelIDParam(c)
	if id == 0 {
		resp.Fail(c, 400, "通道 ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromChannelErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// SetStatus PUT /api/admin/channels/:id/status
func (h *ChannelHandler) SetStatus(c *gin.Context) {
	id := channelIDParam(c)
	if id == 0 {
		resp.Fail(c, 400, "通道 ID 不合法")
		return
	}
	var req dto.ChannelStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(id, req.Status); err != nil {
		failFromChannelErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": req.Status})
}

// GetConfig GET /api/admin/channels/:id/config
func (h *ChannelHandler) GetConfig(c *gin.Context) {
	id := channelIDParam(c)
	if id == 0 {
		resp.Fail(c, 400, "通道 ID 不合法")
		return
	}
	view, err := h.svc.GetConfig(id)
	if err != nil {
		failFromChannelErr(c, err)
		return
	}
	resp.OK(c, view)
}

// SaveConfig PUT /api/admin/channels/:id/config
func (h *ChannelHandler) SaveConfig(c *gin.Context) {
	id := channelIDParam(c)
	if id == 0 {
		resp.Fail(c, 400, "通道 ID 不合法")
		return
	}
	var req dto.ChannelConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SaveConfig(id, req.Config); err != nil {
		failFromChannelErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// OrderHandler 订单相关接口。
type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// List GET /api/admin/orders
func (h *OrderHandler) List(c *gin.Context) {
	var q dto.OrderQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1002, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}
