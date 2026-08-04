package handler

import (
	"bytes"
	"errors"
	"io"

	"github.com/epvia/api/internal/service"
	"github.com/gin-gonic/gin"
)

// ChannelControlNotifyHandler 子商户管控/处置订阅回调入口（风控第三段，公开无 JWT，靠 WECHATPAY2-SHA256-RSA2048 验签鉴权）。
//
// 两条独立回调路由，分别对应微信两套机制：
//   POST /api/notify/channel-control/violation        —— (A) 商户平台处置通知（service_notify_url，4012079216）
//   POST /api/notify/channel-control/merchant-notify  —— (B) 合作伙伴订阅·商户新增管控流水（topic 20000，4016022266）
//
// 应答约定（微信规范）：验签+解密+落库成功 → HTTP 200 空体；失败 → HTTP 5XX + FAIL 报文（触发微信按退避重推最长 48h）。
// 对签名探测（WECHATPAY/SIGNTEST/）流量：验签必然失败 → 返回非 2xx，即"正确应对探测"。
type ChannelControlNotifyHandler struct {
	svc *service.ChannelControlNotifyService
}

func NewChannelControlNotifyHandler(svc *service.ChannelControlNotifyService) *ChannelControlNotifyHandler {
	return &ChannelControlNotifyHandler{svc: svc}
}

// notifyHeaders 从请求头抓取微信验签四要素。
func notifyHeaders(c *gin.Context) service.NotifyHeaders {
	return service.NotifyHeaders{
		Signature: c.GetHeader("Wechatpay-Signature"),
		Timestamp: c.GetHeader("Wechatpay-Timestamp"),
		Nonce:     c.GetHeader("Wechatpay-Nonce"),
		Serial:    c.GetHeader("Wechatpay-Serial"),
	}
}

// ackControlNotify 统一应答：成功 200 空体；失败 5XX + FAIL。
func ackControlNotify(c *gin.Context, err error) {
	if err == nil {
		c.Status(200)
		return
	}
	var ne *service.ChannelNotifyError
	msg := err.Error()
	if errors.As(err, &ne) {
		msg = ne.Msg
	}
	c.JSON(500, gin.H{"code": "FAIL", "message": msg})
}

// Violation POST /api/notify/channel-control/violation  (A) 商户平台处置通知。
func (h *ChannelControlNotifyHandler) Violation(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	err := h.svc.HandleViolation(c.Request.Context(), notifyHeaders(c), body)
	ackControlNotify(c, err)
}

// MerchantNotify POST /api/notify/channel-control/merchant-notify  (B) 合作伙伴订阅·管控流水。
func (h *ChannelControlNotifyHandler) MerchantNotify(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	err := h.svc.HandleMerchantNotify(c.Request.Context(), notifyHeaders(c), body)
	ackControlNotify(c, err)
}
