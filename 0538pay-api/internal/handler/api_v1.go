package handler

import (
	"github.com/epvia/api/internal/service"
	"github.com/gin-gonic/gin"
)

// ApiV1Handler V1 遗留接口 api.php?act= 入口（A-5，GET+明文key，code=1 语义）。
// 路由 /api/v1?act=query|order|orders|settle 与 POST /api/v1?act=refund。
type ApiV1Handler struct {
	svc *service.ApiV1Service
}

func NewApiV1Handler(svc *service.ApiV1Service) *ApiV1Handler {
	return &ApiV1Handler{svc: svc}
}

// Dispatch 按 act 分发（对齐 epay api.php 的 if/elseif 链）。
func (h *ApiV1Handler) Dispatch(c *gin.Context) {
	params := collectParams(c) // 同时收集 GET query 与 POST form
	act := params["act"]
	var out map[string]interface{}
	switch act {
	case "query":
		out = h.svc.Query(params)
	case "order":
		out = h.svc.Order(params)
	case "orders":
		out = h.svc.Orders(params)
	case "settle":
		out = h.svc.Settle(params)
	case "refund":
		out = h.svc.Refund(params)
	default:
		out = map[string]interface{}{"code": -5, "msg": "接口方法不存在"}
	}
	c.JSON(200, out)
}
