package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mrjvadi/creatorbot/log-collector/internal/api"
	"github.com/mrjvadi/creatorbot/log-collector/internal/collector"
	"github.com/mrjvadi/creatorbot/log-collector/internal/status"
	"github.com/mrjvadi/creatorbot/log-collector/internal/store"
	"github.com/mrjvadi/creatorbot/log-collector/internal/telegram"
	sharedmongo "github.com/mrjvadi/creatorbot/shared/pkg/adapters/mongodb"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	MongoURI string `mapstructure:"MONGO_URI"`
	MongoDB  string `mapstructure:"MONGO_DB"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	// تلگرام — اختیاری. اگر TelegramToken یا TelegramChatID خالی باشد، این
	// سرویس فقط در Mongo ذخیره می‌کند و هیچ هشداری به تلگرام نمی‌فرستد.
	TelegramToken    string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	TelegramChatID   int64  `mapstructure:"TELEGRAM_CHAT_ID"`   // باید یک سوپرگروه با Forum فعال باشد
	TelegramBotAPI   string `mapstructure:"LOCAL_BOT_API"`      // اختیاری — سرور local bot API
	MinTelegramLevel string `mapstructure:"MIN_TELEGRAM_LEVEL"` // warn (پیش‌فرض) | error | fatal

	// LogAPIKey برای احراز هویت GET /logs — اجباری (fail-closed اگر خالی باشد).
	LogAPIKey string `mapstructure:"LOG_API_KEY"`

	Port int `mapstructure:"PORT"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.Port == 0 {
		cfg.Port = 8099
	}
	if cfg.LogAPIKey == "" {
		log.Error("LOG_API_KEY not set — GET /logs will reject every request until configured")
	}

	// ── MongoDB ─────────────────────────────────────────────
	mdb, err := sharedmongo.New(sharedmongo.Config{URI: cfg.MongoURI, Database: cfg.MongoDB})
	if err != nil {
		log.Fatal("mongo", ports.F("err", err))
	}
	st := store.New(mdb)
	{
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		st.EnsureIndexes(ctx)
		cancel()
	}

	// ── NATS ────────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
		Name:     "log-collector",
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()
	// خودِ log-collector هم باید در داشبوردِ وضعیتِ خودش دیده شود — بدونِ این،
	// همیشه ❔ «هنوز دیده نشده» می‌ماند چون این تنها سرویسی است که AttachNATS
	// را روی خودش صدا نمی‌زد (فقط برای مصرف‌کردنِ لاگ/heartbeatِ بقیه استفاده
	// می‌شد، نه انتشارِ لاگِ خودش).
	log.AttachNATS(nc, "log-collector")

	// ── Telegram (اختیاری) ─────────────────────────────────
	tg := telegram.New(cfg.TelegramBotAPI, cfg.TelegramToken, cfg.TelegramChatID)
	if !tg.Enabled() {
		log.Warn("telegram not configured — logs will only be stored in MongoDB, no alerts sent")
	}

	col := collector.New(st, tg, log, cfg.MinTelegramLevel)
	if err := nc.Subscribe(logger.SubjLogEvents, col.Handle); err != nil {
		log.Fatal("subscribe logs.events failed", ports.F("err", err))
	}
	log.Info("subscribed", ports.F("subject", logger.SubjLogEvents))

	// ── داشبوردِ زنده‌ی وضعیتِ سرویس‌های اصلی (پیامِ تلگرامی که edit می‌شود) ──
	statusMon := status.NewMonitor()
	if err := nc.Subscribe(logger.SubjHeartbeat, statusMon.Handle); err != nil {
		log.Error("subscribe service.heartbeat failed — status dashboard disabled", ports.F("err", err))
	} else {
		go status.NewReporter(statusMon, tg, st, log).Run()
	}

	// ── HTTP query API ──────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	api.New(st, cfg.LogAPIKey).Register(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("log-collector started", ports.F("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server", ports.F("err", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}
