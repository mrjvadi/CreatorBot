// Package engine یک engine کامل برای هر bot فراهم می‌کند.
// هر bot container این engine را داخل خودش دارد و مستقیماً به DB وصل است.
//
// آنچه داخل engine هر bot هست:
//   - اتصال مستقیم به PostgreSQL (با bot_id filter)
//   - اتصال مستقیم به MongoDB (با instance_id = bot_<bot_id>)
//   - اتصال مستقیم به Redis (با prefix bot_<bot_id>:)
//   - NATS فقط برای heartbeat و رویدادهای cross-service
//
// apimanager دیگر در مسیر hot path نیست.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/configstore"
	"github.com/mrjvadi/creatorbot/shared-core/docstore"
	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/mongodb"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	sharedredis "github.com/mrjvadi/creatorbot/shared/pkg/adapters/redis"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Config تنظیمات engine هر bot.
// همه از env vars تزریق می‌شوند.
type Config struct {
	// از env توکن استخراج می‌شود — نیازی نیست دستی set شود
	BotToken string

	// DB — همه bot ها به همان PostgreSQL و MongoDB وصل می‌شوند
	// bot_id برای filter استفاده می‌شود
	PostgresDSN string
	MongoURI    string
	MongoDB     string
	RedisAddr   string
	RedisPass   string
	RedisDB     int

	// NATS — فقط برای heartbeat و events
	NatsURL  string
	NatsUser string
	NatsPass string
	ServerID string // UUID سرور در جدول servers

	// Heartbeat interval
	HeartbeatSec int

	// LicenseToken توکنی که botmanager هنگام deploy این instance از
	// license-service گرفته و به‌عنوان env var LICENSE_TOKEN تزریق کرده.
	// خالی‌بودن یعنی این instance با یک نسخه‌ی قدیمی‌تر از قبل از راه‌اندازی
	// license-service ساخته شده — engine در این حالت فقط هشدار می‌دهد و
	// کار را متوقف نمی‌کند (fail-open، تا مشتریان موجود قطع نشوند).
	LicenseToken string
}

// Engine موتور اصلی هر bot — شامل همه dependency های لازم.
type Engine struct {
	cfg        Config
	BotID      int64  // از توکن استخراج می‌شود
	InstanceID string // "bot_<BotID>"

	// DB connections
	DB    ports.DB
	Mongo ports.DocumentStore
	Cache ports.Cache

	// Document stores با auto bot_id filter
	Settings *docstore.SettingStore
	Stats    *docstore.StatStore
	Users    *docstore.BotUserStore

	// NATS
	Nats *natsclient.Client

	// Config — در-حافظه config از MongoDB
	Config *configstore.Store

	Log ports.Logger

	// InstanceInfo اطلاعات این instance از جدول bot_instances (PlanID, LockMode).
	// در Start() پر می‌شود. اگر هنوز خوانده نشده یا یافت نشد، nil است.
	InstanceInfo *InstanceInfo
}

// InstanceInfo زیرمجموعه‌ی فیلدهای BotInstance که bot فرعی به آن نیاز دارد
// تا بفهمد قفل کانالش رایگان است یا اجاره‌ای (بدون import مدل کامل botmanager).
type InstanceInfo struct {
	PlanID   string // uuid به‌صورت رشته، خالی = بدون پلن
	LockMode string // "free" | "rented" | "none"
}

// IsFreeLock یعنی این instance بخشی از تبلیغ رایگان پلتفرم است.
func (i *InstanceInfo) IsFreeLock() bool { return i != nil && i.LockMode == "free" }

// IsRentedLock یعنی قفل این instance به یک کمپین اجاره‌ای وصل است.
func (i *InstanceInfo) IsRentedLock() bool { return i != nil && i.LockMode == "rented" }

// New یک engine جدید می‌سازد.
// همه connection ها را برقرار می‌کند.
func New(cfg Config, log ports.Logger) (*Engine, error) {
	// ── استخراج Bot ID از توکن ────────────────────────────────
	botID, err := extractBotID(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("engine: invalid bot token: %w", err)
	}
	instanceID := fmt.Sprintf("bot_%d", botID)

	log.Info("engine starting",
		ports.F("bot_id", botID),
		ports.F("instance_id", instanceID))

	// ── PostgreSQL ────────────────────────────────────────────
	// bot مستقیماً به DB وصل می‌شود
	// همه query ها باید WHERE bot_id = ? داشته باشند
	db, err := postgres.New(postgres.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		return nil, fmt.Errorf("engine: postgres: %w", err)
	}

	// ── MongoDB ───────────────────────────────────────────────
	// instance_id = "bot_<botID>" به‌صورت خودکار به همه query ها اضافه می‌شود
	ds, err := mongodb.New(mongodb.Config{
		URI:      cfg.MongoURI,
		Database: cfg.MongoDB,
	})
	if err != nil {
		return nil, fmt.Errorf("engine: mongodb: %w", err)
	}

	// ── Redis ─────────────────────────────────────────────────
	// prefix = "bot_<botID>:" برای ایزولاسیون
	cache, err := sharedredis.New(sharedredis.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
		// KeyPrefix اضافه می‌شود
	})
	if err != nil {
		return nil, fmt.Errorf("engine: redis: %w", err)
	}

	// ── Document Stores ───────────────────────────────────────
	settings := docstore.NewSettingStore(ds, instanceID)
	stats := docstore.NewStatStore(ds, instanceID)
	users := docstore.NewBotUserStore(ds, instanceID)

	// ── Config از MongoDB ────────────────────────────────────
	cfgStore := configstore.New(ds, instanceID)

	e := &Engine{
		cfg:        cfg,
		BotID:      botID,
		InstanceID: instanceID,
		DB:         db,
		Mongo:      ds,
		Cache:      cache,
		Settings:   settings,
		Stats:      stats,
		Users:      users,
		Config:     cfgStore,
		Log:        log,
	}

	// ── NATS (اختیاری) ────────────────────────────────────────
	if cfg.NatsURL != "" {
		nc, err := natsclient.New(natsclient.Config{
			URL:      cfg.NatsURL,
			Username: cfg.NatsUser,
			Password: cfg.NatsPass,
			Name:     fmt.Sprintf("bot-%d", botID),
		})
		if err != nil {
			// NATS اختیاری است — بدون آن هم bot کار می‌کند
			log.Info("NATS not available — heartbeat disabled",
				ports.F("err", err))
		} else {
			e.Nats = nc
		}
	}

	return e, nil
}

// Start heartbeat را شروع می‌کند (اگه NATS وصل باشد).
func (e *Engine) Start(ctx context.Context) {
	// ── خواندن PlanID/LockMode این instance از bot_instances ──
	e.loadInstanceInfo(ctx)

	// ── بارگذاری config از MongoDB ────────────────────────────
	if _, err := e.Config.Load(ctx); err != nil {
		e.Log.Error("config load failed", ports.F("err", err))
	} else {
		e.Log.Info("config loaded", ports.F("bot_id", e.BotID))
	}

	// ── subscribe به config.updated از NATS ──────────────────
	if e.Nats != nil {
		configstore.RegisterNATSHandler(e.Config, e.Nats, e.Log)
		// fallback poller در صورت قطع NATS
		go e.Config.RunFallbackPoller(ctx)
	}

	if e.Nats == nil || e.cfg.ServerID == "" {
		return
	}

	interval := time.Duration(e.cfg.HeartbeatSec) * time.Second
	if interval == 0 {
		interval = 30 * time.Second
	}

	go e.heartbeatLoop(ctx, interval)
	go licenseclient.RunLicenseLoop(ctx, e.Nats, e.BotID, e.cfg.LicenseToken, e.cfg.ServerID, e.Log)

	e.Log.Info("engine started",
		ports.F("bot_id", e.BotID),
		ports.F("heartbeat", interval))
}

// Close همه connection ها را می‌بندد.
func (e *Engine) Close(ctx context.Context) {
	if e.Nats != nil {
		e.Nats.Close()
	}
	if e.Mongo != nil {
		e.Mongo.Close(ctx)
	}
}

// heartbeatLoop وضعیت bot را به apimanager ارسال می‌کند.
// loadInstanceInfo از جدول bot_instances، PlanID و LockMode این bot را
// با BotID خودش پیدا می‌کند. عمداً raw query است (نه import مدل botmanager)
// تا engine به shared-core/models وابسته نشود.
func (e *Engine) loadInstanceInfo(ctx context.Context) {
	if e.DB == nil {
		return
	}
	var row struct {
		PlanID   *string
		LockMode string
	}
	err := e.DB.Conn().WithContext(ctx).
		Table("bot_instances").
		Select("plan_id, lock_mode").
		Where("bot_id = ?", e.BotID).
		Take(&row).Error
	if err != nil {
		e.Log.Warn("instance info not found — running without plan context",
			ports.F("bot_id", e.BotID), ports.F("err", err))
		return
	}

	info := &InstanceInfo{LockMode: row.LockMode}
	if row.PlanID != nil {
		info.PlanID = *row.PlanID
	}
	e.InstanceInfo = info
	e.Log.Info("instance info loaded",
		ports.F("bot_id", e.BotID), ports.F("lock_mode", info.LockMode))
}

func (e *Engine) heartbeatLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// اول فوری
	e.sendHeartbeat(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.sendHeartbeat(ctx)
		}
	}
}

func (e *Engine) sendHeartbeat(ctx context.Context) {
	msg := protocol.HeartbeatMsg{
		Type:      protocol.MsgHeartbeat,
		ServerID:  e.cfg.ServerID,
		Timestamp: time.Now().Unix(),
		// bot containers فقط خودشان را report می‌کنند
		Containers: []protocol.ContainerStatus{
			{
				Name:  fmt.Sprintf("bot_%d", e.BotID),
				State: "running",
			},
		},
	}

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := e.Nats.Publish(pubCtx,
		protocol.HeartbeatSubject(e.cfg.ServerID), msg); err != nil {
		e.Log.Error("heartbeat failed", ports.F("err", err))
	}
}

// PublishPaymentEvent رویداد پرداخت را به apimanager ارسال می‌کند.
func (e *Engine) PublishPaymentEvent(ctx context.Context, event protocol.PaymentConfirmedEvent) error {
	if e.Nats == nil {
		return nil
	}
	event.BotID = e.BotID
	event.Timestamp = time.Now().Unix()
	return e.Nats.Publish(ctx, protocol.PaymentEventSubject(e.BotID), event)
}

// SubscribeInstanceEvents رویدادهای تغییر instance را subscribe می‌کند.
// مثلاً وقتی apimanager instance را expire می‌کند.
func (e *Engine) SubscribeInstanceEvents(handler func(protocol.InstanceUpdatedEvent)) error {
	if e.Nats == nil {
		return nil
	}
	err := e.Nats.Subscribe(
		protocol.InstanceEventSubject(e.BotID),
		func(data []byte) {
			var event protocol.InstanceUpdatedEvent
			if err := jsonUnmarshal(data, &event); err != nil {
				e.Log.Error("SubscribeInstanceEvents: unmarshal failed",
					ports.F("err", err))
				return
			}
			handler(event)
		},
	)
	return err
}

// ── helpers ───────────────────────────────────────────────

func extractBotID(token string) (int64, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid format: expected '<id>:<hash>'")
	}
	var id int64
	if _, err := fmt.Sscanf(parts[0], "%d", &id); err != nil {
		return 0, fmt.Errorf("non-numeric bot id: %s", parts[0])
	}
	return id, nil
}

func jsonUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
