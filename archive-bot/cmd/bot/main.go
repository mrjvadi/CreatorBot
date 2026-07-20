package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/mongodb"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/webhook"
	"github.com/mrjvadi/creatorbot/shared/pkg/botprofile"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/joinevents"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/store"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/tgbot"
)

type Config struct {
	AppEnv      string `mapstructure:"APP_ENV"`
	ServiceName string `mapstructure:"BOT_SERVICE_NAME"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	BotToken    string `mapstructure:"BOT_TOKEN"`
	ChannelID   int64  `mapstructure:"CHANNEL_ID"`
	// این ربات هیچ Postgres ندارد؛ همه‌ی داده روی MongoDB است (دیتابیس
	// اختصاصیِ نوع سرویس archive-bot).
	MongoURI  string `mapstructure:"MONGO_URI"`
	MongoDB   string `mapstructure:"MONGO_DB"`
	RedisAddr string `mapstructure:"REDIS_ADDR"`
	RedisPass string `mapstructure:"REDIS_PASSWORD"`
	RedisDB   int    `mapstructure:"REDIS_DB"`

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

	// instanceID جدا می‌کند دیتای این deploy را از بقیه‌ی instanceهای archive-bot
	// که همگی روی همان دیتابیسِ مشترکِ Mongo (MONGO_DB=archive_bot) نشسته‌اند —
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

	// SWAP: replace sharedredis.New with any ports.Cache
	cache, err := sharedredis.New(sharedredis.Config{Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	mode := webhook.ParseMode(cfg.BotMode)
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
		log.AttachNATS(nc, "archive-bot", instanceID)
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
	if err := botprofile.Sync(rawBot, botprofile.Config{
		Environment: cfg.AppEnv,
		ServiceName: botprofile.ServiceName(cfg.ServiceName, "Archive Bot"),
	}); err != nil {
		log.Warn("production bot profile sync failed", ports.F("err", err))
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── وضعیتِ اجاره‌ی قفل (اگر این instance رایگان است) ─────────
	// جایگزینِ چیزی که uploader-bot قبلاً با Postgres/bot_instances.lock_mode
	// می‌خواند — archive-bot هرگز چنین چیزی نداشت؛ حالا با همان مکانیزمِ NATS
	// به ads-bot اضافه شد تا این نوع ربات هم بتواند رایگان/اجاره‌ای شود.
	rentalStatus := &memberclient.RentalStatus{}
	var joinPublisher *joinevents.Publisher
	if nc != nil {
		go memberclient.RunStatusLoop(ctx, nc, botID, rentalStatus, log)
		joinPublisher = joinevents.NewPublisher(nc, nil, log)
		joinPublisher.Gate = rentalStatus.IsInCampaign
		joinPublisher.CampaignID = rentalStatus.CampaignID
	}

	h := tgbot.NewHandler(rawBot, st, cache, log, cfg.OwnerID, rawBot.Me.Username,
		botID, nc, rentalStatus, joinPublisher)
	tgbot.Register(rawBot, h)

	if nc != nil {
		go licenseclient.RunLicenseLoop(ctx, nc, botID, cfg.LicenseToken, cfg.ServerID, log)
	}
	go func() { <-ctx.Done(); rawBot.Stop() }()

	log.Info("archive-bot started")
	rawBot.Start()
}
