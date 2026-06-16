package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/engine"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/tgbot"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/fraudclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	BotToken    string `mapstructure:"BOT_TOKEN"`
	LocalBotAPI string `mapstructure:"LOCAL_BOT_API"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	NatsURL     string `mapstructure:"NATS_URL"`
	NatsUser    string `mapstructure:"NATS_USERNAME"`
	NatsPass    string `mapstructure:"NATS_PASSWORD"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// ── PostgreSQL ─────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN))
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(db)

	ctx := context.Background()
	if err := st.SeedCategories(ctx); err != nil {
		log.Fatal("seed categories", ports.F("err", err))
	}

	// ── Redis ──────────────────────────────────────────────
	cache, err := redis.New(redis.Config{Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: 4})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
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

	// ── Telegram Bot ───────────────────────────────────────
	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		URL:    cfg.LocalBotAPI,
	})
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	log.Info("bot connected", ports.F("username", b.Me.Username))

	// ── Engine + Analyzer ────────────────────────────────────
	broadcaster := engine.NewBroadcaster(b)
	fc := fraudclient.New(nc)
	eng := engine.New(st, nc, broadcaster, fc, log)

	// ── Handler ───────────────────────────────────────────
	h := tgbot.NewHandler(b, st, eng, cache, log, cfg.OwnerID)
	tgbot.Register(b, h)

	// ── Start ─────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go eng.RunScheduler(ctx)

	go func() {
		log.Info("ads-bot started", ports.F("owner", cfg.OwnerID))
		b.Start()
	}()

	<-ctx.Done()
	log.Info("shutting down...")
	b.Stop()
}
