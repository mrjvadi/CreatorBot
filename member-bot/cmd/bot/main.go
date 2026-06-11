package main

import (
	"context"
	"os/signal"
	"syscall"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/member-bot/internal/dispatcher"
	"github.com/mrjvadi/creatorbot/member-bot/internal/lock"
	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
	"github.com/mrjvadi/creatorbot/member-bot/internal/scheduler"
	"github.com/mrjvadi/creatorbot/member-bot/internal/store"
	"github.com/mrjvadi/creatorbot/member-bot/internal/tgbot"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/telebot"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	BotToken    string `mapstructure:"BOT_TOKEN"`
	PostgresDSN string `mapstructure:"MASTER_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`
	LockAPIPort int    `mapstructure:"LOCK_API_PORT"`
	LockAPIKey  string `mapstructure:"LOCK_API_SECRET"`
	EncryptKey  string `mapstructure:"ENCRYPTION_KEY"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("db", ports.F("err", err))
	}
	db.Migrate(&models.Owner{}, &models.Lock{}, &models.CheckBot{},
		&models.BotChannelMembership{}, &models.MemberVerification{},
		&models.Payment{}, &models.Setting{})

	cache, err := sharedredis.New(sharedredis.Config{
		Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB,
	})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	rawBot, err := tele.NewBot(tele.Settings{Token: cfg.BotToken, Poller: &tele.LongPoller{Timeout: 10}})
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	var sender ports.BotSender = telebot.New(rawBot)

	st := store.New(db)
	h := tgbot.NewHandler(sender, st, cache, log)
	tgbot.Register(rawBot, h)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// FIX 17: start scheduler
	sched := scheduler.New(st, sender, log)
	sched.Start(ctx)

	// Lock HTTP API
	lockServer := lock.NewServer(cache, log, cfg.LockAPIPort, cfg.LockAPIKey)
	go func() {
		if err := lockServer.Start(); err != nil {
			log.Fatal("lock api", ports.F("err", err))
		}
	}()

	// Worker dispatcher
	disp := dispatcher.New(db, cache, log, cfg.EncryptKey)
	go func() {
		if err := disp.Start(ctx); err != nil && ctx.Err() == nil {
			log.Fatal("dispatcher", ports.F("err", err))
		}
	}()

	go func() { <-ctx.Done(); rawBot.Stop() }()
	log.Info("member-bot started")
	rawBot.Start()
}
