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
	out, err := h.svc.Login(req, c.ClientIP())
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

// merchantUIDParam 解析路径 :uid，失败返回 0。
func merchantUIDParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("uid"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// failFromMerchantErr 透传 MerchantError 的业务码。
func failFromMerchantErr(c *gin.Context, err error) {
	var me *service.MerchantError
	if errors.As(err, &me) {
		resp.Fail(c, me.Code, me.Msg)
		return
	}
	resp.Fail(c, 1003, "操作失败: "+err.Error())
}

// Create POST /api/admin/merchants 添加商户。
func (h *MerchantHandler) Create(c *gin.Context) {
	var req dto.MerchantCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	uid, key, err := h.svc.Create(req)
	if err != nil {
		failFromMerchantErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": uid, "key": key})
}

// Update PUT /api/admin/merchants/:uid 编辑商户。
func (h *MerchantHandler) Update(c *gin.Context) {
	uid := merchantUIDParam(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	var req dto.MerchantEditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(uid, req); err != nil {
		failFromMerchantErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": uid})
}

// Recharge POST /api/admin/merchants/:uid/recharge 余额充值/扣除。
func (h *MerchantHandler) Recharge(c *gin.Context) {
	uid := merchantUIDParam(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	var req dto.MerchantRechargeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Recharge(uid, req); err != nil {
		failFromMerchantErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": uid})
}

// SetGroup PUT /api/admin/merchants/:uid/group 修改用户组 + 有效期。
func (h *MerchantHandler) SetGroup(c *gin.Context) {
	uid := merchantUIDParam(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	var req dto.MerchantGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetGroup(uid, req); err != nil {
		failFromMerchantErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": uid, "gid": req.GID})
}

// SetStatus PUT /api/admin/merchants/:uid/status 权限/状态切换。
func (h *MerchantHandler) SetStatus(c *gin.Context) {
	uid := merchantUIDParam(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	var req dto.MerchantSetStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(uid, req); err != nil {
		failFromMerchantErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": uid})
}

// ResetKey POST /api/admin/merchants/:uid/resetkey 重置通信密钥。
func (h *MerchantHandler) ResetKey(c *gin.Context) {
	uid := merchantUIDParam(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	key, err := h.svc.ResetKey(uid)
	if err != nil {
		failFromMerchantErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": uid, "key": key})
}

// Delete DELETE /api/admin/merchants/:uid 删除商户。
func (h *MerchantHandler) Delete(c *gin.Context) {
	uid := merchantUIDParam(c)
	if uid == 0 {
		resp.Fail(c, 400, "商户号不合法")
		return
	}
	if err := h.svc.Delete(uid); err != nil {
		failFromMerchantErr(c, err)
		return
	}
	resp.OK(c, gin.H{"uid": uid})
}

// GroupHandler 用户组管理接口。
type GroupHandler struct {
	svc *service.GroupService
}

func NewGroupHandler(svc *service.GroupService) *GroupHandler {
	return &GroupHandler{svc: svc}
}

func failFromGroupErr(c *gin.Context, err error) {
	var ge *service.GroupError
	if errors.As(err, &ge) {
		resp.Fail(c, ge.Code, ge.Msg)
		return
	}
	resp.Fail(c, 1011, "操作失败: "+err.Error())
}

// groupGIDParam 解析路径 :gid，失败返回 -1（区别于合法的 gid=0）。
func groupGIDParam(c *gin.Context) int {
	id, err := strconv.Atoi(c.Param("gid"))
	if err != nil {
		return -1
	}
	return id
}

// List GET /api/admin/groups 用户组列表。
func (h *GroupHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		resp.Fail(c, 1011, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// Create POST /api/admin/groups 新增用户组。
func (h *GroupHandler) Create(c *gin.Context) {
	var req dto.GroupSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	gid, err := h.svc.Create(req)
	if err != nil {
		failFromGroupErr(c, err)
		return
	}
	resp.OK(c, gin.H{"gid": gid})
}

// Update PUT /api/admin/groups/:gid 编辑用户组。
func (h *GroupHandler) Update(c *gin.Context) {
	gid := groupGIDParam(c)
	if gid < 0 {
		resp.Fail(c, 400, "用户组 ID 不合法")
		return
	}
	var req dto.GroupSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(gid, req); err != nil {
		failFromGroupErr(c, err)
		return
	}
	resp.OK(c, gin.H{"gid": gid})
}

// SetBuy PUT /api/admin/groups/:gid/buy 上/下架。
func (h *GroupHandler) SetBuy(c *gin.Context) {
	gid := groupGIDParam(c)
	if gid < 0 {
		resp.Fail(c, 400, "用户组 ID 不合法")
		return
	}
	var req dto.GroupBuyStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetBuy(gid, req.IsBuy); err != nil {
		failFromGroupErr(c, err)
		return
	}
	resp.OK(c, gin.H{"gid": gid, "isbuy": req.IsBuy})
}

// Delete DELETE /api/admin/groups/:gid 删除用户组。
func (h *GroupHandler) Delete(c *gin.Context) {
	gid := groupGIDParam(c)
	if gid < 0 {
		resp.Fail(c, 400, "用户组 ID 不合法")
		return
	}
	if err := h.svc.Delete(gid); err != nil {
		failFromGroupErr(c, err)
		return
	}
	resp.OK(c, gin.H{"gid": gid})
}

// GetAssigns GET /api/admin/groups/:gid/assigns 读取该组的通道分配。
func (h *GroupHandler) GetAssigns(c *gin.Context) {
	gid := groupGIDParam(c)
	if gid < 0 {
		resp.Fail(c, 400, "用户组 ID 不合法")
		return
	}
	list, err := h.svc.GetAssigns(gid)
	if err != nil {
		failFromGroupErr(c, err)
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// SaveAssigns PUT /api/admin/groups/:gid/assigns 保存该组的通道分配。
func (h *GroupHandler) SaveAssigns(c *gin.Context) {
	gid := groupGIDParam(c)
	if gid < 0 {
		resp.Fail(c, 400, "用户组 ID 不合法")
		return
	}
	var req dto.GroupAssignSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SaveAssigns(gid, req); err != nil {
		failFromGroupErr(c, err)
		return
	}
	resp.OK(c, gin.H{"gid": gid})
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

// RecordHandler 后台资金流水接口（列表 + 统计，对齐 epay record.php）。
type RecordHandler struct {
	svc *service.RecordService
}

func NewRecordHandler(svc *service.RecordService) *RecordHandler {
	return &RecordHandler{svc: svc}
}

// List GET /api/admin/records 后台资金流水（分页 + 多条件筛选）。
func (h *RecordHandler) List(c *gin.Context) {
	var q dto.RecordQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1005, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Stats GET /api/admin/records/stats 当前筛选条件下的资金明细统计。
func (h *RecordHandler) Stats(c *gin.Context) {
	var q dto.RecordQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	stats, err := h.svc.Stats(q)
	if err != nil {
		resp.Fail(c, 1005, "统计失败: "+err.Error())
		return
	}
	resp.OK(c, stats)
}

// TransferHandler 代付/转账接口（后台 + 商户端，对齐 epay transfer）。
type TransferHandler struct {
	svc *service.TransferService
}

func NewTransferHandler(svc *service.TransferService) *TransferHandler {
	return &TransferHandler{svc: svc}
}

// failFromTransferErr 透传 TransferError 的业务码。
func failFromTransferErr(c *gin.Context, err error) {
	var te *service.TransferError
	if errors.As(err, &te) {
		resp.Fail(c, te.Code, te.Msg)
		return
	}
	resp.Fail(c, 1106, "操作失败: "+err.Error())
}

// List GET /api/admin/transfers 后台代付列表（分页 + 筛选）。
func (h *TransferHandler) List(c *gin.Context) {
	var q dto.TransferQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1006, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Stats GET /api/admin/transfers/stats 后台代付概况统计。
func (h *TransferHandler) Stats(c *gin.Context) {
	var q dto.TransferQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	stats, err := h.svc.Stats(q)
	if err != nil {
		resp.Fail(c, 1006, "统计失败: "+err.Error())
		return
	}
	resp.OK(c, stats)
}

// Create POST /api/admin/transfers 后台管理员发起代付（uid=0，免费不扣款）。
func (h *TransferHandler) Create(c *gin.Context) {
	adminID, _ := c.Get(middleware.CtxUID)
	id, ok := adminID.(uint)
	if !ok || id == 0 {
		resp.Fail(c, 401, "登录态异常")
		return
	}
	var req dto.TransferCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	bizNo, err := h.svc.CreateByAdmin(id, req)
	if err != nil {
		failFromTransferErr(c, err)
		return
	}
	resp.OK(c, gin.H{"biz_no": bizNo})
}

// SetStatus PUT /api/admin/transfers/:biz/status 手动改状态（不动资金）。
func (h *TransferHandler) SetStatus(c *gin.Context) {
	var req dto.TransferStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(c.Param("biz"), req); err != nil {
		failFromTransferErr(c, err)
		return
	}
	resp.OK(c, gin.H{"biz_no": c.Param("biz"), "status": req.Status})
}

// Refund POST /api/admin/transfers/:biz/refund 退回（仅处理中，退回商户扣款）。
func (h *TransferHandler) Refund(c *gin.Context) {
	if err := h.svc.Refund(c.Param("biz")); err != nil {
		failFromTransferErr(c, err)
		return
	}
	resp.OK(c, gin.H{"biz_no": c.Param("biz")})
}

// Delete DELETE /api/admin/transfers/:biz 删除记录（不退款）。
func (h *TransferHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Param("biz")); err != nil {
		failFromTransferErr(c, err)
		return
	}
	resp.OK(c, gin.H{"biz_no": c.Param("biz")})
}

// ProfitHandler 分账接口（列表 + 统计 + 状态操作，对齐 epay profitsharing）。
type ProfitHandler struct {
	svc *service.ProfitService
}

func NewProfitHandler(svc *service.ProfitService) *ProfitHandler {
	return &ProfitHandler{svc: svc}
}

func failFromProfitErr(c *gin.Context, err error) {
	var pe *service.ProfitError
	if errors.As(err, &pe) {
		resp.Fail(c, pe.Code, pe.Msg)
		return
	}
	resp.Fail(c, 1107, "操作失败: "+err.Error())
}

// List GET /api/admin/ps/orders 分账订单列表（分页 + 筛选）。
func (h *ProfitHandler) List(c *gin.Context) {
	var q dto.PsOrderQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1007, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Stats GET /api/admin/ps/orders/stats 分账统计概况。
func (h *ProfitHandler) Stats(c *gin.Context) {
	var q dto.PsOrderQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	stats, err := h.svc.Stats(q)
	if err != nil {
		resp.Fail(c, 1007, "统计失败: "+err.Error())
		return
	}
	resp.OK(c, stats)
}

// Operate POST /api/admin/ps/orders/:id/op 分账状态操作（submit/query/return/cancel/editmoney/delete）。
func (h *ProfitHandler) Operate(c *gin.Context) {
	id := channelIDParam(c) // 复用路径 :id 解析
	if id == 0 {
		resp.Fail(c, 400, "分账记录 ID 不合法")
		return
	}
	var req dto.PsStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Operate(id, req); err != nil {
		failFromProfitErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "action": req.Action})
}

// ===== 风控 / 黑名单 / 域名（C4）=====

func failFromRiskErr(c *gin.Context, err error) {
	var re *service.RiskError
	if errors.As(err, &re) {
		resp.Fail(c, re.Code, re.Msg)
		return
	}
	resp.Fail(c, 1108, "操作失败: "+err.Error())
}

// idParam 解析路径 :id，失败返回 0。
func idParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// RiskHandler 风控记录（只读）。
type RiskHandler struct{ svc *service.RiskService }

func NewRiskHandler(svc *service.RiskService) *RiskHandler { return &RiskHandler{svc: svc} }

// List GET /api/admin/risks 风控记录列表（只读）。
func (h *RiskHandler) List(c *gin.Context) {
	var q dto.RiskQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1008, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// BlacklistHandler 黑名单 CRUD。
type BlacklistHandler struct{ svc *service.BlacklistService }

func NewBlacklistHandler(svc *service.BlacklistService) *BlacklistHandler {
	return &BlacklistHandler{svc: svc}
}

// List GET /api/admin/blacklist
func (h *BlacklistHandler) List(c *gin.Context) {
	var q dto.BlacklistQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1008, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Stats GET /api/admin/blacklist/stats
func (h *BlacklistHandler) Stats(c *gin.Context) {
	total, account, ip, permanent, err := h.svc.Stats()
	if err != nil {
		resp.Fail(c, 1008, "统计失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"total": total, "account": account, "ip": ip, "permanent": permanent})
}

// Add POST /api/admin/blacklist
func (h *BlacklistHandler) Add(c *gin.Context) {
	var req dto.BlacklistAddReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Add(req); err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Delete DELETE /api/admin/blacklist/:id
func (h *BlacklistHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// BatchDelete POST /api/admin/blacklist/batch-delete
func (h *BlacklistHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	n, err := h.svc.BatchDelete(req.IDs)
	if err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"deleted": n})
}

// DomainHandler 授权域名 CRUD + 审核。
type DomainHandler struct{ svc *service.DomainService }

func NewDomainHandler(svc *service.DomainService) *DomainHandler {
	return &DomainHandler{svc: svc}
}

// List GET /api/admin/domains
func (h *DomainHandler) List(c *gin.Context) {
	var q dto.DomainQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1008, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Stats GET /api/admin/domains/stats
func (h *DomainHandler) Stats(c *gin.Context) {
	total, pending, normal, rejected, err := h.svc.Stats()
	if err != nil {
		resp.Fail(c, 1008, "统计失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"total": total, "pending": pending, "normal": normal, "rejected": rejected})
}

// Add POST /api/admin/domains
func (h *DomainHandler) Add(c *gin.Context) {
	var req dto.DomainAddReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Add(req); err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// SetStatus PUT /api/admin/domains/:id/status
func (h *DomainHandler) SetStatus(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "ID 不合法")
		return
	}
	var req dto.DomainStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(id, req.Status); err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": req.Status})
}

// Delete DELETE /api/admin/domains/:id
func (h *DomainHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// BatchOp POST /api/admin/domains/batch  批量通过(1)/拒绝(2)/删除(3)
func (h *DomainHandler) BatchOp(c *gin.Context) {
	var req struct {
		IDs    []uint `json:"ids"`
		Status int8   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	n, err := h.svc.BatchOp(req.IDs, req.Status)
	if err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"affected": n})
}

// ===== 统计 / 日志 / 邀请码（C5）=====

// StatHandler 商户支付统计（只读聚合）。
type StatHandler struct{ svc *service.StatService }

func NewStatHandler(svc *service.StatService) *StatHandler { return &StatHandler{svc: svc} }

// PayStat GET /api/admin/stat/pay 商户支付统计交叉透视表。
func (h *StatHandler) PayStat(c *gin.Context) {
	var q dto.StatQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	res, err := h.svc.PayStat(q)
	if err != nil {
		resp.Fail(c, 1009, "统计失败: "+err.Error())
		return
	}
	resp.OK(c, res)
}

// LogHandler 登录日志（只读）。
type LogHandler struct{ svc *service.LogService }

func NewLogHandler(svc *service.LogService) *LogHandler { return &LogHandler{svc: svc} }

// List GET /api/admin/logs 登录日志列表。
func (h *LogHandler) List(c *gin.Context) {
	var q dto.LogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1009, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// InviteHandler 邀请码 CRUD。
type InviteHandler struct{ svc *service.InviteService }

func NewInviteHandler(svc *service.InviteService) *InviteHandler { return &InviteHandler{svc: svc} }

// List GET /api/admin/invitecodes
func (h *InviteHandler) List(c *gin.Context) {
	var q dto.InviteQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1009, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Generate POST /api/admin/invitecodes/generate
func (h *InviteHandler) Generate(c *gin.Context) {
	var req dto.InviteGenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	n, err := h.svc.Generate(req.Num)
	if err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"generated": n})
}

// Delete DELETE /api/admin/invitecodes/:id
func (h *InviteHandler) Delete(c *gin.Context) {
	id := idParam(c)
	if id == 0 {
		resp.Fail(c, 400, "ID 不合法")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Clear POST /api/admin/invitecodes/clear  body {mode: "all"|"used"}
func (h *InviteHandler) Clear(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	var n int64
	var err error
	if req.Mode == "used" {
		n, err = h.svc.ClearUsed()
	} else {
		n, err = h.svc.ClearAll()
	}
	if err != nil {
		failFromRiskErr(c, err)
		return
	}
	resp.OK(c, gin.H{"deleted": n})
}

// ===== 官网 CMS（自研）=====

// SiteConfigHandler 官网 CMS 内容读写。
type SiteConfigHandler struct{ svc *service.SiteConfigService }

func NewSiteConfigHandler(svc *service.SiteConfigService) *SiteConfigHandler {
	return &SiteConfigHandler{svc: svc}
}

// Get GET /api/site/config/:key 读取 CMS 文档（公开，官网读）。
// 返回 {value: <原始JSON字符串>}；无记录时 value 为空串，前端回退默认。
func (h *SiteConfigHandler) Get(c *gin.Context) {
	val, err := h.svc.Get(c.Param("key"))
	if err != nil {
		var se *service.SiteConfigError
		if errors.As(err, &se) {
			resp.Fail(c, 400, se.Msg)
			return
		}
		resp.Fail(c, 1010, "读取失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"value": val})
}

// Save PUT /api/admin/site/config/:key 保存 CMS 文档（后台鉴权写）。
func (h *SiteConfigHandler) Save(c *gin.Context) {
	var req dto.SiteConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Set(c.Param("key"), req.Value); err != nil {
		var se *service.SiteConfigError
		if errors.As(err, &se) {
			resp.Fail(c, 400, se.Msg)
			return
		}
		resp.Fail(c, 1010, "保存失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"key": c.Param("key")})
}

// ConfigHandler 系统设置读写接口。
type ConfigHandler struct {
	svc *service.ConfigService
}

func NewConfigHandler(svc *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{svc: svc}
}

// GetGroup GET /api/admin/config/:group 读取某分组配置（回填设置页）。
func (h *ConfigHandler) GetGroup(c *gin.Context) {
	kv, err := h.svc.GetGroup(c.Param("group"))
	if err != nil {
		var ce *service.ConfigError
		if errors.As(err, &ce) {
			resp.Fail(c, 400, ce.Msg)
			return
		}
		resp.Fail(c, 1012, "读取失败: "+err.Error())
		return
	}
	resp.OK(c, kv)
}

// SaveGroup PUT /api/admin/config/:group 保存某分组配置（白名单键）。
func (h *ConfigHandler) SaveGroup(c *gin.Context) {
	var kv map[string]string
	if err := c.ShouldBindJSON(&kv); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SaveGroup(c.Param("group"), kv); err != nil {
		var ce *service.ConfigError
		if errors.As(err, &ce) {
			resp.Fail(c, 400, ce.Msg)
			return
		}
		resp.Fail(c, 1012, "保存失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"group": c.Param("group")})
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

// failFromOrderErr 透传 OrderError 的业务码。
func failFromOrderErr(c *gin.Context, err error) {
	var oe *service.OrderError
	if errors.As(err, &oe) {
		resp.Fail(c, oe.Code, oe.Msg)
		return
	}
	resp.Fail(c, 1002, "操作失败: "+err.Error())
}

// tradeNoParam 取路径 :trade_no。
func tradeNoParam(c *gin.Context) string { return c.Param("trade_no") }

// SetStatus PUT /api/admin/orders/:trade_no/status 裸改状态（改未完成/已完成）。
func (h *OrderHandler) SetStatus(c *gin.Context) {
	var req dto.OrderStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(tradeNoParam(c), req.Status); err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": tradeNoParam(c), "status": req.Status})
}

// Freeze POST /api/admin/orders/:trade_no/freeze 冻结订单。
func (h *OrderHandler) Freeze(c *gin.Context) {
	if err := h.svc.Freeze(tradeNoParam(c)); err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": tradeNoParam(c)})
}

// Unfreeze POST /api/admin/orders/:trade_no/unfreeze 解冻订单。
func (h *OrderHandler) Unfreeze(c *gin.Context) {
	if err := h.svc.Unfreeze(tradeNoParam(c)); err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": tradeNoParam(c)})
}

// RefundInfo GET /api/admin/orders/:trade_no/refund-info 退款前查可退金额。
func (h *OrderHandler) RefundInfo(c *gin.Context) {
	info, err := h.svc.RefundInfo(tradeNoParam(c))
	if err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, info)
}

// Refund POST /api/admin/orders/refund 退款（手动 / API 原路）。
func (h *OrderHandler) Refund(c *gin.Context) {
	adminID, _ := c.Get(middleware.CtxUID)
	id, _ := adminID.(uint)
	var req dto.OrderRefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Refund(id, req); err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": req.TradeNo})
}

// FillOrder POST /api/admin/orders/:trade_no/fill 手动补单。
func (h *OrderHandler) FillOrder(c *gin.Context) {
	if err := h.svc.FillOrder(tradeNoParam(c)); err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": tradeNoParam(c)})
}

// Renotify POST /api/admin/orders/:trade_no/notify 重新通知商户。
func (h *OrderHandler) Renotify(c *gin.Context) {
	if err := h.svc.Renotify(tradeNoParam(c)); err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": tradeNoParam(c)})
}

// Delete DELETE /api/admin/orders/:trade_no 删除订单。
func (h *OrderHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(tradeNoParam(c)); err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"trade_no": tradeNoParam(c)})
}

// Batch POST /api/admin/orders/batch 批量操作。
func (h *OrderHandler) Batch(c *gin.Context) {
	adminID, _ := c.Get(middleware.CtxUID)
	id, _ := adminID.(uint)
	var req dto.OrderBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	n, err := h.svc.Batch(id, req)
	if err != nil {
		failFromOrderErr(c, err)
		return
	}
	resp.OK(c, gin.H{"affected": n})
}
