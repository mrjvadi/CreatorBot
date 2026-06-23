package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	tele "gopkg.in/telebot.v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/botpay/internal/chainguard"
	"github.com/mrjvadi/creatorbot/botpay/internal/consensus"
	"github.com/mrjvadi/creatorbot/botpay/internal/payresponder"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/tgbot"
	"github.com/mrjvadi/creatorbot/botpay/internal/ton"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	BotToken    string `mapstructure:"BOT_TOKEN"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	// Redis — botpay موجودی را مستقیم در Redis می‌نویسد
	RedisAddr string `mapstructure:"REDIS_ADDR"`
	RedisPass string `mapstructure:"REDIS_PASSWORD"`
	RedisDB   int    `mapstructure:"REDIS_DB"`

	// TON
	TONMasterAddress string `mapstructure:"TON_MASTER_ADDRESS"`
	TONAPIKey        string `mapstructure:"TON_API_KEY"`
	TONNetwork       string `mapstructure:"TON_NETWORK"`
	ConsensusDBDir   string `mapstructure:"CONSENSUS_DB_DIR"`

	// DefaultLang زبان پیش‌فرض ربات وقتی کاربر هنوز زبانی انتخاب نکرده.
	DefaultLang string `mapstructure:"DEFAULT_LANG"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	var err error
	log := logger.MustNew(false)

	// ── PostgreSQL ────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN))
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(db)

	// ── NATS ─────────────────────────────────────────────
	// NATS اختیاری است — اگه down باشد، botpay بدون event publishing ادامه می‌دهد
	var nc *natsclient.Client
	if cfg.NatsURL != "" {
		nc, err = natsclient.New(natsclient.Config{
			URL:      cfg.NatsURL,
			Username: cfg.NatsUser,
			Password: cfg.NatsPass,
			Name:     "botpay",
		})
		if err != nil {
			log.Error("nats unavailable — running in standalone mode", ports.F("err", err))
			nc = nil
		} else {
			defer nc.Close()
			log.Info("nats connected")
		}
	} else {
		log.Warn("NATS_URL not set — event publishing disabled")
	}

	// ── Consensus Engine ────────────────────────────────────
	dbDir := cfg.ConsensusDBDir
	if dbDir == "" {
		dbDir = "./data/consensus"
	}
	consensusEngine := consensus.NewEngine(consensus.Config{
		Threshold: 3,
		Timeout:   5 * time.Second,
		DBDir:     dbDir,
	}, log)
	if err := consensus.SetupWorkers(consensusEngine, dbDir); err != nil {
		log.Fatal("consensus workers", ports.F("err", err))
	}
	guard := consensus.NewGuard(consensusEngine, log)
	log.Info("consensus ready", ports.F("workers", consensusEngine.WorkerCount()))

	// ── Wallet Service ────────────────────────────────────
	walletSvc := wallet.New(st, nc, log, cfg.TONMasterAddress, guard)

	// ── TON Watcher ───────────────────────────────────────
	watcher := ton.New(
		ton.Config{
			MasterAddress: cfg.TONMasterAddress,
			APIKey:        cfg.TONAPIKey,
			Network:       cfg.TONNetwork,
			PollInterval:  15 * time.Second,
		},
		walletSvc.HandleDeposit,
		nc, log,
	)

	// ── NATS Responder (pay.* request/reply) ──────────────
	// همه‌ی سرویس‌ها برای موجودی/پرداخت فقط از این طریق با botpay حرف می‌زنند.
	// REST API حذف شده — ارتباط بین‌سرویسی کاملاً روی NATS است.
	if nc != nil {
		// Redis اختیاری — اگر در دسترس نباشد، botpay بدون cache ادامه می‌دهد
		var payCache ports.Cache
		if cfg.RedisAddr != "" {
			rc, rerr := sharedredis.New(sharedredis.Config{
				Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB,
			})
			if rerr != nil {
				log.Error("redis unavailable — balance cache disabled", ports.F("err", rerr))
			} else {
				payCache = rc
			}
		}
		resp := payresponder.New(nc, walletSvc, payCache, log)
		if err := resp.Start(); err != nil {
			log.Error("payresponder start failed", ports.F("err", err))
		}
	} else {
		log.Warn("NATS unavailable — pay request/reply disabled")
	}

	// ── Telegram Bot ──────────────────────────────────────
	rawBot, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10},
		URL:    "http://141.95.210.17:8081",
	})
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	h := tgbot.New(walletSvc, st, cfg.OwnerID, cfg.DefaultLang, log)
	tgbot.Register(rawBot, h)
	h.SetBot(rawBot) // فعال‌سازی push notification

	// ── Start ─────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── ChainGuard: پایش یکپارچگی زنجیره‌ی پرداخت‌ها ────────
	cg := chainguard.New(st, nc, log, cfg.OwnerID)
	cg.SetNotifier(func(telegramID int64, msg string) {
		_, _ = rawBot.Send(&tele.User{ID: telegramID}, msg)
	})
	go cg.Start(ctx)

	go watcher.Run(ctx)
	go func() { <-ctx.Done(); rawBot.Stop() }()

	metrics.ServeMetrics(":9091")
	log.Info("botpay started",
		ports.F("bot", rawBot.Me.Username),
		ports.F("ton_address", cfg.TONMasterAddress))
	rawBot.Start()
}
