// Package handler contains all HTTP handlers for apimanager.
package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// maxAuthAge حداکثر عمرِ قابل‌قبول برای auth_date ویجت لاگین تلگرام، برای
// جلوگیری از replay یک payload معتبرِ قدیمی.
const maxAuthAge = 24 * time.Hour

// Handler holds dependencies for all API endpoints.
type Handler struct {
	store          *store.Store
	docker         *sharedocker.Manager
	nc             *natsclient.Client
	log            ports.Logger
	accessSecret   string
	refreshSecret  string
	encryptKey     string
	agentAPIKey    string
	telegramBotTok string
	// payClient اختیاری است — اگر SERVICE_HMAC_SECRET تنظیم نشده باشد nil می‌ماند و
	// endpoint هایی که نیازش دارند (مثل AddUserCredit) با 503 fail-closed می‌شوند،
	// نه با یک panic یا رفتار نامشخص.
	payClient *natspayclient.Client
	license   *licenseclient.Client
	// imageRegistryURL/imageRegistryAdminKey برای اتصال به سرویسِ جداگانه‌ی Image Registry —
	// رجوع به image_registry.go. اگر URL خالی باشد آن endpoint ها 503 برمی‌گردانند.
	imageRegistryURL      string
	imageRegistryAdminKey string
}

func New(
	st *store.Store,
	docker *sharedocker.Manager,
	nc *natsclient.Client,
	log ports.Logger,
	accessSecret, refreshSecret, encryptKey string,
	agentAPIKey string,
	telegramBotToken string,
	payClient *natspayclient.Client,
	license *licenseclient.Client,
	imageRegistryURL, imageRegistryAdminKey string,
) *Handler {
	return &Handler{
		store:                 st,
		docker:                docker,
		nc:                    nc,
		log:                   log,
		accessSecret:          accessSecret,
		refreshSecret:         refreshSecret,
		encryptKey:            encryptKey,
		agentAPIKey:           agentAPIKey,
		telegramBotTok:        telegramBotToken,
		payClient:             payClient,
		license:               license,
		imageRegistryURL:      imageRegistryURL,
		imageRegistryAdminKey: imageRegistryAdminKey,
	}
}

// ── helpers ────────────────────────────────────────────────

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"ok": false, "message": msg})
}

// ── Auth ──────────────────────────────────────────────────

// POST /api/v1/auth/telegram
func (h *Handler) TelegramAuth(c *gin.Context) {
	var req struct {
		ID         int64  `json:"id"`
		TelegramID int64  `json:"telegram_id"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		Username   string `json:"username"`
		PhotoURL   string `json:"photo_url"`
		AuthDate   int64  `json:"auth_date"`
		Hash       string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	telegramID := req.ID
	signedIDKey := "id"
	if telegramID == 0 {
		telegramID = req.TelegramID
		signedIDKey = "telegram_id"
	}
	if telegramID <= 0 {
		fail(c, http.StatusBadRequest, "telegram id is required")
		return
	}

	// ── تأیید امضای HMAC ویجت لاگین تلگرام ─────────────────
	// بدونِ توکنِ ربات پیکربندی‌شده، fail-closed: هیچ لاگینی پذیرفته نمی‌شود.
	if h.telegramBotTok == "" {
		fail(c, http.StatusServiceUnavailable, "telegram auth not configured")
		return
	}
	now := time.Now()
	if !validTelegramAuthTime(now, req.AuthDate) {
		fail(c, http.StatusUnauthorized, "auth data expired")
		return
	}
	fields := map[string]string{
		signedIDKey:  strconv.FormatInt(telegramID, 10),
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"username":   req.Username,
		"photo_url":  req.PhotoURL,
		"auth_date":  strconv.FormatInt(req.AuthDate, 10),
	}
	if !verifyTelegramAuth(fields, req.Hash, h.telegramBotTok) {
		fail(c, http.StatusUnauthorized, "invalid telegram signature; dev login token must match apimanager BOT_TOKEN")
		return
	}

	ctx := c.Request.Context()
	u, err := h.store.FindUserByTelegramID(ctx, telegramID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if u == nil {
		u = &models.User{
			TelegramID: telegramID,
			FirstName:  req.FirstName,
			Username:   req.Username,
			Role:       models.RoleUser,
		}
		if err := h.store.UpsertUser(ctx, u); err != nil {
			fail(c, http.StatusInternalServerError, "user create failed")
			return
		}
	}
	if u.IsBlocked {
		fail(c, http.StatusForbidden, "user is blocked")
		return
	}

	accessToken, err := auth.GenerateAccessToken(u.ID.String(), string(u.Role), auth.JWTConfig{AccessSecret: h.accessSecret, AccessTTL: 15 * time.Minute})
	if err != nil {
		fail(c, http.StatusInternalServerError, "token error")
		return
	}
	refreshToken, _ := auth.GenerateRefreshToken(u.ID.String(), string(u.Role), auth.JWTConfig{RefreshSecret: h.refreshSecret, RefreshTTL: 30 * 24 * time.Hour})

	ok(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user_id":       u.ID,
		"role":          u.Role,
	})
}

// POST /api/v1/auth/refresh
func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	claims, err := auth.ParseRefreshToken(req.RefreshToken, h.refreshSecret)
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	uid, parseErr := uuid.Parse(claims.UserID)
	if parseErr != nil {
		fail(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	u, lookupErr := h.store.FindUserByID(c.Request.Context(), uid)
	if lookupErr != nil || u == nil || u.IsBlocked {
		fail(c, http.StatusUnauthorized, "user is unavailable")
		return
	}
	accessToken, _ := auth.GenerateAccessToken(u.ID.String(), string(u.Role), auth.JWTConfig{AccessSecret: h.accessSecret, AccessTTL: 15 * time.Minute})
	ok(c, gin.H{"access_token": accessToken})
}

// ── Me ────────────────────────────────────────────────────

// GET /api/v1/me
func (h *Handler) Me(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString("user_id")
	uid, _ := uuid.Parse(userID)

	u, _ := h.store.FindUserByID(ctx, uid)
	if u == nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}

	sub, _ := h.store.GetActiveSubscription(ctx, u.ID)
	instances, _ := h.store.ListInstancesByOwner(ctx, u.ID)

	ok(c, gin.H{
		"user":         u,
		"subscription": sub,
		"bot_count":    len(instances),
	})
}

// ── Servers (admin) ───────────────────────────────────────

// serverResponse یک Server را برای پاسخ HTTP آماده می‌کند: LastContainers (متن JSON خام در
// دیتابیس) این‌جا parse می‌شود به آرایه‌ی واقعی، و online_seconds از OnlineSince محاسبه
// می‌شود (فقط وقتی سرور واقعاً آنلاین است — برای سرور آفلاین معنا ندارد). CPUPercent/
// MemoryUsedMB/MemoryTotalMB فعلاً همیشه nil هستند چون agentmanager هنوز آن‌ها را در
// heartbeat نمی‌فرستد — رجوع به کامنت روی HeartbeatMsg در shared-core/protocol. فرانت‌اند
// باید nil را «گزارش نشده» نشان بدهد، نه صفر.
// tagsToString/tagsFromString — Server.Tags روی دیتابیس comma-separated ذخیره می‌شود (مثل
// یک متن ساده، بدون نیاز به جدول جدا)؛ این دو فقط تبدیل رفت‌وبرگشت را برای handler و پاسخ
// HTTP انجام می‌دهند.
func tagsToString(tags []string) string {
	clean := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			clean = append(clean, t)
		}
	}
	return strings.Join(clean, ",")
}

func tagsFromString(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func serverResponse(s *models.Server) gin.H {
	var containers []protocol.ContainerStatus
	if s.LastContainers != "" {
		_ = json.Unmarshal([]byte(s.LastContainers), &containers)
	}
	if containers == nil {
		containers = []protocol.ContainerStatus{}
	}

	var onlineSeconds *int64
	if s.IsOnline && s.OnlineSince != nil {
		secs := int64(time.Since(*s.OnlineSince).Seconds())
		if secs < 0 {
			secs = 0
		}
		onlineSeconds = &secs
	}

	return gin.H{
		"id":              s.ID,
		"created_at":      s.CreatedAt,
		"updated_at":      s.UpdatedAt,
		"name":            s.Name,
		"ip":              s.IP,
		"is_online":       s.IsOnline,
		"last_seen":       s.LastSeen,
		"channel":         s.Channel,
		"online_since":    s.OnlineSince,
		"online_seconds":  onlineSeconds,
		"cpu_percent":     s.CPUPercent,
		"memory_used_mb":  s.MemoryUsedMB,
		"memory_total_mb": s.MemoryTotalMB,
		"containers":      containers,
		"tags":            tagsFromString(s.Tags),
		"max_containers":  s.MaxContainers,
	}
}

func serversResponse(list []models.Server) []gin.H {
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, serverResponse(&list[i]))
	}
	return out
}

func (h *Handler) ListServers(c *gin.Context) {
	servers, err := h.store.ListServers(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, serversResponse(servers))
}

func (h *Handler) CreateServer(c *gin.Context) {
	var req struct {
		Name          string   `json:"name" binding:"required"`
		IP            string   `json:"ip" binding:"required"`
		Tags          []string `json:"tags"`
		MaxContainers int      `json:"max_containers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.MaxContainers < 0 {
		req.MaxContainers = 0
	}
	srv := &models.Server{
		Name:          req.Name,
		IP:            req.IP,
		Channel:       fmt.Sprintf("server_%s", strings.ReplaceAll(req.Name, " ", "_")),
		Tags:          tagsToString(req.Tags),
		MaxContainers: req.MaxContainers,
	}
	if err := h.store.CreateServer(c.Request.Context(), srv); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, serverResponse(srv))
}

// PATCH /api/v1/admin/servers/:id — تگ‌ها و سقف container (بازخورد کاربر ۲۰۲۶-۰۷-۰۵).
// نام/آی‌پی هم قابل ویرایش‌اند چون تا الان هیچ راه ویرایشی برای سرور وجود نداشت (فقط
// ساخت/حذف)، ولی تمرکز اصلی همان تگ و سقف container است.
func (h *Handler) UpdateServer(c *gin.Context) {
	ctx := c.Request.Context()
	srv, err := h.store.FindServerByID(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if srv == nil {
		fail(c, http.StatusNotFound, "server not found")
		return
	}

	var req struct {
		Name          *string  `json:"name"`
		IP            *string  `json:"ip"`
		Tags          []string `json:"tags"`
		MaxContainers *int     `json:"max_containers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		srv.Name = *req.Name
	}
	if req.IP != nil {
		srv.IP = *req.IP
	}
	if req.Tags != nil {
		srv.Tags = tagsToString(req.Tags)
	}
	if req.MaxContainers != nil {
		if *req.MaxContainers < 0 {
			*req.MaxContainers = 0
		}
		srv.MaxContainers = *req.MaxContainers
	}

	if err := h.store.UpdateServer(ctx, srv); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, serverResponse(srv))
}

func (h *Handler) DeleteServer(c *gin.Context) {
	if err := h.store.DeleteServer(c.Request.Context(), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// GET /api/v1/admin/servers/:id/instances — چه instance هایی روی این سرور دیپلوی شده‌اند
// (بخش ۳ سند API_DESIGN.md). store متد اختصاصی «بر اساس سرور» ندارد، پس از همان
// ListAllInstances (که AdminStats هم استفاده می‌کند) فیلتر می‌کنیم.
func (h *Handler) ListServerInstances(c *gin.Context) {
	ctx := c.Request.Context()
	serverID := c.Param("id")

	instances, err := h.store.ListAllInstances(ctx)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}

	filtered := make([]interface{}, 0)
	for _, inst := range instances {
		if inst.ServerID.String() == serverID {
			filtered = append(filtered, inst)
		}
	}
	ok(c, filtered)
}

// POST /api/v1/admin/instances/:id/migrate — انتقال یک instance به سرور دیگر (بازخورد کاربر
// ۲۰۲۶-۰۷-۰۵: «بتونم کانتینر رو از یه سرور به سرور دیگه منتقل کنم»). فرایند: stop روی سرورِ
// فعلی (بدون remove — کانتینرِ متوقف‌شده روی سرورِ قبلی به‌عنوان یک نسخه‌ی پشتیبان تا وقتی
// خودِ ادمین صریحاً حذفش کند باقی می‌ماند)، سپس deploy تازه روی سرورِ مقصد با همان
// env overrides ذخیره‌شده، و در آخر به‌روزرسانی ServerID در دیتابیس. عمداً محدودیت تگ/سقف
// سرورهای مقصد (SelectLeastLoadedServer) این‌جا اعمال نمی‌شود چون این یک override دستیِ
// ادمین است، نه انتخاب خودکار.
func (h *Handler) MigrateInstance(c *gin.Context) {
	ctx := c.Request.Context()
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}

	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	newServer, err := h.store.FindServerByID(ctx, req.ServerID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if newServer == nil {
		fail(c, http.StatusNotFound, "target server not found")
		return
	}
	if !newServer.IsOnline {
		fail(c, http.StatusBadRequest, "target server is not online")
		return
	}
	if newServer.ID == inst.ServerID {
		fail(c, http.StatusBadRequest, "instance is already on this server")
		return
	}

	tmpl, err := h.store.FindTemplate(ctx, inst.TemplateID.String())
	if err != nil || tmpl == nil {
		fail(c, http.StatusInternalServerError, "template not found")
		return
	}
	plainToken, err := auth.Decrypt(inst.BotToken, h.encryptKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, "decrypt failed")
		return
	}

	// ── متوقف کردن روی سرورِ فعلی (بدون حذف) ─────────────────
	if err := h.publishCommand(ctx, inst, protocol.MsgStop); err != nil {
		h.log.Warn("migrate: stop on old server failed", ports.F("instance", inst.ID), ports.F("err", err))
	}

	// ── deploy روی سرورِ مقصد ──────────────────────────────
	if err := h.publishDeploy(ctx, inst, tmpl, newServer, plainToken); err != nil {
		fail(c, http.StatusInternalServerError, "deploy on target server failed")
		return
	}

	oldServerID := inst.ServerID
	if err := h.store.UpdateInstanceServer(ctx, inst.ID, newServer.ID); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)

	h.auditAdminAction(c, models.AuditAdminAction, inst.ID.String(), "instance_migrate",
		fmt.Sprintf("moved from server %s to %s", oldServerID, newServer.ID))

	ok(c, gin.H{"status": models.StatusPending, "server": serverResponse(newServer)})
}

// PATCH /api/v1/admin/instances/:id — ویرایش دستیِ یک instance از پنل اصلی ادمین (بازخورد
// کاربر ۲۰۲۶-۰۷-۰۵: «باید بتونم ربات‌ها رو از پنل ادمین ادیت کنم»). قبلاً هیچ endpoint ای
// برای ویرایش نبود، فقط start/stop/restart/delete/logs/settings/migrate. عمداً فقط فیلدهای
// «امن برای ویرایش دستی» را قابل‌تغییر می‌کند — container_name/server_id/bot_token از این
// مسیر دست‌نخورده می‌مانند چون تغییرشان بدون هماهنگی با کانتینر واقعی معنا ندارد (server_id
// از طریق endpoint اختصاصی migrate عوض می‌شود، نه این‌جا).
func (h *Handler) UpdateInstanceAdmin(c *gin.Context) {
	ctx := c.Request.Context()
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}

	var req struct {
		ExpiresAt *string `json:"expires_at"` // RFC3339؛ رشته‌ی خالی یعنی «بدون انقضا»
		PlanID    *string `json:"plan_id"`    // رشته‌ی خالی یعنی «بدون پلن»
		LockMode  *string `json:"lock_mode"`  // free | rented | none
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			inst.ExpiresAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				fail(c, http.StatusBadRequest, "invalid expires_at (must be RFC3339)")
				return
			}
			inst.ExpiresAt = &t
		}
	}
	if req.PlanID != nil {
		if *req.PlanID == "" {
			inst.PlanID = nil
		} else {
			pid, err := uuid.Parse(*req.PlanID)
			if err != nil {
				fail(c, http.StatusBadRequest, "invalid plan_id")
				return
			}
			inst.PlanID = &pid
		}
	}
	if req.LockMode != nil {
		mode := models.InstanceLockMode(*req.LockMode)
		if mode != models.LockModeFree && mode != models.LockModeRented && mode != models.LockModeNone {
			fail(c, http.StatusBadRequest, "invalid lock_mode")
			return
		}
		inst.LockMode = mode
	}

	if err := h.store.UpdateInstance(ctx, inst); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	h.auditAdminAction(c, models.AuditAdminAction, inst.ID.String(), "instance_edit", "admin edited instance fields")
	ok(c, inst)
}

// GET /api/v1/admin/instances?search=&status= — نمای کلی همه‌ی instance های پلتفرم.
// قبل از این، ادمین هیچ راه مستقیمی برای دیدن/پیداکردن یک instance نداشت جز غیرمستقیم از
// طریق «سرورها» یا «کاربران» — AdminStats همین ListAllInstances را برای شمارش صدا می‌زد،
// این‌جا همان لیست کامل (نه فقط شمارش) برمی‌گردد.
func (h *Handler) ListAllInstancesAdmin(c *gin.Context) {
	ctx := c.Request.Context()
	instances, err := h.store.ListAllInstances(ctx)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}

	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))

	if search == "" && status == "" {
		ok(c, instances)
		return
	}

	filtered := make([]interface{}, 0, len(instances))
	for _, inst := range instances {
		if status != "" && strings.ToLower(string(inst.Status)) != status {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(inst.ContainerName), search) {
			continue
		}
		filtered = append(filtered, inst)
	}
	ok(c, filtered)
}

// ── Templates (admin) ─────────────────────────────────────

// TemplateConfigField یک فیلدِ قابل‌تنظیم توسط کاربرِ نهایی برای instance های ساخته‌شده
// از یک قالب است — مثلاً «آیدی کانال». ادمین این‌ها را روی هر قالب تعریف می‌کند؛ مقدارِ
// واقعیِ هر کاربر جای دیگری (BotInstance.EnvOverrides) ذخیره می‌شود.
type TemplateConfigField struct {
	Key      string   `json:"key" binding:"required"`
	Label    string   `json:"label" binding:"required"`
	Type     string   `json:"type" binding:"required"` // string | number | boolean | select
	Required bool     `json:"required"`
	Default  string   `json:"default,omitempty"`
	Options  []string `json:"options,omitempty"` // فقط برای type=select
}

var validConfigFieldTypes = map[string]bool{"string": true, "number": true, "boolean": true, "select": true}

func validateConfigSchema(fields []TemplateConfigField) error {
	seen := make(map[string]bool, len(fields))
	for _, f := range fields {
		if f.Key == "" || f.Label == "" {
			return fmt.Errorf("each field needs a key and a label")
		}
		if seen[f.Key] {
			return fmt.Errorf("duplicate field key: %s", f.Key)
		}
		seen[f.Key] = true
		if !validConfigFieldTypes[f.Type] {
			return fmt.Errorf("invalid field type %q for key %q (must be string/number/boolean/select)", f.Type, f.Key)
		}
		if f.Type == "select" && len(f.Options) == 0 {
			return fmt.Errorf("field %q is type=select but has no options", f.Key)
		}
	}
	return nil
}

// templateResponse یک BotTemplate را برای پاسخ HTTP آماده می‌کند — ConfigSchema که در
// دیتابیس یک رشته‌ی متنی JSON است، این‌جا parse می‌شود تا در پاسخ یک آرایه‌ی واقعی باشد
// (نه یک رشته‌ی JSON تودرتو که فرانت‌اند مجبور باشد خودش دوباره parse کند).
func templateResponse(t *models.BotTemplate) gin.H {
	schema := []TemplateConfigField{}
	if t.ConfigSchema != "" {
		_ = json.Unmarshal([]byte(t.ConfigSchema), &schema)
	}
	return gin.H{
		"id":            t.ID,
		"created_at":    t.CreatedAt,
		"updated_at":    t.UpdatedAt,
		"name":          t.Name,
		"type":          t.Type,
		"image_name":    t.ImageName,
		"image_tag":     t.ImageTag,
		"description":   t.Description,
		"is_active":     t.IsActive,
		"is_free":       t.IsFree,
		"config_schema": schema,
	}
}

func templatesResponse(list []models.BotTemplate) []gin.H {
	out := make([]gin.H, 0, len(list))
	for i := range list {
		out = append(out, templateResponse(&list[i]))
	}
	return out
}

func (h *Handler) ListTemplates(c *gin.Context) {
	templates, err := h.store.ListTemplates(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, templatesResponse(templates))
}

func (h *Handler) CreateTemplate(c *gin.Context) {
	var req struct {
		Name         string                `json:"name" binding:"required"`
		Type         string                `json:"type" binding:"required"`
		ImageName    string                `json:"image_name" binding:"required"`
		ImageTag     string                `json:"image_tag" binding:"required"`
		ConfigSchema []TemplateConfigField `json:"config_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateConfigSchema(req.ConfigSchema); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	t := &models.BotTemplate{
		Name:      req.Name,
		Type:      req.Type,
		ImageName: req.ImageName,
		ImageTag:  req.ImageTag,
		IsActive:  true,
	}
	if len(req.ConfigSchema) > 0 {
		schemaJSON, _ := json.Marshal(req.ConfigSchema)
		t.ConfigSchema = string(schemaJSON)
	}
	if err := h.store.CreateTemplate(c.Request.Context(), t); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, templateResponse(t))
}

// PATCH /api/v1/admin/templates/:id — فعال/غیرفعال، تغییر image tag، تعریف تنظیمات قابل‌تغییر
// توسط کاربر (بخش ۳ سند API_DESIGN.md + بازخورد کاربر ۲۰۲۶-۰۷-۰۳ درباره‌ی تنظیمات قالب‌ها).
func (h *Handler) UpdateTemplate(c *gin.Context) {
	ctx := c.Request.Context()
	tmpl, err := h.store.FindTemplate(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if tmpl == nil {
		fail(c, http.StatusNotFound, "template not found")
		return
	}

	var req struct {
		Name         *string                `json:"name"`
		ImageName    *string                `json:"image_name"`
		ImageTag     *string                `json:"image_tag"`
		IsActive     *bool                  `json:"is_active"`
		IsFree       *bool                  `json:"is_free"`
		ConfigSchema *[]TemplateConfigField `json:"config_schema"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		tmpl.Name = *req.Name
	}
	if req.ImageName != nil {
		tmpl.ImageName = *req.ImageName
	}
	if req.ImageTag != nil {
		tmpl.ImageTag = *req.ImageTag
	}
	if req.IsActive != nil {
		tmpl.IsActive = *req.IsActive
	}
	if req.IsFree != nil {
		tmpl.IsFree = *req.IsFree
	}
	if req.ConfigSchema != nil {
		if err := validateConfigSchema(*req.ConfigSchema); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if len(*req.ConfigSchema) == 0 {
			tmpl.ConfigSchema = ""
		} else {
			schemaJSON, _ := json.Marshal(*req.ConfigSchema)
			tmpl.ConfigSchema = string(schemaJSON)
		}
	}

	if err := h.store.UpdateTemplate(ctx, tmpl); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, templateResponse(tmpl))
}

// DELETE /api/v1/admin/templates/:id
func (h *Handler) DeleteTemplate(c *gin.Context) {
	if err := h.store.DeleteTemplate(c.Request.Context(), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// ── Plans ─────────────────────────────────────────────────

func (h *Handler) ListPlans(c *gin.Context) {
	plans, err := h.store.ListPlans(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, plans)
}

// ── Wallet / خرید پلن ────────────────────────────────────────
//
// natspayclient.Client (CreateInvoice/InvoiceStatus/DeductForService/Balance) از قبل کاملاً
// پیاده‌سازی شده بود و فقط برای AddUserCredit ادمین استفاده می‌شد — هیچ endpoint ای برای
// خودِ کاربر برای شارژِ کیف‌پول یا خریدِ واقعیِ یک پلن وجود نداشت. یعنی صفحه‌ی «پلن‌ها» فقط
// نمایشی بود (بدون دکمه‌ی خرید) و جدول Payment همیشه خالی می‌ماند — دقیقاً همان چیزی که
// بازخورد کاربر ۲۰۲۶-۰۷-۰۵ می‌گفت («پرداختی‌ها کامل نشده»).

func (h *Handler) currentUser(c *gin.Context) (*models.User, error) {
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		return nil, fmt.Errorf("invalid user")
	}
	return h.store.FindUserByID(c.Request.Context(), uid)
}

// GET /api/v1/wallet/balance
func (h *Handler) GetWalletBalance(c *gin.Context) {
	if h.payClient == nil {
		fail(c, http.StatusServiceUnavailable, "pay client not configured")
		return
	}
	u, err := h.currentUser(c)
	if err != nil || u == nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	bal, err := h.payClient.Balance(c.Request.Context(), u.TelegramID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "balance fetch failed: "+err.Error())
		return
	}
	ok(c, bal)
}

// POST /api/v1/wallet/topup — ساخت invoice واریز TON (کاربر باید AmountTON را به
// MasterAddress با comment=Code بفرستد؛ این یک لینک پرداخت آنلاین نیست، بلکه یک واریز
// مستقیم TON با تطبیقِ comment تراکنش است — طبق معماری واقعیِ botpay).
func (h *Handler) CreateWalletTopup(c *gin.Context) {
	if h.payClient == nil {
		fail(c, http.StatusServiceUnavailable, "pay client not configured")
		return
	}
	u, err := h.currentUser(c)
	if err != nil || u == nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}

	var req struct {
		AmountTON float64 `json:"amount_ton" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.AmountTON <= 0 {
		fail(c, http.StatusBadRequest, "amount_ton must be positive")
		return
	}

	inv, err := h.payClient.CreateInvoice(c.Request.Context(), u.TelegramID, req.AmountTON, "wallet_topup")
	if err != nil {
		fail(c, http.StatusInternalServerError, "invoice creation failed: "+err.Error())
		return
	}
	ok(c, gin.H{
		"code":           inv.Code,
		"master_address": inv.MasterAddress,
		"amount_ton":     inv.AmountTON,
		"expires_at":     inv.ExpiresAt,
	})
}

// GET /api/v1/wallet/topup/:code/status
func (h *Handler) GetWalletTopupStatus(c *gin.Context) {
	if h.payClient == nil {
		fail(c, http.StatusServiceUnavailable, "pay client not configured")
		return
	}
	u, err := h.currentUser(c)
	if err != nil || u == nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	status, err := h.payClient.InvoiceStatus(c.Request.Context(), u.TelegramID, c.Param("code"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "status check failed: "+err.Error())
		return
	}
	ok(c, status)
}

// POST /api/v1/plans/:id/buy — خریدِ واقعیِ یک پلن. برای پلن‌های غیررایگان، از همان
// natspayclient.DeductForService استفاده می‌کند که از قبل برای دقیقاً همین کار نوشته شده بود
// کلید idempotency هر خرید از هدر Idempotency-Key می‌آید؛ نبود هدر یک UUID جدید می‌سازد.
// بنابراین retry آگاهانه با همان هدر کسر دوباره ندارد، ولی خرید مستقل همان plan هزینه‌ی جدا دارد.
func (h *Handler) BuyPlan(c *gin.Context) {
	ctx := c.Request.Context()
	u, err := h.currentUser(c)
	if err != nil || u == nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}

	plan, err := h.store.FindPlan(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if plan == nil {
		fail(c, http.StatusNotFound, "plan not found")
		return
	}
	if !plan.IsActive {
		fail(c, http.StatusBadRequest, "plan is not active")
		return
	}

	attemptID := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if attemptID == "" {
		attemptID = uuid.NewString()
	}
	charged := false
	var payment *models.Payment
	if !plan.IsFree && plan.Price > 0 {
		if h.payClient == nil {
			fail(c, http.StatusServiceUnavailable, "pay client not configured")
			return
		}
		if _, err := h.payClient.DeductForService(ctx, u.TelegramID, plan.Price, plan.ID.String(), attemptID); err != nil {
			if natspayclient.IsInsufficientBalance(err) {
				fail(c, http.StatusPaymentRequired, "insufficient balance")
				return
			}
			fail(c, http.StatusInternalServerError, "payment failed: "+err.Error())
			return
		}
		charged = true

		// ثبتِ رکورد پرداخت برای تاریخچه — best-effort: پول از قبل کسر شده، پس شکستِ ثبتِ
		// این رکورد نباید کل خرید را ناموفق نشان بدهد.
		now := time.Now()
		payment = &models.Payment{
			UserID:      u.ID,
			PlanID:      &plan.ID,
			Amount:      plan.Price,
			Currency:    "TON",
			Status:      models.PaymentDone,
			ConfirmedAt: &now,
			InvoiceID:   "plan:" + attemptID,
		}
	}

	var expiresAt *time.Time
	if plan.DurationDay > 0 {
		t := time.Now().AddDate(0, 0, plan.DurationDay)
		expiresAt = &t
	}
	sub := &models.Subscription{
		UserID:    u.ID,
		PlanID:    plan.ID,
		StartedAt: time.Now(),
		ExpiresAt: expiresAt,
		IsActive:  true,
	}
	if err := h.store.ActivateSubscription(ctx, sub); err != nil {
		if charged {
			if refundErr := h.payClient.RefundService(ctx, u.TelegramID, plan.Price, plan.ID.String()+":"+attemptID+":sub_create_failed"); refundErr != nil {
				h.log.Error("plan activation and refund failed", ports.F("err", err), ports.F("refund_err", refundErr), ports.F("user", u.ID))
			}
		}
		fail(c, http.StatusInternalServerError, "subscription creation failed")
		return
	}
	if payment != nil {
		if err := h.store.CreatePayment(ctx, payment); err != nil {
			h.log.Warn("create payment record failed", ports.F("err", err))
		}
	}
	if h.nc != nil {
		_ = h.nc.PublishCore("plan.upgraded", map[string]any{
			"user_id": u.ID, "telegram_id": u.TelegramID, "plan_id": plan.ID,
			"plan_name": plan.Name, "max_bots": plan.MaxBots,
		})
	}
	h.createAudit(c, u.ID, string(u.Role), models.AuditBuyPlan, plan.ID.String(), "plan", plan.Name)

	ok(c, gin.H{"subscription": sub, "attempt_id": attemptID})
}

// ── Service types / templates (کاربر عادی) ─────────────────
//
// این دو endpoint قبلاً وجود نداشتند — کاربر عادی هیچ راهی نداشت بفهمد موقع ساخت
// instance چه template_id ای باید بفرستد، جز این‌که یک UUID خام را از یک‌جای دیگر
// (مثلاً از ادمین) بگیرد. حالا از همان جریانی که در botmanager هست («انتخاب نوع سرویس،
// کاملاً پویا از DB → انتخاب تگ/نسخه»، PROJECT_UNDERSTANDING بخش ۳.۲) استفاده می‌کنیم —
// با متدهای store ای که از قبل دقیقاً برای همین‌کار وجود داشتند.

// GET /api/v1/service-types
func (h *Handler) ListServiceTypes(c *gin.Context) {
	types, err := h.store.ListServiceTypes(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, types)
}

// GET /api/v1/templates?type=uploader
func (h *Handler) ListTemplatesByType(c *gin.Context) {
	serviceType := c.Query("type")
	if serviceType == "" {
		fail(c, http.StatusBadRequest, "type query param is required")
		return
	}
	templates, err := h.store.ListTemplatesByType(c.Request.Context(), serviceType)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, templatesResponse(templates))
}

// ── Instances ────────────────────────────────────────────

// GET /api/v1/instances
func (h *Handler) ListInstances(c *gin.Context) {
	ctx := c.Request.Context()
	userID, _ := uuid.Parse(c.GetString("user_id"))
	instances, err := h.store.ListInstancesByOwner(ctx, userID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, instances)
}

// POST /api/v1/instances — ساخت و deploy instance جدید
func (h *Handler) CreateInstance(c *gin.Context) {
	var req struct {
		BotToken   string `json:"bot_token" binding:"required"`
		TemplateID string `json:"template_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	userID, _ := uuid.Parse(c.GetString("user_id"))

	// ── بررسی capacity ──────────────────────────────────────
	tmpl, err := h.store.FindTemplate(ctx, req.TemplateID)
	if err != nil || tmpl == nil {
		fail(c, http.StatusNotFound, "template not found")
		return
	}

	canCreate, current, limit, err := h.store.CanCreateInstance(ctx, userID, tmpl.Type)
	if err != nil {
		fail(c, http.StatusInternalServerError, "capacity check failed")
		return
	}
	if !canCreate {
		fail(c, http.StatusPaymentRequired, fmt.Sprintf(
			"bot limit reached (%d/%d) for type %s", current, limit, tmpl.Type))
		return
	}

	// ── پیدا کردن بهترین سرور ──────────────────────────────
	// SelectLeastLoadedServer (بار را بین سرورها پخش می‌کند و سقف MaxContainers را رعایت
	// می‌کند) به‌جای FindBestOnlineServer قبلی که فقط تازگیِ heartbeat را می‌دید و اصلاً بار
	// سرور را در نظر نمی‌گرفت. اگر قالب رایگان است، اول سراغ سرورهای تگ‌خورده با "free"
	// می‌رویم؛ اگر چنین سروری نبود، به هر سرورِ واجدشرایط برمی‌گردیم (بازخورد کاربر
	// ۲۰۲۶-۰۷-۰۵: «سرور با تگ free فقط پنل‌های رایگان را بگیرد»).
	var server *models.Server
	if tmpl.IsFree {
		server, err = h.store.SelectLeastLoadedServer(ctx, "free")
		if err == nil && server == nil {
			server, err = h.store.SelectLeastLoadedServer(ctx, "")
		}
	} else {
		server, err = h.store.SelectLeastLoadedServer(ctx, "")
	}
	if err != nil || server == nil {
		fail(c, http.StatusServiceUnavailable, "no server available")
		return
	}

	// ── استخراج Bot ID از توکن ──────────────────────────────
	botID, err := models.BotIDFromToken(req.BotToken)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid bot token format")
		return
	}

	// بررسی تکراری نبودن
	if existing, _ := h.store.FindInstanceByBotID(ctx, botID); existing != nil {
		fail(c, http.StatusConflict, fmt.Sprintf("bot already deployed: instance %s", existing.ID))
		return
	}

	// ── رمزنگاری توکن ──────────────────────────────────────
	encToken, err := auth.Encrypt(req.BotToken, h.encryptKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, "encryption failed")
		return
	}

	// ── ساخت instance در DB ─────────────────────────────────
	u, _ := h.store.FindUserByID(ctx, userID)
	var planID *uuid.UUID
	lockMode := models.LockModeNone
	if sub, _ := h.store.GetActiveSubscription(ctx, userID); sub != nil {
		planID = &sub.PlanID
		if plan, _ := h.store.FindPlan(ctx, sub.PlanID.String()); plan != nil && plan.IsFree {
			lockMode = models.LockModeFree
		}
	}
	if tmpl.IsFree {
		lockMode = models.LockModeFree
	}
	containerName := fmt.Sprintf("%s-%d", tmpl.Type, botID)
	inst := &models.BotInstance{
		OwnerID:       userID,
		TemplateID:    tmpl.ID,
		ServerID:      server.ID,
		BotToken:      encToken,
		BotID:         botID,
		ContainerName: containerName,
		DBSchema:      fmt.Sprintf("inst_%d", botID),
		Status:        models.StatusPending,
		PlanID:        planID,
		LockMode:      lockMode,
	}
	if err := h.store.CreateInstance(ctx, inst); err != nil {
		fail(c, http.StatusInternalServerError, "create instance failed")
		return
	}
	planIDStr := ""
	if planID != nil {
		planIDStr = planID.String()
	}
	if h.nc != nil {
		if lockMode == models.LockModeFree {
			_ = h.nc.PublishCore(protocol.SubjFreeBotCreated, protocol.FreeBotCreatedEvent{InstanceID: inst.ID.String(), BotID: botID})
		}
		_ = h.nc.PublishCore(protocol.ServiceCreationRequested, protocol.ServiceProvisionPayload{
			InstanceID: inst.ID.String(), OwnerID: userID.String(), ServiceType: tmpl.Type,
			PlanID: planIDStr,
		})
	}

	// ── ارسال deploy command به agentmanager ────────────────
	if err := h.publishDeploy(ctx, inst, tmpl, server, req.BotToken); err != nil {
		h.log.Error("deploy publish failed", ports.F("err", err))
		h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusError)
		fail(c, http.StatusInternalServerError, "deploy failed")
		return
	}

	_ = u
	h.log.Info("instance created",
		ports.F("instance", inst.ID),
		ports.F("server", server.Name),
		ports.F("type", tmpl.Type))

	ok(c, gin.H{
		"instance_id":    inst.ID,
		"container_name": containerName,
		"server":         server.Name,
		"status":         models.StatusPending,
	})
}

// POST /api/v1/instances/:id/start
func (h *Handler) StartInstance(c *gin.Context) {
	ctx := c.Request.Context()
	userID, parseErr := uuid.Parse(c.GetString("user_id"))
	if parseErr != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if inst.OwnerID != userID {
		fail(c, http.StatusForbidden, "instance access denied")
		return
	}
	if inst.Status == models.StatusRunning {
		fail(c, http.StatusConflict, "already running")
		return
	}

	if err := h.publishCommand(ctx, inst, protocol.MsgStart); err != nil {
		fail(c, http.StatusInternalServerError, "start command failed")
		return
	}

	h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)
	ok(c, gin.H{"status": models.StatusPending})
}

// POST /api/v1/instances/:id/stop
func (h *Handler) StopInstance(c *gin.Context) {
	ctx := c.Request.Context()
	userID, parseErr := uuid.Parse(c.GetString("user_id"))
	if parseErr != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if inst.OwnerID != userID {
		fail(c, http.StatusForbidden, "instance access denied")
		return
	}

	if err := h.publishCommand(ctx, inst, protocol.MsgStop); err != nil {
		fail(c, http.StatusInternalServerError, "stop command failed")
		return
	}

	h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusStopped)
	ok(c, gin.H{"status": models.StatusStopped})
}

// POST /api/v1/instances/:id/restart
func (h *Handler) RestartInstance(c *gin.Context) {
	ctx := c.Request.Context()
	userID, parseErr := uuid.Parse(c.GetString("user_id"))
	if parseErr != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if inst.OwnerID != userID {
		fail(c, http.StatusForbidden, "instance access denied")
		return
	}
	if err := h.publishCommand(ctx, inst, protocol.MsgRestart); err != nil {
		fail(c, http.StatusInternalServerError, "restart failed")
		return
	}

	h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)
	ok(c, gin.H{"status": models.StatusPending})
}

// DELETE /api/v1/instances/:id
func (h *Handler) DeleteInstance(c *gin.Context) {
	ctx := c.Request.Context()
	userID, parseErr := uuid.Parse(c.GetString("user_id"))
	if parseErr != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if inst.OwnerID != userID {
		fail(c, http.StatusForbidden, "instance access denied")
		return
	}

	// ارسال remove command
	h.publishCommand(ctx, inst, protocol.MsgRemove)

	// حذف از DB
	h.store.DeleteInstance(ctx, inst.ID)

	h.log.Info("instance deleted", ports.F("instance", inst.ID))
	ok(c, nil)
}

// GET /api/v1/instances/:id/logs
func (h *Handler) GetInstanceLogs(c *gin.Context) {
	ctx := c.Request.Context()
	userID, parseErr := uuid.Parse(c.GetString("user_id"))
	if parseErr != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if inst.OwnerID != userID {
		fail(c, http.StatusForbidden, "instance access denied")
		return
	}

	server, _ := h.store.FindServerByID(ctx, inst.ServerID.String())
	if server == nil {
		fail(c, http.StatusNotFound, "server not found")
		return
	}

	// logs را از Docker می‌خواهیم
	logs, err := h.docker.Logs(ctx, server.ID.String(), inst.ContainerID, 100)
	if err != nil {
		fail(c, http.StatusInternalServerError, "logs failed")
		return
	}

	ok(c, gin.H{"logs": logs})
}

// ── Instance settings (کاربر عادی) ──────────────────────────
//
// قالب می‌تواند فیلدهای قابل‌تنظیم تعریف کند (BotTemplate.ConfigSchema)؛ این دو endpoint به
// مالکِ instance اجازه می‌دهند مقدارِ آن فیلدها را برای رباتِ خودش ببیند/ذخیره کند. مقدارها در
// BotInstance.EnvOverrides ذخیره می‌شوند و روی publishDeploy تزریق می‌شوند — یعنی فقط بعد از
// یک start/restart واقعاً روی کانتینر اعمال می‌شوند (بازخورد کاربر ۲۰۲۶-۰۷-۰۳).
//
// بر خلافِ start/stop/restart/delete/logس که پیش‌تر بدون بررسیِ مالکیت پیاده‌سازی شده بودند،
// این‌جا OwnerID با کاربرِ JWT مقایسه می‌شود — چون تنظیمات می‌تواند اطلاعات حساس‌تری (مثلاً
// آیدی کانال خصوصی) داشته باشد. آن endpoint های قدیمی‌تر دست‌نخورده ماندند چون تغییرشان
// درخواست نشده بود؛ رجوع به memory برای این gap شناخته‌شده.

// GET /api/v1/instances/:id/settings
func (h *Handler) GetInstanceSettings(c *gin.Context) {
	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}

	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	// ادمین/owner می‌تواند تنظیمات هر instance ای را ببیند (برای پشتیبانی — بازخورد کاربر
	// ۲۰۲۶-۰۷-۰۵: از صفحه‌ی «همه‌ی ربات‌ها» فقط لاگ در دسترس بود، نه تنظیمات).
	role := c.GetString("role")
	isAdmin := role == "admin" || role == "owner"
	if inst.OwnerID != uid && !isAdmin {
		fail(c, http.StatusForbidden, "not your instance")
		return
	}

	tmpl, err := h.store.FindTemplate(ctx, inst.TemplateID.String())
	if err != nil || tmpl == nil {
		fail(c, http.StatusInternalServerError, "template not found")
		return
	}

	schema := []TemplateConfigField{}
	if tmpl.ConfigSchema != "" {
		_ = json.Unmarshal([]byte(tmpl.ConfigSchema), &schema)
	}
	values := map[string]string{}
	if inst.EnvOverrides != "" {
		_ = json.Unmarshal([]byte(inst.EnvOverrides), &values)
	}

	ok(c, gin.H{"schema": schema, "values": values})
}

// PUT /api/v1/instances/:id/settings
func (h *Handler) UpdateInstanceSettings(c *gin.Context) {
	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}

	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	role := c.GetString("role")
	isAdmin := role == "admin" || role == "owner"
	if inst.OwnerID != uid && !isAdmin {
		fail(c, http.StatusForbidden, "not your instance")
		return
	}

	tmpl, err := h.store.FindTemplate(ctx, inst.TemplateID.String())
	if err != nil || tmpl == nil {
		fail(c, http.StatusInternalServerError, "template not found")
		return
	}
	schema := []TemplateConfigField{}
	if tmpl.ConfigSchema != "" {
		_ = json.Unmarshal([]byte(tmpl.ConfigSchema), &schema)
	}
	if len(schema) == 0 {
		fail(c, http.StatusBadRequest, "this bot type has no configurable settings")
		return
	}

	var req struct {
		Values map[string]string `json:"values" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	allowed := make(map[string]bool, len(schema))
	for _, f := range schema {
		allowed[f.Key] = true
	}
	for k := range req.Values {
		if !allowed[k] {
			fail(c, http.StatusBadRequest, "unknown setting: "+k)
			return
		}
	}

	cleaned := map[string]string{}
	for _, f := range schema {
		v, present := req.Values[f.Key]
		if !present || v == "" {
			if f.Required {
				fail(c, http.StatusBadRequest, "missing required field: "+f.Key)
				return
			}
			continue
		}
		cleaned[f.Key] = v
	}

	envJSON, err := json.Marshal(cleaned)
	if err != nil {
		fail(c, http.StatusInternalServerError, "encode error")
		return
	}
	if err := h.store.UpdateInstanceEnvOverrides(ctx, inst.ID, string(envJSON)); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}

	if isAdmin && inst.OwnerID != uid {
		h.auditAdminAction(c, models.AuditAdminAction, inst.ID.String(), "instance_settings",
			"admin edited another user's bot settings")
	}

	ok(c, gin.H{"values": cleaned, "applied": false})
}

// ── Admin — Users ──────────────────────────────────────────

// GET /api/v1/admin/users?search=&role=
// فیلتر/جست‌وجو در همین‌جا (سمت apimanager) روی نتیجه‌ی store.ListUsers انجام می‌شود، چون
// store متدی برای فیلتر سمت DB ندارد؛ برای تعداد فعلی کاربران این پلتفرم (قبل از ده‌ها هزار
// رکورد) مشکلی ایجاد نمی‌کند. اگر تعداد کاربران خیلی زیاد شد، باید این فیلتر به لایه‌ی store
// منتقل شود.
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.store.ListUsers(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}

	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	roleFilter := strings.ToLower(strings.TrimSpace(c.Query("role")))

	if search == "" && roleFilter == "" {
		ok(c, users)
		return
	}

	filtered := make([]interface{}, 0, len(users))
	for _, u := range users {
		if roleFilter != "" && strings.ToLower(string(u.Role)) != roleFilter {
			continue
		}
		if search != "" {
			// فقط فیلدهایی که مستقیم در همین فایل (TelegramAuth) دیده شده و مطمئنیم روی
			// models.User وجود دارند: FirstName, Username, TelegramID, Role.
			haystack := strings.ToLower(u.FirstName + " " + u.Username + " " +
				strconv.FormatInt(u.TelegramID, 10))
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, u)
	}
	ok(c, filtered)
}

// GET /api/v1/admin/users/:id — جزئیات یک کاربر + لیست instance هایش (بخش ۳ سند API_DESIGN.md).
func (h *Handler) GetUser(c *gin.Context) {
	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}

	u, err := h.store.FindUserByID(ctx, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if u == nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}

	instances, _ := h.store.ListInstancesByOwner(ctx, u.ID)
	sub, _ := h.store.GetActiveSubscription(ctx, u.ID)

	ok(c, gin.H{
		"user":         u,
		"instances":    instances,
		"subscription": sub,
	})
}

// POST /api/v1/admin/users/:id/role — تغییر نقش کاربر (بخش ۳ سند API_DESIGN.md).
func (h *Handler) SetUserRole(c *gin.Context) {
	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	role := models.UserRole(req.Role)
	switch role {
	case models.RoleUser, models.RoleAdmin, models.RoleOwner:
	default:
		fail(c, http.StatusBadRequest, "invalid role")
		return
	}

	if err := h.store.SetUserRole(ctx, uid, role); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}

	h.auditAdminAction(c, models.AuditAdminAction, uid.String(), "user", fmt.Sprintf("role -> %s", role))
	ok(c, gin.H{"id": uid, "role": role})
}

// POST /api/v1/admin/users/:id/block
func (h *Handler) BlockUser(c *gin.Context) {
	h.setUserBlocked(c, true)
}

// POST /api/v1/admin/users/:id/unblock
func (h *Handler) UnblockUser(c *gin.Context) {
	h.setUserBlocked(c, false)
}

func (h *Handler) setUserBlocked(c *gin.Context, blocked bool) {
	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.store.SetUserBlocked(ctx, uid, blocked); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	action := models.AuditBlockUser
	h.auditAdminAction(c, action, uid.String(), "user", fmt.Sprintf("blocked=%v", blocked))
	ok(c, gin.H{"id": uid, "is_blocked": blocked})
}

// POST /api/v1/admin/users/:id/credit — افزودن اعتبار دستی (بخش ۳ سند API_DESIGN.md).
// از همان مسیر پولی رسمی پلتفرم رد می‌شود (natspayclient.Credit -> pay.credit روی botpay
// با auth ServiceID+ServiceKey) — apimanager هرگز مستقیم به موجودی کاربر نمی‌نویسد،
// طبق قانون بنیادی پروژه («botpay تنها نویسنده‌ی موجودی است»، رجوع PROJECT_UNDERSTANDING بخش ۵).
func (h *Handler) AddUserCredit(c *gin.Context) {
	if h.payClient == nil {
		fail(c, http.StatusServiceUnavailable, "pay client not configured")
		return
	}

	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		AmountTON float64 `json:"amount_ton" binding:"required"`
		Reason    string  `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.AmountTON <= 0 {
		fail(c, http.StatusBadRequest, "amount_ton must be positive")
		return
	}

	u, err := h.store.FindUserByID(ctx, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if u == nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}

	metadata := fmt.Sprintf(`{"admin_actor":%q}`, c.GetString("user_id"))
	if err := h.payClient.Credit(ctx, u.TelegramID, req.AmountTON, "admin_credit:"+req.Reason, metadata); err != nil {
		h.log.Error("admin credit failed", ports.F("user", uid), ports.F("err", err))
		fail(c, http.StatusBadGateway, "credit failed")
		return
	}

	h.auditAdminAction(c, models.AuditAdminAction, uid.String(), "user",
		fmt.Sprintf("credit +%.4f TON (%s)", req.AmountTON, req.Reason))
	ok(c, gin.H{"id": uid, "credited_ton": req.AmountTON})
}

// auditAdminAction تلاش best-effort برای ثبت audit log — شکستش نباید عملیات اصلی را fail کند.
func (h *Handler) auditAdminAction(c *gin.Context, action models.AuditAction, targetID, targetType, description string) {
	actorID, _ := uuid.Parse(c.GetString("user_id"))
	log := &models.AuditLog{
		ActorID:     actorID,
		ActorRole:   c.GetString("role"),
		Action:      action,
		TargetID:    targetID,
		TargetType:  targetType,
		Description: description,
		IPAddress:   c.ClientIP(),
	}
	if err := h.store.CreateAuditLog(c.Request.Context(), log); err != nil {
		h.log.Warn("audit log write failed", ports.F("err", err))
	}
}

// ── Admin — Plans ──────────────────────────────────────────

// GET /api/v1/admin/plans — همه‌ی پلن‌ها (فعال و غیرفعال)، برخلاف GET /plans کاربری که فقط فعال‌ها را می‌دهد.
func (h *Handler) ListAllPlans(c *gin.Context) {
	plans, err := h.store.ListAllPlans(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, plans)
}

// POST /api/v1/admin/plans
func (h *Handler) CreatePlan(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		DurationDay int     `json:"duration_day"`
		Price       float64 `json:"price"`
		MaxBots     int     `json:"max_bots"`
		IsFree      bool    `json:"is_free"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	p := &models.Plan{
		Name:        req.Name,
		DurationDay: req.DurationDay,
		Price:       req.Price,
		MaxBots:     req.MaxBots,
		IsFree:      req.IsFree,
		IsActive:    true,
	}
	if err := h.store.CreatePlan(c.Request.Context(), p); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, p)
}

// PATCH /api/v1/admin/plans/:id
func (h *Handler) UpdatePlan(c *gin.Context) {
	ctx := c.Request.Context()
	plan, err := h.store.FindPlan(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if plan == nil {
		fail(c, http.StatusNotFound, "plan not found")
		return
	}

	var req struct {
		Name        *string  `json:"name"`
		DurationDay *int     `json:"duration_day"`
		Price       *float64 `json:"price"`
		MaxBots     *int     `json:"max_bots"`
		IsFree      *bool    `json:"is_free"`
		IsActive    *bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.DurationDay != nil {
		plan.DurationDay = *req.DurationDay
	}
	if req.Price != nil {
		plan.Price = *req.Price
	}
	if req.MaxBots != nil {
		plan.MaxBots = *req.MaxBots
	}
	if req.IsFree != nil {
		plan.IsFree = *req.IsFree
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}

	if err := h.store.UpdatePlan(ctx, plan); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, plan)
}

// PATCH /api/v1/admin/plans/:id/limits — معادل دکمه‌های ➕➖ محدودیت هر نوع ربات (بخش ۳ سند).
func (h *Handler) UpdatePlanLimit(c *gin.Context) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid plan id")
		return
	}

	var req struct {
		BotType string `json:"bot_type" binding:"required"`
		MaxBots int    `json:"max_bots"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.MaxBots < 0 {
		req.MaxBots = 0
	}

	if err := h.store.SetPlanLimit(c.Request.Context(), planID, req.BotType, req.MaxBots); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"plan_id": planID, "bot_type": req.BotType, "max_bots": req.MaxBots})
}

// DELETE /api/v1/admin/plans/:id
func (h *Handler) DeletePlan(c *gin.Context) {
	if err := h.store.DeletePlan(c.Request.Context(), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// ── Admin — Stats ─────────────────────────────────────────

func (h *Handler) AdminStats(c *gin.Context) {
	ctx := c.Request.Context()
	instances, _ := h.store.ListAllInstances(ctx)
	users, _ := h.store.ListUsers(ctx)
	servers, _ := h.store.ListServers(ctx)

	running, stopped, pending, errored := 0, 0, 0, 0
	for _, inst := range instances {
		switch inst.Status {
		case models.StatusRunning:
			running++
		case models.StatusStopped:
			stopped++
		case models.StatusPending:
			pending++
		case models.StatusError:
			errored++
		}
	}

	onlineSrv := 0
	for _, s := range servers {
		if s.IsOnline {
			onlineSrv++
		}
	}

	ok(c, gin.H{
		"instances": gin.H{
			"total": len(instances), "running": running,
			"stopped": stopped, "pending": pending, "error": errored,
		},
		"servers": gin.H{"total": len(servers), "online": onlineSrv},
		"users":   len(users),
	})
}

func validTelegramAuthTime(now time.Time, authDate int64) bool {
	if authDate == 0 {
		return false
	}
	authTime := time.Unix(authDate, 0)
	return now.Sub(authTime) <= maxAuthAge && !authTime.After(now.Add(5*time.Minute))
}

// verifyTelegramAuth دقیقاً طبق مستنداتِ Telegram Login Widget عمل می‌کند:
// data-check-string = تمامِ فیلدهای دریافتی (به‌جز hash) که خالی نیستند،
// به‌صورت "key=value" مرتب‌شده‌ی الفبایی و جداشده با "\n"؛
// secret_key = SHA256(bot_token)؛ امضا = HMAC-SHA256(secret_key, data-check-string).
func verifyTelegramAuth(fields map[string]string, hash, botToken string) bool {
	if hash == "" {
		return false
	}
	keys := make([]string, 0, len(fields))
	for k, v := range fields {
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, k+"="+fields[k])
	}
	dataCheckString := strings.Join(pairs, "\n")

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(dataCheckString))
	expected := hex.EncodeToString(mac.Sum(nil))

	return subtle.ConstantTimeCompare([]byte(expected), []byte(hash)) == 1
}

// ── Payments ──────────────────────────────────────────────

// GET /api/v1/payments — تاریخچه‌ی پرداخت‌های خودِ کاربر لاگین‌شده.
func (h *Handler) ListMyPayments(c *gin.Context) {
	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	payments, err := h.store.ListPaymentsByUser(ctx, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, payments)
}

// GET /api/v1/admin/payments — همه‌ی پرداخت‌های پلتفرم (فقط ادمین)، با فیلتر
// اختیاری status. فعلاً فقط نمایشی است؛ تغییر وضعیت دستی (مثلاً تأیید کارت‌به‌کارت)
// به‌عمد اضافه نشده چون مشخص نیست چه سرویسی الان واقعاً از این مدل برای نوشتن
// استفاده می‌کند — نباید بدون تأیید جریان نوشتن، از این‌جا state را تغییر داد.
func (h *Handler) ListAllPaymentsAdmin(c *gin.Context) {
	ctx := c.Request.Context()
	payments, err := h.store.ListAllPayments(ctx)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}

	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status == "" {
		ok(c, payments)
		return
	}

	filtered := make([]models.Payment, 0, len(payments))
	for _, p := range payments {
		if strings.ToLower(string(p.Status)) == status {
			filtered = append(filtered, p)
		}
	}
	ok(c, filtered)
}

// ── Promo codes ───────────────────────────────────────────

// GET /api/v1/admin/promo-codes
func (h *Handler) ListPromoCodesAdmin(c *gin.Context) {
	codes, err := h.store.ListPromoCodes(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, codes)
}

// POST /api/v1/admin/promo-codes
func (h *Handler) CreatePromoCode(c *gin.Context) {
	var req struct {
		Code      string  `json:"code" binding:"required"`
		AmountTON float64 `json:"amount_ton" binding:"required"`
		MaxUses   int     `json:"max_uses"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	var createdBy int64
	if actorID, err := uuid.Parse(c.GetString("user_id")); err == nil {
		if actor, _ := h.store.FindUserByID(c.Request.Context(), actorID); actor != nil {
			createdBy = actor.TelegramID
		}
	}

	p := &models.PromoCode{
		Code:      strings.TrimSpace(req.Code),
		AmountTON: req.AmountTON,
		MaxUses:   req.MaxUses,
		IsActive:  true,
		CreatedBy: createdBy,
	}
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, *req.ExpiresAt); err == nil {
			p.ExpiresAt = &t
		}
	}

	if err := h.store.CreatePromoCode(c.Request.Context(), p); err != nil {
		fail(c, http.StatusInternalServerError, "db error: "+err.Error())
		return
	}
	h.auditAdminAction(c, models.AuditAdminAction, p.ID.String(), "promo_code", "created code "+p.Code)
	ok(c, p)
}

// PATCH /api/v1/admin/promo-codes/:id — فعلاً فقط toggle فعال/غیرفعال (همان
// چیزی که SetPromoCodeActive پشتیبانی می‌کند؛ فیلدهای دیگر تغییرناپذیرند تا با
// منطقِ RedeemPromoCode که روی همین رکورد قفل می‌گیرد تداخل نکند).
func (h *Handler) SetPromoCodeActive(c *gin.Context) {
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SetPromoCodeActive(c.Request.Context(), c.Param("id"), req.IsActive); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	h.auditAdminAction(c, models.AuditAdminAction, c.Param("id"), "promo_code", fmt.Sprintf("is_active -> %v", req.IsActive))
	ok(c, gin.H{"success": true})
}

// DELETE /api/v1/admin/promo-codes/:id
func (h *Handler) DeletePromoCode(c *gin.Context) {
	if err := h.store.DeletePromoCode(c.Request.Context(), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	h.auditAdminAction(c, models.AuditAdminAction, c.Param("id"), "promo_code", "deleted")
	ok(c, gin.H{"success": true})
}

// ── NATS publish helpers ──────────────────────────────────

func (h *Handler) publishDeploy(ctx context.Context, inst *models.BotInstance, tmpl *models.BotTemplate, server *models.Server, plainToken string) error {
	owner, _ := h.store.FindUserByID(ctx, inst.OwnerID)
	ownerTelegram := int64(0)
	if owner != nil {
		ownerTelegram = owner.TelegramID
	}
	planID := ""
	if inst.PlanID != nil {
		planID = inst.PlanID.String()
	}
	jwtToken := ""
	if owner != nil {
		jwtToken, _ = auth.GenerateAccessToken(owner.ID.String(), string(owner.Role), auth.JWTConfig{AccessSecret: h.accessSecret, AccessTTL: 15 * time.Minute})
	}
	licenseToken := ""
	if h.license != nil {
		issued, err := h.license.Issue(ctx, inst.BotID, "bot_"+strconv.FormatInt(inst.BotID, 10), inst.OwnerID.String(), server.ID.String(), planID)
		if err != nil {
			h.log.Warn("license issue failed", ports.F("err", err), ports.F("instance", inst.ID))
		} else {
			licenseToken = issued
		}
	}
	serviceName := strings.TrimSpace(tmpl.Name)
	if serviceName == "" {
		serviceName = tmpl.Type
	}
	envVars := map[string]string{
		"BOT_TOKEN": plainToken, "INSTANCE_ID": fmt.Sprintf("bot_%d", inst.BotID),
		"OWNER_TELEGRAM": strconv.FormatInt(ownerTelegram, 10),
		"OWNER_ID":       strconv.FormatInt(ownerTelegram, 10), "PLAN_ID": planID,
		"JWT_TOKEN": jwtToken, "LICENSE_TOKEN": licenseToken, "SERVER_ID": server.ID.String(),
		"APP_ENV": "production", "BOT_SERVICE_NAME": serviceName,
	}
	// تنظیمات کاربرمحوری که کاربر از طریق UpdateInstanceSettings ذخیره کرده — این تنها جایی
	// است که EnvOverrides واقعاً روی کانتینر اعمال می‌شود، پس فقط با start/restart جدید اثر
	// می‌کند (بازخورد کاربر ۲۰۲۶-۰۷-۰۳ درباره‌ی تنظیمات قالب‌ها).
	if inst.EnvOverrides != "" {
		var overrides map[string]string
		if err := json.Unmarshal([]byte(inst.EnvOverrides), &overrides); err == nil {
			for k, v := range overrides {
				envVars[k] = v
			}
		}
	}

	cmd := protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ServerID:      server.ID.String(),
		ContainerName: inst.ContainerName,
		ImageName:     tmpl.ImageName,
		ImageTag:      tmpl.ImageTag,
		EnvVars:       envVars,
	}

	pubCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return h.docker.Send(pubCtx, server.ID.String(), cmd)
}

func (h *Handler) publishCommand(ctx context.Context, inst *models.BotInstance, msgType protocol.MsgType) error {
	server, err := h.store.FindServerByID(ctx, inst.ServerID.String())
	if err != nil || server == nil {
		return fmt.Errorf("server not found")
	}

	cmd := protocol.DeployCommand{
		Type:          msgType,
		ServerID:      server.ID.String(),
		ContainerName: inst.ContainerName,
		ContainerID:   firstNonEmpty(inst.ContainerID, inst.ContainerName),
	}

	pubCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return h.docker.Send(pubCtx, server.ID.String(), cmd)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
