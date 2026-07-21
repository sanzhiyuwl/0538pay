package service

import (
	"context"
	"time"
)

// notifyBackoff 返回第 n 次重试距离上次的时间间隔（对齐 epay cron notify 的退避档位）。
// epay：notify=2→2min、3→16min、4→36min、5→1hour，超过则放弃(notify=-1)。
// 我们把首次失败记为 notify=1（1min 后重试），后续 2~5 沿用 epay 档位。
// 返回 0 表示不应再重试（放弃）。
func notifyBackoff(n int) time.Duration {
	switch n {
	case 1:
		return 1 * time.Minute
	case 2:
		return 2 * time.Minute
	case 3:
		return 16 * time.Minute
	case 4:
		return 36 * time.Minute
	case 5:
		return 1 * time.Hour
	default:
		return 0 // 放弃
	}
}

// maxNotifyRetry 最大重试计数，达到后置 notify=-1 放弃（对齐 epay 到 5 后 -1）。
const maxNotifyRetry = 5

// RetryNotify 扫一批待重试通知的订单并重发（供定时任务调用）。
// 对齐 epay cron do=notify：取 notify>0 且到期 且近一天完成的订单，重发商户通知：
//   - 成功 → notify=0 清零；
//   - 失败 → 递增重试计数并按退避表安排下次；超过上限 → notify=-1 放弃。
// 返回本次处理条数（成功+失败），便于日志与测试断言。
func (s *PayService) RetryNotify(ctx context.Context, limit int) (int, error) {
	orders, err := s.orders.FindNotifyPending(limit)
	if err != nil {
		return 0, err
	}
	for i := range orders {
		o := &orders[i]
		m, err := s.merchants.FindByUID(o.UID)
		if err != nil || m == nil {
			continue
		}
		params := buildCallbackParams(o, m)
		if doNotify(ctx, appendQuery(o.NotifyURL, params)) {
			_ = s.orders.SetNotifySuccess(o.TradeNo)
			continue
		}
		// 失败：递增重试计数
		next := int(o.Notify) + 1
		if next > maxNotifyRetry {
			_ = s.orders.SetNotifyRetry(o.TradeNo, 0, time.Time{}) // n<=0 → notify=-1 放弃
			continue
		}
		_ = s.orders.SetNotifyRetry(o.TradeNo, next, time.Now().Add(notifyBackoff(next)))
	}
	return len(orders), nil
}

// CleanExpiredOrders 清理超时未支付订单（对齐 epay cron do=order 的删除逻辑）。
// before 之前创建且仍未支付的订单直接删除（无资金影响）。返回清理条数。
func (s *PayService) CleanExpiredOrders(before time.Time) (int64, error) {
	return s.orders.CleanExpiredUnpaid(before)
}

// ReconcileUnpaid 定时对账：扫最近窗口内的未支付真实渠道订单，主动向渠道查单，
// 渠道确认已付则走 QueryStatus 的改单+入账(幂等)。返回处理条数。
// 复用 QueryStatus，避免与回调/收银台轮询逻辑分叉。
func (s *PayService) ReconcileUnpaid(ctx context.Context, since time.Time, limit int) (int, error) {
	orders, err := s.orders.FindUnpaidForReconcile(since, limit)
	if err != nil {
		return 0, err
	}
	for i := range orders {
		_, _ = s.QueryStatus(ctx, orders[i].TradeNo)
	}
	return len(orders), nil
}
