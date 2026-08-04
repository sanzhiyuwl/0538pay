package handler

import (
	"errors"
	"strconv"

	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// ChannelControlHandler 子商户管控（风控第二段）后台接口。
// 仅后台 /admin：风控总览页读列表/概览、单个/批量刷新微信管控状态。
type ChannelControlHandler struct {
	svc *service.ChannelControlService
}

func NewChannelControlHandler(svc *service.ChannelControlService) *ChannelControlHandler {
	return &ChannelControlHandler{svc: svc}
}

func failCC(c *gin.Context, err error) {
	var ce *service.ChannelControlError
	if errors.As(err, &ce) {
		resp.Fail(c, 1121, ce.Msg)
		return
	}
	resp.Fail(c, 1121, "操作失败: "+err.Error())
}

// List GET /api/admin/channel-controls 风控总览（概览 + 已开通子商户管控列表，读本地快照不现查微信）。
func (h *ChannelControlHandler) List(c *gin.Context) {
	res, err := h.svc.List(c.Request.Context())
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// Refresh POST /api/admin/channel-controls/:id/refresh 单个商户刷新管控状态（现查微信落快照）。
func (h *ChannelControlHandler) Refresh(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	res, err := h.svc.RefreshOne(c.Request.Context(), uint(id))
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// RefreshAll POST /api/admin/channel-controls/refresh-all 批量刷新全部已开通商户（串行限速）。
func (h *ChannelControlHandler) RefreshAll(c *gin.Context) {
	res, err := h.svc.RefreshAll(c.Request.Context())
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// Flows GET /api/admin/channel-controls/:id/flows 某进件单的管控流水时间线（风控第三段处置/订阅回调）。
func (h *ChannelControlHandler) Flows(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	res, err := h.svc.ListFlows(uint(id))
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}
