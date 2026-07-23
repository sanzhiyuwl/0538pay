package handler

import (
	"strconv"
	"strings"

	"github.com/0538pay/api/internal/dto"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// PaypageHandler 公开聚合收款页（对齐 epay paypage/index.php，无需登录）。
type PaypageHandler struct {
	svc *service.MerchantSelfService
}

func NewPaypageHandler(svc *service.MerchantSelfService) *PaypageHandler {
	return &PaypageHandler{svc: svc}
}

// Info GET /api/paypage/info?merchant=xxx 收款页信息（收款方 + 可选支付方式）
func (h *PaypageHandler) Info(c *gin.Context) {
	merchant := c.Query("merchant")
	out, err := h.svc.PaypageInfo(merchant, deviceFromUA(c))
	if err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, out)
}

// Submit POST /api/paypage/submit 收款页下单（输金额 + 选方式 → 走收单链）
func (h *PaypageHandler) Submit(c *gin.Context) {
	var req dto.PaypageSubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	out, err := h.svc.PaypageSubmit(req)
	if err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, out)
}

// MessageHandler 后台站内信下发管理（我方新增）。
type MessageHandler struct {
	svc *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

// List GET /api/admin/messages 已下发站内信列表
func (h *MessageHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	list, total, err := h.svc.AdminList(page, pageSize)
	if err != nil {
		resp.Fail(c, 1102, "查询失败: "+err.Error())
		return
	}
	resp.Page(c, list, total, page, pageSize)
}

// Send POST /api/admin/messages 下发站内信（UID=0 全体广播）
func (h *MessageHandler) Send(c *gin.Context) {
	var req dto.MessageSendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Fail(c, 400, "参数错误: "+err.Error())
		return
	}
	if err := h.svc.Send(req); err != nil {
		resp.Fail(c, 1102, errMsg(err))
		return
	}
	resp.OK(c, gin.H{"ok": true})
}

// Delete DELETE /api/admin/messages/:id 删除站内信
func (h *MessageHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.Fail(c, 400, "参数错误")
		return
	}
	if err := h.svc.AdminDelete(uint(id)); err != nil {
		resp.Fail(c, 1102, "删除失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"id": id})
}

// deviceFromUA 从请求 User-Agent 粗判是否移动端，返回 service 侧 isMobileDevice 认得的标识
// （"mobile" 或 ""=PC），用于收银可选支付方式列表的设备过滤（B1-63，对齐 epay checkmobile()）。
func deviceFromUA(c *gin.Context) string {
	ua := strings.ToLower(c.GetHeader("User-Agent"))
	for _, kw := range []string{"android", "iphone", "ipad", "ipod", "windows phone", "mobile", "micromessenger", "alipayclient"} {
		if strings.Contains(ua, kw) {
			return "mobile"
		}
	}
	return ""
}

// errMsg 提取 service 业务错误消息（MerchantAuthError/RiskError 携带友好提示）。
func errMsg(err error) string {
	type msgErr interface{ Error() string }
	if e, ok := err.(msgErr); ok {
		return e.Error()
	}
	return "操作失败"
}
