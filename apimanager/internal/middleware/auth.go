package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

// JWTAuth بررسی می‌کند که Bearer token معتبر باشد.
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "missing token"})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ParseAccessToken(token, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid token"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// RequireRole بررسی می‌کند که کاربر یکی از نقش‌های مجاز را داشته باشد.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role := c.GetString("role")
		if !allowed[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "message": "forbidden"})
			return
		}
		c.Next()
	}
}

// AgentKeyAuth بررسی می‌کند که X-Agent-Key درست باشد.
func AgentKeyAuth(agentKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Agent-Key")
		if key != agentKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid agent key"})
			return
		}
		c.Next()
	}
}
