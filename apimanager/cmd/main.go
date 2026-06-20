package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mrjvadi/creatorbot/apimanager/internal/handler"
	"github.com/mrjvadi/creatorbot/apimanager/internal/middleware"
	pgadapter "github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"encoding/json"

)

type Config struct {
	PostgresDSN   string `mapstructure:"POSTGRES_DSN"`
	NatsURL       string `mapstructure:"NATS_URL"`
	NatsUser      string `mapstructure:"NATS_USERNAME"`
	NatsPass      string `mapstructure:"NATS_PASSWORD"`
	Port          string `mapstructure:"PORT"`
	AccessSecret  string `mapstructure:"JWT_ACCESS_SECRET"`
	RefreshSecret string `mapstructure:"JWT_REFRESH_SECRET"`
	EncryptionKey string `mapstructure:"ENCRYPTION_KEY"`
	AgentAPIKey   string `mapstructure:"AGENT_API_KEY"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// ── PostgreSQL ─────────────────────────────────────────
	pg, err := pgadapter.New(pgadapter.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := pg.Conn().AutoMigrate(models.AllModels()...); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(pg)

	// ── NATS ───────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()

	// ── Docker Manager (NATS-based) ────────────────────────
	dockerManager := sharedocker.NewManager(nc)

	// ── Handler ────────────────────────────────────────────
	h := handler.New(
		st, dockerManager, nc, log,
		cfg.AccessSecret, cfg.RefreshSecret,
		cfg.EncryptionKey, cfg.AgentAPIKey,
	)

	// ── NATS Listeners ─────────────────────────────────────
	// Heartbeat از agentmanager
	nc.Subscribe("agent.*.heartbeat", func(data []byte) {
		handleHeartbeat(data, st, log)
	})

	// نتیجه deploy/stop/remove از agentmanager
	nc.Subscribe("agent.*.result", func(data []byte) {
		handleResult(data, st, log)
	})

	log.Info("NATS listeners started")

	// ── Routes ────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "service": "apimanager"})
	})

	v1 := r.Group("/api/v1")

	// Public
	v1.POST("/auth/telegram", h.TelegramAuth)
	v1.POST("/auth/refresh", h.RefreshToken)
	v1.POST("/agent/auth", h.AgentAuth)

	// Agent webhook (با AgentAPIKey)
	agent := v1.Group("/agent", middleware.AgentKeyAuth(cfg.AgentAPIKey))
	agent.POST("/heartbeat", h.AgentHeartbeat)
	agent.POST("/result", h.AgentResult)

	// User (JWT)
	// rate limiting: 60 req/min per IP
	limiter := middleware.NewSimpleLimiter(60)
	v1.Use(middleware.RateLimit(limiter, 60))

	user := v1.Group("", middleware.JWTAuth(cfg.AccessSecret))
	user.GET("/me", h.Me)
	user.GET("/instances", h.ListInstances)
	user.POST("/instances", h.CreateInstance)
	user.POST("/instances/:id/start", h.StartInstance)
	user.POST("/instances/:id/stop", h.StopInstance)
	user.POST("/instances/:id/restart", h.RestartInstance)
	user.DELETE("/instances/:id", h.DeleteInstance)
	user.GET("/instances/:id/logs", h.GetInstanceLogs)
	user.GET("/plans", h.ListPlans)

	// Admin (JWT + Admin role)
	admin := v1.Group("/admin",
		middleware.JWTAuth(cfg.AccessSecret),
		middleware.RequireRole("admin", "owner"))
	admin.GET("/stats", h.AdminStats)
	admin.GET("/servers", h.ListServers)
	admin.POST("/servers", h.CreateServer)
	admin.DELETE("/servers/:id", h.DeleteServer)
	admin.GET("/templates", h.ListTemplates)
	admin.POST("/templates", h.CreateTemplate)

	// ── Start server ───────────────────────────────────────
	addr := ":" + cfg.Port
	srv := &http.Server{Addr: addr, Handler: r}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		metrics.ServeMetrics(":9090")
	log.Info("metrics server started", ports.F("addr", ":9090"))
	log.Info("apimanager started", ports.F("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("api server", ports.F("err", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}

// ── NATS event handlers ───────────────────────────────────

func handleHeartbeat(data []byte, st *store.Store, log ports.Logger) {
	// agentmanager heartbeat رو handle کن — server را آنلاین نشان بده
	var hb protocol.HeartbeatMsg
	if err := parseJSON(data, &hb); err != nil {
		return
	}
	ctx := context.Background()
	st.MarkServerOnlineByServerID(ctx, hb.ServerID)
}

func handleResult(data []byte, st *store.Store, log ports.Logger) {
	var result protocol.ResultMsg
	if err := parseJSON(data, &result); err != nil {
		return
	}

	ctx := context.Background()
	inst, err := st.FindInstanceByContainerName(ctx, result.ContainerName)
	if err != nil || inst == nil {
		return
	}

	switch {
	case result.Success && result.CommandType == string(protocol.MsgDeploy):
		st.UpdateInstanceStatus(ctx, inst.ID, models.StatusRunning)
		log.Info("instance running",
			ports.F("instance", inst.ID),
			ports.F("container", result.ContainerName))
	case !result.Success:
		st.UpdateInstanceStatus(ctx, inst.ID, models.StatusError)
		log.Error("instance failed",
			ports.F("instance", inst.ID),
			ports.F("err", result.Error))
	case result.CommandType == string(protocol.MsgStop):
		st.UpdateInstanceStatus(ctx, inst.ID, models.StatusStopped)
	case result.CommandType == string(protocol.MsgRemove):
		st.DeleteInstance(ctx, inst.ID)
	}
}

func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
