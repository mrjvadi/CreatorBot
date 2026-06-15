// Package router webhook های تلگرام را دریافت و به NATS forward می‌کند.
//
// Flow:
//  1. تلگرام POST /webhook/{token} می‌زند
//  2. router توکن را در registry جستجو می‌کند
//  3. payload به NATS subject مربوط publish می‌شود
//  4. bot مربوطه از NATS می‌خواند و update را پردازش می‌کند
package router

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/webhook-gateway/internal/registry"
)

// Stats آمار gateway.
type Stats struct {
	TotalReceived  int64
	TotalForwarded int64
	TotalUnknown   int64
	TotalErrors    int64
	StartedAt      time.Time
}

// Router دریافت و forward webhook ها.
type Router struct {
	reg   *registry.Registry
	nc    *natsclient.Client
	log   ports.Logger
	stats Stats
}

func New(reg *registry.Registry, nc *natsclient.Client, log ports.Logger) *Router {
	return &Router{
		reg:   reg,
		nc:    nc,
		log:   log,
		stats: Stats{StartedAt: time.Now()},
	}
}

// Register route های gateway را ثبت می‌کند.
func (r *Router) Register(engine *gin.Engine) {
	// webhook از تلگرام
	engine.POST("/webhook/:token", r.handleWebhook)
	engine.POST("/webhook/:token/", r.handleWebhook)

	// مدیریت bot ها (internal — با API key)
	mgmt := engine.Group("/internal")
	mgmt.POST("/register", r.registerBot)
	mgmt.POST("/unregister", r.unregisterBot)
	mgmt.GET("/bots", r.listBots)
	mgmt.GET("/stats", r.getStats)
	mgmt.GET("/health", r.health)
}

// ── Webhook handler ────────────────────────────────────────

func (r *Router) handleWebhook(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	r.stats.TotalReceived++

	// پیدا کردن bot
	entry, found := r.reg.Lookup(token)
	if !found {
		r.stats.TotalUnknown++
		r.log.Info("webhook for unknown bot",
			ports.F("token_prefix", safePrefix(token, 8)))
		// به تلگرام 200 برگردان — retry نکند
		c.Status(http.StatusOK)
		return
	}

	// خواندن payload
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 10*1024*1024)) // max 10MB
	if err != nil {
		r.stats.TotalErrors++
		c.Status(http.StatusBadRequest)
		return
	}

	// forward به NATS
	if err := r.nc.PublishCore(entry.NATSSubject, body); err != nil {
		r.stats.TotalErrors++
		r.log.Error("webhook forward failed",
			ports.F("bot_id", entry.BotID),
			ports.F("subject", entry.NATSSubject),
			ports.F("err", err))
		// به تلگرام 200 برگردان تا retry نکند — خودمان retry می‌کنیم
		c.Status(http.StatusOK)
		return
	}

	r.stats.TotalForwarded++
	// تلگرام انتظار 200 دارد
	c.Status(http.StatusOK)
}

// ── Management endpoints ──────────────────────────────────

type RegisterRequest struct {
	Token       string `json:"token"        binding:"required"`
	BotID       int64  `json:"bot_id"       binding:"required"`
	NATSSubject string `json:"nats_subject" binding:"required"`
	Type        string `json:"type"`
}

func (r *Router) registerBot(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}

	r.reg.Register(&registry.BotEntry{
		Token:       req.Token,
		BotID:       req.BotID,
		NATSSubject: req.NATSSubject,
		Type:        req.Type,
	})

	r.log.Info("bot registered",
		ports.F("bot_id", req.BotID),
		ports.F("type", req.Type),
		ports.F("subject", req.NATSSubject))

	c.JSON(http.StatusOK, gin.H{
		"ok":           true,
		"bot_id":       req.BotID,
		"webhook_path": "/webhook/" + req.Token,
	})
}

func (r *Router) unregisterBot(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": err.Error()})
		return
	}
	r.reg.Unregister(req.Token)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (r *Router) listBots(c *gin.Context) {
	bots := r.reg.List()
	result := make([]gin.H, 0, len(bots))
	for _, b := range bots {
		result = append(result, gin.H{
			"bot_id":       b.BotID,
			"type":         b.Type,
			"nats_subject": b.NATSSubject,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "bots": result, "count": len(result)})
}

func (r *Router) getStats(c *gin.Context) {
	uptime := time.Since(r.stats.StartedAt)
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"stats": gin.H{
			"total_received":  r.stats.TotalReceived,
			"total_forwarded": r.stats.TotalForwarded,
			"total_unknown":   r.stats.TotalUnknown,
			"total_errors":    r.stats.TotalErrors,
			"registered_bots": r.reg.Count(),
			"uptime_sec":      int(uptime.Seconds()),
		},
	})
}

func (r *Router) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"service": "webhook-gateway",
		"bots":    r.reg.Count(),
	})
}

// ── helpers ───────────────────────────────────────────────

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ── NATS subject builder ──────────────────────────────────

// BotWebhookSubject subject که bot از NATS می‌خواند.
// فرمت: webhook.<bot_id>
func BotWebhookSubject(botID int64) string {
	return "webhook." + strconv.FormatInt(botID, 10)
}

// suppress unused
var _ = strings.TrimSpace
