package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/shared-core/store"
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

// UserState نقش و وضعیت مسدودی را در هر درخواست از منبع حقیقت DB تازه می‌کند.
// به این ترتیب block/demotion منتظر انقضای access/refresh token نمی‌ماند.
func UserState(st *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, err := uuid.Parse(c.GetString("user_id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid user"})
			return
		}
		u, err := st.FindUserByID(c.Request.Context(), uid)
		if err != nil || u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "user not found"})
			return
		}
		if u.IsBlocked {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "message": "user is blocked"})
			return
		}
		c.Set("role", string(u.Role))
		c.Set("current_user", u)
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
