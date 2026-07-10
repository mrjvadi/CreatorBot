package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/sourceworker"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot"
	"github.com/mrjvadi/creatorbot/shared-core/agentlistener"
	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared-core/ton"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	BotToken    string `mapstructure:"BOT_TOKEN"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	LocalBotAPI string `mapstructure:"LOCAL_BOT_API"`

	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	EncryptKey string `mapstructure:"ENCRYPTION_KEY"`
	// TON payment
	TONWallet   string `mapstructure:"TON_WALLET_ADDRESS"`
	TONAPIKey   string `mapstructure:"TON_API_KEY"`
	TONNetwork  string `mapstructure:"TON_NETWORK"`
	BotpayURL   string `mapstructure:"BOTPAY_URL"`
	BotpayKey   string `mapstructure:"BOTPAY_API_KEY"`
	BotpaySvcID string `mapstructure:"BOTPAY_SERVICE_ID"`

	// ServiceHMACSecret همان راز مادری است که botpay هم دارد — این سرویس
	// کلید pay.* خودش را از آن مشتق می‌کند (auth.ComputeServiceKey)، به‌جای
	// یک کلید ثابت جداگانه که باید دستی هماهنگ نگه داشته شود.
	ServiceHMACSecret string `mapstructure:"SERVICE_HMAC_SECRET"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// ── PostgreSQL ────────────────────────────────────────────
	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	_ = db.Migrate(models.AllModels()...)

	// ── Redis (wizard state) ───────────────────────────────────
	cache, err := sharedredis.New(sharedredis.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	// ── NATS ─────────────────────────────────────────────────
	var nc *natsclient.Client
	if cfg.NatsURL != "" {
		nc, err = natsclient.New(natsclient.Config{
			URL:      cfg.NatsURL,
			Username: cfg.NatsUser,
			Password: cfg.NatsPass,
			Name:     "botmanager",
		})
		if err != nil {
			log.Fatal("nats", ports.F("err", err))
		}
		defer nc.Close()
		log.Info("nats connected")
		log.AttachNATS(nc, "botmanager")
	} else {
		log.Info("NATS not configured — docker commands disabled")
	}

	// ── Docker Manager (از طریق NATS) ────────────────────────
	dockerManager := sharedocker.NewManager(nc)

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
	log.Info("bot connected", ports.F("username", rawBot.Me.Username))

	// ── Wire ──────────────────────────────────────────────────
	st := store.New(db)
	tonClient := ton.New(ton.Config{
		WalletAddress: cfg.TONWallet,
		APIKey:        cfg.TONAPIKey,
		Network:       cfg.TONNetwork,
	})
	var payClient *natspayclient.Client
	var licenseClient *licenseclient.Client
	if nc != nil {
		if cfg.ServiceHMACSecret == "" {
			log.Error("SERVICE_HMAC_SECRET not set — botpay/license-service will reject all requests from botmanager")
		}
		payClient = natspayclient.New(nc, cache, natspayclient.Config{
			ServiceID:  "botmanager",
			ServiceKey: auth.ComputeServiceKey(cfg.ServiceHMACSecret, "botmanager"),
		})
		log.Info("botpay connected via NATS")
		if err := payClient.SubscribeWalletUpdates(); err != nil {
			log.Error("wallet updates subscription failed", ports.F("err", err))
		}
		licenseClient = licenseclient.New(nc, licenseclient.Config{
			ServiceID:  "botmanager",
			ServiceKey: auth.ComputeServiceKey(cfg.ServiceHMACSecret, "botmanager"),
		})
	}
	h := tgbot.NewHandler(rawBot, st, cache, dockerManager, log, cfg.OwnerID, cfg.EncryptKey, tonClient, payClient, nc, licenseClient)
	tgbot.Register(rawBot, h)

	// ── NATS: دریافت heartbeat و نتایج Docker ─────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if nc != nil {
		// heartbeat — queue group "managers": فقط یک سرویس از بین botmanager/apimanager پردازش می‌کند
		nc.QueueSubscribe("agent.*.heartbeat", "managers", func(data []byte) {
			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			agentlistener.HandleHeartbeat(cctx, data, st, log)
		})

		// نتایج دستورات (Deploy/Stop/Remove) — منطق کامل در shared-core/agentlistener
		nc.QueueSubscribe("agent.*.result", "managers", func(data []byte) {
			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			agentlistener.HandleResult(cctx, data, st, log)
		})

		log.Info("NATS listeners started")

		// source-service worker contract (source.worker.register/heartbeat/update)
		// — شرح در shared-core/protocol/source_worker.go و
		// shared/PENDING_CHANGES.md. تا امروز پیاده نشده بود؛ اگر
		// SERVICE_HMAC_SECRET ست نباشد، پاسخ‌ها همیشه unauthorized خواهند بود
		// (همان رفتاری که pay.*/license.* هم دارند).
		if err := sourceworker.Register(nc, st, sourceworker.Config{
			ServiceHMACSecret: cfg.ServiceHMACSecret,
			EncryptKey:        cfg.EncryptKey,
		}, log); err != nil {
			log.Error("source-service worker responders failed to register", ports.F("err", err))
		}
	}

	// ── یادآورِ انقضای سرویس‌ها (job پس‌زمینه) ────────────────
	h.StartExpiryReminders(ctx)

	// ── سرورهای بی‌heartbeat را offline نشان بده ─────────────
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := st.MarkStaleServersOffline(context.Background(), 60); err != nil {
					log.Warn("mark stale servers offline", ports.F("err", err))
				}
			}
		}
	}()

	go func() {
		<-ctx.Done()
		log.Info("shutting down botmanager...")
		rawBot.Stop()
	}()

	log.Info("botmanager started", ports.F("owner", cfg.OwnerID))
	rawBot.Start()
}
