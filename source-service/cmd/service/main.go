package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	sharednats "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"

	"github.com/mrjvadi/creatorbot/source-service/internal/botmanager"
	"github.com/mrjvadi/creatorbot/source-service/internal/bus"
	"github.com/mrjvadi/creatorbot/source-service/internal/models"
	"github.com/mrjvadi/creatorbot/source-service/internal/store"
	"github.com/mrjvadi/creatorbot/source-service/internal/task"
	"github.com/mrjvadi/creatorbot/source-service/internal/telegram"
	"github.com/mrjvadi/creatorbot/source-service/internal/userbot"
	"github.com/mrjvadi/creatorbot/source-service/internal/worker"
)

type Config struct {
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	RedisAddr   string `mapstructure:"REDIS_ADDR"`
	RedisPass   string `mapstructure:"REDIS_PASSWORD"`
	RedisDB     int    `mapstructure:"REDIS_DB"`
	NatsURL     string `mapstructure:"NATS_URL"`

	// ServiceID/ServiceKey authenticate this instance to botmanager as a
	// trusted core service — the same convention shared-core/protocol
	// already uses for license.issue/pay.credit. source-service is an
	// internal tool only core services may call, not a customer BotInstance.
	ServiceID  string `mapstructure:"SERVICE_ID"`
	ServiceKey string `mapstructure:"SERVICE_KEY"`

	// LicenseKey identifies this instance to botmanager, which activates it
	// and hands back a worker ID + the Telegram account (app id/hash/phone,
	// session encryption key) to run as. Give every instance its own key to
	// run more than one worker — see README.
	LicenseKey        string `mapstructure:"LICENSE_KEY"`
	ServiceHMACSecret string `mapstructure:"SERVICE_HMAC_SECRET"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	if cfg.ServiceID == "" {
		cfg.ServiceID = "source-service"
	}
	log := logger.MustNew(false)
	if cfg.ServiceHMACSecret == "" {
		log.Fatal("SERVICE_HMAC_SECRET is required for source task authorization")
	}

	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("db", ports.F("err", err))
	}
	db.Migrate(&models.ArchiveFile{}, &models.BotFileCache{}, &models.TelegramSession{}, &models.ChannelWatch{}, &models.NatsWatch{}, &models.Rule{})
	st := store.New(db)

	cache, err := sharedredis.New(sharedredis.Config{Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: cfg.RedisDB})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	nc, err := sharednats.New(sharednats.Config{URL: cfg.NatsURL})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	log.AttachNATS(nc, "source-service")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	bmCfg := botmanager.Config{ServiceID: cfg.ServiceID, ServiceKey: cfg.ServiceKey}

	// Activate this instance's license with botmanager to learn who we are
	// (worker ID), which Telegram account we operate, and the key that
	// encrypts its session at rest. Uses the shared contract in
	// shared-core/protocol (SubjSourceWorkerRegister) — the same package
	// botmanager imports.
	identity, err := worker.Register(ctx, nc, worker.Config{
		ServiceID:  cfg.ServiceID,
		ServiceKey: cfg.ServiceKey,
		LicenseKey: cfg.LicenseKey,
	})
	if err != nil {
		log.Fatal("worker registration", ports.F("err", err))
	}
	log.Info("worker registered", ports.F("worker_id", identity.ID))
	identity.StartHeartbeat(ctx, nc, log, 30*time.Second)

	// ⚠️ UserBot (MTProto) usage may violate Telegram's ToS — personal
	// archiving/automation use only.
	//
	// Session is stored encrypted in Postgres (not a file): a Docker volume
	// can be lost, but the database survives, so this worker never needs a
	// fresh Telegram login just because its container was recreated.
	sessionStorage, err := telegram.NewDBSessionStorage(st, identity.Telegram.Phone, identity.Telegram.SessionKey)
	if err != nil {
		log.Fatal("session storage", ports.F("err", err))
	}
	codeSource := telegram.NewNATSCodeSource(nc, worker.AuthCodeSubject(identity.ID), 5*time.Minute)
	tgClient := telegram.New(telegram.Config{
		AppID:          identity.Telegram.AppID,
		AppHash:        identity.Telegram.AppHash,
		Phone:          identity.Telegram.Phone,
		SessionStorage: sessionStorage,
	}, codeSource, log)

	go func() {
		if err := tgClient.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("telegram client stopped", ports.F("err", err))
		}
	}()

	// Every capability this worker exposes is registered here. Add new ones
	// in internal/userbot.Register — nothing else needs to change. userbot
	// also builds the generic rules.Engine (create_rule/list_rules/
	// delete_rule) on top of this same registry, so any task type is
	// automatically usable as a rule's "run_task" action.
	tasks := task.NewRegistry()
	ub := userbot.New(tgClient, st, nc, bmCfg, tasks, log, identity.Telegram.Phone)
	ub.Register(tasks)

	// Reload real-time rules from a previous run — fixed channel/NATS
	// watches plus generic rules — once the client is authorized/connected.
	go func() {
		select {
		case <-tgClient.Ready():
			ub.RestoreWatches(ctx, log)
			ub.RestoreNatsWatches(ctx)
			ub.RestoreRules(ctx)
		case <-ctx.Done():
		}
	}()

	b := bus.New(nc, cache, st, log, tasks, identity.ID, cfg.ServiceHMACSecret)
	log.Info("source-service worker started", ports.F("worker_id", identity.ID))
	if err := b.Start(ctx); err != nil {
		log.Fatal("bus", ports.F("err", err))
	}
}
