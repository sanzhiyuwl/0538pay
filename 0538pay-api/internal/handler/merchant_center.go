package handler

import (
	"errors"
	"strconv"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/middleware"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// MerchantCenterHandler 商户中心业务接口（工作台/订单/流水/结算/提现/退款）。
type MerchantCenterHandler struct {
	svc      *service.MerchantCenterService
	orderSvc *service.OrderService
	recSvc   *service.RecordService
}

func NewMerchantCenterHandler(
	svc *service.MerchantCenterService,
	orderSvc *service.OrderService,
	recSvc *service.RecordService,
) *MerchantCenterHandler {
	return &MerchantCenterHandler{svc: svc, orderSvc: orderSvc, recSvc: recSvc}
}

// currentUID 从鉴权上下文取当前商户号（middleware 注入，不信任前端传参）。
func currentUID(c *gin.Context) (uint, bool) {
	v, _ := c.Get(middleware.CtxUID)
	id, ok := v.(uint)
	return id, ok && id != 0
}

func failMC(c *gin.Context, err error) {
	var ae *service.MerchantAuthError
	if errors.As(err, &ae) {
		resp.Fail(c, 1102, ae.Msg)
		return
	}
	resp.Fail(c, 1102, "操作失败: "+err.Error())
}

// Dashboard GET /api/merchant/dashboard 工作台聚合
func (h *MerchantCenterHandler) Dashboard(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	data, err := h.svc.Dashboard(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, data)
}

// Orders GET /api/merchant/orders 商户订单查询（分页）
func (h *MerchantCenterHandler) Orders(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var q dto.OrderQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.orderSvc.ListByMerchant(uid, q)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Records GET /api/merchant/records 资金流水（分页）
func (h *MerchantCenterHandler) Records(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var q dto.RecordQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.recSvc.ListByMerchant(uid, q)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Settles GET /api/merchant/settles 结算记录（分页）
func (h *MerchantCenterHandler) Settles(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	var status *int
	if s := c.Query("status"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			status = &n
		}
	}
	list, total, err := h.svc.Settles(uid, status, page, pageSize)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	resp.Page(c, list, total, page, pageSize)
}

// ApplyInfo GET /api/merchant/apply/info 申请提现页信息
func (h *MerchantCenterHandler) ApplyInfo(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	info, err := h.svc.ApplyInfo(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, info)
}

// Apply POST /api/merchant/apply 申请提现
func (h *MerchantCenterHandler) Apply(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.ApplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Apply(uid, req); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Refund POST /api/merchant/order/refund 订单退款（全额）
func (h *MerchantCenterHandler) Refund(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.RefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Refund(uid, req.TradeNo); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": req.TradeNo, "status": 2})
}

// Renotify POST /api/merchant/order/notify 重新通知（补单）
func (h *MerchantCenterHandler) Renotify(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.RefundReq // 复用 {trade_no}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Renotify(uid, req.TradeNo); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": req.TradeNo})
}
