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
	JWT            *jwtauth.Manager
	Auth           *handler.AuthHandler
	Order          *handler.OrderHandler
	Merchant       *handler.MerchantHandler
	Channel        *handler.ChannelHandler
	Pay            *handler.PayHandler
	Settle         *handler.SettleHandler
	MerchantAuth   *handler.MerchantAuthHandler
	MerchantCenter *handler.MerchantCenterHandler
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
			authed.GET("/channels", d.Channel.List)
			authed.POST("/channels", d.Channel.Create)
			authed.PUT("/channels/:id", d.Channel.Update)
			authed.DELETE("/channels/:id", d.Channel.Delete)
			authed.PUT("/channels/:id/status", d.Channel.SetStatus)
			authed.GET("/channels/:id/config", d.Channel.GetConfig)
			authed.PUT("/channels/:id/config", d.Channel.SaveConfig)

			// 结算管理（C2 结算域）
			authed.GET("/settles", d.Settle.List)
			authed.PUT("/settles/:id/status", d.Settle.SetStatus)
			authed.GET("/settle/batches", d.Settle.Batches)
			authed.POST("/settle/batch", d.Settle.CreateBatch)
			authed.POST("/settle/batch/:batch/complete", d.Settle.CompleteBatch)
		}
	}

	// 对外收单 API（公开，无 JWT，靠 MD5 签名鉴权）
	pay := api.Group("/pay")
	{
		pay.POST("/submit", d.Pay.Submit)
		// 收银台中间页查单（公开，仅安全字段）
		pay.GET("/order/:trade_no", d.Pay.Cashier)
		// 收银台主动查单（公开）：未付时向渠道 Query，确认已付则改单入账
		pay.GET("/query/:trade_no", d.Pay.Query)
		// 第三方渠道回调（GET/POST 均支持，验签在渠道层）
		pay.POST("/notify/:trade_no", d.Pay.Notify)
		pay.GET("/notify/:trade_no", d.Pay.Notify)
	}

	// 商户中心（阶段D）
	merchant := api.Group("/merchant")
	{
		merchant.POST("/login", d.MerchantAuth.Login) // 无需鉴权

		mAuthed := merchant.Group("")
		mAuthed.Use(middleware.Auth(d.JWT, "merchant"))
		{
			mAuthed.GET("/info", d.MerchantAuth.Info)
			// D2 查询与操作
			mAuthed.GET("/dashboard", d.MerchantCenter.Dashboard)
			mAuthed.GET("/orders", d.MerchantCenter.Orders)
			mAuthed.GET("/records", d.MerchantCenter.Records)
			mAuthed.GET("/settles", d.MerchantCenter.Settles)
			mAuthed.GET("/apply/info", d.MerchantCenter.ApplyInfo)
			mAuthed.POST("/apply", d.MerchantCenter.Apply)
			mAuthed.POST("/order/refund", d.MerchantCenter.Refund)
			mAuthed.POST("/order/notify", d.MerchantCenter.Renotify)
			// D3 资料/密钥/密码
			mAuthed.GET("/apikey", d.MerchantCenter.ApiInfo)
			mAuthed.POST("/apikey/reset", d.MerchantCenter.ResetKey)
			mAuthed.PUT("/profile", d.MerchantCenter.UpdateProfile)
			mAuthed.PUT("/password", d.MerchantCenter.ChangePassword)
		}
	}

	// console 分组后续补
}
