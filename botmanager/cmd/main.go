package main

import (
	"context"
	"encoding/json"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot"
	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared-core/ton"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
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
	if nc != nil {
		payClient = natspayclient.New(nc, cache, natspayclient.Config{
			ServiceID:  "botmanager",
			ServiceKey: cfg.BotpayKey,
		})
		log.Info("botpay connected via NATS")
		if err := payClient.SubscribeWalletUpdates(); err != nil {
			log.Error("wallet updates subscription failed", ports.F("err", err))
		}
	}
	h := tgbot.NewHandler(rawBot, st, cache, dockerManager, log, cfg.OwnerID, cfg.EncryptKey, tonClient, payClient, nc)
	tgbot.Register(rawBot, h)

	// ── NATS: دریافت heartbeat و نتایج Docker ─────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if nc != nil {
		// heartbeat
		nc.QueueSubscribe("agent.*.heartbeat", "botmanager", func(data []byte) {
			var msg protocol.HeartbeatMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			_ = st.MarkServerOnlineByServerID(cctx, msg.ServerID)
			for _, c := range msg.Containers {
				switch c.State {
				case "running":
					_ = st.UpdateInstanceStatusByContainerName(cctx, c.Name, models.StatusRunning)
				case "exited", "dead":
					_ = st.UpdateInstanceStatusByContainerName(cctx, c.Name, models.StatusStopped)
				}
			}
		})

		// نتایج دستورات
		nc.QueueSubscribe("agent.*.result", "botmanager", func(data []byte) {
			var msg protocol.ResultMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return
			}
			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if msg.Success {
				_ = st.UpdateInstanceStatusByContainerName(cctx, msg.ContainerName, models.StatusRunning)
			} else {
				_ = st.UpdateInstanceStatusByContainerName(cctx, msg.ContainerName, models.StatusError)
			}
			log.Info("docker result",
				ports.F("cmd", msg.CommandType),
				ports.F("success", msg.Success),
				ports.F("container", msg.ContainerName))
		})

		log.Info("NATS listeners started")
	}

	// ── یادآورِ انقضای سرویس‌ها (job پس‌زمینه) ────────────────
	h.StartExpiryReminders(ctx)

	go func() {
		<-ctx.Done()
		log.Info("shutting down botmanager...")
		rawBot.Stop()
	}()

	log.Info("botmanager started", ports.F("owner", cfg.OwnerID))
	rawBot.Start()
}
