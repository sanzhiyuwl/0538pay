package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// SettleHandler 结算相关接口。
type SettleHandler struct {
	svc *service.SettleService
}

func NewSettleHandler(svc *service.SettleService) *SettleHandler {
	return &SettleHandler{svc: svc}
}

// failFromSettleErr 把 service.SettleError 的业务码透传，其它错误按通用失败处理。
func failFromSettleErr(c *gin.Context, err error) {
	var se *service.SettleError
	if errors.As(err, &se) {
		resp.Fail(c, se.Code, se.Msg)
		return
	}
	resp.Fail(c, 1105, "操作失败: "+err.Error())
}

func settleIDParam(c *gin.Context) uint {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// List GET /api/admin/settles 结算明细列表
func (h *SettleHandler) List(c *gin.Context) {
	var q dto.SettleQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1105, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Batches GET /api/admin/settle/batches 结算批次列表
func (h *SettleHandler) Batches(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	list, total, err := h.svc.ListBatches(page, pageSize)
	if err != nil {
		resp.Fail(c, 1105, "查询失败: "+err.Error())
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

// CreateBatch POST /api/admin/settle/batch 生成结算批次（收当前所有待结算记录）
func (h *SettleHandler) CreateBatch(c *gin.Context) {
	batch, count, err := h.svc.CreateBatch(time.Now())
	if err != nil {
		failFromSettleErr(c, err)
		return
	}
	resp.OK(c, gin.H{"batch": batch, "count": count})
}

// CompleteBatch POST /api/admin/settle/batch/:batch/complete 批次一键完成
func (h *SettleHandler) CompleteBatch(c *gin.Context) {
	batch := c.Param("batch")
	if batch == "" {
		resp.Fail(c, 400, "批次号不能为空")
		return
	}
	n, err := h.svc.CompleteBatch(batch)
	if err != nil {
		failFromSettleErr(c, err)
		return
	}
	resp.OK(c, gin.H{"affected": n})
}

// SetStatus PUT /api/admin/settles/:id/status 变更单条结算状态（含删除退回）
func (h *SettleHandler) SetStatus(c *gin.Context) {
	id := settleIDParam(c)
	if id == 0 {
		resp.Fail(c, 400, "无效的记录ID")
		return
	}
	var req dto.SettleStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(id, req); err != nil {
		failFromSettleErr(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "status": req.Status})
}
