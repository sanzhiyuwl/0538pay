package middleware

import (
	"sync"
	"time"

	"github.com/epvia/api/pkg/resp"
	"github.com/gin-gonic/gin"
)

// 内存滑动窗口限流（按 IP）。用于公开无鉴权接口（如客户自助进件公开页）防刷/防爆破，
// 无外部依赖、单机足够。多实例部署需换 Redis，届时替换本实现即可，调用处不变。

// ipLimiter 一个限流器实例：window 时间窗内每 IP 最多 max 次。
type ipLimiter struct {
	mu     sync.Mutex
	hits   map[string][]int64 // ip -> 命中时间戳（纳秒）列表
	max    int
	window time.Duration
}

// visited 记录一次访问并判断是否超限。返回 true=放行，false=超限。
func (l *ipLimiter) visited(ip string) bool {
	now := time.Now().UnixNano()
	cutoff := now - int64(l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	// 保留窗口内的时间戳
	kept := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t > cutoff {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.hits[ip] = kept
		return false
	}
	l.hits[ip] = append(kept, now)
	return true
}

// gc 周期清理空/过期条目，避免 map 无限膨胀。
func (l *ipLimiter) gc() {
	for range time.Tick(l.window) {
		now := time.Now().UnixNano()
		cutoff := now - int64(l.window)
		l.mu.Lock()
		for ip, ts := range l.hits {
			alive := ts[:0]
			for _, t := range ts {
				if t > cutoff {
					alive = append(alive, t)
				}
			}
			if len(alive) == 0 {
				delete(l.hits, ip)
			} else {
				l.hits[ip] = alive
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit 返回一个按 IP 限流的中间件：window 时间窗内每 IP 最多 max 次，超限回 429。
// 每次调用创建独立限流器（各接口互不干扰）；后台启动一个 goroutine 定期 gc。
func RateLimit(max int, window time.Duration) gin.HandlerFunc {
	l := &ipLimiter{hits: make(map[string][]int64), max: max, window: window}
	go l.gc()
	return func(c *gin.Context) {
		if !l.visited(c.ClientIP()) {
			resp.Abort(c, 429, 429, "操作过于频繁，请稍后再试")
			return
		}
		c.Next()
	}
}
