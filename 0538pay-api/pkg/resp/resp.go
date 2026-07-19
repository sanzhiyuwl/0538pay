// Package resp 提供统一 API 响应封装。
// 约定：code=0 成功，非 0 失败。data 为业务数据。
// 注：不沿用 epay 的 code=1 成功约定，自研 code=0，顺带拉开相似度。
package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Body 是所有接口的统一响应体。
type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// PageData 是列表类接口 data 的标准形状。
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// OK 返回成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: 0, Msg: "ok", Data: data})
}

// Page 返回分页成功响应。
func Page(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	OK(c, PageData{List: list, Total: total, Page: page, PageSize: pageSize})
}

// Fail 返回业务失败响应（HTTP 仍 200，靠 code 区分）。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Body{Code: code, Msg: msg, Data: nil})
}

// Abort 返回失败并中断后续 handler（用于中间件鉴权失败等）。
func Abort(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, Body{Code: code, Msg: msg, Data: nil})
}
