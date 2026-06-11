// Package middleware provides Gin middleware for apimanager.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

// JWT returns a Gin middleware that validates Bearer tokens.
// accessSecret is injected at construction time — swap auth strategy in main.go.
func JWT(accessSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "missing token"})
			return
		}
		claims, err := auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer "), accessSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "invalid token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// Role returns a middleware that allows only the specified roles.
func Role(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if !allowed[role.(string)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"ok": false, "message": "forbidden"})
			return
		}
		c.Next()
	}
}
