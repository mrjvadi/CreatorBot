package main

import (
	"context"
	"encoding/json"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/engine"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/ads-bot/internal/tgbot"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/fraudclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
)

type Config struct {
	BotToken      string `mapstructure:"BOT_TOKEN"`
	OwnerID       int64  `mapstructure:"OWNER_ID"`
	PostgresDSN   string `mapstructure:"POSTGRES_DSN"`
	RedisAddr     string `mapstructure:"REDIS_ADDR"`
	RedisPass     string `mapstructure:"REDIS_PASSWORD"`
	NatsURL       string `mapstructure:"NATS_URL"`
	NatsUser      string `mapstructure:"NATS_USERNAME"`
	NatsPass      string `mapstructure:"NATS_PASSWORD"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	// ── PostgreSQL ─────────────────────────────────────────
	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN))
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(db)
	
	ctx := context.Background()
	if err := st.SeedCategories(ctx); err != nil {
		log.Fatal("seed categories", ports.F("err", err))
	}

	// ── Redis ──────────────────────────────────────────────
	cache, err := redis.New(redis.Config{Addr: cfg.RedisAddr, Password: cfg.RedisPass, DB: 4})
	if err != nil {
		log.Fatal("redis", ports.F("err", err))
	}

	// ── NATS ──────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()

	// ── Telegram Bot ───────────────────────────────────────
	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal("bot", ports.F("err", err))
	}
	log.Info("bot connected", ports.F("username", b.Me.Username))

	// ── Engine + Analyzer ────────────────────────────────────
	broadcaster := engine.NewBroadcaster(b)
	fc := fraudclient.New(nc)
	eng := engine.New(st, nc, broadcaster, fc, log)

	// ── Handler ───────────────────────────────────────────
	payClient := natspayclient.New(nc, cache, natspayclient.Config{
		ServiceID:  "ads-bot",
		ServiceKey: "", // احراز پویا بر اساس service_id (نگاه کنید به botpay/payresponder)
	})

	h := tgbot.NewHandler(b, st, eng, cache, log, cfg.OwnerID, payClient)
	tgbot.Register(b, h)

	// ── subscribe: instance رایگان جدید از botmanager ────────
	// وقتی botmanager یک instance با قفل رایگان می‌سازد، اینجا به‌عنوان
	// یک FreeBotSlot ثبت می‌شود تا بعداً قابل اجاره به خریداران باشد.
	if nc != nil {
		_ = nc.Subscribe(protocol.SubjFreeBotCreated, func(data []byte) {
			var ev protocol.FreeBotCreatedEvent
			if err := json.Unmarshal(data, &ev); err != nil {
				log.Error("freebot.created: bad payload", ports.F("err", err))
				return
			}
			instID, err := uuid.Parse(ev.InstanceID)
			if err != nil {
				log.Error("freebot.created: bad instance_id", ports.F("err", err))
				return
			}
			slot := &store.FreeBotSlot{
				ID:            uuid.New(),
				BotInstanceID: instID,
				BotID:         ev.BotID,
			}
			if err := st.UpsertFreeBotSlot(context.Background(), slot); err != nil {
				log.Error("upsert free bot slot failed", ports.F("err", err))
				return
			}
			log.Info("free bot slot registered", ports.F("bot_id", ev.BotID))
		})

		// ── subscribe: عضویت واقعی کاربر (از member-bot) ─────────
		// اگر کانال هدف به یک کمپین اجاره‌ای فعال وصل باشد، پاداش
		// per-join همان لحظه پرداخت می‌شود (فاز ۶).
		_ = nc.Subscribe(protocol.SubjMembershipJoined, func(data []byte) {
			var ev protocol.MembershipJoinedEvent
			if err := json.Unmarshal(data, &ev); err != nil {
				log.Error("membership.joined: bad payload", ports.F("err", err))
				return
			}
			h.HandleMembershipJoined(context.Background(), ev.TelegramID, ev.CommunityID)
		})

		// ── subscribe: تشخیص تقلب (از fraud-engine) ──────────────
		// اگر کاربری که برای یک کانال هدف پاداش pending دارد fraud
		// تشخیص داده شود، آن پاداش قبل از تسویه لغو می‌شود.
		_ = nc.Subscribe("fraud.detected", func(data []byte) {
			var ev struct {
				TelegramID  int64 `json:"telegram_id"`
				CommunityID int64 `json:"community_id"`
			}
			if err := json.Unmarshal(data, &ev); err != nil {
				log.Error("fraud.detected: bad payload", ports.F("err", err))
				return
			}
			h.HandleFraudDetected(context.Background(), ev.TelegramID, ev.CommunityID)
		})

		// ── responder: bot فرعی می‌گوید «در کانال خریدار ادمین شدم» ──
		_ = nc.Respond(protocol.SubjConfirmChannelAdmin, func(data []byte) (any, error) {
			var req protocol.ConfirmChannelAdminRequest
			if err := json.Unmarshal(data, &req); err != nil {
				return protocol.ConfirmChannelAdminResponse{Error: "bad request"}, nil
			}
			if err := h.ConfirmChannelAdminByBotID(context.Background(), req.BotID); err != nil {
				log.Error("confirm channel admin failed", ports.F("err", err))
				return protocol.ConfirmChannelAdminResponse{Error: err.Error()}, nil
			}
			log.Info("channel admin confirmed", ports.F("bot_id", req.BotID))
			return protocol.ConfirmChannelAdminResponse{Success: true}, nil
		})
	}

	// ── Start ─────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go eng.RunScheduler(ctx)
	go h.RunSettlementScheduler(ctx)

	go func() {
		log.Info("ads-bot started", ports.F("owner", cfg.OwnerID))
		b.Start()
	}()

	<-ctx.Done()
	log.Info("shutting down...")
	b.Stop()
}
