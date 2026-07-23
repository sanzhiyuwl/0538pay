package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/0538pay/api/internal/config"
	"github.com/0538pay/api/internal/handler"
	"github.com/0538pay/api/internal/model"
	"github.com/0538pay/api/internal/repository"
	"github.com/0538pay/api/internal/router"
	"github.com/0538pay/api/internal/scheduler"
	"github.com/0538pay/api/internal/service"
	"github.com/0538pay/api/pkg/jwtauth"
	"github.com/0538pay/api/pkg/sign"
	"github.com/gin-gonic/gin"

	// 支付渠道：匿名导入以触发各渠道 init() 自注册到 registry。
	_ "github.com/0538pay/api/internal/channel/alipayf2f"
	_ "github.com/0538pay/api/internal/channel/alipaypage"
	_ "github.com/0538pay/api/internal/channel/alipaywap"
	_ "github.com/0538pay/api/internal/channel/epay"
	_ "github.com/0538pay/api/internal/channel/epayn"
	_ "github.com/0538pay/api/internal/channel/mock"
	_ "github.com/0538pay/api/internal/channel/wxnative"
	_ "github.com/0538pay/api/internal/channel/wxjsapi"
	_ "github.com/0538pay/api/internal/channel/wxh5"
)

func main() {
	configPath := flag.String("config", "./configs", "配置目录路径")
	flag.Parse()

	// 1. 配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 数据库 + 自动建表
	db, err := model.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("自动建表失败: %v", err)
	}

	// 3. 依赖装配（repo → service → handler）
	jm := jwtauth.New(cfg.JWT.Secret, cfg.JWT.ExpireHours)

	adminRepo := repository.NewAdminRepo(db)
	orderRepo := repository.NewOrderRepo(db)
	merchantRepo := repository.NewMerchantRepo(db)
	accountRepo := repository.NewAccountRepo(db)
	channelRepo := repository.NewChannelRepo(db)

	groupRepo := repository.NewGroupRepo(db)

	// 系统配置域：全量加载进内存缓存，业务服务的常量由它刷新（对齐 epay pre_config）。
	configSvc := service.NewConfigService(repository.NewConfigRepo(db))
	if err := configSvc.Load(); err != nil {
		log.Fatalf("加载系统配置失败: %v", err)
	}
	service.ApplyConfig(configSvc)                          // 初始刷新业务常量
	configSvc.OnChange(func() { service.ApplyConfig(configSvc) }) // 设置保存后自动刷新

	authSvc := service.NewAuthService(adminRepo, jm)
	orderSvc := service.NewOrderService(orderRepo)
	merchantSvc := service.NewMerchantService(merchantRepo, accountRepo, groupRepo)
	groupSvc := service.NewGroupService(groupRepo, merchantRepo)
	channelSvc := service.NewChannelService(channelRepo)
	// 通道轮询组 / 子通道服务（repo 已在选通道分发器处创建，复用）。
	paySvc := service.NewPayService(merchantRepo, orderRepo, accountRepo, channelRepo)
	paySvc.SetConfigService(configSvc) // 注入配置：V2回调平台私钥RSA签名/全局金额限额/mode=1加费兜底/随机金额
	// 选通道分发：用户组通道分配 / 轮询组 / 子通道 / 组级费率覆盖（对齐 epay getSubmitInfo）。
	rollRepo := repository.NewRollRepo(db)
	subChannelRepo := repository.NewSubChannelRepo(db)
	paySvc.SetSelector(service.NewChannelSelector(channelRepo, rollRepo, subChannelRepo, groupRepo))
	rollSvc := service.NewRollService(rollRepo, channelRepo)
	subChannelSvc := service.NewSubChannelService(subChannelRepo, channelRepo, merchantRepo)
	payTypeSvc := service.NewPayTypeService(repository.NewPayTypeRepo(db), channelRepo)
	weixinSvc := service.NewWeixinService(repository.NewWeixinRepo(db))
	weworkSvc := service.NewWeworkService(repository.NewWeworkRepo(db))
	merchantSvc.SetSubChannelRepo(subChannelRepo) // 删商户级联删子通道
	channelSvc.SetSubChannelRepo(subChannelRepo)  // 删主通道级联删子通道
	channelSvc.SetOrderRepo(orderRepo)            // 通道列表今昨收款/成功率实时聚合
	channelSvc.SetPayService(paySvc) // 后台测试支付定向下测试单走收单链
	orderSvc.SetWriteDeps(accountRepo, channelRepo, adminRepo, paySvc) // 订单写操作依赖
	settleRepo := repository.NewSettleRepo(db)
	settleSvc := service.NewSettleService(settleRepo, merchantRepo)
	settleSvc.SetAdminRepo(adminRepo) // 删除退回需管理员支付密码二次校验
	settleSvc.SetConfigService(configSvc) // C-4 银行导出取 transfer_desc 打款备注
	recordRepo := repository.NewRecordRepo(db)
	recordSvc := service.NewRecordService(recordRepo)
	transferRepo := repository.NewTransferRepo(db)
	transferSvc := service.NewTransferService(transferRepo, merchantRepo, adminRepo)
	transferSvc.SetOrderRepo(orderRepo) // A-8：settle_type=1 代付可用余额扣当日已收 realmoney
	profitRepo := repository.NewProfitRepo(db)
	profitSvc := service.NewProfitService(profitRepo, channelRepo)
	profitSvc.SetMerchantRepo(merchantRepo) // 分账规则管理校验绑定商户存在性（C-1）
	paySvc.SetProfitService(profitSvc) // 注入分账：下单匹配规则 + 支付成功建分账单
	riskSvc := service.NewRiskService(repository.NewRiskRepo(db))
	blacklistRepo := repository.NewBlacklistRepo(db)
	blacklistSvc := service.NewBlacklistService(blacklistRepo)
	domainSvc := service.NewDomainService(repository.NewDomainRepo(db))
	paySvc.SetRiskServices(riskSvc, blacklistSvc, domainSvc) // 注入下单拦截：黑名单/域名白名单/关键词风控
	statSvc := service.NewStatService(repository.NewStatRepo(db), channelRepo)
	logSvc := service.NewLogService(repository.NewLogRepo(db))
	inviteSvc := service.NewInviteService(repository.NewInviteRepo(db))
	siteConfigSvc := service.NewSiteConfigService(repository.NewSiteConfigRepo(db))
	merchantAuthSvc := service.NewMerchantAuthService(merchantRepo, jm)
	authSvc.SetLogService(logSvc)         // 后台登录写日志
	merchantAuthSvc.SetLogService(logSvc) // 商户登录写日志
	// 商户自助流程：图形验证码 + 注册/完善资料/找回密码（复用 invite 核销 + config reg 分组）。
	captchaSvc := service.NewCaptchaService()
	merchantRegSvc := service.NewMerchantRegService(merchantRepo, configSvc, inviteSvc, captchaSvc)
	merchantCenterSvc := service.NewMerchantCenterService(
		merchantRepo, orderRepo, recordRepo, settleRepo, accountRepo, channelRepo, groupRepo, paySvc,
	)
	merchantCenterSvc.SetCertVerify(service.NewCertVerifyService(configSvc)) // 实名第三方核验
	// 邀请返现：支付成功钩子返现到上级余额 + 统计（对齐 epay invite.php + functions.php）。
	inviteRewardSvc := service.NewInviteRewardService(merchantRepo, accountRepo, recordRepo, configSvc)
	inviteRewardSvc.SetGroupRepo(groupRepo) // 邀请返现读上级所在组的组级 invite_* 覆盖
	paySvc.SetInviteReward(inviteRewardSvc)
	// 商户中心其余自助流程：测试支付 + 聚合收款码 + 公开收款页（复用 CreateInternalOrder 走收单链）。
	merchantSelfSvc := service.NewMerchantSelfService(
		merchantRepo, channelRepo, repository.NewPayTypeRepo(db), paySvc, configSvc,
	)
	// 站内信（我方新增，epay 无此实体）。
	messageSvc := service.NewMessageService(repository.NewMessageRepo(db))
	// 后台仪表盘全平台聚合（对齐 epay admin/index.php + ajax getcount）。
	dashboardSvc := service.NewDashboardService(repository.NewDashboardRepo(db), repository.NewDomainRepo(db), profitRepo)
	dashboardSvc.SetAdminRepo(adminRepo) // 弱密码/默认密码安全告警
	// 网站公告（对齐 epay gonggao.php）。
	announceSvc := service.NewAnnounceService(repository.NewAnnounceRepo(db))
	// 数据清理（对齐 epay clean.php）。
	cleanSvc := service.NewCleanService(db)
	// 风控自动关停（对齐 epay cron do=check）。
	riskAutoSvc := service.NewRiskAutoService(repository.NewRiskAutoRepo(db), configSvc)
	// 商户 handler 单列（SSO 需注入 JWT）。
	merchantHandler := handler.NewMerchantHandler(merchantSvc)
	merchantHandler.SetJWT(jm)

	// 定时任务调度器（先建，供 cron 手动触发端点复用）。
	regCodeRepo := repository.NewRegCodeRepo(db)
	sch := scheduler.New(paySvc, settleSvc)
	sch.SetProfit(profitSvc)      // 分账自动执行（对齐 epay cron do=profitsharing）
	sch.SetRiskAuto(riskAutoSvc)  // 风控自动关停（对齐 epay cron do=check）
	sch.SetMaintenanceRepos(regCodeRepo, blacklistRepo) // B-7 清理过期验证码/黑名单
	sch.SetChannelRepo(channelRepo) // 单日限额 daystatus 每日重置（对齐 epay cron.php:152）
	// V2 REST 接口族（mapi）：统一验签 + 回包 RSA 签名，复用 Pay/Transfer 核心。
	refundOrderRepo := repository.NewRefundOrderRepo(db)
	if err := configSvc.EnsurePlatformKeys(sign.GenerateRSAKeyPair); err != nil {
		log.Fatalf("初始化平台 RSA 密钥失败: %v", err)
	}
	mapiSvc := service.NewMapiService(merchantRepo, orderRepo, refundOrderRepo, accountRepo, channelRepo, configSvc, paySvc, transferSvc)
	apiV1Svc := service.NewApiV1Service(merchantRepo, orderRepo, settleRepo, configSvc, mapiSvc) // A-5 V1 兼容层

	merchantAuthHandler := handler.NewMerchantAuthHandler(merchantAuthSvc)
	merchantAuthHandler.SetRegService(merchantRegSvc, captchaSvc) // 注入自助流程 + 图形验证码
	// 快捷登录 OAuth（QQ/微信/支付宝，复用密码/密钥登录做绑定校验）。
	oauthSvc := service.NewOAuthService(merchantRepo, configSvc, jm, merchantAuthSvc)
	merchantAuthHandler.SetOAuthService(oauthSvc)
	// 短信 OTP + 极验滑块（与图形验证码并存，凭证到位即真发/真校验）。
	smsSvc := service.NewSmsService(regCodeRepo, configSvc)
	geetestSvc := service.NewGeetestService(configSvc)
	merchantAuthHandler.SetSmsGeetest(smsSvc, geetestSvc)

	deps := router.Deps{
		JWT:            jm,
		Auth:           handler.NewAuthHandler(authSvc),
		Admin:          handler.NewAdminHandler(service.NewAdminService(adminRepo)),
		Order:          handler.NewOrderHandler(orderSvc),
		Merchant:       merchantHandler,
		Group:          handler.NewGroupHandler(groupSvc),
		Config:         handler.NewConfigHandler(configSvc),
		Channel:        handler.NewChannelHandler(channelSvc),
		Roll:           handler.NewRollHandler(rollSvc),
		SubChannel:     handler.NewSubChannelHandler(subChannelSvc),
		PayType:        handler.NewPayTypeHandler(payTypeSvc),
		Weixin:         handler.NewWeixinHandler(weixinSvc),
		Wework:         handler.NewWeworkHandler(weworkSvc),
		Pay:            handler.NewPayHandler(paySvc),
		Settle:         handler.NewSettleHandler(settleSvc),
		Record:         handler.NewRecordHandler(recordSvc),
		Transfer:       handler.NewTransferHandler(transferSvc),
		Profit:         handler.NewProfitHandler(profitSvc),
		Risk:           handler.NewRiskHandler(riskSvc),
		Blacklist:      handler.NewBlacklistHandler(blacklistSvc),
		Domain:         handler.NewDomainHandler(domainSvc),
		Stat:           handler.NewStatHandler(statSvc),
		Log:            handler.NewLogHandler(logSvc),
		Invite:         handler.NewInviteHandler(inviteSvc),
		SiteConfig:     handler.NewSiteConfigHandler(siteConfigSvc),
		MerchantAuth: merchantAuthHandler,
		MerchantCenter: handler.NewMerchantCenterHandler(
			merchantCenterSvc, orderSvc, recordSvc, transferSvc,
			merchantSelfSvc, inviteRewardSvc, domainSvc, messageSvc, configSvc,
		),
		Mapi:      handler.NewMapiHandler(mapiSvc),
		ApiV1:     handler.NewApiV1Handler(apiV1Svc),
		Paypage:   handler.NewPaypageHandler(merchantSelfSvc),
		Message:   handler.NewMessageHandler(messageSvc),
		Dashboard: handler.NewDashboardHandler(dashboardSvc),
		Announce:  handler.NewAnnounceHandler(announceSvc),
		Clean:     handler.NewCleanHandler(cleanSvc),
		Cron:      handler.NewCronHandler(sch, configSvc),
	}

	// 4. 定时任务（阶段E）：通知重试 + 对账 + 超时关单 + 自动结算 + 分账 + 风控。
	sch.Start()

	// 5. 路由 + 启动（HTTP 起独立协程，主协程等信号做优雅停机）。
	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()
	router.Setup(r, deps)

	srv := &http.Server{Addr: cfg.Server.Addr, Handler: r}
	go func() {
		log.Printf("0538pay-api 启动于 %s", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 等待 SIGINT/SIGTERM，先停调度器再停 HTTP，做到优雅退出。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Printf("收到停止信号，正在优雅关闭…")
	sch.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP 优雅关闭超时: %v", err)
	}
	log.Printf("已退出")
}
