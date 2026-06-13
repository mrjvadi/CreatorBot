// Package middleware middleware های webhook-gateway.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// InternalAuth بررسی X-Internal-Key برای management endpoints.
func InternalAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Internal-Key")
		if key == "" || key != secret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":      false,
				"message": "invalid internal key",
			})
			return
		}
		c.Next()
	}
}

// RateLimit محدود کردن تعداد request از یک IP.
// ساده — برای production از redis-based limiter استفاده شود.
func RateLimit(maxPerSec int) gin.HandlerFunc {
	// فعلاً pass-through — در production پیاده‌سازی می‌شود
	return func(c *gin.Context) {
		c.Next()
	}
}
