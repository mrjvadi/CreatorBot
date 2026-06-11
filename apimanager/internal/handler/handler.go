// Package handler contains all HTTP handlers for apimanager.
// Depends only on shared-core/store, shared-core/docker, and ports interfaces.
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Handler holds dependencies for all API endpoints.
type Handler struct {
	store            *store.Store
	docker           *sharedocker.Manager
	log              ports.Logger
	accessSecret     string
	refreshSecret    string
	encryptKey       string
	agentAPIKey      string // کلید مخفی مشترک بین apimanager و agentmanager
}

func New(
	st *store.Store,
	docker *sharedocker.Manager,
	log ports.Logger,
	accessSecret, refreshSecret, encryptKey string,
	agentAPIKey string,
) *Handler {
	return &Handler{
		store:            st,
		docker:           docker,
		log:              log,
		accessSecret:     accessSecret,
		refreshSecret:    refreshSecret,
		encryptKey:       encryptKey,
		agentAPIKey:      agentAPIKey,
	}
}

// ---- helpers ----

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"ok": false, "message": msg})
}

// ---- Auth ----

// POST /api/v1/auth/telegram
// Body: { telegram_id, first_name, username, hash } (Telegram Login Widget data)
func (h *Handler) TelegramAuth(c *gin.Context) {
	var req struct {
		TelegramID int64  `json:"telegram_id" binding:"required"`
		FirstName  string `json:"first_name"`
		Username   string `json:"username"`
		// TODO: validate Telegram hash before trusting
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.store.FindUserByTelegramID(c.Request.Context(), req.TelegramID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	if user == nil {
		user = &models.User{
			TelegramID: req.TelegramID,
			FirstName:  req.FirstName,
			Username:   req.Username,
		}
		if err := h.store.UpsertUser(c.Request.Context(), user); err != nil {
			fail(c, http.StatusInternalServerError, "create user failed")
			return
		}
	}

	jwtCfg := auth.JWTConfig{
		AccessSecret:  h.accessSecret,
		RefreshSecret: h.refreshSecret,
		AccessTTL:     60 * time.Minute,
		RefreshTTL:    30 * 24 * time.Hour,
	}
	access, _ := auth.GenerateAccessToken(user.ID.String(), string(user.Role), jwtCfg)
	refresh, _ := auth.GenerateRefreshToken(user.ID.String(), string(user.Role), jwtCfg)

	ok(c, gin.H{"access_token": access, "refresh_token": refresh})
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
	jwtCfg := auth.JWTConfig{
		AccessSecret: h.accessSecret,
		AccessTTL:    60 * time.Minute,
	}
	access, _ := auth.GenerateAccessToken(claims.UserID, claims.Role, jwtCfg)
	ok(c, gin.H{"access_token": access})
}

// ---- Me ----

// GET /api/v1/me
func (h *Handler) Me(c *gin.Context) {
	// TODO: fetch user from DB using c.GetString("user_id")
	ok(c, gin.H{"user_id": c.GetString("user_id"), "role": c.GetString("role")})
}

// ---- Servers ----

// GET /api/v1/admin/servers
func (h *Handler) ListServers(c *gin.Context) {
	servers, err := h.store.ListServers(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, servers)
}

// POST /api/v1/admin/servers
func (h *Handler) CreateServer(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		IP   string `json:"ip"   binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	srv := &models.Server{Name: req.Name, IP: req.IP}
	if err := h.store.CreateServer(c.Request.Context(), srv); err != nil {
		fail(c, http.StatusInternalServerError, "create server failed")
		return
	}
	ok(c, srv)
}

// DELETE /api/v1/admin/servers/:id
func (h *Handler) DeleteServer(c *gin.Context) {
	if err := h.store.DeleteServer(c.Request.Context(), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, "delete failed")
		return
	}
	ok(c, nil)
}

// ---- Templates ----

// GET /api/v1/admin/templates
func (h *Handler) ListTemplates(c *gin.Context) {
	templates, err := h.store.ListTemplates(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, templates)
}

// POST /api/v1/admin/templates
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req struct {
		Name        string `json:"name"        binding:"required"`
		Type        string `json:"type"        binding:"required"`
		ImageName   string `json:"image_name"  binding:"required"`
		ImageTag    string `json:"image_tag"   binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	t := &models.BotTemplate{
		Name:        req.Name,
		Type:        req.Type,
		ImageName:   req.ImageName,
		ImageTag:    req.ImageTag,
		Description: req.Description,
	}
	if err := h.store.CreateTemplate(c.Request.Context(), t); err != nil {
		fail(c, http.StatusInternalServerError, "create template failed")
		return
	}
	ok(c, t)
}

// ---- Instances ----

// GET /api/v1/instances
func (h *Handler) ListInstances(c *gin.Context) {
	ownerID := c.GetString("user_id")
	instances, err := h.store.ListInstancesByOwner(c.Request.Context(), ownerID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, instances)
}

// POST /api/v1/instances
func (h *Handler) CreateInstance(c *gin.Context) {
	// TODO: validate plan, deduct balance, deploy via docker.Manager
	ok(c, gin.H{"message": "TODO: deploy instance"})
}

// POST /api/v1/instances/:id/start
func (h *Handler) StartInstance(c *gin.Context) {
	inst, err := h.store.FindInstance(c.Request.Context(), c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if err := h.docker.Start(c.Request.Context(), inst.ServerID.String(), inst.ContainerID); err != nil {
		fail(c, http.StatusInternalServerError, "start failed")
		return
	}
	h.store.UpdateInstanceStatus(c.Request.Context(), inst.ID, models.StatusRunning)
	ok(c, nil)
}

// POST /api/v1/instances/:id/stop
func (h *Handler) StopInstance(c *gin.Context) {
	inst, err := h.store.FindInstance(c.Request.Context(), c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if err := h.docker.Stop(c.Request.Context(), inst.ServerID.String(), inst.ContainerID); err != nil {
		fail(c, http.StatusInternalServerError, "stop failed")
		return
	}
	h.store.UpdateInstanceStatus(c.Request.Context(), inst.ID, models.StatusStopped)
	ok(c, nil)
}

// DELETE /api/v1/instances/:id
func (h *Handler) DeleteInstance(c *gin.Context) {
	inst, err := h.store.FindInstance(c.Request.Context(), c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	h.docker.Remove(c.Request.Context(), inst.ServerID.String(), inst.ContainerID)
	h.store.DeleteInstance(c.Request.Context(), inst.ID)
	ok(c, nil)
}
