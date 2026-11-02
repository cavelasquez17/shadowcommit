package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiter is a per-key fixed-window request counter. Hand-rolled instead of pulling
// in a dependency: at this scale it's arithmetic, not infrastructure.
type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	buckets  map[string]*bucket
	stop     chan struct{}
	stopOnce sync.Once
}

type bucket struct {
	count   int
	resetAt time.Time
}

// newRateLimiter allows up to limit requests per key within window, and starts a
// background sweep that evicts expired buckets so the map doesn't grow forever. Call
// Stop to shut the sweep down.
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]*bucket),
		stop:    make(chan struct{}),
	}
	go rl.sweep()
	return rl
}

func (rl *rateLimiter) sweep() {
	t := time.NewTicker(rl.window)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			now := time.Now()
			rl.mu.Lock()
			for k, b := range rl.buckets {
				if now.After(b.resetAt) {
					delete(rl.buckets, k)
				}
			}
			rl.mu.Unlock()
		case <-rl.stop:
			return
		}
	}
}

// Stop halts the background sweep. Safe to call more than once.
func (rl *rateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stop) })
}

func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.resetAt) {
		rl.buckets[key] = &bucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

// middleware 429s once a client IP exceeds the limiter's budget for the current window.
func (rl *rateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			fail(c, http.StatusTooManyRequests, "rate_limited",
				"Slow down. The machine took your job, not your patience.")
			return
		}
		c.Next()
	}
}
