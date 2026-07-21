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

	"github.com/0538pay/api/internal/service"
)

// Scheduler 定时任务调度器。
type Scheduler struct {
	pay    *service.PayService
	settle *service.SettleService
	cancel context.CancelFunc
	done   chan struct{}

	// 可配置的间隔与批量（起步用固定默认值，后续可接配置）。
	notifyInterval    time.Duration
	reconcileInterval time.Duration
	cleanupInterval   time.Duration
	settleInterval    time.Duration
	batchLimit        int
	settleLimit       int
	reconcileWindow   time.Duration
	orderExpiry       time.Duration // 未支付订单超时时长，超过则清理
}

// New 创建调度器，使用对齐 epay 语义的默认参数。
func New(pay *service.PayService, settle *service.SettleService) *Scheduler {
	return &Scheduler{
		pay:               pay,
		settle:            settle,
		notifyInterval:    1 * time.Minute,  // 每分钟扫一次待重试通知
		reconcileInterval: 5 * time.Minute,  // 每 5 分钟对账一次未支付单
		cleanupInterval:   1 * time.Hour,    // 每小时清理一次超时未付单
		settleInterval:    1 * time.Hour,    // 每小时检查一次自动结算（服务层每日只跑一次）
		batchLimit:        20,               // 每批处理条数，对齐 epay limit=20
		settleLimit:       100,              // 单次自动结算处理商户数上限
		reconcileWindow:   30 * time.Minute, // 只对账最近 30 分钟创建的未支付单
		orderExpiry:       24 * time.Hour,   // 未付超 24 小时清理，对齐 epay
	}
}

// Start 启动后台任务协程。非阻塞。
func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.run(ctx)
	log.Printf("[scheduler] 已启动：通知重试每 %s、对账每 %s、超时关单每 %s、自动结算每 %s", s.notifyInterval, s.reconcileInterval, s.cleanupInterval, s.settleInterval)
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
	reconcileTicker := time.NewTicker(s.reconcileInterval)
	cleanupTicker := time.NewTicker(s.cleanupInterval)
	settleTicker := time.NewTicker(s.settleInterval)
	defer notifyTicker.Stop()
	defer reconcileTicker.Stop()
	defer cleanupTicker.Stop()
	defer settleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-notifyTicker.C:
			s.runNotify(ctx)
		case <-reconcileTicker.C:
			s.runReconcile(ctx)
		case <-cleanupTicker.C:
			s.runCleanup()
		case <-settleTicker.C:
			s.runSettle(ctx)
		}
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
}
