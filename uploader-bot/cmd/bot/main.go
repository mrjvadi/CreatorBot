package main

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/webhook"
	"github.com/mrjvadi/creatorbot/shared/pkg/botprofile"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/joinevents"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/tgbot"
)

type Config struct {
	AppEnv      string `mapstructure:"APP_ENV"`
	ServiceName string `mapstructure:"BOT_SERVICE_NAME"`
	BotToken    string `mapstructure:"BOT_TOKEN"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	ChannelID   int64  `mapstructure:"CHANNEL_ID"`
	LocalBotAPI string `mapstructure:"LOCAL_BOT_API"`

	// EncryptKey برای رمزنگاری BotToken قفل‌های نوع «ربات» قبل از ذخیره در Mongo.
	EncryptKey string `mapstructure:"ENCRYPTION_KEY"`

	// DB — مستقیم، بدون واسط. این ربات هیچ داده‌ای در Postgres ندارد؛ همه‌چیز
	// روی MongoDB است (رجوع internal/models/models.go).
	MongoURI string `mapstructure:"MONGO_URI"`
	MongoDB     string `mapstructure:"MONGO_DB"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`

	// NATS — فقط برای heartbeat و events
	NatsURL    string `mapstructure:"NATS_URL"`
	NatsUser   string `mapstructure:"NATS_USERNAME"`
	NatsPass   string `mapstructure:"NATS_PASSWORD"`
	BotMode    string `mapstructure:"BOT_MODE"`
	GatewayURL string `mapstructure:"GATEWAY_URL"`
	ServerID   string `mapstructure:"SERVER_ID"`

	HeartbeatSec int `mapstructure:"HEARTBEAT_INTERVAL_SEC"`

	// LicenseToken توکنی که botmanager هنگام deploy از license-service
	// گرفته و به‌عنوان env var تزریق کرده — برای ضدکپی/ضدکلون.
	LicenseToken string `mapstructure:"LICENSE_TOKEN"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// ── Engine — همه DB connections و business logic ───────────
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
		LicenseToken: cfg.LicenseToken,
	}, log)
	if err != nil {
		log.Fatal("engine init failed", ports.F("err", err))
	}
	if eng.Nats != nil {
		log.AttachNATS(eng.Nats, "uploader-bot", fmt.Sprintf("bot_%d", eng.BotID))
	}

	// ── بررسی لایسنس در startup — fail-closed ────────────────
	// اگر LICENSE_TOKEN نباشد یا license-service آن را نپذیرد، ربات شروع نمی‌شود.
	{
		lctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		if err := licenseclient.RequireValid(lctx, eng.Nats, eng.BotID, cfg.LicenseToken, cfg.ServerID); err != nil {
			cancel()
			log.Fatal("license check failed — bot will not start", ports.F("err", err))
		}
		cancel()
		log.Info("license verified", ports.F("bot_id", eng.BotID))
	}

	// ── بدون migration پوستگرس ────────────────────────────────
	// این ربات هیچ داده‌ای در PostgreSQL نمی‌نویسد؛ همه‌ی داده‌ها در
	// MongoDB (engine.Mongo) به‌صورت سند ذخیره می‌شوند و schema لازم ندارند.

	// ── Telegram Bot ──────────────────────────────────────────
	mode := webhook.ParseMode(cfg.BotMode)
	poller := webhook.BuildPoller(webhook.PollerConfig{
		Mode: mode, BotID: eng.BotID, Token: cfg.BotToken,
		GatewayURL: cfg.GatewayURL, NATS: eng.Nats, Log: log,
	})
	settings := tele.Settings{
		Token:  cfg.BotToken,
		Poller: poller,
		// هندلر خطای سراسری — «message is not modified» بی‌خطر است و نادیده گرفته می‌شود.
		OnError: func(e error, _ tele.Context) {
			if e == nil || strings.Contains(e.Error(), "message is not modified") {
				return
			}
			log.Error("bot handler error", ports.F("err", e))
		},
	}
	if cfg.LocalBotAPI != "" {
		settings.URL = cfg.LocalBotAPI
	}
	rawBot, err := tele.NewBot(settings)
	if err != nil {
		log.Fatal("bot init failed", ports.F("err", err))
	}
	if err := botprofile.Sync(rawBot, botprofile.Config{
		Environment: cfg.AppEnv,
		ServiceName: botprofile.ServiceName(cfg.ServiceName, "Uploader Bot"),
	}); err != nil {
		log.Warn("production bot profile sync failed", ports.F("err", err))
	}
	if err == nil {
		if e := webhook.Setup(context.Background(), rawBot, webhook.PollerConfig{
			Mode: mode, Token: cfg.BotToken, GatewayURL: cfg.GatewayURL,
		}); e != nil {
			log.Error("webhook setup", ports.F("err", e))
		}
	}
	log.Info("uploader-bot starting",
		ports.F("bot_id", eng.BotID),
		ports.F("instance_id", eng.InstanceID))

	// ── Wire ──────────────────────────────────────────────────
	if cfg.EncryptKey == "" {
		log.Warn("ENCRYPTION_KEY not set — bot-lock tokens cannot be saved until configured")
	}

	// ── Graceful shutdown context (زودتر ساخته می‌شود چون RunStatusLoop زیر به آن نیاز دارد) ──
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── وضعیتِ اجاره‌ی قفل (اگر این instance رایگان است) ─────────
	// جایگزینِ کوئریِ مستقیمِ Postgres/bot_instances.lock_mode قبلی که با
	// قطعِ کاملِ Postgres از این ربات از کار افتاده بود — حالا ads-bot
	// (مالکِ واقعیِ FreeBotSlot/LockRentalCampaign) روی NATS پاسخ می‌دهد.
	rentalStatus := &memberclient.RentalStatus{}
	var joinPublisher *joinevents.Publisher
	if eng.Nats != nil {
		go memberclient.RunStatusLoop(ctx, eng.Nats, eng.BotID, rentalStatus, log)
		joinPublisher = joinevents.NewPublisher(eng.Nats, nil, log)
		joinPublisher.Gate = rentalStatus.IsInCampaign
		joinPublisher.CampaignID = rentalStatus.CampaignID
	}

	h := tgbot.NewHandler(tgbot.Deps{
		Engine:        eng,
		Bot:           rawBot,
		OwnerID:       cfg.OwnerID,
		ChannelID:     cfg.ChannelID,
		EncryptKey:    cfg.EncryptKey,
		RentalStatus:  rentalStatus,
		JoinPublisher: joinPublisher,
	})
	tgbot.Register(rawBot, h)
	h.EnsureDefaults(context.Background()) // ست‌کردن تنظیمات پیش‌فرض

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
