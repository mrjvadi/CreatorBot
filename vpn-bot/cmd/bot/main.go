package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/marzban"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/nowpayments"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/webhook"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/zarinpal"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"

	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/payment"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/scheduler"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/store"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/tgbot"
)

type Config struct {
	BotToken  string `mapstructure:"BOT_TOKEN"`
	ChannelID int64  `mapstructure:"CHANNEL_ID"`
	AdminID   int64  `mapstructure:"OWNER_ID"`

	PostgresDSN string `mapstructure:"MASTER_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`

	PanelType string `mapstructure:"PANEL_TYPE"`
	PanelURL  string `mapstructure:"PANEL_URL"`
	PanelUser string `mapstructure:"PANEL_USERNAME"`
	PanelPass string `mapstructure:"PANEL_PASSWORD"`

	PaymentGateway   string `mapstructure:"PAYMENT_GATEWAY"`
	ZarinpalMerchant string `mapstructure:"ZARINPAL_MERCHANT"`
	NowpaymentsKey   string `mapstructure:"NOWPAYMENTS_KEY"`
	CardNumber       string `mapstructure:"CARD_NUMBER"`
	CardOwner        string `mapstructure:"CARD_OWNER"`

	// حالت دریافت update: polling (dev) یا webhook (production)
	BotMode    string `mapstructure:"BOT_MODE"`
	GatewayURL string `mapstructure:"GATEWAY_URL"`
	NatsURL    string `mapstructure:"NATS_URL"`
	NatsUser   string `mapstructure:"NATS_USERNAME"`
	NatsPass   string `mapstructure:"NATS_PASSWORD"`
	ServerID   string `mapstructure:"SERVER_ID"`

	EncryptKey string `mapstructure:"ENCRYPTION_KEY"`

	// LicenseToken توکنی که botmanager هنگام deploy از license-service
	// گرفته و به‌عنوان env var تزریق کرده — برای ضدکپی/ضدکلون.
	LicenseToken string `mapstructure:"LICENSE_TOKEN"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("db", ports.F("err", err))
	}
	db.Migrate(&models.User{}, &models.Panel{}, &models.Plan{}, &models.Subscription{},
		&models.DiscountCode{}, &models.Payment{}, &models.Setting{})

	cache, err := sharedredis.New(sharedredis.Config{
		Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB,
	})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	// VPN Panel — FIX 20: call Login() right after creation
	var panel ports.VPNPanel
	switch cfg.PanelType {
	case "marzban":
		panel = marzban.New(cfg.PanelURL, cfg.PanelUser, cfg.PanelPass)
	default:
		log.Fatal("unknown PANEL_TYPE", ports.F("type", cfg.PanelType))
	}
	loginCtx := context.Background()
	if err := panel.Login(loginCtx); err != nil {
		log.Fatal("panel login failed", ports.F("err", err))
	}

	// ── انتخاب حالت: polling (dev) یا webhook (production) ────
	mode := webhook.ParseMode(cfg.BotMode)
	botID := webhook.BotIDFromToken(cfg.BotToken)

	// نکته: قبلاً nc فقط در حالت webhook ساخته می‌شد و بدون username/password
	// (auth) — یعنی اگر NATS واقعاً auth الزامی داشت، این اتصال از اول رد
	// می‌شد. حالا همیشه با auth کامل ساخته می‌شود، هم برای وب‌هوک و هم برای
	// license check-in دوره‌ای (که در هر دو حالت لازم است).
	var nc *natsclient.Client
	if cfg.NatsURL != "" {
		nc, err = natsclient.New(natsclient.Config{
			URL: cfg.NatsURL, Username: cfg.NatsUser, Password: cfg.NatsPass, Name: "vpn-bot",
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
		log.AttachNATS(nc, "vpn-bot")
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
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	// در حالت webhook روی تلگرام SetWebhook می‌زنیم؛ در polling webhook قبلی حذف می‌شود
	if err := webhook.Setup(context.Background(), rawBot, webhook.PollerConfig{
		Mode: mode, Token: cfg.BotToken, GatewayURL: cfg.GatewayURL,
	}); err != nil {
		log.Error("webhook setup", ports.F("err", err))
	}
	var gateway ports.PaymentGateway
	switch cfg.PaymentGateway {
	case "zarinpal":
		gateway = zarinpal.New(cfg.ZarinpalMerchant)
	case "nowpayments":
		gateway = nowpayments.New(cfg.NowpaymentsKey)
	case "card":
		gateway = payment.NewCardGateway(cfg.CardNumber, cfg.CardOwner, rawBot, cfg.AdminID)
	default:
		log.Fatal("unknown PAYMENT_GATEWAY", ports.F("type", cfg.PaymentGateway))
	}

	st := store.New(db)
	h := tgbot.NewHandler(rawBot, st, panel, gateway, cache, log, cfg.ChannelID, cfg.AdminID, cfg.EncryptKey)
	tgbot.Register(rawBot, h)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if nc != nil {
		go licenseclient.RunLicenseLoop(ctx, nc, botID, cfg.LicenseToken, cfg.ServerID, log)
	}

	// FIX 16: start scheduler
	sched := scheduler.New(st, panel, rawBot, log)
	sched.Start(ctx)

	go func() { <-ctx.Done(); rawBot.Stop() }()
	log.Info("vpn-bot started")
	rawBot.Start()
}
