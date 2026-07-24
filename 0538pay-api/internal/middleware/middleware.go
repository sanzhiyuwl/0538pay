package middleware

import (
	"strings"

	"github.com/epvia/api/pkg/jwtauth"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// 上下文键
const (
	CtxUID   = "uid"
	CtxName  = "name"
	CtxRole  = "role"
	CtxScope = "scope"
)

// CORS 开发期放开跨域，前端 vite dev server 直连。生产按域名收紧。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// Auth 校验 JWT。scope 限定该组接口所属端（admin/merchant/console），不匹配则拒绝。
func Auth(jm *jwtauth.Manager, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			resp.Abort(c, 401, 401, "未登录或缺少 token")
			return
		}
		claims, err := jm.Parse(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			resp.Abort(c, 401, 401, "token 无效或已过期")
			return
		}
		if scope != "" && claims.Scope != scope {
			resp.Abort(c, 403, 403, "无权访问该端接口")
			return
		}
		c.Set(CtxUID, claims.UID)
		c.Set(CtxName, claims.Name)
		c.Set(CtxRole, claims.Role)
		c.Set(CtxScope, claims.Scope)
		c.Next()
	}
}
