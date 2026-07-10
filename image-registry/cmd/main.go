package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/mrjvadi/creatorbot/image-registry/internal/api"
	"github.com/mrjvadi/creatorbot/image-registry/internal/ipallow"
	"github.com/mrjvadi/creatorbot/image-registry/internal/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	PostgresDSN string `mapstructure:"POSTGRES_DSN"`
	Port        int    `mapstructure:"PORT"`

	// AdminKey برای مدیریت callerها (`/v1/callers`) — این تنها راهِ bootstrap
	// است، چون قبل از ثبت اولین AllowedCaller هیچ IP ای اجازه‌ی نوشتن ندارد.
	AdminKey string `mapstructure:"ADMIN_KEY"`

	// SeedCallerCIDR/Label — اختیاری: اگر ست شود و جدول callerها خالی باشد،
	// یک caller اولیه با CanWrite=true ساخته می‌شود (مثلاً IP ماشین ادمین)،
	// تا حتی نیازی به X-Admin-Key برای اولین قدم هم نباشد. خالی گذاشتنش هم
	// مشکلی ندارد — همیشه می‌شود با X-Admin-Key این کار را دستی انجام داد.
	SeedCallerCIDR  string `mapstructure:"SEED_CALLER_CIDR"`
	SeedCallerLabel string `mapstructure:"SEED_CALLER_LABEL"`

	// StorageDir محلی روی دیسک این سرویس که فایل‌های واقعی image (خروجی
	// `docker save`) در آن نگه‌داری می‌شوند — agentmanager دیگر از یک
	// registry بیرونی pull نمی‌کند، مستقیماً از همین سرویس دانلود می‌کند.
	// در production حتماً باید یک volume پایدار (نه دیسک موقت کانتینر) روی
	// این مسیر mount شود، وگرنه با هر restart/redeploy همه‌ی فایل‌ها از دست
	// می‌روند (فقط متادیتای DB باقی می‌ماند، بدون خودِ آرتیفکت).
	StorageDir string `mapstructure:"IMAGE_STORAGE_DIR"`

	// MaxImageFileSizeMB سقف اندازه‌ی فایل قابل‌آپلود (مگابایت) — بدون این،
	// یک آپلود عمدی/اشتباهِ خیلی بزرگ می‌تواند دیسک IMAGE_STORAGE_DIR را پر
	// کند. صفر یا منفی یعنی صریحاً بدون سقف (خاموش کردن عمدی، نه پیش‌فرض).
	MaxImageFileSizeMB int64 `mapstructure:"MAX_IMAGE_FILE_SIZE_MB"`

	// NATS اختیاری — فقط برای log.AttachNATS (جمع‌آوری لاگ مرکزی)، این
	// سرویس هیچ منطق کسب‌وکاری روی NATS ندارد (رجوع README برای اینکه چرا).
	NatsURL  string `mapstructure:"NATS_URL"`
	NatsUser string `mapstructure:"NATS_USERNAME"`
	NatsPass string `mapstructure:"NATS_PASSWORD"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.Port == 0 {
		cfg.Port = 8100
	}
	if cfg.AdminKey == "" {
		log.Error("ADMIN_KEY not set — /v1/callers management is unusable until configured")
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "data/images"
	}
	if err := os.MkdirAll(cfg.StorageDir, 0o755); err != nil {
		log.Fatal("image storage dir", ports.F("err", err), ports.F("dir", cfg.StorageDir))
	}
	if cfg.MaxImageFileSizeMB == 0 {
		cfg.MaxImageFileSizeMB = 8192 // پیش‌فرض ۸ گیگابایت؛ برای بدون‌سقف، صریحاً منفی بگذارید
	}

	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN))
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := store.AutoMigrate(db); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(db)

	if cfg.SeedCallerCIDR != "" {
		seedInitialCaller(db, cfg, log)
	}

	if cfg.NatsURL != "" {
		if nc, err := natsclient.New(natsclient.Config{
			URL: cfg.NatsURL, Username: cfg.NatsUser, Password: cfg.NatsPass, Name: "image-registry",
		}); err == nil {
			log.AttachNATS(nc, "image-registry")
			defer nc.Close()
		} else {
			log.Warn("nats unavailable — central log collection disabled", ports.F("err", err))
		}
	}

	checker := ipallow.New(st)
	var maxBytes int64
	if cfg.MaxImageFileSizeMB > 0 {
		maxBytes = cfg.MaxImageFileSizeMB * 1024 * 1024
	}
	h := api.New(st, checker, log, cfg.AdminKey, cfg.StorageDir, maxBytes)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	// حد پیش‌فرض gin برای بافر multipart در حافظه (۳۲ مگابایت) خیلی کوچک‌تر
	// از فایل‌های واقعی image است؛ بالاتر از این حد، gin خودش خودکار روی
	// فایل موقت دیسک spill می‌کند، پس این فقط برای کاهش I/O روی آپلودهای
	// معمولی است، نه یک سقف واقعی روی اندازه‌ی فایل.
	r.MaxMultipartMemory = 64 << 20 // 64 MiB
	h.Register(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("image-registry started", ports.F("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server", ports.F("err", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}

// seedInitialCaller یک AllowedCaller با CanWrite=true می‌سازد اگر جدول
// callerها کاملاً خالی باشد — idempotent (اگر از قبل چیزی هست، کاری نمی‌کند)
// تا هر بار restart سرویس رکورد تکراری نسازد.
func seedInitialCaller(db *gorm.DB, cfg Config, log ports.Logger) {
	var count int64
	db.Model(&store.AllowedCaller{}).Count(&count)
	if count > 0 {
		return
	}
	label := cfg.SeedCallerLabel
	if label == "" {
		label = "seed-admin"
	}
	seed := &store.AllowedCaller{
		Label: label, CIDR: cfg.SeedCallerCIDR, CanWrite: true, IsActive: true,
	}
	if err := db.Create(seed).Error; err != nil {
		log.Error("seed caller failed", ports.F("err", err))
		return
	}
	log.Info("seed caller created", ports.F("label", label), ports.F("cidr", cfg.SeedCallerCIDR))
}
