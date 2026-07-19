package router

import (
	"github.com/0538pay/api/internal/handler"
	"github.com/0538pay/api/internal/middleware"
	"github.com/0538pay/api/pkg/jwtauth"
	"github.com/0538pay/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// Deps 汇聚路由所需的 handler 与依赖。
type Deps struct {
	JWT      *jwtauth.Manager
	Auth     *handler.AuthHandler
	Order    *handler.OrderHandler
	Merchant *handler.MerchantHandler
}

// Setup 注册所有路由。
func Setup(r *gin.Engine, d Deps) {
	r.Use(middleware.CORS())

	// 探活
	r.GET("/health", func(c *gin.Context) { resp.OK(c, gin.H{"status": "up"}) })

	api := r.Group("/api")

	// 运营后台
	admin := api.Group("/admin")
	{
		admin.POST("/login", d.Auth.Login) // 无需鉴权

		authed := admin.Group("")
		authed.Use(middleware.Auth(d.JWT, "admin"))
		{
			authed.GET("/orders", d.Order.List)
			authed.GET("/merchants", d.Merchant.List)
		}
	}

	// merchant / console / pay 分组后续补
}
