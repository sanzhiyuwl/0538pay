package middleware

import (
	"bytes"
	"encoding/json"

	"github.com/epvia/api/internal/service"
	"github.com/gin-gonic/gin"
)

// 商户操作日志埋点相关的 gin.Context 键。
// handler/service 层可在业务处理后 c.Set 这些键，让中间件写出更精确的 target/detail（第二期用）。
const (
	CtxOpTarget = "op_target" // string：精确对象摘要，覆盖动作映射表默认
	CtxOpDetail = "op_detail" // string(JSON)：明细
	CtxOpSkip   = "op_skip"   // bool：置 true 则本次不记日志
)

// bodyLogWriter 包装 gin ResponseWriter，旁路捕获响应体以判定业务 code。
type bodyLogWriter struct {
	gin.ResponseWriter
	buf *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.buf.Write(b) // 旁路缓存一份（响应体通常很小）
	return w.ResponseWriter.Write(b)
}

// OpLog 商户写操作自动埋点中间件。挂在 merchant 写操作路由组上。
// 请求完成后按响应 code 记一条操作日志（成功/失败都记）；动作元数据来自 service.opActions 映射表，
// 未登记的路由不记（避免噪音）。写入异步、吞错，不阻断主流程。
func OpLog(oplog *service.OpLogService) gin.HandlerFunc {
	return opLogFor(oplog, service.ScopeMerchant, "/api/merchant", service.LookupAction)
}

// AdminOpLog 管理端写操作自动埋点中间件。挂在 admin 写操作路由组上。
// 动作元数据由「资源段 × 动词」派生（service.LookupAdminAction），未识别资源不记。
func AdminOpLog(oplog *service.OpLogService) gin.HandlerFunc {
	return opLogFor(oplog, service.ScopeAdmin, "/api/admin", service.LookupAdminAction)
}

// AgentOpLog 代理端写操作自动埋点中间件。挂在 agent 写操作路由组上。
// 动作元数据来自 service.agentActions 显式映射，未登记的路由不记（避免噪音）。
func AgentOpLog(oplog *service.OpLogService) gin.HandlerFunc {
	return opLogFor(oplog, service.ScopeAgent, "/api/agent", service.LookupAgentAction)
}

// ConsoleOpLog 代理进件控制台写操作自动埋点中间件。挂在 console 路由组上（走 admin token，前缀 /api/console）。
// 动作元数据由「资源段 × 动词」派生（service.LookupConsoleAction），未识别资源不记。
func ConsoleOpLog(oplog *service.OpLogService) gin.HandlerFunc {
	return opLogFor(oplog, service.ScopeConsole, "/api/console", service.LookupConsoleAction)
}

// opLogFor 构造某端的埋点中间件。scope 区分商户/管理端；prefix 为路由前缀（用于剥离得到路由模板）；
// lookup 为该端的动作元数据查询（未登记/未识别返回 ok=false 时直接放行不记）。
// 仅对写操作(POST/PUT/DELETE)埋点，GET 一律放行。
func opLogFor(
	oplog *service.OpLogService,
	scope, prefix string,
	lookup func(string) (service.OpActionMeta, bool),
) gin.HandlerFunc {
	return func(c *gin.Context) {
		m := c.Request.Method
		if m != "POST" && m != "PUT" && m != "DELETE" {
			c.Next()
			return
		}
		// 按 "METHOD 路由模板" 查动作，未登记直接放行不埋点。
		key := m + " " + routeTemplate(c, prefix)
		meta, ok := lookup(key)
		if !ok {
			c.Next()
			return
		}

		blw := bodyLogWriter{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = blw
		c.Next()

		// handler 显式要求跳过（如幂等无变更）。
		if skip, _ := c.Get(CtxOpSkip); skip == true {
			return
		}

		result := "ok"
		if !isSuccessBody(blw.buf.Bytes()) {
			result = "fail"
		}

		target, _ := c.Get(CtxOpTarget)
		detail, _ := c.Get(CtxOpDetail)
		targetStr, _ := target.(string)
		detailStr, _ := detail.(string)

		oplog.Record(service.OpContext{
			Scope:    scope,
			UID:      currentUID(c),
			Operator: c.GetString(CtxName),
			IP:       c.ClientIP(),
			Meta:     meta,
			Result:   result,
			Target:   targetStr,
			Detail:   detailStr,
		})
	}
}

// routeTemplate 返回注册时的路由模板（去掉端前缀），如 "/domains/:id"。
func routeTemplate(c *gin.Context, prefix string) string {
	full := c.FullPath() // 如 /api/merchant/domains/:id
	if len(full) > len(prefix) && full[:len(prefix)] == prefix {
		return full[len(prefix):]
	}
	return full
}

// isSuccessBody 解析响应体判断业务是否成功（统一响应 code=0 为成功）。
// 解析失败（非 JSON/空）保守视为成功，避免误记失败。
func isSuccessBody(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return true
	}
	return body.Code == 0
}

// currentUID 从 context 取当前商户 uid（0 表示无效）。与 handler 层同语义。
func currentUID(c *gin.Context) uint {
	v, ok := c.Get(CtxUID)
	if !ok {
		return 0
	}
	uid, _ := v.(uint)
	return uid
}
