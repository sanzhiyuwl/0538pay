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
	Record         *handler.RecordHandler
	Transfer       *handler.TransferHandler
	Profit         *handler.ProfitHandler
	Risk           *handler.RiskHandler
	Blacklist      *handler.BlacklistHandler
	Domain         *handler.DomainHandler
	Stat           *handler.StatHandler
	Log            *handler.LogHandler
	Invite         *handler.InviteHandler
	SiteConfig     *handler.SiteConfigHandler
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

			// 资金流水（C2 尾巴：后台资金明细页）
			authed.GET("/records", d.Record.List)
			authed.GET("/records/stats", d.Record.Stats)

			// 代付 / 转账（C3）
			authed.GET("/transfers", d.Transfer.List)
			authed.GET("/transfers/stats", d.Transfer.Stats)
			authed.POST("/transfers", d.Transfer.Create)
			authed.PUT("/transfers/:biz/status", d.Transfer.SetStatus)
			authed.POST("/transfers/:biz/refund", d.Transfer.Refund)
			authed.DELETE("/transfers/:biz", d.Transfer.Delete)

			// 分账（C3）
			authed.GET("/ps/orders", d.Profit.List)
			authed.GET("/ps/orders/stats", d.Profit.Stats)
			authed.POST("/ps/orders/:id/op", d.Profit.Operate)

			// 风控（C4，只读）
			authed.GET("/risks", d.Risk.List)
			// 黑名单（C4）
			authed.GET("/blacklist", d.Blacklist.List)
			authed.GET("/blacklist/stats", d.Blacklist.Stats)
			authed.POST("/blacklist", d.Blacklist.Add)
			authed.DELETE("/blacklist/:id", d.Blacklist.Delete)
			authed.POST("/blacklist/batch-delete", d.Blacklist.BatchDelete)
			// 授权域名（C4）
			authed.GET("/domains", d.Domain.List)
			authed.GET("/domains/stats", d.Domain.Stats)
			authed.POST("/domains", d.Domain.Add)
			authed.PUT("/domains/:id/status", d.Domain.SetStatus)
			authed.DELETE("/domains/:id", d.Domain.Delete)
			authed.POST("/domains/batch", d.Domain.BatchOp)

			// 统计（C5，只读聚合）
			authed.GET("/stat/pay", d.Stat.PayStat)
			// 登录日志（C5，只读）
			authed.GET("/logs", d.Log.List)
			// 邀请码（C5）
			authed.GET("/invitecodes", d.Invite.List)
			authed.POST("/invitecodes/generate", d.Invite.Generate)
			authed.DELETE("/invitecodes/:id", d.Invite.Delete)
			authed.POST("/invitecodes/clear", d.Invite.Clear)

			// 官网 CMS 内容保存（后台鉴权写）
			authed.PUT("/site/config/:key", d.SiteConfig.Save)
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

	// 官网 CMS 内容读取（公开，官网前端读）
	site := api.Group("/site")
	{
		site.GET("/config/:key", d.SiteConfig.Get)
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
			mAuthed.POST("/apikey/rsa", d.MerchantCenter.GenRSAKey)      // V2 生成 RSA 密钥对
			mAuthed.PUT("/apikey/keytype", d.MerchantCenter.SetKeyType)  // V2 设置签名模式
			mAuthed.PUT("/profile", d.MerchantCenter.UpdateProfile)
			mAuthed.PUT("/password", d.MerchantCenter.ChangePassword)
			// 代付（C3 商户端）
			mAuthed.GET("/transfers", d.MerchantCenter.Transfers)
			mAuthed.POST("/transfer", d.MerchantCenter.TransferCreate)
			// 保证金 / 购买会员（D3 增值）
			mAuthed.GET("/deposit", d.MerchantCenter.DepositInfo)
			mAuthed.POST("/deposit/recharge", d.MerchantCenter.DepositRecharge)
			mAuthed.POST("/deposit/withdraw", d.MerchantCenter.DepositWithdraw)
			mAuthed.GET("/groups", d.MerchantCenter.GroupPlans)
			mAuthed.POST("/groups/buy", d.MerchantCenter.BuyGroup)
			mAuthed.POST("/recharge", d.MerchantCenter.Recharge)
			mAuthed.GET("/cert", d.MerchantCenter.CertInfo)
			mAuthed.POST("/cert", d.MerchantCenter.CertSubmit)
		}
	}

	// console 分组后续补
}
