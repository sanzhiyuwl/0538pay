package handler

import (
	"strconv"

	"github.com/0538pay/api/internal/service"
	"github.com/gin-gonic/gin"
)

// MapiHandler V2 REST 接口族入口（对齐 epay api.php?s= → ApiHelper::load_api 反射分发）。
// 路由 /api/mapi/:class/:action，按 class/action 分发到 MapiService 各方法。
// 统一验签(除 pay/create 自行在 Submit 内验签) + 回包 RSA 签名。
type MapiHandler struct {
	svc *service.MapiService
}

func NewMapiHandler(svc *service.MapiService) *MapiHandler {
	return &MapiHandler{svc: svc}
}

// mapiJSON 输出 epay 风格 JSON（不走统一 resp 包，因 mapi 成功 code=0 且需回包签名）。
func mapiJSON(c *gin.Context, data map[string]string) {
	// map[string]string → map[string]interface{}，code 转 int 便于商户解析。
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		out[k] = v
	}
	if _, ok := data["code"]; ok {
		out["code"] = atoiSafe(data["code"])
	}
	c.JSON(200, out)
}

// mapiFail 输出错误 {code,msg}（对齐 epay echojsonmsg）。
func mapiFail(c *gin.Context, err error) {
	code := -1
	msg := err.Error()
	if me, ok := err.(*service.MapiError); ok {
		code = me.Code
		msg = me.Msg
	}
	c.JSON(200, gin.H{"code": code, "msg": msg})
}

func atoiSafe(s string) int {
	n := 0
	neg := false
	for i, ch := range s {
		if i == 0 && ch == '-' {
			neg = true
			continue
		}
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	if neg {
		n = -n
	}
	return n
}

// Classic 经典 mapi.php 兼容下单端点（对齐 epay mapi.php → Pay::create() version=0 分支）。
// 契约：POST 表单参数 → {code:1, trade_no, payurl|html|qrcode|urlscheme}，不签名。
// 老商户 SDK 直连入口，与 /api/mapi/pay/create（新版 code=0+签名）区分。
func (h *MapiHandler) Classic(c *gin.Context) {
	params := collectParams(c)
	params["_ip"] = c.ClientIP()
	params["_siteurl"] = reqBaseURL(c) // B1-18：未知支付形态时回落收银台 URL 用
	// 对齐 epay mapi.php:2 未传 pid 直接报 -4。
	if _, ok := params["pid"]; !ok {
		c.JSON(200, gin.H{"code": -4, "msg": "未传入任何参数"})
		return
	}
	out, err := h.svc.PayCreateClassic(c.Request.Context(), params)
	if err != nil {
		mapiFail(c, err)
		return
	}
	c.JSON(200, out)
}

// Dispatch 反射式分发 /api/mapi/:class/:action。
func (h *MapiHandler) Dispatch(c *gin.Context) {
	class := c.Param("class")
	action := c.Param("action")
	params := collectParams(c)
	params["_ip"] = c.ClientIP()
	params["_siteurl"] = reqBaseURL(c) // B1-18：未知支付形态时回落收银台 URL 用

	key := class + "/" + action

	// pay/create 走白名单：验签在 PayService.Submit 内做（与老 submit 一致），不经统一 verify。
	if key == "pay/create" {
		out, err := h.svc.PayCreate(c.Request.Context(), params)
		if err != nil {
			mapiFail(c, err)
			return
		}
		mapiJSON(c, h.svc.SignResponse(out))
		return
	}

	// 其余接口统一验签。
	m, err := h.svc.Verify(params)
	if err != nil {
		mapiFail(c, err)
		return
	}

	var out map[string]string
	switch key {
	case "pay/query":
		out, err = h.svc.PayQuery(m, params)
	case "pay/refund":
		out, err = h.svc.PayRefund(m, params)
	case "pay/refundquery":
		out, err = h.svc.PayRefundQuery(m, params)
	case "merchant/info":
		out, err = h.svc.MerchantInfo(m)
	case "merchant/orders":
		// orders 返回含 data 列表。对齐 epay ApiHelper::load_api：整个 result 进 makeSign，
		// getSignContent 跳过数组值(data)但 code/timestamp/sign_type 等标量仍参与签名并回包 sign。
		res, e := h.svc.MerchantOrders(m, params)
		if e != nil {
			mapiFail(c, e)
			return
		}
		// 拆出标量字段走 SignResponse 签名，data 数组作为额外字段合并回输出（数组不参与签名，对齐 epay）。
		scalar := map[string]string{}
		extra := map[string]interface{}{}
		for k, v := range res {
			switch vv := v.(type) {
			case string:
				scalar[k] = vv
			case int:
				scalar[k] = strconv.Itoa(vv)
			default:
				extra[k] = v // data 列表等非标量，不参与签名
			}
		}
		signed := h.svc.SignResponse(scalar)
		out := make(map[string]interface{}, len(signed)+len(extra))
		for k, v := range signed {
			out[k] = v
		}
		if _, ok := signed["code"]; ok {
			out["code"] = atoiSafe(signed["code"])
		}
		for k, v := range extra {
			out[k] = v
		}
		c.JSON(200, out)
		return
	case "transfer/submit":
		out, err = h.svc.TransferSubmit(m, params)
	case "transfer/query":
		out, err = h.svc.TransferQuery(m, params)
	case "transfer/proof":
		out, err = h.svc.TransferProof(m, params)
	case "transfer/balance":
		out, err = h.svc.TransferBalance(m)
	default:
		c.JSON(200, gin.H{"code": -5, "msg": "接口方法不存在"})
		return
	}
	if err != nil {
		mapiFail(c, err)
		return
	}
	mapiJSON(c, h.svc.SignResponse(out))
}
