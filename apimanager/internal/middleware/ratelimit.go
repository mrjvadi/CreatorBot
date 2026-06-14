package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter interface — پیاده‌سازی با Redis.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// RateLimit middleware ساده برای محدود کردن request در ثانیه.
// در production از Redis sliding window استفاده شود.
func RateLimit(limiter RateLimiter, maxPerMin int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}

		// key از IP + endpoint
		key := fmt.Sprintf("rl:%s:%s", c.ClientIP(), c.FullPath())

		ok, err := limiter.Allow(context.Background(), key)
		if err != nil || !ok {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"ok":         false,
				"message":    "rate limit exceeded",
				"retry_after": "60s",
			})
			return
		}
		c.Next()
	}
}

// SimpleInMemoryLimiter برای development — در production با Redis جایگزین شود.
type SimpleInMemoryLimiter struct {
	counts   map[string][]time.Time
	maxPerMin int
}

func NewSimpleLimiter(maxPerMin int) *SimpleInMemoryLimiter {
	return &SimpleInMemoryLimiter{
		counts:    make(map[string][]time.Time),
		maxPerMin: maxPerMin,
	}
}

func (l *SimpleInMemoryLimiter) Allow(_ context.Context, key string) (bool, error) {
	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// حذف قدیمی‌ها
	var recent []time.Time
	for _, t := range l.counts[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= l.maxPerMin {
		return false, nil
	}

	l.counts[key] = append(recent, now)
	return true, nil
}
