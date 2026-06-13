package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"bytes"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/community-service/internal/api"
	"github.com/mrjvadi/creatorbot/community-service/internal/engine"
	"github.com/mrjvadi/creatorbot/community-service/internal/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/fraudclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	NatsURL     string `mapstructure:"NATS_URL"`
	NatsUser    string `mapstructure:"NATS_USERNAME"`
	NatsPass    string `mapstructure:"NATS_PASSWORD"`
	Port        int    `mapstructure:"PORT"`
	AdminKey    string `mapstructure:"ADMIN_KEY"`
	BotpayURL   string `mapstructure:"BOTPAY_URL"`
	BotpayKey   string `mapstructure:"BOTPAY_API_KEY"`
	FraudURL    string `mapstructure:"FRAUD_ENGINE_URL"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.Port == 0 { cfg.Port = 8093 }

	// ── PostgreSQL ─────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN))
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(db)

	// ── NATS ──────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()

	// ── External clients ───────────────────────────────────
	payClient := newBotpayClient(cfg.BotpayURL, cfg.BotpayKey)
	fc := fraudclient.New(nc)

	// ── Engine ────────────────────────────────────────────
	eng := engine.New(st, nc, payClient, fc, log)
	eng.RegisterNATSListeners(nc)

	// ── API ───────────────────────────────────────────────
	apiHandler := api.New(st, eng, cfg.AdminKey)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	apiHandler.Register(r)

	// ── Start ─────────────────────────────────────────────
	shutCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go eng.RunValidationChecker(shutCtx)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Info("community-service started", ports.F("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http", ports.F("err", err))
		}
	}()

	<-shutCtx.Done()
	log.Info("shutting down...")
	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(timeout)
}

// ── botpay client ─────────────────────────────────────────

type botpayClient struct{ baseURL, apiKey string; http *http.Client }

func newBotpayClient(url, key string) engine.PayClient {
	if url == "" { return &noopPayClient{} }
	return &botpayClient{baseURL: url, apiKey: key, http: &http.Client{Timeout: 10 * time.Second}}
}

func (b *botpayClient) AddCredit(ctx context.Context, telegramID int64, amountTON float64, desc string) error {
	body, _ := json.Marshal(map[string]any{
		"telegram_id": telegramID, "amount_ton": amountTON, "description": desc,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", b.baseURL+"/api/v1/pay/credit/add",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", b.apiKey)
	resp, err := b.http.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	return nil
}



