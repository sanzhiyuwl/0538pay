package handler

import (
	"bytes"
	"io"

	"github.com/0538pay/api/internal/channel"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// PayHandler 对外收单接口（无 JWT，靠 MD5 签名鉴权）。
type PayHandler struct {
	svc *service.PayService
}

func NewPayHandler(svc *service.PayService) *PayHandler {
	return &PayHandler{svc: svc}
}

// Submit POST /api/pay/submit
// 兼容 epay V1：接收 application/x-www-form-urlencoded 或 query 参数，
// 全量参数进 map 用于验签。返回 JSON（自研 code=0 约定）。
func (h *PayHandler) Submit(c *gin.Context) {
	params := collectParams(c)
	out, err := h.svc.Submit(c.Request.Context(), params)
	if err != nil {
		if pe, ok := err.(*service.PayError); ok {
			resp.Fail(c, pe.Code, pe.Msg)
			return
		}
		resp.Fail(c, 1199, "下单失败: "+err.Error())
		return
	}
	resp.OK(c, out)
}

// Notify POST/GET /api/pay/notify/:trade_no
// 第三方渠道支付回调入口。验签在渠道层做，这里汇集参数交 service 处理。
// 兼容两类回调：
//   - 表单/query 型（易支付 V1、mock）：参数进 params map。
//   - JSON 型（微信 APIv3）：原始报文体 + Wechatpay-* 头注入 params 保留键，渠道层验签+解密。
// 应答：JSON 型渠道 AckContent 为空 → HTTP 200 空体（微信规范）；表单型 → 纯文本 AckContent。
func (h *PayHandler) Notify(c *gin.Context) {
	tradeNo := c.Param("trade_no")

	// 先读原始 body（ParseForm 会消费 body，需在 collectParams 之前抓取）。
	rawBody, _ := io.ReadAll(c.Request.Body)
	// 把 body 塞回去，供后续 collectParams 的 ParseForm 解析表单型回调。
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

	params := collectParams(c)
	// 注入 JSON 型回调所需的原始报文与验签头（微信 APIv3 用）。
	params[channel.RawBody] = string(rawBody)
	params[channel.RawSignature] = c.GetHeader("Wechatpay-Signature")
	params[channel.RawTimestamp] = c.GetHeader("Wechatpay-Timestamp")
	params[channel.RawNonce] = c.GetHeader("Wechatpay-Nonce")
	params[channel.RawSerial] = c.GetHeader("Wechatpay-Serial")

	res, err := h.svc.Notify(c.Request.Context(), tradeNo, params)
	if err != nil {
		// 回调失败：JSON 型渠道回 4XX + FAIL（触发微信重推）；此处统一 500 + fail 文本兼容两类。
		c.JSON(500, gin.H{"code": "FAIL", "message": err.Error()})
		return
	}
	ack := res.AckContent
	if ack == "" {
		// 微信 APIv3：验签通过应答 200，无需报文体。
		c.Status(200)
		return
	}
	c.String(200, ack)
}

// Query GET /api/pay/query/:trade_no
// 收银台轮询查单：本地未付时主动问渠道，渠道确认已付则改单入账。返回 { status }。
func (h *PayHandler) Query(c *gin.Context) {
	tradeNo := c.Param("trade_no")
	status, err := h.svc.QueryStatus(c.Request.Context(), tradeNo)
	if err != nil {
		if pe, ok := err.(*service.PayError); ok {
			resp.Fail(c, pe.Code, pe.Msg)
			return
		}
		resp.Fail(c, 1199, "查单失败: "+err.Error())
		return
	}
	resp.OK(c, gin.H{"status": status})
}

// Cashier GET /api/pay/order/:trade_no
// 收银台中间页读取的公开订单信息（无鉴权，仅安全字段）。
func (h *PayHandler) Cashier(c *gin.Context) {
	tradeNo := c.Param("trade_no")
	view, err := h.svc.GetCashier(tradeNo)
	if err != nil {
		if pe, ok := err.(*service.PayError); ok {
			resp.Fail(c, pe.Code, pe.Msg)
			return
		}
		resp.Fail(c, 1199, "查询失败: "+err.Error())
		return
	}
	resp.OK(c, view)
}

// collectParams 汇集请求全部参数（form + query）为 string map，供验签与业务读取。
// 同名参数以 form 优先。多值取第一个。
func collectParams(c *gin.Context) map[string]string {
	params := map[string]string{}
	// query
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	// form（POST body）覆盖 query
	_ = c.Request.ParseForm()
	for k, v := range c.Request.PostForm {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	return params
}
