package handler

import (
	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// RoleHandler 后台角色管理（RBAC 增强，我方独有）。
type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// List GET /api/admin/roles 角色列表（首次访问播种内置角色）。
func (h *RoleHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"list": list})
}

// Create POST /api/admin/roles 新增角色。
func (h *RoleHandler) Create(c *gin.Context) {
	var req dto.RoleSaveReq
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

// Update PUT /api/admin/roles/:id 编辑角色。
func (h *RoleHandler) Update(c *gin.Context) {
	id, ok := intIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.RoleSaveReq
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

// Delete DELETE /api/admin/roles/:id 删除角色。
func (h *RoleHandler) Delete(c *gin.Context) {
	id, ok := intIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(uint(id)); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"id": id})
}
