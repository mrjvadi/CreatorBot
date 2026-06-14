package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/revenue-service/internal/api"
	revengine "github.com/mrjvadi/creatorbot/revenue-service/internal/engine"
	"github.com/mrjvadi/creatorbot/revenue-service/internal/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	NatsURL     string `mapstructure:"NATS_URL"`
	NatsUser    string `mapstructure:"NATS_USERNAME"`
	NatsPass    string `mapstructure:"NATS_PASSWORD"`
	Port        int    `mapstructure:"PORT"`
	AdminKey           string `mapstructure:"ADMIN_API_KEY"`
	PlatformTelegramID int64  `mapstructure:"PLATFORM_TELEGRAM_ID"`

	// botpay — برای پرداخت سهم‌ها
	BotpayURL   string `mapstructure:"BOTPAY_URL"`
	BotpayKey   string `mapstructure:"BOTPAY_API_KEY"`
	BotpayAdmin string `mapstructure:"BOTPAY_ADMIN_KEY"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.Port == 0 {
		cfg.Port = 8088
	}

	// ── PostgreSQL ─────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN))
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(db)

	// Seed default rules اگه وجود ندارن
	ctx := context.Background()
	if err := st.SeedDefaultRules(ctx); err != nil {
		log.Fatal("seed rules", ports.F("err", err))
	}

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

	// ── botpay client ──────────────────────────────────────
	pay := newBotpayClient(cfg.BotpayURL, cfg.BotpayKey, cfg.BotpayAdmin)

	// ── Revenue Engine ─────────────────────────────────────
	eng := revengine.New(st, pay, log)
	eng.SetPlatformWallet(cfg.PlatformTelegramID)
	eng.SetNC(nc)

	// ── API ────────────────────────────────────────────────
	apiHandler := api.New(eng, st, nc, api.Config{AdminKey: cfg.AdminKey}, log)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "service": "revenue-service"})
	})

	apiHandler.Register(r)

	// ── شروع ──────────────────────────────────────────────
	shutCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// NATS listeners
	apiHandler.RegisterNATSListeners(shutCtx)

	// Worker loop — پردازش pending earnings
	go eng.RunWorker(shutCtx)

	// HTTP server
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Info("revenue-service started",
			ports.F("addr", addr),
			ports.F("nats", cfg.NatsURL))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server", ports.F("err", err))
		}
	}()

	<-shutCtx.Done()
	log.Info("shutting down...")

	timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(timeout)
	log.Info("revenue-service stopped")
}

// ── botpay HTTP client ─────────────────────────────────────

type botpayClient struct {
	baseURL  string
	apiKey   string
	adminKey string
	client   *http.Client
}

func newBotpayClient(url, apiKey, adminKey string) revengine.PayClient {
	return &botpayClient{
		baseURL:  url,
		apiKey:   apiKey,
		adminKey: adminKey,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (b *botpayClient) AddCredit(ctx context.Context, telegramID int64, amountTON float64, desc string) (string, error) {
	return b.post(ctx, "/api/v1/pay/credit/add", map[string]any{
		"telegram_id": telegramID,
		"amount_ton":  amountTON,
		"description": desc,
	}, b.adminKey)
}

func (b *botpayClient) Deduct(ctx context.Context, telegramID int64, amountTON float64, ref, desc string) (string, error) {
	return b.post(ctx, "/api/v1/pay/deduct", map[string]any{
		"telegram_id": telegramID,
		"amount_ton":  amountTON,
		"ref":         ref,
		"description": desc,
	}, b.apiKey)
}

func (b *botpayClient) post(ctx context.Context, path string, body map[string]any, key string) (string, error) {
	if b.baseURL == "" {
		return "", nil // botpay not configured
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", key)
	req.Header.Set("X-Service-ID", "revenue-service")

	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("botpay: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK   bool `json:"ok"`
		Data struct {
			TxID string `json:"tx_id"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.OK {
		return "", fmt.Errorf("botpay: %s", result.Message)
	}
	return result.Data.TxID, nil
}
