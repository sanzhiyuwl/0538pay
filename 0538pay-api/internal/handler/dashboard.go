package handler

import (
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// DashboardHandler 后台仪表盘聚合接口。
type DashboardHandler struct {
	svc *service.DashboardService
}

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// Overview GET /api/admin/dashboard 全平台仪表盘聚合
func (h *DashboardHandler) Overview(c *gin.Context) {
	data, err := h.svc.Overview()
	if err != nil {
		resp.Fail(c, 1102, "加载仪表盘失败: "+err.Error())
		return
	}
	resp.OK(c, data)
}
