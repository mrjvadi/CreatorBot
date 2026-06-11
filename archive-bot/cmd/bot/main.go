package main

import (
	"context"
	"os/signal"
	"syscall"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/telebot"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/store"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/tgbot"
)

type Config struct {
	BotToken    string `mapstructure:"BOT_TOKEN"`
	ChannelID   int64  `mapstructure:"CHANNEL_ID"`
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`
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

	rawBot, err := tele.NewBot(tele.Settings{Token: cfg.BotToken, Poller: &tele.LongPoller{Timeout: 10}})
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	// SWAP: replace telebot.New with any ports.BotSender
	var sender ports.BotSender = telebot.New(rawBot)

	st := store.New(db)
	h := tgbot.NewHandler(sender, st, db, cache, log)
	tgbot.Register(rawBot, h)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); rawBot.Stop() }()

	log.Info("archive-bot started")
	rawBot.Start()
}
