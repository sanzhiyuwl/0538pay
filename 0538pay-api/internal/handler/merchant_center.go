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

// MerchantCenterHandler 商户中心业务接口（工作台/订单/流水/结算/提现/退款/代付）。
type MerchantCenterHandler struct {
	svc         *service.MerchantCenterService
	orderSvc    *service.OrderService
	recSvc      *service.RecordService
	transferSvc *service.TransferService
}

func NewMerchantCenterHandler(
	svc *service.MerchantCenterService,
	orderSvc *service.OrderService,
	recSvc *service.RecordService,
	transferSvc *service.TransferService,
) *MerchantCenterHandler {
	return &MerchantCenterHandler{svc: svc, orderSvc: orderSvc, recSvc: recSvc, transferSvc: transferSvc}
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

// ApiInfo GET /api/merchant/apikey API 信息（V1 MD5 密钥）
func (h *MerchantCenterHandler) ApiInfo(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	info, err := h.svc.ApiInfo(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, info)
}

// ResetKey POST /api/merchant/apikey/reset 重置 MD5 密钥
func (h *MerchantCenterHandler) ResetKey(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	key, err := h.svc.ResetKey(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"mdkey": key})
}

// GenRSAKey POST /api/merchant/apikey/rsa 生成商户 RSA 密钥对（私钥一次性返回）
func (h *MerchantCenterHandler) GenRSAKey(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	priv, err := h.svc.GenRSAKeyPair(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"private_key": priv})
}

// SetKeyType PUT /api/merchant/apikey/keytype 设置签名模式
func (h *MerchantCenterHandler) SetKeyType(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.KeyTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetKeyType(uid, req.KeyType); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"keytype": req.KeyType})
}

// UpdateProfile PUT /api/merchant/profile 修改资料
func (h *MerchantCenterHandler) UpdateProfile(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.MerchantProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.UpdateProfile(uid, req); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// ChangePassword PUT /api/merchant/password 修改登录密码
func (h *MerchantCenterHandler) ChangePassword(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.MerchantPwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.ChangePassword(uid, req); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Transfers GET /api/merchant/transfers 商户代付记录（分页，强制限定本商户）
func (h *MerchantCenterHandler) Transfers(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var q dto.TransferQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.transferSvc.ListByMerchant(uid, q)
	if err != nil {
		resp.Fail(c, 1106, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// TransferCreate POST /api/merchant/transfer 商户发起代付（校验+计费+即时扣款）
func (h *MerchantCenterHandler) TransferCreate(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.TransferCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	bizNo, err := h.transferSvc.CreateByMerchant(uid, req)
	if err != nil {
		var te *service.TransferError
		if errors.As(err, &te) {
			resp.Fail(c, te.Code, te.Msg)
			return
		}
		resp.Fail(c, 1106, "操作失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"biz_no": bizNo})
}

// ===== 保证金 / 购买会员（D3 增值）=====

// DepositInfo GET /api/merchant/deposit 保证金页信息
func (h *MerchantCenterHandler) DepositInfo(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	info, err := h.svc.DepositInfo(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, info)
}

// DepositRecharge POST /api/merchant/deposit/recharge 充值保证金
func (h *MerchantCenterHandler) DepositRecharge(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.DepositReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.DepositRecharge(uid, req); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// DepositWithdraw POST /api/merchant/deposit/withdraw 提取保证金
func (h *MerchantCenterHandler) DepositWithdraw(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.DepositReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.DepositWithdraw(uid, req); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// CertInfo GET /api/merchant/cert 实名认证信息
func (h *MerchantCenterHandler) CertInfo(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	info, err := h.svc.CertInfo(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, info)
}

// CertSubmit POST /api/merchant/cert 提交实名认证
func (h *MerchantCenterHandler) CertSubmit(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.CertSubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.CertSubmit(uid, req); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Recharge POST /api/merchant/recharge 余额充值下单（走渠道，回调入账）
func (h *MerchantCenterHandler) Recharge(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.RechargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.svc.Recharge(uid, req)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, out)
}

// GroupPlans GET /api/merchant/groups 可购买会员套餐 + 当前会员状态
func (h *MerchantCenterHandler) GroupPlans(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	plans, err := h.svc.GroupPlans()
	if err != nil {
		failMC(c, err)
		return
	}
	current, err := h.svc.CurrentGroup(uid)
	if err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"plans": plans, "current": current})
}

// BuyGroup POST /api/merchant/groups/buy 购买会员
func (h *MerchantCenterHandler) BuyGroup(c *gin.Context) {
	uid, ok := currentUID(c)
	if !ok {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.GroupBuyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.BuyGroup(uid, req); err != nil {
		failMC(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
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
