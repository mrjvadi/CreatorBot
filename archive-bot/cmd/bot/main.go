package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/webhook"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/store"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/tgbot"
)

type Config struct {
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	BotToken    string `mapstructure:"BOT_TOKEN"`
	ChannelID   int64  `mapstructure:"CHANNEL_ID"`
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`

	BotMode    string `mapstructure:"BOT_MODE"`
	GatewayURL string `mapstructure:"GATEWAY_URL"`
	NatsURL    string `mapstructure:"NATS_URL"`
	NatsUser   string `mapstructure:"NATS_USERNAME"`
	NatsPass   string `mapstructure:"NATS_PASSWORD"`
	ServerID   string `mapstructure:"SERVER_ID"`

	// LicenseToken توکنی که botmanager هنگام deploy از license-service
	// گرفته و به‌عنوان env var تزریق کرده — برای ضدکپی/ضدکلون.
	LicenseToken string `mapstructure:"LICENSE_TOKEN"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// SWAP: replace postgres.New with any ports.DB
	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("db", ports.F("err", err))
	}
	if err := db.Migrate(&models.User{}, &models.Category{}, &models.File{}, &models.Setting{}); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	// Enable pg_trgm and create GIN index for fuzzy search
	db.Conn().Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm")
	db.Conn().Exec(`CREATE INDEX IF NOT EXISTS idx_files_trgm ON files
		USING GIN ((title || ' ' || tags || ' ' || description) gin_trgm_ops)`)

	// SWAP: replace sharedredis.New with any ports.Cache
	cache, err := sharedredis.New(sharedredis.Config{Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	mode := webhook.ParseMode(cfg.BotMode)
	botID := webhook.BotIDFromToken(cfg.BotToken)
	// نکته: قبلاً این اتصال فقط در حالت webhook و بدون username/password
	// ساخته می‌شد. حالا هر وقت NATS_URL تنظیم باشد ساخته می‌شود (لازم برای
	// license check-in دوره‌ای، صرف‌نظر از polling/webhook).
	var nc *natsclient.Client
	if cfg.NatsURL != "" {
		nc, err = natsclient.New(natsclient.Config{
			URL: cfg.NatsURL, Username: cfg.NatsUser, Password: cfg.NatsPass, Name: "archive-bot",
		})
		if err != nil {
			if mode == webhook.ModeWebhook {
				log.Fatal("nats connect (webhook mode)", ports.F("err", err))
			}
			log.Warn("nats unavailable — license check-in disabled", ports.F("err", err))
			nc = nil
		}
	}
	if nc != nil {
		log.AttachNATS(nc, "archive-bot")
	}

	// ── بررسی لایسنس در startup — fail-closed ────────────────
	{
		lctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if err := licenseclient.RequireValid(lctx, nc, botID, cfg.LicenseToken, cfg.ServerID); err != nil {
			cancel()
			log.Fatal("license check failed — bot will not start", ports.F("err", err))
		}
		cancel()
		log.Info("license verified", ports.F("bot_id", botID))
	}

	poller := webhook.BuildPoller(webhook.PollerConfig{
		Mode: mode, BotID: botID, Token: cfg.BotToken,
		GatewayURL: cfg.GatewayURL, NATS: nc, Log: log,
	})
	rawBot, err := tele.NewBot(tele.Settings{Token: cfg.BotToken, Poller: poller})
	if err == nil {
		if e := webhook.Setup(context.Background(), rawBot, webhook.PollerConfig{
			Mode: mode, Token: cfg.BotToken, GatewayURL: cfg.GatewayURL,
		}); e != nil {
			log.Error("webhook setup", ports.F("err", e))
		}
	}
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	st := store.New(db)
	h := tgbot.NewHandler(rawBot, st, db, cache, log, cfg.OwnerID, rawBot.Me.Username)
	tgbot.Register(rawBot, h)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if nc != nil {
		go licenseclient.RunLicenseLoop(ctx, nc, botID, cfg.LicenseToken, cfg.ServerID, log)
	}
	go func() { <-ctx.Done(); rawBot.Stop() }()

	log.Info("archive-bot started")
	rawBot.Start()
}
