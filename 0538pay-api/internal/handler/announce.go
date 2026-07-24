package handler

import (
	"strconv"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// AnnounceHandler 网站公告管理（对齐 epay admin/gonggao.php）+ 公开读取。
type AnnounceHandler struct {
	svc *service.AnnounceService
}

func NewAnnounceHandler(svc *service.AnnounceService) *AnnounceHandler {
	return &AnnounceHandler{svc: svc}
}

// List GET /api/admin/announces 后台全部公告
func (h *AnnounceHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// Public GET /api/site/announces 官网/公开读取展示中的公告
func (h *AnnounceHandler) Public(c *gin.Context) {
	list, err := h.svc.ListVisible()
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// Create POST /api/admin/announces 新增公告
func (h *AnnounceHandler) Create(c *gin.Context) {
	var req dto.AnnounceSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Create(req); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Update PUT /api/admin/announces/:id 编辑公告
func (h *AnnounceHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req dto.AnnounceSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Update(uint(id), req); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// SetStatus PUT /api/admin/announces/:id/status 显隐切换
func (h *AnnounceHandler) SetStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.SetStatus(uint(id), req.Status); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// Delete DELETE /api/admin/announces/:id 删除公告
func (h *AnnounceHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.svc.Delete(uint(id)); err != nil {
		resp.Fail(c, 1102, "删除失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"id": id})
}
