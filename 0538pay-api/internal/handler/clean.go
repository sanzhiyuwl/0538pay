package handler

import (
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// CleanHandler 数据清理（对齐 epay admin/clean.php）。高风险破坏性操作。
type CleanHandler struct {
	svc *service.CleanService
}

func NewCleanHandler(svc *service.CleanService) *CleanHandler {
	return &CleanHandler{svc: svc}
}

// Clean POST /api/admin/clean 按类型删除 N 天前记录
func (h *CleanHandler) Clean(c *gin.Context) {
	var req struct {
		Target string `json:"target" binding:"required"`
		Days   int    `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	n, err := h.svc.Clean(req.Target, req.Days)
	if err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"deleted": n})
}
