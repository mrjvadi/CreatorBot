package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	tele "gopkg.in/telebot.v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/botpay/internal/api"
	"github.com/mrjvadi/creatorbot/botpay/internal/consensus"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/tgbot"
	"github.com/mrjvadi/creatorbot/botpay/internal/ton"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
)

type Config struct {
	BotToken    string `mapstructure:"BOT_TOKEN"`
	OwnerID     int64  `mapstructure:"OWNER_ID"`
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	APIPort     int    `mapstructure:"API_PORT"`
	AdminAPIKey string `mapstructure:"ADMIN_API_KEY"`

	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`

	// TON
	TONMasterAddress string `mapstructure:"TON_MASTER_ADDRESS"`
	TONAPIKey        string `mapstructure:"TON_API_KEY"`
	TONNetwork       string `mapstructure:"TON_NETWORK"`
	ConsensusDBDir   string `mapstructure:"CONSENSUS_DB_DIR"`

	// Service API keys — هر سرویس یک key دارد
	// فرمت: SERVICE_KEY_<SERVICE_ID>=key
	BotManagerKey string `mapstructure:"SERVICE_KEY_BOTMANAGER"`
	UploaderKey   string `mapstructure:"SERVICE_KEY_UPLOADER"`
	VPNKey        string `mapstructure:"SERVICE_KEY_VPN"`
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

	// ── REST API ──────────────────────────────────────────
	serviceKeys := map[string]string{
		"botmanager": cfg.BotManagerKey,
		"uploader":   cfg.UploaderKey,
		"vpn":        cfg.VPNKey,
	}
	apiHandler := api.New(walletSvc, api.Config{
		ServiceKeys: serviceKeys,
		AdminKey:    cfg.AdminAPIKey,
	}, log)

	r := gin.New()
	r.Use(gin.Recovery())
	apiHandler.Register(r)

	apiAddr := fmt.Sprintf(":%d", cfg.APIPort)
	srv := &http.Server{Addr: apiAddr, Handler: r}

	// ── Telegram Bot ──────────────────────────────────────
	rawBot, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10},
	})
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	h := tgbot.New(walletSvc, st, cfg.OwnerID, log)
	tgbot.Register(rawBot, h)
	h.SetBot(rawBot) // فعال‌سازی push notification

	// ── Start ─────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go watcher.Run(ctx)
	go func() {
		log.Info("botpay API started", ports.F("addr", apiAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("api", ports.F("err", err))
		}
	}()
	go func() { <-ctx.Done(); rawBot.Stop() }()

	metrics.ServeMetrics(":9091")
	log.Info("botpay started",
		ports.F("bot", rawBot.Me.Username),
		ports.F("api", apiAddr),
		ports.F("ton_address", cfg.TONMasterAddress))
	rawBot.Start()

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}
