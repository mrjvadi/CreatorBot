// Package middleware — rate limiting برای webhook-gateway.
// هر bot_id حداکثر N request در ثانیه می‌تواند داشته باشد.
// از token bucket algorithm استفاده می‌کند.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// bucket — token bucket برای یک bot.
type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

// BotRateLimiter rate limiter به ازای هر bot_id.
type BotRateLimiter struct {
	rate     float64 // توکن در ثانیه
	capacity float64 // ظرفیت bucket
	buckets  sync.Map
}

// NewBotRateLimiter یک rate limiter جدید می‌سازد.
// rate = تعداد request مجاز در ثانیه
// burst = حداکثر request های همزمان
func NewBotRateLimiter(ratePerSec, burst float64) *BotRateLimiter {
	r := &BotRateLimiter{
		rate:     ratePerSec,
		capacity: burst,
	}
	// cleanup کردن bucket های قدیمی هر ۵ دقیقه
	go r.cleanup()
	return r
}

func (r *BotRateLimiter) allow(botID string) bool {
	now := time.Now()

	val, _ := r.buckets.LoadOrStore(botID, &bucket{
		tokens:   r.capacity,
		lastSeen: now,
	})
	b := val.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	// refill tokens
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = min(r.capacity, b.tokens+elapsed*r.rate)
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func (r *BotRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		cutoff := time.Now().Add(-10 * time.Minute)
		r.buckets.Range(func(k, v any) bool {
			b := v.(*bucket)
			b.mu.Lock()
			old := b.lastSeen.Before(cutoff)
			b.mu.Unlock()
			if old {
				r.buckets.Delete(k)
			}
			return true
		})
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// WebhookRateLimit middleware برای محدود کردن webhook ها.
// هر bot حداکثر 30 req/sec با burst 100 دارد.
func WebhookRateLimit() gin.HandlerFunc {
	limiter := NewBotRateLimiter(30, 100)

	return func(c *gin.Context) {
		// bot_id از path param
		botID := c.Param("bot_id")
		if botID == "" {
			// از IP اگه bot_id نبود
			botID = c.ClientIP()
		}

		if !limiter.allow(botID) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"ok":          false,
				"description": "Too many requests",
				"retry_after": 1,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GlobalRateLimit محدودیت کلی برای جلوگیری از DDoS.
// حداکثر 1000 req/sec کل
func GlobalRateLimit() gin.HandlerFunc {
	limiter := NewBotRateLimiter(1000, 5000)

	return func(c *gin.Context) {
		if !limiter.allow("global") {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"ok":          false,
				"description": "Service temporarily unavailable",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
