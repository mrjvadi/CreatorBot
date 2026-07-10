package main

import (
	"context"
	"os/signal"
	"syscall"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/license-service/internal/licensing"
	"github.com/mrjvadi/creatorbot/license-service/internal/responder"
	"github.com/mrjvadi/creatorbot/license-service/internal/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	// ServiceHMACSecret همان راز مشترک با botpay — برای احراز اینکه فقط
	// agentmanager/botmanager مجاز به صدور/ابطال لایسنس‌اند.
	ServiceHMACSecret string `mapstructure:"SERVICE_HMAC_SECRET"`

	// LicenseSigningSecret راز مجزا برای امضای JWT توکن لایسنس — عمداً از
	// SERVICE_HMAC_SECRET/ENCRYPTION_KEY جدا نگه داشته می‌شود تا نشتِ یکی
	// باعثِ جعلِ لایسنس نشود.
	LicenseSigningSecret string `mapstructure:"LICENSE_SIGNING_SECRET"`

	MetricsPort string `mapstructure:"METRICS_PORT"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.ServiceHMACSecret == "" {
		log.Error("SERVICE_HMAC_SECRET not set — license.issue/revoke will reject everything")
	}
	if cfg.LicenseSigningSecret == "" {
		log.Fatal("LICENSE_SIGNING_SECRET is required")
	}

	// ── PostgreSQL — جدول‌های این سرویس مستقل، بدون کوئری متقاطع ────
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN))
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(db)

	// ── NATS ─────────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
		Name:     "license-service",
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()
	log.AttachNATS(nc, "license-service")

	svc := licensing.New(st, nc, log, cfg.LicenseSigningSecret, 0)
	resp := responder.New(nc, svc, log, cfg.ServiceHMACSecret)
	if err := resp.Start(); err != nil {
		log.Fatal("responder start failed", ports.F("err", err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := cfg.MetricsPort
	if port == "" {
		port = ":9097"
	}
	metrics.ServeMetrics(port) // شامل /metrics و /health

	log.Info("license-service started")
	<-ctx.Done()
	log.Info("license-service stopped")
}
