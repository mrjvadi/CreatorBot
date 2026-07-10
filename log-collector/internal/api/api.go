// Package api یک REST API ساده برای کوئری لاگ‌های ذخیره‌شده فراهم می‌کند.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mrjvadi/creatorbot/log-collector/internal/store"
)

type Handler struct {
	store  *store.Store
	apiKey string
}

func New(st *store.Store, apiKey string) *Handler {
	return &Handler{store: st, apiKey: apiKey}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "service": "log-collector"})
	})

	logs := r.Group("/logs", h.auth)
	logs.GET("", h.query)
}

// auth یک X-API-Key ساده — fail-closed اگر LOG_API_KEY تنظیم نشده باشد
// (یعنی هیچ کوئری‌ای بدون کلید پذیرفته نمی‌شود، نه اینکه به‌طور پیش‌فرض باز بماند).
func (h *Handler) auth(c *gin.Context) {
	if h.apiKey == "" || c.GetHeader("X-API-Key") != h.apiKey {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "unauthorized"})
		return
	}
	c.Next()
}

// GET /logs?service=&level=&q=&from=&to=&limit=&skip=
func (h *Handler) query(c *gin.Context) {
	f := store.QueryFilter{
		Service: c.Query("service"),
		Level:   c.Query("level"),
		Query:   c.Query("q"),
	}
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Limit = n
		}
	}
	if v := c.Query("skip"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.Skip = n
		}
	}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.To = &t
		}
	}

	results, err := h.store.QueryLogs(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "count": len(results), "logs": results})
}
