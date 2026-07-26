package handler

import (
	"encoding/csv"
	"strconv"

	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// billingMonths 解析并夹取 months 查询参数（默认 0=服务层用默认 6，上限 24）。
func billingMonths(c *gin.Context) int {
	months := 0
	if v := c.Query("months"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			months = n
		}
	}
	if months < 0 {
		months = 0
	}
	if months > 24 {
		months = 24
	}
	return months
}

// BillingHandler 平台月度对账账单接口（我方独有细化项，epay 无）。
type BillingHandler struct {
	svc *service.BillingService
}

func NewBillingHandler(svc *service.BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

// List GET /api/admin/billing 平台近 N 期月度账单聚合。
// 可选 query: months(默认 6，1~24)。
func (h *BillingHandler) List(c *gin.Context) {
	data, err := h.svc.Billing(billingMonths(c))
	if err != nil {
		resp.Fail(c, 1103, "加载账单失败: "+err.Error())
		return
	}
	resp.OK(c, data)
}

// Export GET /api/admin/billing/export 导出账单 CSV（UTF-8 BOM + 汇总列）。
func (h *BillingHandler) Export(c *gin.Context) {
	data, err := h.svc.Billing(billingMonths(c))
	if err != nil {
		resp.Fail(c, 1103, "加载账单失败: "+err.Error())
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=billing.csv")
	_, _ = c.Writer.WriteString("\xEF\xBB\xBF") // UTF-8 BOM，Excel 正确识别中文
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"账期", "已支付订单", "收入合计", "支出合计", "净收入", "状态"})
	for i := range data.Bills {
		b := &data.Bills[i]
		status := "已归档"
		if b.Status == 0 {
			status = "进行中"
		}
		_ = w.Write([]string{
			b.Period, strconv.FormatInt(b.Orders, 10),
			b.Income, b.Expense, b.Net, status,
		})
	}
	w.Flush()
}
