package router

import (
	"time"

	"github.com/epvia/api/internal/handler"
	"github.com/epvia/api/internal/middleware"
	"github.com/epvia/api/internal/service"
	"github.com/epvia/api/pkg/jwtauth"
	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// Deps 汇聚路由所需的 handler 与依赖。
type Deps struct {
	JWT            *jwtauth.Manager
	Auth           *handler.AuthHandler
	Admin          *handler.AdminHandler
	Order          *handler.OrderHandler
	Merchant       *handler.MerchantHandler
	Group          *handler.GroupHandler
	Config         *handler.ConfigHandler
	Channel        *handler.ChannelHandler
	Roll           *handler.RollHandler
	SubChannel     *handler.SubChannelHandler
	ChannelEnroll  *handler.ChannelEnrollHandler
	ChannelControl *handler.ChannelControlHandler
	ChannelCtrlNotify *handler.ChannelControlNotifyHandler
	WxComplaint       *handler.WxComplaintHandler
	WxComplaintNotify *handler.WxComplaintNotifyHandler
	MerchantComplaint *handler.MerchantComplaintHandler
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
	ApiV1          *handler.ApiV1Handler
	Paypage        *handler.PaypageHandler
	Message        *handler.MessageHandler
	Dashboard      *handler.DashboardHandler
	Announce       *handler.AnnounceHandler
	Article        *handler.ArticleHandler
	Clean          *handler.CleanHandler
	Cron           *handler.CronHandler
	Upload         *handler.UploadHandler
	Billing        *handler.BillingHandler
	OpLog          *handler.OpLogHandler
	OpLogSvc       *service.OpLogService
	Role           *handler.RoleHandler
	Console        *handler.ConsoleHandler
	Agent          *handler.AgentHandler
	EnrollPublic   *handler.EnrollPublicHandler
	OCR            *handler.OCRHandler
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
		// 管理端写操作自动埋点（scope=admin，复用商户日志同表；仅 POST/PUT/DELETE 记，GET 放行）。
		authed.Use(middleware.AdminOpLog(d.OpLogSvc))
		{
			// 当前管理员账号资料 / 改密
			authed.GET("/profile", d.Auth.Profile)
			authed.PUT("/profile", d.Auth.UpdateProfile)
			authed.PUT("/password", d.Auth.ChangePassword)

			// 管理员账号 CRUD（RBAC 增强）
			authed.GET("/admins", d.Admin.List)
			authed.POST("/admins", d.Admin.Create)
			authed.PUT("/admins/:id", d.Admin.Update)
			authed.PUT("/admins/:id/status", d.Admin.SetStatus)
			authed.DELETE("/admins/:id", d.Admin.Delete)

			// 角色管理（RBAC 增强，我方独有）
			authed.GET("/roles", d.Role.List)
			authed.POST("/roles", d.Role.Create)
			authed.PUT("/roles/:id", d.Role.Update)
			authed.DELETE("/roles/:id", d.Role.Delete)

			// 仪表盘（全平台聚合）
			authed.GET("/dashboard", d.Dashboard.Overview)

			// 平台月度对账账单（我方独有细化项）
			authed.GET("/billing", d.Billing.List)
			authed.GET("/billing/export", d.Billing.Export)

			// 操作日志（我方独有安全审计增强，查询/导出）：商户端 + 管理端两 scope 复用同表
			authed.GET("/oplogs/merchant", d.OpLog.List)
			authed.GET("/oplogs/merchant/options", d.OpLog.Options)
			authed.GET("/oplogs/merchant/export", d.OpLog.Export)
			authed.GET("/oplogs/admin", d.OpLog.AdminList)
			authed.GET("/oplogs/admin/options", d.OpLog.AdminOptions)
			authed.GET("/oplogs/admin/export", d.OpLog.AdminExport)

			// 订单管理（列表 + 写操作）
			authed.GET("/orders", d.Order.List)
			authed.GET("/orders/export", d.Order.Export) // 全量流式 CSV 导出
			authed.GET("/orders/stats", d.Order.Stats)   // 订单统计概况（全量聚合）
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
			authed.GET("/merchants/:uid/cert", d.Merchant.CertDetail) // 商户实名详情（对齐 epay user_cert）
			authed.DELETE("/merchants/:uid", d.Merchant.Delete)

			// 系统设置（config 域）
			authed.GET("/config/iptype", d.Config.DetectIPType) // F-9 真实IP探测（对齐 epay ajax.php iptype）
			authed.GET("/config/:group", d.Config.GetGroup)
			authed.PUT("/config/:group", d.Config.SaveGroup)
			authed.POST("/config/mail/test", d.Config.TestMail) // K-3 发送测试邮件
			authed.PUT("/paypwd", d.Config.ChangePayPwd) // 修改管理员支付密码（对齐 epay admin_paypwd）
			// OCR 证件识别（营业执照/身份证），后台内可用于核验/录入
			if d.OCR != nil {
				authed.POST("/ocr/license", d.OCR.License)
				authed.POST("/ocr/idcard", d.OCR.IDCard)
			}

			// 用户组管理
			authed.GET("/groups", d.Group.List)
			authed.POST("/groups", d.Group.Create)
			authed.PUT("/groups/:gid", d.Group.Update)
			authed.PUT("/groups/:gid/buy", d.Group.SetBuy)
			authed.DELETE("/groups/:gid", d.Group.Delete)
			// 用户组通道分配（{typeid:{type,channel,rate}}）
			authed.GET("/groups/:gid/assigns", d.Group.GetAssigns)
			authed.PUT("/groups/:gid/assigns", d.Group.SaveAssigns)

			authed.GET("/channels/plugins", d.Channel.PluginMeta)                  // 插件能力/配置元数据
			authed.PUT("/channels/plugins/status", d.Channel.SetPluginsStatus)     // 批量插件启用/禁用（品牌卡一键）
			authed.PUT("/channels/plugins/:key/status", d.Channel.SetPluginStatus) // 单插件启用/禁用开关
			authed.GET("/channels", d.Channel.List)
			authed.POST("/channels", d.Channel.Create)
			authed.PUT("/channels/:id", d.Channel.Update)
			authed.DELETE("/channels/:id", d.Channel.Delete)
			authed.PUT("/channels/:id/status", d.Channel.SetStatus)
			authed.GET("/channels/:id/config", d.Channel.GetConfig)
			authed.PUT("/channels/:id/config", d.Channel.SaveConfig)
			authed.POST("/channels/testpay", d.Channel.TestPay) // 后台通道测试支付（定向下测试单）

			// 通道轮询组（roll）
			authed.GET("/rolls", d.Roll.List)
			authed.POST("/rolls", d.Roll.Create)
			authed.PUT("/rolls/:id", d.Roll.Update)
			authed.PUT("/rolls/:id/status", d.Roll.SetStatus)
			authed.DELETE("/rolls/:id", d.Roll.Delete)

			// 子通道（商户维度，?uid= 指定商户）
			authed.GET("/subchannels", d.SubChannel.List)
			authed.GET("/subchannels/info-fields", d.SubChannel.InfoFormByChannel) // 新建预览：?channel= 按主通道取占位字段
			authed.POST("/subchannels", d.SubChannel.Create)
			authed.PUT("/subchannels/:id", d.SubChannel.Update)
			authed.PUT("/subchannels/:id/status", d.SubChannel.SetStatus)
			authed.GET("/subchannels/:id/info-fields", d.SubChannel.InfoForm) // 编辑：按主通道占位符+插件inputs动态渲染info表单(对齐epay subChannelInfo)
			authed.DELETE("/subchannels/:id", d.SubChannel.Delete)

			// 服务商通道商户进件审核（epay 精仿线：只走商户进件不走二清；通过则自动建子通道回填子商户号）
			if d.ChannelEnroll != nil {
				authed.GET("/channel-enrolls", d.ChannelEnroll.List)
				authed.GET("/channel-enrolls/:id", d.ChannelEnroll.Get)
				authed.POST("/channel-enrolls/:id/sync", d.ChannelEnroll.Sync)       // 主动拉取微信进件状态
				authed.POST("/channel-enrolls/:id/approve", d.ChannelEnroll.Approve) // 管理员手动交付（兜底）
				authed.POST("/channel-enrolls/:id/reject", d.ChannelEnroll.Reject)   // 管理员手动驳回（兜底）

				// 子商户管控（风控第二段）：总览 + 单个/批量刷新微信管控状态
				authed.GET("/channel-controls", d.ChannelControl.List)
				authed.POST("/channel-controls/refresh-all", d.ChannelControl.RefreshAll) // 批量刷新（须在 :id 前）
				authed.POST("/channel-controls/:id/refresh", d.ChannelControl.Refresh)
				authed.GET("/channel-controls/:id/flows", d.ChannelControl.Flows) // 管控流水时间线（风控第三段）
			}

			// 微信支付消费者投诉2.0（自研扩展，挂服务商进件线；admin 全量统一台）
			if d.WxComplaint != nil {
				authed.GET("/wx-complaints", d.WxComplaint.List)
				authed.GET("/wx-complaints/notify-url", d.WxComplaint.GetNotifyURL)     // 回调地址自管理（须在 :id 前）
				authed.PUT("/wx-complaints/notify-url", d.WxComplaint.SetNotifyURL)
				authed.DELETE("/wx-complaints/notify-url", d.WxComplaint.DeleteNotifyURL)
				authed.POST("/wx-complaints/upload", d.WxComplaint.UploadImage)         // 商户反馈图片上传（须在 :id 前）
				authed.POST("/wx-complaints/reconcile", d.WxComplaint.Reconcile)        // 手动轮询兜底对账
				authed.GET("/wx-complaints/:id", d.WxComplaint.Detail)
				authed.POST("/wx-complaints/:id/sync", d.WxComplaint.Sync)              // 现查微信覆盖本地
				authed.GET("/wx-complaints/:id/history", d.WxComplaint.History)         // 协商历史
				authed.POST("/wx-complaints/:id/reply", d.WxComplaint.Reply)            // 回复用户
				authed.POST("/wx-complaints/:id/complete", d.WxComplaint.Complete)      // 反馈处理完成
				authed.POST("/wx-complaints/:id/refund", d.WxComplaint.UpdateRefund)    // 退款审批
				authed.POST("/wx-complaints/:id/immediate", d.WxComplaint.ReplyImmediate) // 即时服务回复
			}

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
			authed.POST("/weworks/:id/kf/refresh", d.Wework.RefreshKf) // K-4 同步客服账号列表
			authed.GET("/wxkf/accounts", d.Wework.ListKf)              // H5 微信客服支付设置页：启用企微下客服账号下拉

			// 结算管理（C2 结算域）
			authed.GET("/settles", d.Settle.List)
			authed.GET("/settle/stats", d.Settle.Stats)
			authed.GET("/settle/export", d.Settle.Export)
			authed.PUT("/settles/:id/status", d.Settle.SetStatus)
			authed.GET("/settle/batches", d.Settle.Batches)
			authed.POST("/settle/batch", d.Settle.CreateBatch)
			authed.POST("/settle/batch/:batch/complete", d.Settle.CompleteBatch)
			authed.GET("/settle/batch/:batch/export", d.Settle.ExportBatch) // C-4 银行专用打款导出

			// 资金流水（C2 尾巴：后台资金明细页）
			authed.GET("/records", d.Record.List)
			authed.GET("/records/stats", d.Record.Stats)
			authed.DELETE("/records/:id", d.Record.Delete)

			// 代付 / 转账（C3）
			authed.GET("/transfers", d.Transfer.List)
			authed.GET("/transfers/stats", d.Transfer.Stats)
			authed.POST("/transfers", d.Transfer.Create)
			authed.POST("/transfers/batch", d.Transfer.CreateBatch) // C-2 批量代付
			authed.PUT("/transfers/:biz/status", d.Transfer.SetStatus)
			authed.POST("/transfers/:biz/refund", d.Transfer.Refund)
			authed.DELETE("/transfers/:biz", d.Transfer.Delete)

			// 分账（C3）
			authed.GET("/ps/orders", d.Profit.List)
			authed.GET("/ps/orders/stats", d.Profit.Stats)
			authed.POST("/ps/orders/:id/op", d.Profit.Operate)
			// 分账规则管理（ps_receiver，C-1）
			authed.GET("/ps/receivers", d.Profit.ListReceivers)
			authed.POST("/ps/receivers", d.Profit.CreateReceiver)
			authed.PUT("/ps/receivers/:id", d.Profit.UpdateReceiver)
			authed.PUT("/ps/receivers/:id/status", d.Profit.SetReceiverStatus)
			authed.DELETE("/ps/receivers/:id", d.Profit.DeleteReceiver)

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
			authed.GET("/stat/buyer", d.Stat.BuyerStat) // C-3 支付用户统计
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
			authed.POST("/clean/cache", d.Clean.CleanCache) // F-14 清理设置缓存（cleancache）

			// 网站公告（对齐 epay gonggao.php）
			authed.GET("/announces", d.Announce.List)
			authed.POST("/announces", d.Announce.Create)
			authed.PUT("/announces/:id", d.Announce.Update)
			authed.PUT("/announces/:id/status", d.Announce.SetStatus)
			authed.DELETE("/announces/:id", d.Announce.Delete)

			// 文章管理（对齐 epay article.php pre_article 行表 CRUD）
			authed.GET("/articles", d.Article.List)
			authed.POST("/articles", d.Article.Create)
			authed.PUT("/articles/:id", d.Article.Update)
			authed.PUT("/articles/:id/status", d.Article.SetActive)
			authed.DELETE("/articles/:id", d.Article.Delete)
			// 文章分类（我方官网扩展）
			authed.GET("/article-categories", d.Article.ListCategories)
			authed.POST("/article-categories", d.Article.CreateCategory)
			authed.PUT("/article-categories/:id", d.Article.UpdateCategory)
			authed.DELETE("/article-categories/:id", d.Article.DeleteCategory)

			// 官网 CMS 内容保存（后台鉴权写）
			authed.PUT("/site/config/:key", d.SiteConfig.Save)

			// 图片上传（文章封面 / 富文本插图，对齐 epay article_upload）
			if d.Upload != nil {
				authed.POST("/upload/image", d.Upload.Image)
			}
		}
	}

	// 上传文件静态访问（/uploads/... → 本地磁盘 ./uploads 目录）
	r.Static("/uploads", "./uploads")

	// 对外收单 API（公开，无 JWT，靠 MD5 签名鉴权）
	pay := api.Group("/pay")
	{
		pay.POST("/submit", d.Pay.Submit)
		// B1-04 收银台选定支付方式：对既有裸单补选通道下单（公开，凭 trade_no，无需商户签名）
		pay.POST("/choose", d.Pay.ChoosePay)
		// JSAPI 收银台微信网页授权：生成授权跳转 URL / code 换买家 openid（公开，凭 trade_no）
		pay.GET("/wx/authurl", d.Pay.WxAuthURL)
		pay.GET("/wx/openid", d.Pay.WxOpenID)
		// 收银台中间页查单（公开，仅安全字段）
		pay.GET("/order/:trade_no", d.Pay.Cashier)
		// 收银台主动查单（公开）：未付时向渠道 Query，确认已付则改单入账
		pay.GET("/query/:trade_no", d.Pay.Query)
		// 第三方渠道回调（GET/POST 均支持，验签在渠道层）
		pay.POST("/notify/:trade_no", d.Pay.Notify)
		pay.GET("/notify/:trade_no", d.Pay.Notify)
	}

	// 公开回调（无 JWT）：腾讯云扫码实名核身完成后回跳（对齐 epay user/alipaycertok.php cert_open==4）。
	// 用户扫脸完在腾讯云页面点完成 → 浏览器带 state=uid&AuthToken=xxx 回跳这里 → 查结果置 cert=1 → 返回结果页。
	pub := api.Group("/pub")
	{
		pub.GET("/cert/qcloud/callback", d.MerchantCenter.CertQcloudCallback)
	}

	// 子商户管控/处置订阅回调（风控第三段，公开无 JWT，靠 WECHATPAY2-SHA256-RSA2048 验签）。
	// A 商户平台处置通知 / B 合作伙伴订阅·管控流水，两条独立路由分别配到微信对应回调地址。
	if d.ChannelCtrlNotify != nil {
		notify := api.Group("/notify/channel-control")
		{
			notify.POST("/violation", d.ChannelCtrlNotify.Violation)             // (A) service_notify_url
			notify.POST("/merchant-notify", d.ChannelCtrlNotify.MerchantNotify) // (B) 合作伙伴订阅 topic 20000
		}
	}

	// 微信支付消费者投诉2.0 回调（自研扩展，公开无 JWT，靠 WECHATPAY2-SHA256-RSA2048 验签鉴权）。
	if d.WxComplaintNotify != nil {
		api.POST("/notify/wx-complaint", d.WxComplaintNotify.Notify)
	}

	// V2 REST 接口族（对齐 epay api.php?s= → ApiHelper 反射分发）。
	// 公开(无 JWT)，靠 MD5/RSA 签名鉴权 + timestamp 防重放。路径 /api/mapi/:class/:action。
	mapi := api.Group("/mapi")
	{
		mapi.POST("/:class/:action", d.Mapi.Dispatch)
		mapi.GET("/:class/:action", d.Mapi.Dispatch)
	}

	// 经典 mapi.php 兼容下单端点（对齐 epay mapi.php，code=1/version=0/不签名，老商户 SDK 直连）。
	// 注册在 api 组与站点根双路径，兼容老 SDK 硬编码 /mapi.php 的场景。
	api.POST("/mapi.php", d.Mapi.Classic)
	r.POST("/mapi.php", d.Mapi.Classic)

	// V1 遗留接口 api.php?act=（A-5，GET+明文key，code=1 语义，兼容老商户）。
	if d.ApiV1 != nil {
		api.GET("/v1", d.ApiV1.Dispatch)
		api.POST("/v1", d.ApiV1.Dispatch)
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
		site.GET("/announces", d.Announce.Public)          // 展示中的公告（官网/商户端读）
		site.GET("/articles", d.Article.Public)            // 分类 + 显示中的文章（官网首页/文章页读）
		site.GET("/articles/:id", d.Article.PublicDetail)  // 文章详情（浏览量 +1）
	}

	// 商户中心（阶段D）
	merchant := api.Group("/merchant")
	{
		merchant.POST("/login", d.MerchantAuth.Login)         // 无需鉴权
		merchant.GET("/captcha", d.MerchantAuth.Captcha)      // 图形验证码（公开）
		merchant.POST("/register", d.MerchantAuth.Register)   // 注册（公开）
		merchant.POST("/findpwd", d.MerchantAuth.FindPwd)     // 找回密码（公开）
		// 快捷登录 OAuth（公开）
		merchant.GET("/oauth/methods", d.MerchantAuth.OAuthMethods) // 开启的快捷登录方式（登录页读，决定显示哪些入口）
		merchant.GET("/oauth/:provider/url", d.MerchantAuth.OAuthURL)
		merchant.POST("/oauth/:provider/callback", d.MerchantAuth.OAuthCallback)
		merchant.POST("/oauth/bind", d.MerchantAuth.OAuthBind)
		// 短信 OTP + 极验（公开；与图形验证码并存，前端二选一）
		merchant.POST("/sms", d.MerchantAuth.SendSms)
		merchant.GET("/geetest", d.MerchantAuth.GeetestInit)
		// 注册方式开关 + 邮箱验证码（公开，注册页读/发码）
		merchant.GET("/reg/methods", d.MerchantAuth.RegMethods)
		merchant.POST("/email-code", d.MerchantAuth.SendEmailCode)
		merchant.GET("/legal", d.MerchantAuth.Legal) // 服务协议/隐私政策正文（公开，登录/注册页弹窗读）

		mAuthed := merchant.Group("")
		mAuthed.Use(middleware.Auth(d.JWT, "merchant"))
		// 商户写操作自动埋点（按动作映射表过滤，只记登记的写操作，GET 一律放行）。
		mAuthed.Use(middleware.OpLog(d.OpLogSvc))
		{
			mAuthed.GET("/info", d.MerchantAuth.Info)
			mAuthed.POST("/complete", d.MerchantAuth.Complete) // 完善资料（需登录）
			// D2 查询与操作
			mAuthed.GET("/dashboard", d.MerchantCenter.Dashboard)
			mAuthed.GET("/orders", d.MerchantCenter.Orders)
			mAuthed.GET("/orders/stats", d.MerchantCenter.OrderStats) // F-13 商户订单统计（服务端聚合+platformProfit）
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
			mAuthed.GET("/msgconfig", d.MerchantCenter.MsgConfig)      // D-3 消息提醒配置
			mAuthed.PUT("/msgconfig", d.MerchantCenter.SaveMsgConfig)  // D-3
			mAuthed.GET("/channelinfo", d.MerchantCenter.ChannelInfo)  // F-20 自定义接口信息设置
			mAuthed.PUT("/channelinfo", d.MerchantCenter.SaveChannelInfo) // F-20
			mAuthed.POST("/rebind", d.MerchantCenter.Rebind)           // D-3 换绑手机/邮箱
			mAuthed.POST("/oauth/unbind", d.MerchantAuth.OAuthUnbind)  // F-10 解绑第三方账号
			mAuthed.PUT("/password", d.MerchantCenter.ChangePassword)
			// 代付（C3 商户端）
			mAuthed.GET("/transfers", d.MerchantCenter.Transfers)
			mAuthed.POST("/transfer", d.MerchantCenter.TransferCreate)
			// 保证金 / 购买会员（D3 增值）
			mAuthed.GET("/features", d.MerchantCenter.Features) // 商户端全局功能开关（导航/页面守卫用）
			mAuthed.GET("/deposit", d.MerchantCenter.DepositInfo)
			mAuthed.POST("/deposit/recharge", d.MerchantCenter.DepositRecharge)
			mAuthed.POST("/deposit/withdraw", d.MerchantCenter.DepositWithdraw)
			mAuthed.GET("/groups", d.MerchantCenter.GroupPlans)
			mAuthed.POST("/groups/buy", d.MerchantCenter.BuyGroup)
			mAuthed.POST("/recharge", d.MerchantCenter.Recharge)
			mAuthed.GET("/cert", d.MerchantCenter.CertInfo)
			mAuthed.POST("/cert", d.MerchantCenter.CertSubmit)
			// 实名认证证件识别：上传营业执照/身份证 → OCR 回填公司名/姓名/证件号
			if d.OCR != nil {
				mAuthed.POST("/ocr/license", d.OCR.License)
				mAuthed.POST("/ocr/idcard", d.OCR.IDCard)
			}
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
			// 服务商通道商户进件自助（epay 精仿线：选服务商通道进件拿自己的子商户号，非二清）
			if d.ChannelEnroll != nil {
				mAuthed.GET("/channel-enrolls", d.ChannelEnroll.MyList)
				mAuthed.GET("/channel-enrolls/channels", d.ChannelEnroll.MyChannels) // 可进件的服务商通道选项（须在 :id 前）
				mAuthed.GET("/channel-enrolls/:id", d.ChannelEnroll.MyGet)
				mAuthed.POST("/channel-enrolls", d.ChannelEnroll.MyCreate)
				mAuthed.POST("/channel-enrolls/:id/media", d.ChannelEnroll.MyUploadMedia) // 上传资料图片换 media_id
				mAuthed.POST("/channel-enrolls/:id/video", d.ChannelEnroll.MyUploadVideo) // 上传资料视频换 media_id
				mAuthed.POST("/channel-enrolls/:id/material", d.ChannelEnroll.MyFillMaterial)
				mAuthed.POST("/channel-enrolls/:id/submit", d.ChannelEnroll.MySubmit) // 提交微信进件（applyment4sub）
				mAuthed.POST("/channel-enrolls/:id/sync", d.ChannelEnroll.MySync)     // 拉取微信进件状态
				mAuthed.POST("/channel-enrolls/:id/toggle", d.ChannelEnroll.MyToggle)   // 支付开关：启停已开通渠道
				mAuthed.DELETE("/channel-enrolls/:id", d.ChannelEnroll.MyDelete)       // 删除进件单（提交前放弃，仅草稿/被驳回）
			}
			// 消费者投诉2.0 商户端自助（自研扩展，挂进件线；只看/处理自己名下投诉，无回调地址/对账运维）
			if d.MerchantComplaint != nil {
				mAuthed.GET("/complaints", d.MerchantComplaint.MyList)
				mAuthed.POST("/complaints/upload", d.MerchantComplaint.MyUploadImage) // 反馈图片上传（须在 :id 前）
				mAuthed.GET("/complaints/:id", d.MerchantComplaint.MyDetail)
				mAuthed.GET("/complaints/:id/history", d.MerchantComplaint.MyHistory)
				mAuthed.POST("/complaints/:id/sync", d.MerchantComplaint.MySync)
				mAuthed.POST("/complaints/:id/reply", d.MerchantComplaint.MyReply)
				mAuthed.POST("/complaints/:id/complete", d.MerchantComplaint.MyComplete)
				mAuthed.POST("/complaints/:id/refund", d.MerchantComplaint.MyUpdateRefund)
				mAuthed.POST("/complaints/:id/immediate", d.MerchantComplaint.MyReplyImmediate)
			}
		}
	}

	// 公开聚合收款页（对齐 epay paypage/index.php，无需登录，靠加密 merchant 标识）
	paypage := api.Group("/paypage")
	{
		paypage.GET("/info", d.Paypage.Info)
		paypage.POST("/submit", d.Paypage.Submit)
	}

	// 客户自助进件公开页（/enroll/:code，免登录，靠邀请 code；自研扩展，epay 无）。
	// 图形验证码防脚本 + 限流防刷：落地/验证码宽松，提交严格（每 IP 5 分钟最多 10 次）。
	if d.EnrollPublic != nil {
		enrollPub := api.Group("/enroll")
		{
			enrollPub.GET("/captcha", d.EnrollPublic.Captcha)
			enrollPub.GET("/:code", d.EnrollPublic.Info)
			enrollPub.POST("/:code", middleware.RateLimit(10, 5*time.Minute), d.EnrollPublic.Submit)
		}
	}

	// 代理控制台（平台运营视角，管所有代理进件；自研扩展，epay 无）。
	// 前端 admin 与 console 共用 admin_token，故走 admin scope 鉴权。
	console := api.Group("/console")
	console.Use(middleware.Auth(d.JWT, "admin"))
	// 控制台平台运营写操作自动埋点（scope=console，复用同表；仅 POST/PUT/DELETE 记，GET 放行）。
	console.Use(middleware.ConsoleOpLog(d.OpLogSvc))
	{
		// 权限点清单（供平台勾选）
		console.GET("/agent-permissions", d.Console.Permissions)
		// 代理管理
		console.GET("/agents", d.Console.ListAgents)
		console.POST("/agents", d.Console.CreateAgent)
		console.GET("/agents/:id", d.Console.GetAgent)
		console.PUT("/agents/:id", d.Console.UpdateAgent)
		console.PUT("/agents/:id/status", d.Console.SetAgentStatus)
		console.PUT("/agents/:id/permissions", d.Console.SetAgentPermissions)
		console.DELETE("/agents/:id", d.Console.DeleteAgent)
		// 名额管理
		console.GET("/agents/:id/quota", d.Console.AgentWallet)
		console.POST("/agents/:id/quota", d.Console.AdjustQuota)
		console.GET("/quota-logs", d.Console.QuotaLogs)
		// 进件申请
		console.GET("/enrolls", d.Console.ListEnrolls)
		console.POST("/enrolls", d.Console.CreateEnroll)
		console.GET("/enrolls/:id", d.Console.GetEnroll)
		console.POST("/enrolls/:id/submit", d.Console.SubmitEnroll)
		console.POST("/enrolls/:id/sync", d.Console.SyncEnroll)
		console.POST("/enrolls/:id/refund", d.Console.RefundEnroll)
		console.GET("/enrolls/:id/material", d.Console.GetMaterial)
		console.POST("/enrolls/:id/material", d.Console.FillMaterial)
		console.POST("/enrolls/:id/media", d.Console.UploadMedia)
		console.POST("/enrolls/:id/video", d.Console.UploadVideo)
		// 进件填料证件识别：执照/身份证 OCR 回填
		if d.OCR != nil {
			console.POST("/ocr/license", d.OCR.License)
			console.POST("/ocr/idcard", d.OCR.IDCard)
		}
		// 结算账户管理（进件成功后售后）
		console.GET("/enrolls/:id/settlement", d.Console.GetSettlement)
		console.POST("/enrolls/:id/settlement", d.Console.ModifySettlement)
		console.GET("/enrolls/:id/settlement/application", d.Console.GetSettlementApplication)
		// 邀请链接
		console.GET("/enroll-invites", d.Console.ListInvites)
		console.POST("/enroll-invites", d.Console.CreateInvite)
		console.PUT("/enroll-invites/:id/status", d.Console.SetInviteStatus)
		console.DELETE("/enroll-invites/:id", d.Console.DeleteInvite)
		// 佣金结算
		console.GET("/enroll-settlements", d.Console.ListSettlements)

		// 微信服务商凭证（脱敏读 / 保存；私钥·公钥落 secrets/ 文件，不入库不对外）。
		console.GET("/wx-partner", d.Console.GetWxPartner)
		console.PUT("/wx-partner", d.Console.SaveWxPartner)

		// 审计日志三类（平台在控制台查看；复用 pay_oplog 表按 scope 区分）：
		// 代理操作日志（scope=agent，代理在 /agent 端写操作）。
		console.GET("/agent-oplogs", d.OpLog.AgentList)
		console.GET("/agent-oplogs/options", d.OpLog.AgentOptions)
		console.GET("/agent-oplogs/export", d.OpLog.AgentExport)
		// 管理日志（scope=console，平台运营在 /console 的写操作）。
		console.GET("/oplogs", d.OpLog.ConsoleList)
		console.GET("/oplogs/options", d.OpLog.ConsoleOptions)
		console.GET("/oplogs/export", d.OpLog.ConsoleExport)
		// 运维日志（scope=system，系统事件：提交微信/开通/驳回/名额三态/超时关单等）。
		console.GET("/system-oplogs", d.OpLog.SystemList)
		console.GET("/system-oplogs/options", d.OpLog.SystemOptions)
		console.GET("/system-oplogs/export", d.OpLog.SystemExport)
	}

	// 独立代理端（代理只看/只碰自己名下；自研扩展，epay 无）。
	// 登录公开签发 scope=agent 的独立 token（与 admin/merchant token 互不通用）；
	// 其余接口走 agent scope 鉴权，agent_id 从 token 取，写操作按 permissions 门控。
	agent := api.Group("/agent")
	{
		agent.POST("/login", d.Agent.Login)

		authed := agent.Group("")
		authed.Use(middleware.Auth(d.JWT, "agent"))
		authed.Use(middleware.AgentOpLog(d.OpLogSvc)) // 代理写操作自动埋点（scope=agent）
		{
			authed.GET("/profile", d.Agent.Profile)
			authed.GET("/permissions", d.Agent.Permissions)
			// 名额钱包
			authed.GET("/quota", d.Agent.Wallet)
			authed.GET("/quota-logs", d.Agent.QuotaLogs)
			// 进件申请
			authed.GET("/enrolls", d.Agent.ListEnrolls)
			authed.POST("/enrolls", d.Agent.CreateEnroll)
			authed.GET("/enrolls/:id", d.Agent.GetEnroll)
			authed.POST("/enrolls/:id/submit", d.Agent.SubmitEnroll)
			authed.POST("/enrolls/:id/sync", d.Agent.SyncEnroll)
			authed.POST("/enrolls/:id/refund", d.Agent.RefundEnroll)
			authed.GET("/enrolls/:id/material", d.Agent.GetMaterial)
			authed.POST("/enrolls/:id/material", d.Agent.FillMaterial)
			authed.POST("/enrolls/:id/media", d.Agent.UploadMedia)
			authed.POST("/enrolls/:id/video", d.Agent.UploadVideo)
			// 进件填料证件识别：执照/身份证 OCR 回填
			if d.OCR != nil {
				authed.POST("/ocr/license", d.OCR.License)
				authed.POST("/ocr/idcard", d.OCR.IDCard)
			}
			// 结算账户管理（settle_account 权限）
			authed.GET("/enrolls/:id/settlement", d.Agent.GetSettlement)
			authed.POST("/enrolls/:id/settlement", d.Agent.ModifySettlement)
			authed.GET("/enrolls/:id/settlement/application", d.Agent.GetSettlementApplication)
			// 邀请链接
			authed.GET("/enroll-invites", d.Agent.ListInvites)
			authed.POST("/enroll-invites", d.Agent.CreateInvite)
			authed.PUT("/enroll-invites/:id/status", d.Agent.SetInviteStatus)
			authed.DELETE("/enroll-invites/:id", d.Agent.DeleteInvite)
			// 佣金结算
			authed.GET("/enroll-settlements", d.Agent.ListSettlements)
		}
	}
}
