package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mrjvadi/creatorbot/apimanager/internal/handler"
	"github.com/mrjvadi/creatorbot/apimanager/internal/middleware"
	"github.com/mrjvadi/creatorbot/shared-core/agentlistener"
	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	pgadapter "github.com/mrjvadi/creatorbot/shared/pkg/adapters/postgres"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/config"
	"github.com/mrjvadi/creatorbot/shared/pkg/logger"
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	PostgresDSN   string `mapstructure:"POSTGRES_DSN"`
	NatsURL       string `mapstructure:"NATS_URL"`
	NatsUser      string `mapstructure:"NATS_USERNAME"`
	NatsPass      string `mapstructure:"NATS_PASSWORD"`
	Port          string `mapstructure:"API_PORT"`
	AccessSecret  string `mapstructure:"JWT_ACCESS_SECRET"`
	RefreshSecret string `mapstructure:"JWT_REFRESH_SECRET"`
	EncryptionKey string `mapstructure:"ENCRYPTION_KEY"`
	AgentAPIKey   string `mapstructure:"AGENT_API_KEY"`
	BotToken      string `mapstructure:"BOT_TOKEN"`
	// ServiceHMACSecret پایه‌ی الگوی auth سرویس‌به‌سرویس (ComputeServiceKey/ValidateServiceKey)
	// است — برای اینکه apimanager بتواند از طرف خودش با botpay/pay.credit صحبت کند
	// (POST /admin/users/:id/credit). اگر خالی باشد، آن endpoint fail-closed می‌شود
	// (رجوع به Handler.payClient در internal/handler/handler.go).
	ServiceHMACSecret string `mapstructure:"SERVICE_HMAC_SECRET"`
	// IMAGE_REGISTRY_URL/IMAGE_REGISTRY_ADMIN_KEY برای اتصال به سرویسِ جداگانه‌ی
	// image-registry — رجوع به internal/handler/image_registry.go برای جزئیات و هشدارِ
	// «مشخصات این سرویس هنوز تأیید نشده». اگر URL خالی باشد، آن endpoint ها 503 می‌دهند.
	ImageRegistryURL      string `mapstructure:"IMAGE_REGISTRY_URL"`
	ImageRegistryAdminKey string `mapstructure:"IMAGE_REGISTRY_ADMIN_KEY"`
}

func main() {
	var cfg Config
	config.MustLoad(&cfg)
	log := logger.MustNew(false)

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// ── PostgreSQL ─────────────────────────────────────────
	pg, err := pgadapter.New(pgadapter.Config{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("postgres", ports.F("err", err))
	}
	if err := pg.Conn().AutoMigrate(models.AllModels()...); err != nil {
		log.Fatal("migrate", ports.F("err", err))
	}
	st := store.New(pg)

	// ── NATS ───────────────────────────────────────────────
	nc, err := natsclient.New(natsclient.Config{
		URL:      cfg.NatsURL,
		Username: cfg.NatsUser,
		Password: cfg.NatsPass,
	})
	if err != nil {
		log.Fatal("nats", ports.F("err", err))
	}
	defer nc.Close()
	log.AttachNATS(nc, "apimanager")

	// ── Docker Manager (NATS-based) ────────────────────────
	dockerManager := sharedocker.NewManager(nc)

	// ── Pay client (فقط برای عملیات ادمین مثل افزودن اعتبار دستی) ─────
	// اگر SERVICE_HMAC_SECRET تنظیم نشده، به‌جای panic/رفتار نامشخص، nil می‌ماند و
	// Handler خودش fail-closed می‌شود (رجوع به AddUserCredit).
	var payClient *natspayclient.Client
	if cfg.ServiceHMACSecret != "" {
		payClient = natspayclient.New(nc, nil, natspayclient.Config{
			ServiceID:  "apimanager",
			ServiceKey: auth.ComputeServiceKey(cfg.ServiceHMACSecret, "apimanager"),
		})
	} else {
		log.Warn("SERVICE_HMAC_SECRET not set — admin manual-credit endpoint disabled")
	}

	// ── Handler ────────────────────────────────────────────
	h := handler.New(
		st, dockerManager, nc, log,
		cfg.AccessSecret, cfg.RefreshSecret,
		cfg.EncryptionKey, cfg.AgentAPIKey,
		cfg.BotToken, payClient,
		cfg.ImageRegistryURL, cfg.ImageRegistryAdminKey,
	)

	// heartbeat/result از NATS توسط botmanager (queue group "managers") پردازش می‌شود.
	// apimanager فقط HTTP fallback endpoints دارد (/agent/heartbeat, /agent/result با AgentAPIKey).

	// ── Routes ────────────────────────────────────────────
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "service": "apimanager"})
	})

	v1 := r.Group("/api/v1")

	// Public
	v1.POST("/auth/telegram", h.TelegramAuth)
	v1.POST("/auth/refresh", h.RefreshToken)
	v1.POST("/agent/auth", h.AgentAuth)

	// Agent webhook (با AgentAPIKey)
	agent := v1.Group("/agent", middleware.AgentKeyAuth(cfg.AgentAPIKey))
	agent.POST("/heartbeat", h.AgentHeartbeat)
	agent.POST("/result", h.AgentResult)

	// User (JWT)
	// rate limiting: 60 req/min per IP
	limiter := middleware.NewSimpleLimiter(60)
	v1.Use(middleware.RateLimit(limiter, 60))

	user := v1.Group("", middleware.JWTAuth(cfg.AccessSecret))
	user.GET("/me", h.Me)
	user.GET("/instances", h.ListInstances)
	user.POST("/instances", h.CreateInstance)
	user.POST("/instances/:id/start", h.StartInstance)
	user.POST("/instances/:id/stop", h.StopInstance)
	user.POST("/instances/:id/restart", h.RestartInstance)
	user.DELETE("/instances/:id", h.DeleteInstance)
	user.GET("/instances/:id/logs", h.GetInstanceLogs)
	user.GET("/instances/:id/settings", h.GetInstanceSettings)
	user.PUT("/instances/:id/settings", h.UpdateInstanceSettings)
	// مدیریتِ محتوای uploader-bot از پنل وب، بدون بازکردن تلگرام — رجوع
	// internal/handler/uploader_proxy.go و apimanager/NEEDS.md.
	user.GET("/instances/:id/uploader/codes", h.ListUploaderCodes)
	user.DELETE("/instances/:id/uploader/codes/:codeId", h.DeleteUploaderCode)
	user.GET("/instances/:id/uploader/folders", h.ListUploaderFolders)
	user.POST("/instances/:id/uploader/folders", h.CreateUploaderFolder)
	user.DELETE("/instances/:id/uploader/folders/:folderId", h.DeleteUploaderFolder)
	user.GET("/plans", h.ListPlans)
	user.POST("/plans/:id/buy", h.BuyPlan)
	user.GET("/wallet/balance", h.GetWalletBalance)
	user.POST("/wallet/topup", h.CreateWalletTopup)
	user.GET("/wallet/topup/:code/status", h.GetWalletTopupStatus)
	user.GET("/service-types", h.ListServiceTypes)
	user.GET("/templates", h.ListTemplatesByType)
	user.GET("/payments", h.ListMyPayments)

	// Admin (JWT + Admin role)
	admin := v1.Group("/admin",
		middleware.JWTAuth(cfg.AccessSecret),
		middleware.RequireRole("admin", "owner"))
	admin.GET("/stats", h.AdminStats)
	admin.GET("/instances", h.ListAllInstancesAdmin)
	admin.GET("/servers", h.ListServers)
	admin.POST("/servers", h.CreateServer)
	admin.PATCH("/servers/:id", h.UpdateServer)
	admin.DELETE("/servers/:id", h.DeleteServer)
	admin.GET("/servers/:id/instances", h.ListServerInstances)
	admin.POST("/instances/:id/migrate", h.MigrateInstance)
	admin.PATCH("/instances/:id", h.UpdateInstanceAdmin)
	admin.GET("/templates", h.ListTemplates)
	admin.POST("/templates", h.CreateTemplate)
	admin.PATCH("/templates/:id", h.UpdateTemplate)
	admin.DELETE("/templates/:id", h.DeleteTemplate)
	admin.GET("/users", h.ListUsers)
	admin.GET("/users/:id", h.GetUser)
	admin.POST("/users/:id/role", h.SetUserRole)
	admin.POST("/users/:id/block", h.BlockUser)
	admin.POST("/users/:id/unblock", h.UnblockUser)
	admin.POST("/users/:id/credit", h.AddUserCredit)
	admin.GET("/plans", h.ListAllPlans)
	admin.POST("/plans", h.CreatePlan)
	admin.PATCH("/plans/:id", h.UpdatePlan)
	admin.PATCH("/plans/:id/limits", h.UpdatePlanLimit)
	admin.DELETE("/plans/:id", h.DeletePlan)
	admin.GET("/payments", h.ListAllPaymentsAdmin)
	admin.GET("/promo-codes", h.ListPromoCodesAdmin)
	admin.POST("/promo-codes", h.CreatePromoCode)
	admin.PATCH("/promo-codes/:id", h.SetPromoCodeActive)
	admin.DELETE("/promo-codes/:id", h.DeletePromoCode)
	// image-registry proxy — رجوع به internal/handler/image_registry.go برای هشدارِ
	// «مشخصات این سرویس هنوز با خودش تست نشده».
	admin.GET("/images", h.ListRegistryImages)
	admin.POST("/images", h.CreateRegistryImage)
	admin.POST("/images/:id/file", h.UploadRegistryImageFile)
	admin.PATCH("/images/:id", h.UpdateRegistryImage)
	admin.DELETE("/images/:id", h.DeleteRegistryImage)
	admin.GET("/images/check", h.CheckRegistryImage)
	admin.GET("/image-callers", h.ListRegistryCallers)
	admin.POST("/image-callers", h.CreateRegistryCaller)
	admin.PATCH("/image-callers/:id", h.UpdateRegistryCaller)
	admin.DELETE("/image-callers/:id", h.DeleteRegistryCaller)

	// ── Start server ───────────────────────────────────────
	addr := ":" + cfg.Port
	srv := &http.Server{Addr: addr, Handler: r}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// این جاروبِ دوره‌ای قبلاً اصلاً صدا زده نمی‌شد — یعنی is_online یک سرور بعد از این‌که
	// heartbeat هایش واقعاً قطع می‌شدند، هیچ‌وقت به false برنمی‌گشت (تا ابد «آنلاین» می‌ماند).
	// بازخورد کاربر ۲۰۲۶-۰۷-۰۳ («بخش سرورها رو درست کن») همین را هم شامل می‌شود؛ آستانه‌ی
	// ۶۰ ثانیه هماهنگ با پنجره‌ی ۳۰ ثانیه‌ای FindBestOnlineServer است (کمی سخاوتمندانه‌تر تا
	// یک/دو heartbeat جاافتاده باعث false positive نشود).
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := st.MarkStaleServersOffline(context.Background(), 60); err != nil {
					log.Warn("mark stale servers offline failed", ports.F("err", err))
				}
			}
		}
	}()

	go func() {
		metrics.ServeMetrics(":9090")
		log.Info("metrics server started", ports.F("addr", ":9090"))
		log.Info("apimanager started", ports.F("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("api server", ports.F("err", err))
		}
	}()

	<-ctx.Done()
	log.Info("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}

// ── NATS event handlers — delegate به shared-core/agentlistener ────────────

func handleHeartbeat(data []byte, st *store.Store, log ports.Logger) {
	agentlistener.HandleHeartbeat(context.Background(), data, st, log)
}

func handleResult(data []byte, st *store.Store, log ports.Logger) {
	agentlistener.HandleResult(context.Background(), data, st, log)
}
