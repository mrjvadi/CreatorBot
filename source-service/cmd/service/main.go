package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"

	"github.com/mrjvadi/creatorbot/source-service/internal/api"
	"github.com/mrjvadi/creatorbot/source-service/internal/models"
	"github.com/mrjvadi/creatorbot/source-service/internal/store"
	"github.com/mrjvadi/creatorbot/source-service/internal/userbot"
)

type Config struct {
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`
	APIPort     int    `mapstructure:"API_PORT"`
	APISecret   string `mapstructure:"API_SECRET"`
	// Userbot (gotd/td)
	TgAppID           int    `mapstructure:"TG_APP_ID"`
	TgAppHash         string `mapstructure:"TG_APP_HASH"`
	TgPhone           string `mapstructure:"TG_PHONE"`
	TgSessionFile     string `mapstructure:"TG_SESSION_FILE"`
	TgSourceChannel   int64  `mapstructure:"TG_SOURCE_CHANNEL"`
	TgDeliveryChannel int64  `mapstructure:"TG_DELIVERY_CHANNEL"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("db", ports.F("err", err))
	}
	db.Migrate(&models.ArchiveFile{}, &models.BotFileCache{})

	cache, err := sharedredis.New(sharedredis.Config{Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ⚠️ Userbot violates Telegram ToS — use only for personal archiving.
	ub := userbot.New(userbot.Config{
		AppID:           cfg.TgAppID,
		AppHash:         cfg.TgAppHash,
		Phone:           cfg.TgPhone,
		SessionFile:     cfg.TgSessionFile,
		SourceChannel:   cfg.TgSourceChannel,
		DeliveryChannel: cfg.TgDeliveryChannel,
	}, log)
	go ub.Start(ctx)

	st := store.New(db)
	srv := api.NewServer(st, cache, ub, log, cfg.APIPort, cfg.APISecret)
	log.Info("source-service started")
	if err := srv.Start(ctx); err != nil {
		log.Fatal("api", ports.F("err", err))
	}
}
