package handler

import (
	"github.com/epvia/api/internal/scheduler"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// CronHandler 对外手动触发定时任务入口（对齐 epay cron.php），由 cronkey 保护。
type CronHandler struct {
	sch *scheduler.Scheduler
	cfg *service.ConfigService
}

func NewCronHandler(sch *scheduler.Scheduler, cfg *service.ConfigService) *CronHandler {
	return &CronHandler{sch: sch, cfg: cfg}
}

// Run GET/POST /api/cron/:task?key=xxx 手动触发一个 cron 任务。
// 对齐 epay cron.php：cronkey 为空则整个入口关闭(exit)；key 不符拒绝。
func (h *CronHandler) Run(c *gin.Context) {
	cronkey := h.cfg.Str("cronkey")
	if cronkey == "" {
		// 未配置 cronkey：对外触发入口关闭（对齐 epay empty(cronkey) exit）。
		resp.Fail(c, 403, "计划任务访问密钥未配置，对外触发入口已关闭")
		return
	}
	if c.Query("key") != cronkey {
		resp.Fail(c, 403, "计划任务访问密钥不正确")
		return
	}
	task := c.Param("task")
	if !h.sch.RunTask(task) {
		resp.Fail(c, 400, "未知的计划任务: "+task)
		return
	}
	resp.OK(c, gin.H{"task": task, "triggered": true})
}
