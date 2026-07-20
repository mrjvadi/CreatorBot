package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"encoding/json"

	"github.com/gin-gonic/gin"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/webhook-gateway/internal/middleware"
	"github.com/mrjvadi/creatorbot/webhook-gateway/internal/registry"
	"github.com/mrjvadi/creatorbot/webhook-gateway/internal/router"
)

type Config struct {
	Port              int    `mapstructure:"PORT"`
	InternalKey       string `mapstructure:"INTERNAL_KEY"`
	ServiceHMACSecret string `mapstructure:"SERVICE_HMAC_SECRET"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	// bot هایی که در startup ثبت می‌شوند
	BotmanagerToken string `mapstructure:"BOTMANAGER_TOKEN"`
	BotmanagerBotID int64  `mapstructure:"BOTMANAGER_BOT_ID"`
	BotpayToken     string `mapstructure:"BOTPAY_TOKEN"`
	BotpayBotID     int64  `mapstructure:"BOTPAY_BOT_ID"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.Port == 0 {
		cfg.Port = 8090
	}
	if cfg.InternalKey == "" {
		log.Fatal("INTERNAL_KEY is required", ports.F("hint", "set INTERNAL_KEY in .env"))
	}
	if cfg.ServiceHMACSecret == "" {
		log.Fatal("SERVICE_HMAC_SECRET is required for gateway control messages")
	}

	// ── NATS ─────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()
	log.AttachNATS(nc, "webhook-gateway")

	// ── Registry ─────────────────────────────────────────
	reg := registry.New()

	// ثبت bot های ثابت از env
	if cfg.BotmanagerToken != "" && cfg.BotmanagerBotID != 0 {
		reg.Register(&registry.BotEntry{
			Token:       cfg.BotmanagerToken,
			BotID:       cfg.BotmanagerBotID,
			NATSSubject: router.BotWebhookSubject(cfg.BotmanagerBotID),
			Type:        "botmanager",
		})
		log.Info("botmanager registered", ports.F("bot_id", cfg.BotmanagerBotID))
	}

	if cfg.BotpayToken != "" && cfg.BotpayBotID != 0 {
		reg.Register(&registry.BotEntry{
			Token:       cfg.BotpayToken,
			BotID:       cfg.BotpayBotID,
			NATSSubject: router.BotWebhookSubject(cfg.BotpayBotID),
			Type:        "botpay",
		})
		log.Info("botpay registered", ports.F("bot_id", cfg.BotpayBotID))
	}

	// ── NATS: ثبت bot های dynamic ─────────────────────────
	// agentmanager یا apimanager می‌تواند از طریق NATS بخواهد bot جدید ثبت شود
	nc.Subscribe("gateway.register", func(data []byte) {
		var req struct {
			Token       string `json:"token"`
			BotID       int64  `json:"bot_id"`
			NATSSubject string `json:"nats_subject"`
			Type        string `json:"type"`
			ServiceID   string `json:"service_id"`
			ServiceKey  string `json:"service_key"`
		}
		if err := unmarshalJSON(data, &req); err != nil || req.Token == "" {
			return
		}
		// اگر HMAC تنظیم شده، ارسال‌کننده باید اثبات کند که secret را می‌داند
		if cfg.ServiceHMACSecret != "" {
			if !auth.ValidateServiceKey(cfg.ServiceHMACSecret, req.ServiceID, req.ServiceKey) {
				log.Warn("gateway.register: invalid service key — ignoring",
					ports.F("service_id", req.ServiceID))
				return
			}
		}
		reg.Register(&registry.BotEntry{
			Token:       req.Token,
			BotID:       req.BotID,
			NATSSubject: req.NATSSubject,
			Type:        req.Type,
		})
		log.Info("bot registered via NATS",
			ports.F("bot_id", req.BotID),
			ports.F("type", req.Type))
	})

	nc.Subscribe("gateway.unregister", func(data []byte) {
		var req struct {
			Token      string `json:"token"`
			ServiceID  string `json:"service_id"`
			ServiceKey string `json:"service_key"`
		}
		if err := unmarshalJSON(data, &req); err != nil || req.Token == "" {
			return
		}
		if !auth.ValidateServiceKey(cfg.ServiceHMACSecret, req.ServiceID, req.ServiceKey) {
			log.Warn("gateway.unregister: invalid service key — ignoring", ports.F("service_id", req.ServiceID))
			return
		}
		reg.Unregister(req.Token)
	})

	// ── HTTP Server ───────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := router.New(reg, nc, log)

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.GlobalRateLimit())

	// InternalAuth روی /internal/* داخل r.Register اعمال می‌شود (همان جایی
	// که route ها ثبت می‌شوند) — قبلاً این‌جا جدا و روی یک گروه استفاده‌نشده
	// تنظیم می‌شد که هیچ‌وقت واقعاً اجرا نمی‌شد.
	r.Register(engine, cfg.InternalKey)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: addr, Handler: engine}

	// ── Graceful Shutdown ────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("webhook-gateway started",
			ports.F("addr", addr),
			ports.F("bots", reg.Count()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server", ports.F("err", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(timeout)
	log.Info("webhook-gateway stopped")
}

func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
