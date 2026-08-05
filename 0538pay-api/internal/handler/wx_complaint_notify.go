package handler

import (
	"bytes"
	"errors"
	"io"

	"github.com/epvia/api/internal/service"
	"github.com/gin-gonic/gin"
)

// WxComplaintNotifyHandler 消费者投诉2.0 回调入口（自研扩展，公开无 JWT，靠 WECHATPAY2-SHA256-RSA2048 验签鉴权）。
//
// 路由：POST /api/notify/wx-complaint —— 微信投诉通知回调（4012076174）。
// 应答约定（微信规范）：验签+解密+落库成功 → HTTP 200 空体；失败 → HTTP 5XX + FAIL（触发微信按 60S*10 次 / 之后 300S、
// 最长 2 小时重推）。对 WECHATPAY/SIGNTEST/ 探测流量走验签即天然拒绝（正确应对探测）。
type WxComplaintNotifyHandler struct {
	svc *service.WxComplaintService
}

func NewWxComplaintNotifyHandler(svc *service.WxComplaintService) *WxComplaintNotifyHandler {
	return &WxComplaintNotifyHandler{svc: svc}
}

// Notify POST /api/notify/wx-complaint 微信投诉通知回调。
func (h *WxComplaintNotifyHandler) Notify(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	err := h.svc.HandleNotify(c.Request.Context(), notifyHeaders(c), body)
	if err == nil {
		c.Status(200)
		return
	}
	var we *service.WxComplaintError
	msg := err.Error()
	if errors.As(err, &we) {
		msg = we.Msg
	}
	c.JSON(500, gin.H{"code": "FAIL", "message": msg})
}
