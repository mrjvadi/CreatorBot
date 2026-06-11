package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mrjvadi/creatorbot/apimanager/internal/handler"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	APIPort     int    `mapstructure:"API_PORT"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	AccessSecret  string `mapstructure:"JWT_ACCESS_SECRET"`
	RefreshSecret string `mapstructure:"JWT_REFRESH_SECRET"`
	EncryptKey    string `mapstructure:"ENCRYPTION_KEY"`
	AgentAPIKey   string `mapstructure:"AGENT_API_KEY"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// ── PostgreSQL ────────────────────────────────────────────
	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	db.Migrate(models.AllModels()...)

	// ── NATS ──────────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()

	st := store.New(db)

	// ── NATS Listeners ────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// heartbeat از همه agentmanager ها
	nc.QueueSubscribe("agent.*.heartbeat", "apimanager", func(data []byte) {
		var msg protocol.HeartbeatMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		st.MarkServerOnlineByServerID(ctx, msg.ServerID)
		for _, c := range msg.Containers {
			switch c.State {
			case "running":
				st.UpdateInstanceStatusByContainerName(ctx, c.Name, models.StatusRunning)
			case "exited", "dead":
				st.UpdateInstanceStatusByContainerName(ctx, c.Name, models.StatusStopped)
			}
		}
		log.Info("heartbeat",
			ports.F("server", msg.ServerID),
			ports.F("containers", len(msg.Containers)))
	})

	// نتیجه دستورات Docker
	nc.QueueSubscribe("agent.*.result", "apimanager", func(data []byte) {
		var msg protocol.ResultMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			return
		}
		if msg.Success {
			st.UpdateInstanceStatusByContainerName(ctx, msg.ContainerName, models.StatusRunning)
		} else {
			st.UpdateInstanceStatusByContainerName(ctx, msg.ContainerName, models.StatusError)
		}
		log.Info("command result",
			ports.F("cmd", msg.CommandType),
			ports.F("success", msg.Success),
			ports.F("container", msg.ContainerName))
	})

	// رویدادهای پرداخت از bot ها
	nc.Subscribe("event.payment.*", func(data []byte) {
		var event protocol.PaymentConfirmedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			return
		}
		log.Info("payment confirmed",
			ports.F("bot_id", event.BotID),
			ports.F("amount", event.Amount))
		// TODO: ثبت پرداخت در PostgreSQL و تمدید اشتراک
	})

	log.Info("NATS listeners started")

	// ── HTTP API ──────────────────────────────────────────────
	h := handler.New(st, nil, log,
		cfg.AccessSecret, cfg.RefreshSecret, cfg.EncryptKey,
		cfg.AgentAPIKey)

	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	v1.POST("/auth/telegram", h.TelegramAuth)
	v1.POST("/auth/refresh", h.RefreshToken)
	v1.POST("/agent/auth", h.AgentAuth)
	v1.POST("/bot/deploy", deployHandler(nc, st, cfg.EncryptKey, log))
	v1.POST("/bot/stop", stopHandler(nc, st, log))
	v1.POST("/bot/remove", removeHandler(nc, st, log))

	addr := fmt.Sprintf(":%d", cfg.APIPort)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Info("apimanager started", ports.F("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("api server", ports.F("err", err))
		}
	}()

	<-ctx.Done()
	log.Info("apimanager stopping...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}

// deployHandler دستور deploy را به agentmanager ارسال می‌کند.
func deployHandler(nc *natsclient.Client, st *store.Store, encryptKey string, log ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			InstanceID string `json:"instance_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"ok": false, "message": err.Error()})
			return
		}

		ctx := c.Request.Context()
		inst, err := st.FindInstance(ctx, req.InstanceID)
		if err != nil || inst == nil {
			c.JSON(404, gin.H{"ok": false, "message": "instance not found"})
			return
		}

		server, err := st.ListServers(ctx)
		if err != nil || len(server) == 0 {
			c.JSON(500, gin.H{"ok": false, "message": "no server available"})
			return
		}
		// پیدا کردن سرور instance
		var targetServer *models.Server
		for _, s := range server {
			if s.ID == inst.ServerID {
				targetServer = &s
				break
			}
		}
		if targetServer == nil {
			c.JSON(404, gin.H{"ok": false, "message": "server not found"})
			return
		}

		tmpl, _ := st.FindTemplate(ctx, inst.TemplateID.String())
		if tmpl == nil {
			c.JSON(404, gin.H{"ok": false, "message": "template not found"})
			return
		}

		// decrypt توکن
		// botToken, _ := auth.Decrypt(inst.BotToken, encryptKey)

		cmd := protocol.DeployCommand{
			Type:          protocol.MsgDeploy,
			ServerID:      targetServer.ID.String(),
			ContainerName: inst.ContainerName,
			ImageName:     tmpl.ImageName,
			ImageTag:      tmpl.ImageTag,
			EnvVars: map[string]string{
				"BOT_TOKEN": inst.BotToken, // TODO: decrypt
			},
		}

		pubCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := nc.Publish(pubCtx, protocol.DeploySubject(targetServer.ID.String()), cmd); err != nil {
			c.JSON(500, gin.H{"ok": false, "message": "publish failed"})
			return
		}

		st.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)
		c.JSON(200, gin.H{"ok": true, "message": "deploy command sent"})
	}
}

func stopHandler(nc *natsclient.Client, st *store.Store, log ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: مشابه deployHandler
		c.JSON(200, gin.H{"ok": true})
	}
}

func removeHandler(nc *natsclient.Client, st *store.Store, log ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: مشابه deployHandler
		c.JSON(200, gin.H{"ok": true})
	}
}
