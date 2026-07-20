package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/member-bot/internal/dispatcher"
	"github.com/mrjvadi/creatorbot/member-bot/internal/lock"
	"github.com/mrjvadi/creatorbot/member-bot/internal/memberresponder"
	"github.com/mrjvadi/creatorbot/member-bot/internal/scheduler"
	"github.com/mrjvadi/creatorbot/member-bot/internal/store"
	"github.com/mrjvadi/creatorbot/member-bot/internal/tgbot"
	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/mongodb"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/webhook"
	"github.com/mrjvadi/creatorbot/shared/pkg/botprofile"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/fraudclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/joinevents"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	AppEnv      string `mapstructure:"APP_ENV"`
	ServiceName string `mapstructure:"BOT_SERVICE_NAME"`
	NatsURL     string `mapstructure:"NATS_URL"`
	NatsUser    string `mapstructure:"NATS_USERNAME"`
	NatsPass    string `mapstructure:"NATS_PASSWORD"`
	FraudURL    string `mapstructure:"FRAUD_ENGINE_URL"`
	BotToken    string `mapstructure:"BOT_TOKEN"`
	LocalBotAPI string `mapstructure:"LOCAL_BOT_API"`
	// این ربات هیچ Postgres ندارد؛ همه‌ی داده روی MongoDB است (دیتابیس
	// اختصاصیِ نوع سرویس member-bot).
	MongoURI    string `mapstructure:"MONGO_URI"`
	MongoDB     string `mapstructure:"MONGO_DB"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`
	LockAPIPort int    `mapstructure:"LOCK_API_PORT"`
	LockAPIKey  string `mapstructure:"LOCK_API_SECRET"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	EncryptKey  string `mapstructure:"ENCRYPTION_KEY"`
	BotMode     string `mapstructure:"BOT_MODE"`
	GatewayURL  string `mapstructure:"GATEWAY_URL"`
	ServerID    string `mapstructure:"SERVER_ID"`

	// LicenseToken توکنی که botmanager هنگام deploy از license-service
	// گرفته و به‌عنوان env var تزریق کرده — برای ضدکپی/ضدکلون.
	LicenseToken string `mapstructure:"LICENSE_TOKEN"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// instanceID جدا می‌کند دیتای این deploy را از بقیه‌ی instanceهای member-bot
	// که همگی روی همان دیتابیسِ مشترکِ Mongo (MONGO_DB=member_bot) نشسته‌اند —
	// همان الگویی که uploader-bot با shared-core/docstore پیاده می‌کند.
	botID := webhook.BotIDFromToken(cfg.BotToken)
	instanceID := fmt.Sprintf("bot_%d", botID)

	mdb, err := mongodb.New(mongodb.Config{URI: cfg.MongoURI, Database: cfg.MongoDB})
	if err != nil {
		log.Fatal("mongodb", ports.F("err", err))
	}
	st := store.New(mdb.Database(), instanceID)
	if err := st.EnsureIndexes(context.Background()); err != nil {
		log.Fatal("mongo indexes", ports.F("err", err))
	}

	cache, err := sharedredis.New(sharedredis.Config{
		Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB,
	})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	mode := webhook.ParseMode(cfg.BotMode)
	// نکته: قبلاً این اتصال فقط در حالت webhook ساخته می‌شد، یعنی در حالت
	// polling نه memberresponder (member.check) نه license check-in کار
	// می‌کردند. حالا در هر دو حالت، وقتی NATS_URL تنظیم شده باشد، ساخته می‌شود.
	var wnc *natsclient.Client
	if cfg.NatsURL != "" {
		wnc, err = natsclient.New(natsclient.Config{URL: cfg.NatsURL, Username: cfg.NatsUser, Password: cfg.NatsPass, Name: "member-bot"})
		if err != nil {
			if mode == webhook.ModeWebhook {
				log.Fatal("nats connect (webhook mode)", ports.F("err", err))
			}
			log.Warn("nats unavailable — member.check/license check-in disabled", ports.F("err", err))
			wnc = nil
		}
	}
	if wnc != nil {
		log.AttachNATS(wnc, "member-bot", instanceID)
	}
	poller := webhook.BuildPoller(webhook.PollerConfig{
		Mode: mode, BotID: botID, Token: cfg.BotToken,
		GatewayURL: cfg.GatewayURL, NATS: wnc, Log: log,
	})
	botSettings := tele.Settings{Token: cfg.BotToken, Poller: poller}
	if cfg.LocalBotAPI != "" {
		botSettings.URL = cfg.LocalBotAPI
	}
	rawBot, err := tele.NewBot(botSettings)
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
	if err := botprofile.Sync(rawBot, botprofile.Config{
		Environment: cfg.AppEnv,
		ServiceName: botprofile.ServiceName(cfg.ServiceName, "Member Bot"),
	}); err != nil {
		log.Warn("production bot profile sync failed", ports.F("err", err))
	}
	h := tgbot.NewHandler(rawBot, st, cache, log, cfg.OwnerID, cfg.EncryptKey)
	tgbot.Register(rawBot, h)

	// events publisher — join/leave → membership.joined/left + activity → community.activity.updated
	// قبلاً هرگز ساخته نمی‌شد: همه‌ی رویدادهای عضویت بی‌صدا از دست می‌رفتند.
	if wnc != nil {
		fc := fraudclient.New(wnc)
		pub := joinevents.NewPublisher(wnc, fc, log)
		pub.Register(rawBot)
		h.SetActivityPublisher(pub)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if wnc != nil {
		go licenseclient.RunLicenseLoop(ctx, wnc, botID, cfg.LicenseToken, cfg.ServerID, log)
	}

	// FIX 17: start scheduler
	sched := scheduler.New(st, rawBot, log)
	sched.Start(ctx)

	// Lock HTTP API
	lockServer := lock.NewServer(cache, log, cfg.LockAPIPort, cfg.LockAPIKey)
	go func() {
		if err := lockServer.Start(); err != nil {
			log.Fatal("lock api", ports.F("err", err))
		}
	}()

	// NATS responder (member.check) — مسیر متمرکز چک عضویت برای bot های فرعی
	if wnc != nil {
		mresp := memberresponder.New(wnc, cache, log)
		if err := mresp.Start(); err != nil {
			log.Error("member responder start failed", ports.F("err", err))
		}
	}

	// Worker dispatcher
	disp := dispatcher.New(st, cache, log, cfg.EncryptKey)
	go func() {
		if err := disp.Start(ctx); err != nil && ctx.Err() == nil {
			log.Fatal("dispatcher", ports.F("err", err))
		}
	}()

	go func() { <-ctx.Done(); rawBot.Stop() }()
	log.Info("member-bot started")
	rawBot.Start()
}
