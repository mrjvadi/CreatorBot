package main

import (
	"context"
	"os/signal"
	"strings"
	"syscall"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/scheduler"
	"github.com/mrjvadi/creatorbot/admanager-bot/internal/store"
	"github.com/mrjvadi/creatorbot/admanager-bot/internal/tgbot"
	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/webhook"
	"github.com/mrjvadi/creatorbot/shared/pkg/botprofile"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Config متغیرهای محیطی ربات.
type Config struct {
	AppEnv      string `mapstructure:"APP_ENV"`
	ServiceName string `mapstructure:"BOT_SERVICE_NAME"`
	BotToken    string `mapstructure:"BOT_TOKEN"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	LocalBotAPI string `mapstructure:"LOCAL_BOT_API"`

	MongoURI    string `mapstructure:"MONGO_URI"`
	MongoDB     string `mapstructure:"MONGO_DB"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	BotMode    string `mapstructure:"BOT_MODE"`
	GatewayURL string `mapstructure:"GATEWAY_URL"`
	ServerID   string `mapstructure:"SERVER_ID"`

	HeartbeatSec int `mapstructure:"HEARTBEAT_INTERVAL_SEC"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// ── Engine ────────────────────────────────────────────────
	eng, err := engine.New(engine.Config{
		BotToken:     cfg.BotToken,
		MongoURI:     cfg.MongoURI,
		MongoDB:      cfg.MongoDB,
		RedisAddr:    cfg.RedisAddr,
		RedisPass:    cfg.RedisPass,
		RedisDB:      cfg.RedisDB,
		NatsURL:      cfg.NatsURL,
		NatsUser:     cfg.NatsUser,
		NatsPass:     cfg.NatsPass,
		ServerID:     cfg.ServerID,
		HeartbeatSec: cfg.HeartbeatSec,
	}, log)
	if err != nil {
		log.Fatal("engine init failed", ports.F("err", err))
	}
	if eng.Nats != nil {
		log.AttachNATS(eng.Nats, "admanager-bot")
	}

	// ── Telegram Bot ──────────────────────────────────────────
	mode := webhook.ParseMode(cfg.BotMode)
	poller := webhook.BuildPoller(webhook.PollerConfig{
		Mode: mode, BotID: eng.BotID, Token: cfg.BotToken,
		GatewayURL: cfg.GatewayURL, NATS: eng.Nats, Log: log,
	})
	settings := tele.Settings{
		Token:  cfg.BotToken,
		Poller: poller,
		OnError: func(e error, _ tele.Context) {
			if e == nil || strings.Contains(e.Error(), "message is not modified") {
				return
			}
			log.Error("bot error", ports.F("err", e))
		},
	}
	if cfg.LocalBotAPI != "" {
		settings.URL = cfg.LocalBotAPI
	}
	rawBot, err := tele.NewBot(settings)
	if err != nil {
		log.Fatal("bot init failed", ports.F("err", err))
	}
	if err2 := webhook.Setup(context.Background(), rawBot, webhook.PollerConfig{
		Mode: mode, Token: cfg.BotToken, GatewayURL: cfg.GatewayURL,
	}); err2 != nil {
		log.Error("webhook setup", ports.F("err", err2))
	}
	if err := botprofile.Sync(rawBot, botprofile.Config{
		Environment: cfg.AppEnv,
		ServiceName: botprofile.ServiceName(cfg.ServiceName, "Ad Manager Bot"),
	}); err != nil {
		log.Warn("production bot profile sync failed", ports.F("err", err))
	}

	log.Info("admanager-bot starting",
		ports.F("bot_id", eng.BotID),
		ports.F("instance_id", eng.InstanceID),
	)

	// ── Wire ──────────────────────────────────────────────────
	_ = tgbot.NewHandler(tgbot.Deps{
		Engine:  eng,
		Bot:     rawBot,
		OwnerID: cfg.OwnerID,
	})

	// ── Scheduler ─────────────────────────────────────────────
	st := store.New(eng.Mongo, eng.InstanceID, eng.Cache)
	sched := scheduler.New(st, rawBot, log, cfg.OwnerID)
	sched.Start()

	// ── Graceful shutdown ─────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	eng.Start(ctx)

	go func() {
		<-ctx.Done()
		log.Info("admanager-bot stopping...")
		sched.Stop()
		rawBot.Stop()
		eng.Close(context.Background())
	}()

	log.Info("admanager-bot started", ports.F("bot_id", eng.BotID))
	rawBot.Start()
}
