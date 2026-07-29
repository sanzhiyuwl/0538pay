package handler

import (
	"encoding/csv"
	"strconv"

	"github.com/epvia/api/internal/dto"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// OpLogHandler 商户操作日志查询/导出（我方独有安全审计）。
type OpLogHandler struct {
	svc *service.OpLogService
}

func NewOpLogHandler(svc *service.OpLogService) *OpLogHandler {
	return &OpLogHandler{svc: svc}
}

// List GET /api/admin/oplogs/merchant 商户操作日志列表（多维筛选 + 分页）。
func (h *OpLogHandler) List(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.List(q)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// Options GET /api/admin/oplogs/merchant/options 动作下拉选项（前端筛选用）。
func (h *OpLogHandler) Options(c *gin.Context) {
	resp.OK(c, gin.H{"actions": h.svc.OpActionOptions()})
}

// Export GET /api/admin/oplogs/merchant/export 按筛选流式导出 CSV（UTF-8 BOM）。
func (h *OpLogHandler) Export(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	rows, err := h.svc.ExportRows(q)
	if err != nil {
		resp.Fail(c, 1102, "导出失败: "+err.Error())
		return
	}
	writeOpLogCSV(c, "merchant_oplogs.csv", "商户号", rows)
}

// ===== 管理端操作日志（scope=admin，复用同一 svc/表）=====

// AdminList GET /api/admin/oplogs/admin 管理端操作日志列表（多维筛选 + 分页）。
func (h *OpLogHandler) AdminList(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.AdminList(q)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// AdminOptions GET /api/admin/oplogs/admin/options 管理端动作下拉选项（已落库去重）。
func (h *OpLogHandler) AdminOptions(c *gin.Context) {
	resp.OK(c, gin.H{"actions": h.svc.AdminOpActionOptions()})
}

// AdminExport GET /api/admin/oplogs/admin/export 管理端操作日志导出 CSV（UTF-8 BOM）。
func (h *OpLogHandler) AdminExport(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	rows, err := h.svc.AdminExportRows(q)
	if err != nil {
		resp.Fail(c, 1102, "导出失败: "+err.Error())
		return
	}
	writeOpLogCSV(c, "admin_oplogs.csv", "管理员ID", rows)
}

// ===== 代理端操作日志（scope=agent，复用同一 svc/表；平台在控制台查看）=====

// AgentList GET /api/console/agent-oplogs 代理端操作日志列表（多维筛选 + 分页）。
func (h *OpLogHandler) AgentList(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.AgentList(q)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// AgentOptions GET /api/console/agent-oplogs/options 代理端动作下拉选项。
func (h *OpLogHandler) AgentOptions(c *gin.Context) {
	resp.OK(c, gin.H{"actions": h.svc.AgentOpActionOptions()})
}

// AgentExport GET /api/console/agent-oplogs/export 代理端操作日志导出 CSV（UTF-8 BOM）。
func (h *OpLogHandler) AgentExport(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	rows, err := h.svc.AgentExportRows(q)
	if err != nil {
		resp.Fail(c, 1102, "导出失败: "+err.Error())
		return
	}
	writeOpLogCSV(c, "agent_oplogs.csv", "代理ID", rows)
}

// ===== 控制台管理日志（scope=console，复用同一 svc/表；平台运营在控制台查看）=====

// ConsoleList GET /api/console/oplogs 控制台管理日志列表（多维筛选 + 分页）。
func (h *OpLogHandler) ConsoleList(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.ConsoleList(q)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// ConsoleOptions GET /api/console/oplogs/options 控制台动作下拉选项（已落库去重）。
func (h *OpLogHandler) ConsoleOptions(c *gin.Context) {
	resp.OK(c, gin.H{"actions": h.svc.ConsoleOpActionOptions()})
}

// ConsoleExport GET /api/console/oplogs/export 控制台管理日志导出 CSV（UTF-8 BOM）。
func (h *OpLogHandler) ConsoleExport(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	rows, err := h.svc.ConsoleExportRows(q)
	if err != nil {
		resp.Fail(c, 1102, "导出失败: "+err.Error())
		return
	}
	writeOpLogCSV(c, "console_oplogs.csv", "运营ID", rows)
}

// ===== 系统运维日志（scope=system，复用同一 svc/表；平台在控制台查看）=====

// SystemList GET /api/console/system-oplogs 系统运维日志列表（多维筛选 + 分页）。
func (h *OpLogHandler) SystemList(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	list, total, err := h.svc.SystemList(q)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, q.Page, q.PageSize)
}

// SystemOptions GET /api/console/system-oplogs/options 系统事件动作下拉选项。
func (h *OpLogHandler) SystemOptions(c *gin.Context) {
	resp.OK(c, gin.H{"actions": h.svc.SystemOpActionOptions()})
}

// SystemExport GET /api/console/system-oplogs/export 系统运维日志导出 CSV（UTF-8 BOM）。
func (h *OpLogHandler) SystemExport(c *gin.Context) {
	var q dto.OpLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	rows, err := h.svc.SystemExportRows(q)
	if err != nil {
		resp.Fail(c, 1102, "导出失败: "+err.Error())
		return
	}
	writeOpLogCSV(c, "system_oplogs.csv", "来源", rows)
}

// writeOpLogCSV 流式写操作日志 CSV（UTF-8 BOM）。idCol 为第二列表头（商户号/管理员ID）。
func writeOpLogCSV(c *gin.Context, filename, idCol string, rows []dto.OpLogView) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	_, _ = c.Writer.WriteString("\xEF\xBB\xBF") // UTF-8 BOM，Excel 正确识别中文
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"ID", idCol, "操作者", "操作", "分类", "级别", "对象", "结果", "IP", "时间"})
	for i := range rows {
		r := &rows[i]
		_ = w.Write([]string{
			strconv.FormatUint(uint64(r.ID), 10), strconv.FormatUint(uint64(r.UID), 10),
			r.Operator, r.Action, r.Category, r.Level, r.Target, r.Result, r.IP, r.Date,
		})
	}
	w.Flush()
}
