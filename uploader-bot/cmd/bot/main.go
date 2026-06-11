package main

import (
	"context"
	"os/signal"
	"syscall"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/telebot"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/tgbot"
)

type Config struct {
	BotToken    string `mapstructure:"BOT_TOKEN"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	ChannelID   int64  `mapstructure:"CHANNEL_ID"`
	LocalBotAPI string `mapstructure:"LOCAL_BOT_API"`

	// DB — مستقیم، بدون واسط
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	MongoURI    string `mapstructure:"MONGO_URI"`
	MongoDB     string `mapstructure:"MONGO_DB"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`

	// NATS — فقط برای heartbeat و events
	NatsURL  string `mapstructure:"NATS_URL"`
	ServerID string `mapstructure:"SERVER_ID"`

	HeartbeatSec int `mapstructure:"HEARTBEAT_INTERVAL_SEC"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// ── Engine — همه DB connections و business logic ───────────
	eng, err := engine.New(engine.Config{
		BotToken:     cfg.BotToken,
		PostgresDSN:  cfg.PostgresDSN,
		MongoURI:     cfg.MongoURI,
		MongoDB:      cfg.MongoDB,
		RedisAddr:    cfg.RedisAddr,
		RedisPass:    cfg.RedisPass,
		RedisDB:      cfg.RedisDB,
		NatsURL:      cfg.NatsURL,
		ServerID:     cfg.ServerID,
		HeartbeatSec: cfg.HeartbeatSec,
	}, log)
	if err != nil {
		log.Fatal("engine init failed", ports.F("err", err))
	}

	// ── Telegram Bot ──────────────────────────────────────────
	settings := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10},
	}
	if cfg.LocalBotAPI != "" {
		settings.URL = cfg.LocalBotAPI
	}
	rawBot, err := tele.NewBot(settings)
	if err != nil {
		log.Fatal("bot init failed", ports.F("err", err))
	}
	var sender ports.BotSender = telebot.New(rawBot)

	log.Info("uploader-bot starting",
		ports.F("bot_id", eng.BotID),
		ports.F("instance_id", eng.InstanceID))

	// ── Wire ──────────────────────────────────────────────────
	h := tgbot.NewHandler(tgbot.Deps{
		Engine:    eng,
		Sender:    sender,
		OwnerID:   cfg.OwnerID,
		ChannelID: cfg.ChannelID,
	})
	tgbot.Register(rawBot, h)

	// ── Graceful shutdown ─────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	eng.Start(ctx) // heartbeat شروع می‌شود

	go func() {
		<-ctx.Done()
		log.Info("uploader-bot stopping...")
		rawBot.Stop()
		eng.Close(context.Background())
	}()

	log.Info("uploader-bot started",
		ports.F("bot_id", eng.BotID),
		ports.F("channel", cfg.ChannelID))
	rawBot.Start()
}
