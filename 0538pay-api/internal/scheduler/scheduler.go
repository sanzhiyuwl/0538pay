// Package scheduler 提供轻量的定时任务驱动（ticker），承接阶段 E 的后台任务：
//   - 商户通知重试（RetryNotify）：扫到期的待重试订单重发回调。
//   - 未支付订单对账（ReconcileUnpaid）：主动向渠道查单兜底回调遗漏。
//
// 设计：单进程内 goroutine + time.Ticker，Start 起协程、Stop 优雅退出（context 取消）。
// 不引第三方 cron 库；任务本身幂等，重启不会重复入账（靠 MarkPaid 条件 UPDATE）。
package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/epvia/api/internal/repository"
	"github.com/epvia/api/internal/service"
)

// Scheduler 定时任务调度器。
type Scheduler struct {
	pay      *service.PayService
	settle   *service.SettleService
	profit   *service.ProfitService   // 分账自动执行（可空）
	riskAuto *service.RiskAutoService // 风控自动关停（可空）
	regcodes *repository.RegCodeRepo  // 过期验证码清理（可空，B-7）
	blacks   *repository.BlacklistRepo // 过期黑名单清理（可空，B-7）
	channels *repository.ChannelRepo   // 单日限额 daystatus 每日重置（可空）
	cancel   context.CancelFunc
	done     chan struct{}

	lastDayReset string // 上次执行 daystatus 重置的日期（YYYY-MM-DD），跨天触发一次

	// 可配置的间隔与批量（起步用固定默认值，后续可接配置）。
	notifyInterval    time.Duration
	notify2Interval   time.Duration
	reconcileInterval time.Duration
	cleanupInterval   time.Duration
	settleInterval    time.Duration
	profitInterval    time.Duration
	riskInterval      time.Duration
	batchLimit        int
	settleLimit       int
	profitLimit       int
	reconcileWindow   time.Duration
	orderExpiry       time.Duration // 未支付订单超时时长，超过则清理
}

// New 创建调度器，使用对齐 epay 语义的默认参数。
func New(pay *service.PayService, settle *service.SettleService) *Scheduler {
	return &Scheduler{
		pay:               pay,
		settle:            settle,
		notifyInterval:    1 * time.Minute,  // 每分钟扫一次待重试通知
		notify2Interval:   30 * time.Minute, // 每 30 分钟兜底重发一次已放弃(notify=-1)单
		reconcileInterval: 5 * time.Minute,  // 每 5 分钟对账一次未支付单
		cleanupInterval:   1 * time.Hour,    // 每小时清理一次超时未付单
		settleInterval:    1 * time.Hour,    // 每小时检查一次自动结算（服务层每日只跑一次）
		profitInterval:    5 * time.Minute,  // 每 5 分钟自动执行一次待分账单
		riskInterval:      10 * time.Minute, // 每 10 分钟跑一次风控自动关停检查
		batchLimit:        20,               // 每批处理条数，对齐 epay limit=20
		settleLimit:       100,              // 单次自动结算处理商户数上限
		profitLimit:       50,               // 单次自动分账处理条数上限
		reconcileWindow:   30 * time.Minute, // 只对账最近 30 分钟创建的未支付单
		orderExpiry:       24 * time.Hour,   // 未付超 24 小时清理，对齐 epay
	}
}

// SetProfit 注入分账服务（自动执行待分账单）。nil 则不跑分账自动执行。
func (s *Scheduler) SetProfit(p *service.ProfitService) { s.profit = p }

// SetRiskAuto 注入风控自动关停服务。nil 则不跑风控自动检查。
func (s *Scheduler) SetRiskAuto(r *service.RiskAutoService) { s.riskAuto = r }

// SetMaintenanceRepos 注入过期记录清理仓储（B-7：清理过期验证码/黑名单，随超时关单任务一起跑）。
func (s *Scheduler) SetMaintenanceRepos(rc *repository.RegCodeRepo, bl *repository.BlacklistRepo) {
	s.regcodes, s.blacks = rc, bl
}

// SetChannelRepo 注入通道仓储（单日限额 daystatus 每日重置，对齐 epay cron.php:152）。
func (s *Scheduler) SetChannelRepo(c *repository.ChannelRepo) { s.channels = c }

// Start 启动后台任务协程。非阻塞。
func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.run(ctx)
	log.Printf("[scheduler] 已启动：通知重试每 %s、对账每 %s、超时关单每 %s、自动结算每 %s、自动分账每 %s", s.notifyInterval, s.reconcileInterval, s.cleanupInterval, s.settleInterval, s.profitInterval)
}

// Stop 优雅停止，等待当前循环退出。
func (s *Scheduler) Stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
	log.Printf("[scheduler] 已停止")
}

func (s *Scheduler) run(ctx context.Context) {
	defer close(s.done)
	notifyTicker := time.NewTicker(s.notifyInterval)
	notify2Ticker := time.NewTicker(s.notify2Interval)
	reconcileTicker := time.NewTicker(s.reconcileInterval)
	cleanupTicker := time.NewTicker(s.cleanupInterval)
	settleTicker := time.NewTicker(s.settleInterval)
	profitTicker := time.NewTicker(s.profitInterval)
	riskTicker := time.NewTicker(s.riskInterval)
	defer notifyTicker.Stop()
	defer notify2Ticker.Stop()
	defer reconcileTicker.Stop()
	defer cleanupTicker.Stop()
	defer settleTicker.Stop()
	defer profitTicker.Stop()
	defer riskTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-notifyTicker.C:
			s.runNotify(ctx)
		case <-notify2Ticker.C:
			s.runNotify2(ctx)
		case <-reconcileTicker.C:
			s.runReconcile(ctx)
		case <-cleanupTicker.C:
			s.runCleanup()
		case <-settleTicker.C:
			s.runSettle(ctx)
		case <-profitTicker.C:
			s.runProfit()
		case <-riskTicker.C:
			s.runRiskAuto()
		}
	}
}

// RunTask 手动触发一个任务（对齐 epay cron.php?act=xxx，由 cronkey 保护的对外端点调用）。
// 返回是否为已知任务。
func (s *Scheduler) RunTask(task string) bool {
	ctx := context.Background()
	switch task {
	case "notify":
		s.runNotify(ctx)
	case "notify2":
		s.runNotify2(ctx)
	case "reconcile":
		s.runReconcile(ctx)
	case "order", "cleanup":
		s.runCleanup()
	case "settle":
		s.runSettle(ctx)
	case "profitsharing", "profit":
		s.runProfit()
	case "check", "risk":
		s.runRiskAuto()
	default:
		return false
	}
	return true
}

func (s *Scheduler) runRiskAuto() {
	if s.riskAuto == nil {
		return
	}
	n, err := s.riskAuto.Run()
	if err != nil {
		log.Printf("[scheduler] 风控自动检查出错: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[scheduler] 风控自动关停 %d 个商户", n)
	}
}

func (s *Scheduler) runProfit() {
	if s.profit == nil {
		return
	}
	n, err := s.profit.AutoExecute(s.profitLimit)
	if err != nil {
		log.Printf("[scheduler] 自动分账出错: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[scheduler] 自动分账执行 %d 单", n)
	}
}

func (s *Scheduler) runSettle(ctx context.Context) {
	if s.settle == nil {
		return
	}
	n, err := s.settle.RunAutoSettle(ctx, s.settleLimit)
	if err != nil {
		log.Printf("[scheduler] 自动结算出错: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[scheduler] 自动结算生成 %d 单", n)
	}
}

func (s *Scheduler) runNotify(ctx context.Context) {
	n, err := s.pay.RetryNotify(ctx, s.batchLimit)
	if err != nil {
		log.Printf("[scheduler] 通知重试出错: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[scheduler] 通知重试处理 %d 单", n)
	}
}

func (s *Scheduler) runNotify2(ctx context.Context) {
	n, err := s.pay.RetryNotifyAbandoned(ctx, s.batchLimit)
	if err != nil {
		log.Printf("[scheduler] 兜底重发(notify2)出错: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[scheduler] 兜底重发(notify2)处理 %d 单", n)
	}
}

func (s *Scheduler) runReconcile(ctx context.Context) {
	since := time.Now().Add(-s.reconcileWindow)
	n, err := s.pay.ReconcileUnpaid(ctx, since, s.batchLimit)
	if err != nil {
		log.Printf("[scheduler] 对账出错: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[scheduler] 对账处理 %d 单", n)
	}
}

func (s *Scheduler) runCleanup() {
	before := time.Now().Add(-s.orderExpiry)
	n, err := s.pay.CleanExpiredOrders(before)
	if err != nil {
		log.Printf("[scheduler] 超时关单出错: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[scheduler] 清理超时未付单 %d 条", n)
	}
	// B-7：清理过期验证码（保留近 24h）与已过期黑名单（对齐 epay cron order 清理项）。
	now := time.Now()
	if s.regcodes != nil {
		if rn, err := s.regcodes.CleanExpired(now.Add(-24 * time.Hour)); err == nil && rn > 0 {
			log.Printf("[scheduler] 清理过期验证码 %d 条", rn)
		}
	}
	if s.blacks != nil {
		if bn, err := s.blacks.CleanExpired(now); err == nil && bn > 0 {
			log.Printf("[scheduler] 清理过期黑名单 %d 条", bn)
		}
	}
	// 单日限额每日重置：跨天后把所有 daystatus 置 0（对齐 epay cron.php:152 "update pre_channel set daystatus=0"）。
	if s.channels != nil {
		today := now.Format("2006-01-02")
		if s.lastDayReset != today {
			if n, err := s.channels.ResetAllDayStatus(); err == nil {
				s.lastDayReset = today
				if n > 0 {
					log.Printf("[scheduler] 单日限额每日重置：解除 %d 个通道暂停", n)
				}
			}
		}
	}
}
