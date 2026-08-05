package handler

import (
	"strconv"

	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// MerchantChannelControlHandler 子商户管控（风控第二段）商户端「业务受限」面板。
// 只读：只看自己名下已开通子商户的管控状态 + 解脱指引，不开放批量刷新/解脱代办（平台运维能力，仅 admin）。
type MerchantChannelControlHandler struct {
	svc *service.ChannelControlService
}

func NewMerchantChannelControlHandler(svc *service.ChannelControlService) *MerchantChannelControlHandler {
	return &MerchantChannelControlHandler{svc: svc}
}

// MyList GET /api/merchant/channel-controls 我的业务受限面板（强制按登录商户 uid 隔离）。
func (h *MerchantChannelControlHandler) MyList(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	res, err := h.svc.ListForMerchant(c.Request.Context(), uid)
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}

// MyGet GET /api/merchant/channel-controls/:id 我名下单个进件单的管控快照（进件详情「业务受限」
// 就地快照，两处不重复造轮子：与本面板列表共用同一份快照数据）。归属校验：enroll_id 须属登录商户。
func (h *MerchantChannelControlHandler) MyGet(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	v, err := h.svc.GetByEnrollIDForMerchant(uint(id), uid)
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, gin.H{"view": v})
}

// MyRefresh POST /api/merchant/channel-controls/:id/refresh 刷新自己名下单个子商户的管控状态。
func (h *MerchantChannelControlHandler) MyRefresh(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 1121, "进件单号无效")
		return
	}
	res, err := h.svc.RefreshOneForMerchant(c.Request.Context(), uint(id), uid)
	if err != nil {
		failCC(c, err)
		return
	}
	resp.OK(c, res)
}
