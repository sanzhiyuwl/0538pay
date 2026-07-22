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
	Group          *handler.GroupHandler
	Config         *handler.ConfigHandler
	Channel        *handler.ChannelHandler
	Roll           *handler.RollHandler
	SubChannel     *handler.SubChannelHandler
	PayType        *handler.PayTypeHandler
	Weixin         *handler.WeixinHandler
	Wework         *handler.WeworkHandler
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
	Mapi           *handler.MapiHandler
	Paypage        *handler.PaypageHandler
	Message        *handler.MessageHandler
	Dashboard      *handler.DashboardHandler
	Announce       *handler.AnnounceHandler
	Clean          *handler.CleanHandler
	Cron           *handler.CronHandler
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
			// 仪表盘（全平台聚合）
			authed.GET("/dashboard", d.Dashboard.Overview)

			// 订单管理（列表 + 写操作）
			authed.GET("/orders", d.Order.List)
			authed.GET("/orders/export", d.Order.Export) // 全量流式 CSV 导出
			authed.POST("/orders/refund", d.Order.Refund)
			authed.POST("/orders/batch", d.Order.Batch)
			authed.PUT("/orders/:trade_no/status", d.Order.SetStatus)
			authed.POST("/orders/:trade_no/freeze", d.Order.Freeze)
			authed.POST("/orders/:trade_no/unfreeze", d.Order.Unfreeze)
			authed.GET("/orders/:trade_no/refund-info", d.Order.RefundInfo)
			authed.POST("/orders/:trade_no/fill", d.Order.FillOrder)
			authed.POST("/orders/:trade_no/notify", d.Order.Renotify)
			authed.DELETE("/orders/:trade_no", d.Order.Delete)

			// 商户管理（写操作）
			authed.GET("/merchants", d.Merchant.List)
			authed.POST("/merchants", d.Merchant.Create)
			authed.PUT("/merchants/:uid", d.Merchant.Update)
			authed.POST("/merchants/:uid/recharge", d.Merchant.Recharge)
			authed.PUT("/merchants/:uid/group", d.Merchant.SetGroup)
			authed.PUT("/merchants/:uid/status", d.Merchant.SetStatus)
			authed.POST("/merchants/:uid/resetkey", d.Merchant.ResetKey)
			authed.GET("/merchants/:uid/sso", d.Merchant.SSO) // 免密进入商户中心（短时 token）
			authed.DELETE("/merchants/:uid", d.Merchant.Delete)

			// 系统设置（config 域）
			authed.GET("/config/:group", d.Config.GetGroup)
			authed.PUT("/config/:group", d.Config.SaveGroup)

			// 用户组管理
			authed.GET("/groups", d.Group.List)
			authed.POST("/groups", d.Group.Create)
			authed.PUT("/groups/:gid", d.Group.Update)
			authed.PUT("/groups/:gid/buy", d.Group.SetBuy)
			authed.DELETE("/groups/:gid", d.Group.Delete)
			// 用户组通道分配（{typeid:{type,channel,rate}}）
			authed.GET("/groups/:gid/assigns", d.Group.GetAssigns)
			authed.PUT("/groups/:gid/assigns", d.Group.SaveAssigns)

			authed.GET("/channels", d.Channel.List)
			authed.POST("/channels", d.Channel.Create)
			authed.PUT("/channels/:id", d.Channel.Update)
			authed.DELETE("/channels/:id", d.Channel.Delete)
			authed.PUT("/channels/:id/status", d.Channel.SetStatus)
			authed.GET("/channels/:id/config", d.Channel.GetConfig)
			authed.PUT("/channels/:id/config", d.Channel.SaveConfig)

			// 通道轮询组（roll）
			authed.GET("/rolls", d.Roll.List)
			authed.POST("/rolls", d.Roll.Create)
			authed.PUT("/rolls/:id", d.Roll.Update)
			authed.PUT("/rolls/:id/status", d.Roll.SetStatus)
			authed.DELETE("/rolls/:id", d.Roll.Delete)

			// 子通道（商户维度，?uid= 指定商户）
			authed.GET("/subchannels", d.SubChannel.List)
			authed.POST("/subchannels", d.SubChannel.Create)
			authed.PUT("/subchannels/:id", d.SubChannel.Update)
			authed.PUT("/subchannels/:id/status", d.SubChannel.SetStatus)
			authed.DELETE("/subchannels/:id", d.SubChannel.Delete)

			// 支付方式 pay_type
			authed.GET("/paytypes", d.PayType.List)
			authed.POST("/paytypes", d.PayType.Create)
			authed.PUT("/paytypes/:id", d.PayType.Update)
			authed.PUT("/paytypes/:id/status", d.PayType.SetStatus)
			authed.DELETE("/paytypes/:id", d.PayType.Delete)

			// 微信公众号/小程序 pay_weixin（无状态开关）
			authed.GET("/weixins", d.Weixin.List)
			authed.POST("/weixins", d.Weixin.Create)
			authed.PUT("/weixins/:id", d.Weixin.Update)
			authed.DELETE("/weixins/:id", d.Weixin.Delete)

			// 企业微信 pay_wework
			authed.GET("/weworks", d.Wework.List)
			authed.POST("/weworks", d.Wework.Create)
			authed.PUT("/weworks/:id", d.Wework.Update)
			authed.PUT("/weworks/:id/status", d.Wework.SetStatus)
			authed.DELETE("/weworks/:id", d.Wework.Delete)

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

			// 站内信下发（我方新增）
			authed.GET("/messages", d.Message.List)
			authed.POST("/messages", d.Message.Send)
			authed.DELETE("/messages/:id", d.Message.Delete)

			// 数据清理（对齐 epay clean.php，高风险破坏性）
			authed.POST("/clean", d.Clean.Clean)

			// 网站公告（对齐 epay gonggao.php）
			authed.GET("/announces", d.Announce.List)
			authed.POST("/announces", d.Announce.Create)
			authed.PUT("/announces/:id", d.Announce.Update)
			authed.PUT("/announces/:id/status", d.Announce.SetStatus)
			authed.DELETE("/announces/:id", d.Announce.Delete)

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

	// V2 REST 接口族（对齐 epay api.php?s= → ApiHelper 反射分发）。
	// 公开(无 JWT)，靠 MD5/RSA 签名鉴权 + timestamp 防重放。路径 /api/mapi/:class/:action。
	mapi := api.Group("/mapi")
	{
		mapi.POST("/:class/:action", d.Mapi.Dispatch)
		mapi.GET("/:class/:action", d.Mapi.Dispatch)
	}

	// 对外手动触发计划任务（对齐 epay cron.php，由 cronkey 校验，无 JWT）
	if d.Cron != nil {
		cron := api.Group("/cron")
		{
			cron.GET("/:task", d.Cron.Run)
			cron.POST("/:task", d.Cron.Run)
		}
	}

	// 官网 CMS 内容读取（公开，官网前端读）
	site := api.Group("/site")
	{
		site.GET("/config/:key", d.SiteConfig.Get)
		site.GET("/announces", d.Announce.Public) // 展示中的公告（官网/商户端读）
	}

	// 商户中心（阶段D）
	merchant := api.Group("/merchant")
	{
		merchant.POST("/login", d.MerchantAuth.Login)         // 无需鉴权
		merchant.GET("/captcha", d.MerchantAuth.Captcha)      // 图形验证码（公开）
		merchant.POST("/register", d.MerchantAuth.Register)   // 注册（公开）
		merchant.POST("/findpwd", d.MerchantAuth.FindPwd)     // 找回密码（公开）
		// 快捷登录 OAuth（公开）
		merchant.GET("/oauth/:provider/url", d.MerchantAuth.OAuthURL)
		merchant.POST("/oauth/:provider/callback", d.MerchantAuth.OAuthCallback)
		merchant.POST("/oauth/bind", d.MerchantAuth.OAuthBind)

		mAuthed := merchant.Group("")
		mAuthed.Use(middleware.Auth(d.JWT, "merchant"))
		{
			mAuthed.GET("/info", d.MerchantAuth.Info)
			mAuthed.POST("/complete", d.MerchantAuth.Complete) // 完善资料（需登录）
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
			// 自助流程：测试支付 / 聚合收款码 / 邀请返现 / 授权域名 / 使用说明 / 站内信
			mAuthed.GET("/test", d.MerchantCenter.TestPayInfo)
			mAuthed.POST("/test", d.MerchantCenter.TestPay)
			mAuthed.GET("/onecode", d.MerchantCenter.OnecodeInfo)
			mAuthed.POST("/onecode/name", d.MerchantCenter.SaveCodeName)
			mAuthed.GET("/invite", d.MerchantCenter.InviteInfo)
			mAuthed.GET("/domains", d.MerchantCenter.DomainList)
			mAuthed.POST("/domains", d.MerchantCenter.DomainAdd)
			mAuthed.DELETE("/domains/:id", d.MerchantCenter.DomainDelete)
			mAuthed.GET("/help", d.MerchantCenter.Help)
			mAuthed.GET("/messages", d.MerchantCenter.Messages)
			mAuthed.POST("/messages/:id/read", d.MerchantCenter.MessageRead)
		}
	}

	// 公开聚合收款页（对齐 epay paypage/index.php，无需登录，靠加密 merchant 标识）
	paypage := api.Group("/paypage")
	{
		paypage.GET("/info", d.Paypage.Info)
		paypage.POST("/submit", d.Paypage.Submit)
	}

	// console 分组后续补
}
