// Package handler contains all HTTP handlers for apimanager.
package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
)

// Handler holds dependencies for all API endpoints.
type Handler struct {
	store         *store.Store
	docker        *sharedocker.Manager
	nc            *natsclient.Client
	log           ports.Logger
	accessSecret  string
	refreshSecret string
	encryptKey    string
	agentAPIKey   string
}

func New(
	st *store.Store,
	docker *sharedocker.Manager,
	nc *natsclient.Client,
	log ports.Logger,
	accessSecret, refreshSecret, encryptKey string,
	agentAPIKey string,
) *Handler {
	return &Handler{
		store:         st,
		docker:        docker,
		nc:            nc,
		log:           log,
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		encryptKey:    encryptKey,
		agentAPIKey:   agentAPIKey,
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
		TelegramID int64  `json:"telegram_id" binding:"required"`
		FirstName  string `json:"first_name"`
		Username   string `json:"username"`
		Hash       string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	u, err := h.store.FindUserByTelegramID(ctx, req.TelegramID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if u == nil {
		u = &models.User{
			TelegramID: req.TelegramID,
			FirstName:  req.FirstName,
			Username:   req.Username,
			Role:       models.RoleUser,
		}
		if err := h.store.UpsertUser(ctx, u); err != nil {
			fail(c, http.StatusInternalServerError, "user create failed")
			return
		}
	}

	accessToken, err := auth.GenerateAccessToken(u.ID.String(), string(u.Role), auth.JWTConfig{AccessSecret: h.accessSecret, AccessTTL: 15*time.Minute})
	if err != nil {
		fail(c, http.StatusInternalServerError, "token error")
		return
	}
	refreshToken, _ := auth.GenerateRefreshToken(u.ID.String(), string(u.Role), auth.JWTConfig{RefreshSecret: h.refreshSecret, RefreshTTL: 30*24*time.Hour})

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

	accessToken, _ := auth.GenerateJWT(claims.UserID, claims.Role, h.accessSecret, 15*time.Minute)
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

func (h *Handler) ListServers(c *gin.Context) {
	servers, err := h.store.ListServers(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, servers)
}

func (h *Handler) CreateServer(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		IP   string `json:"ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	srv := &models.Server{
		Name:    req.Name,
		IP:      req.IP,
		Channel: fmt.Sprintf("server_%s", strings.ReplaceAll(req.Name, " ", "_")),
	}
	if err := h.store.CreateServer(c.Request.Context(), srv); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, srv)
}

func (h *Handler) DeleteServer(c *gin.Context) {
	if err := h.store.DeleteServer(c.Request.Context(), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// ── Templates (admin) ─────────────────────────────────────

func (h *Handler) ListTemplates(c *gin.Context) {
	templates, err := h.store.ListTemplates(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, templates)
}

func (h *Handler) CreateTemplate(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type" binding:"required"`
		ImageName string `json:"image_name" binding:"required"`
		ImageTag  string `json:"image_tag" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
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
	if err := h.store.CreateTemplate(c.Request.Context(), t); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, t)
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
	server, err := h.store.FindBestOnlineServer(ctx)
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
	}
	if err := h.store.CreateInstance(ctx, inst); err != nil {
		fail(c, http.StatusInternalServerError, "create instance failed")
		return
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
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if inst.Status == models.StatusRunning {
		fail(c, http.StatusConflict, "already running")
		return
	}

	tmpl, _ := h.store.FindTemplate(ctx, inst.TemplateID.String())
	server, _ := h.store.FindServerByID(ctx, inst.ServerID.String())
	if tmpl == nil || server == nil {
		fail(c, http.StatusInternalServerError, "missing template or server")
		return
	}

	// decrypt توکن
	plainToken, err := auth.Decrypt(inst.BotToken, h.encryptKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, "decrypt failed")
		return
	}

	if err := h.publishDeploy(ctx, inst, tmpl, server, plainToken); err != nil {
		fail(c, http.StatusInternalServerError, "start command failed")
		return
	}

	h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)
	ok(c, gin.H{"status": models.StatusPending})
}

// POST /api/v1/instances/:id/stop
func (h *Handler) StopInstance(c *gin.Context) {
	ctx := c.Request.Context()
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
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
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}

	tmpl, _ := h.store.FindTemplate(ctx, inst.TemplateID.String())
	server, _ := h.store.FindServerByID(ctx, inst.ServerID.String())
	if tmpl == nil || server == nil {
		fail(c, http.StatusInternalServerError, "missing template or server")
		return
	}

	plainToken, err := auth.Decrypt(inst.BotToken, h.encryptKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, "decrypt failed")
		return
	}

	// stop → deploy
	h.publishCommand(ctx, inst, protocol.MsgStop)
	time.Sleep(2 * time.Second)
	if err := h.publishDeploy(ctx, inst, tmpl, server, plainToken); err != nil {
		fail(c, http.StatusInternalServerError, "restart failed")
		return
	}

	h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)
	ok(c, gin.H{"status": models.StatusPending})
}

// DELETE /api/v1/instances/:id
func (h *Handler) DeleteInstance(c *gin.Context) {
	ctx := c.Request.Context()
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
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
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
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

// ── NATS publish helpers ──────────────────────────────────

func (h *Handler) publishDeploy(ctx context.Context, inst *models.BotInstance, tmpl *models.BotTemplate, server *models.Server, plainToken string) error {
	cmd := protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ServerID:      server.ID.String(),
		ContainerName: inst.ContainerName,
		ImageName:     tmpl.ImageName,
		ImageTag:      tmpl.ImageTag,
		EnvVars: map[string]string{
			"BOT_TOKEN":   plainToken,
			"INSTANCE_ID": fmt.Sprintf("bot_%d", inst.BotID),
		},
	}

	pubCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return h.nc.Publish(pubCtx, protocol.DeploySubject(server.ID.String()), cmd)
}

func (h *Handler) publishCommand(ctx context.Context, inst *models.BotInstance, msgType protocol.MessageType) error {
	server, err := h.store.FindServerByID(ctx, inst.ServerID.String())
	if err != nil || server == nil {
		return fmt.Errorf("server not found")
	}

	cmd := protocol.DeployCommand{
		Type:          msgType,
		ServerID:      server.ID.String(),
		ContainerName: inst.ContainerName,
	}

	pubCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return h.nc.Publish(pubCtx, protocol.DeploySubject(server.ID.String()), cmd)
}
