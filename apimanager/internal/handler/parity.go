package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// RedeemPromoCode همتای HTTP مسیر redeem در botmanager است.
func (h *Handler) RedeemPromoCode(c *gin.Context) {
	if h.payClient == nil {
		fail(c, http.StatusServiceUnavailable, "pay client not configured")
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	u, err := h.currentUser(c)
	if err != nil || u == nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	promo, err := h.store.FindPromoCodeByCode(c.Request.Context(), code)
	if err != nil {
		fail(c, http.StatusInternalServerError, "promo lookup failed")
		return
	}
	if promo == nil {
		fail(c, http.StatusNotFound, "promo code not found")
		return
	}
	if err := h.store.RedeemPromoCode(c.Request.Context(), promo.ID, u.ID); err != nil {
		switch {
		case errors.Is(err, store.ErrPromoAlreadyRedeemed):
			fail(c, http.StatusConflict, "promo code already redeemed")
		case errors.Is(err, store.ErrPromoNotRedeemable):
			fail(c, http.StatusGone, "promo code expired or exhausted")
		default:
			fail(c, http.StatusInternalServerError, "promo redeem failed")
		}
		return
	}
	ref := "promo:" + code
	if err := h.payClient.Credit(c.Request.Context(), u.TelegramID, promo.AmountTON, ref, fmt.Sprintf(`{"promo_id":%q}`, promo.ID)); err != nil {
		h.log.Error("promo credit failed after claim", ports.F("err", err), ports.F("user", u.ID), ports.F("promo", promo.ID))
		fail(c, http.StatusBadGateway, "promo claimed but wallet credit needs reconciliation")
		return
	}
	h.createAudit(c, u.ID, string(u.Role), models.AuditAdminAction, promo.ID.String(), "promo", "redeem:"+code)
	ok(c, gin.H{"amount_ton": promo.AmountTON, "code": code})
}

// RenewInstance همتای HTTP تمدید سرویس در botmanager است.
func (h *Handler) RenewInstance(c *gin.Context) {
	ctx := c.Request.Context()
	u, err := h.currentUser(c)
	if err != nil || u == nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil || inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return
	}
	if inst.OwnerID != u.ID {
		fail(c, http.StatusForbidden, "instance access denied")
		return
	}
	if inst.PlanID == nil {
		fail(c, http.StatusConflict, "instance has no renewable plan")
		return
	}
	plan, err := h.store.FindPlan(ctx, inst.PlanID.String())
	if err != nil || plan == nil {
		fail(c, http.StatusNotFound, "plan not found")
		return
	}
	attemptID := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if attemptID == "" {
		attemptID = uuid.NewString()
	}
	charged := false
	if plan.Price > 0 {
		if h.payClient == nil {
			fail(c, http.StatusServiceUnavailable, "pay client not configured")
			return
		}
		if _, err := h.payClient.DeductForService(ctx, u.TelegramID, plan.Price, "renew:"+plan.ID.String(), attemptID); err != nil {
			if natspayclient.IsInsufficientBalance(err) {
				fail(c, http.StatusPaymentRequired, "insufficient balance")
			} else {
				fail(c, http.StatusBadGateway, "payment failed")
			}
			return
		}
		charged = true
	}
	base := time.Now()
	if inst.ExpiresAt != nil && inst.ExpiresAt.After(base) {
		base = *inst.ExpiresAt
	}
	var expiresAt *time.Time
	if plan.DurationDay > 0 {
		t := base.AddDate(0, 0, plan.DurationDay)
		expiresAt = &t
	}
	if err := h.store.UpdateInstanceExpiry(ctx, inst.ID, expiresAt); err != nil {
		if charged {
			_ = h.payClient.RefundService(ctx, u.TelegramID, plan.Price, "renew:"+inst.ID.String()+":"+attemptID+":expiry_failed")
		}
		fail(c, http.StatusInternalServerError, "renewal update failed")
		return
	}
	if charged {
		now := time.Now()
		_ = h.store.CreatePayment(ctx, &models.Payment{UserID: u.ID, PlanID: &plan.ID, InstanceID: &inst.ID, Amount: plan.Price, Currency: "TON", Status: models.PaymentDone, ConfirmedAt: &now, InvoiceID: "renew:" + attemptID})
	}
	startQueued := true
	if inst.Status != models.StatusRunning {
		if err := h.publishCommand(ctx, inst, protocol.MsgStart); err != nil {
			startQueued = false
			h.log.Error("renewed but start command failed", ports.F("err", err), ports.F("instance", inst.ID))
		} else {
			_ = h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusPending)
		}
	}
	h.createAudit(c, u.ID, string(u.Role), models.AuditBuyPlan, inst.ID.String(), "instance", "renew")
	ok(c, gin.H{"expires_at": expiresAt, "start_queued": startQueued, "attempt_id": attemptID})
}

func (h *Handler) ListMyAuditLogs(c *gin.Context) {
	u, err := h.currentUser(c)
	if err != nil || u == nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return
	}
	logs, err := h.store.ListAuditLogs(c.Request.Context(), u.ID.String(), boundedLimit(c, 100))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, logs)
}

func (h *Handler) ListAdminAuditLogs(c *gin.Context) {
	logs, err := h.store.ListAdminAuditLogs(c.Request.Context(), c.Query("action"), c.Query("target_type"), boundedLimit(c, 200))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	ok(c, logs)
}

func boundedLimit(c *gin.Context, fallback int) int {
	n, err := strconv.Atoi(c.Query("limit"))
	if err != nil || n <= 0 {
		return fallback
	}
	if n > 500 {
		return 500
	}
	return n
}

func (h *Handler) ListSourceWorkers(c *gin.Context) {
	list, err := h.store.ListSourceWorkerConfigs(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	result := make([]gin.H, 0, len(list))
	for i := range list {
		result = append(result, sourceWorkerView(&list[i]))
	}
	ok(c, result)
}

func sourceWorkerView(sw *models.SourceWorkerConfig) gin.H {
	return gin.H{
		"id": sw.ID, "label": sw.Label, "worker_id": sw.WorkerID,
		"app_id": sw.AppID, "phone": sw.Phone, "is_active": sw.IsActive,
		"last_heartbeat_at": sw.LastHeartbeatAt, "last_status": sw.LastStatus,
		"is_online": sw.IsOnline(90 * time.Second),
	}
}

func (h *Handler) CreateSourceWorker(c *gin.Context) {
	var req struct {
		Label   string `json:"label"`
		AppID   int    `json:"app_id" binding:"required"`
		AppHash string `json:"app_hash" binding:"required"`
		Phone   string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AppID <= 0 {
		fail(c, http.StatusBadRequest, "valid app_id, app_hash and phone are required")
		return
	}
	sessionKey, err := secureRandomHex(32)
	if err != nil {
		fail(c, http.StatusInternalServerError, "credential generation failed")
		return
	}
	appHash, err1 := auth.Encrypt(strings.TrimSpace(req.AppHash), h.encryptKey)
	encSession, err2 := auth.Encrypt(sessionKey, h.encryptKey)
	if err1 != nil || err2 != nil {
		fail(c, http.StatusInternalServerError, "credential encryption failed")
		return
	}
	sw := &models.SourceWorkerConfig{
		Label: strings.TrimSpace(req.Label), LicenseKey: uuid.NewString(),
		WorkerID: "sw_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12],
		AppID:    req.AppID, AppHash: appHash, Phone: strings.TrimSpace(req.Phone),
		SessionKey: encSession, IsActive: true,
	}
	if err := h.store.CreateSourceWorkerConfig(c.Request.Context(), sw); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	h.auditAdminAction(c, models.AuditAdminAction, sw.ID.String(), "source_worker", "created")
	view := sourceWorkerView(sw)
	view["license_key"] = sw.LicenseKey // فقط پاسخ ساخت؛ در list هرگز secret برنمی‌گردد.
	ok(c, view)
}

func secureRandomHex(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *Handler) SetSourceWorkerActive(c *gin.Context) {
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SetSourceWorkerConfigActive(c.Request.Context(), c.Param("id"), req.IsActive); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	h.auditAdminAction(c, models.AuditAdminAction, c.Param("id"), "source_worker", fmt.Sprintf("is_active -> %v", req.IsActive))
	ok(c, gin.H{"is_active": req.IsActive})
}

func (h *Handler) DeleteSourceWorker(c *gin.Context) {
	if err := h.store.DeleteSourceWorkerConfig(c.Request.Context(), c.Param("id")); err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return
	}
	h.auditAdminAction(c, models.AuditAdminAction, c.Param("id"), "source_worker", "deleted")
	ok(c, nil)
}

// BroadcastText قابلیت broadcast متنی botmanager را برای وب ارائه می‌کند.
func (h *Handler) BroadcastText(c *gin.Context) {
	if h.telegramBotTok == "" {
		fail(c, http.StatusServiceUnavailable, "telegram bot not configured")
		return
	}
	var req struct {
		Message string `json:"message" binding:"required"`
		Filter  string `json:"filter"`
		PlanID  string `json:"plan_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		fail(c, http.StatusBadRequest, "message is required")
		return
	}
	if len([]rune(req.Message)) > 4096 {
		fail(c, http.StatusBadRequest, "message is too long")
		return
	}
	users, err := h.broadcastAudience(c.Request.Context(), req.Filter, req.PlanID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "audience lookup failed")
		return
	}
	message := req.Message
	go h.runHTTPBroadcast(users, message)
	h.auditAdminAction(c, models.AuditAdminAction, req.PlanID, "broadcast", fmt.Sprintf("queued %d recipients filter=%s", len(users), req.Filter))
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "data": gin.H{"queued": len(users)}})
}

func (h *Handler) broadcastAudience(ctx context.Context, filter, planID string) ([]models.User, error) {
	switch filter {
	case "no_plan":
		return h.store.ListUsersWithoutActivePlan(ctx)
	case "plan":
		if strings.TrimSpace(planID) == "" {
			return nil, fmt.Errorf("plan_id is required")
		}
		return h.store.ListUsersByActivePlan(ctx, planID)
	default:
		return h.store.ListUsers(ctx)
	}
}

func (h *Handler) runHTTPBroadcast(users []models.User, message string) {
	ticker := time.NewTicker(time.Second / 25)
	defer ticker.Stop()
	sent, failed := 0, 0
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", h.telegramBotTok)
	client := &http.Client{Timeout: 12 * time.Second}
	for _, user := range users {
		if user.IsBlocked || user.TelegramID == 0 {
			continue
		}
		<-ticker.C
		form := url.Values{"chat_id": {strconv.FormatInt(user.TelegramID, 10)}, "text": {message}}
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode >= 300 {
			failed++
		} else {
			sent++
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}
	h.log.Info("web broadcast finished", ports.F("sent", sent), ports.F("failed", failed))
}

func (h *Handler) createAudit(c *gin.Context, actorID uuid.UUID, role string, action models.AuditAction, targetID, targetType, description string) {
	_ = h.store.CreateAuditLog(c.Request.Context(), &models.AuditLog{
		ActorID: actorID, ActorRole: role, Action: action, TargetID: targetID,
		TargetType: targetType, Description: description, IPAddress: c.ClientIP(),
	})
}
