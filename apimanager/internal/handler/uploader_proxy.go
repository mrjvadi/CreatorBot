package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/shared-core/models"
)

// ── Uploader content proxy (NATS) ───────────────────────────
//
// بازخورد کاربر ۲۰۲۶-۰۷-۰۵ (رجوع apimanager/NEEDS.md، بخش «از uploader-bot»): مالکِ ربات
// باید بتواند کدها/پوشه‌ها را بدون بازکردن خودِ تلگرام مدیریت کند. uploader-bot حالا
// روی uploader.codes.*.<botID> و uploader.folders.*.<botID> پاسخ می‌دهد (رجوع
// uploader-bot/internal/tgbot/nats_query.go) — این‌جا فقط یک لایه‌ی نازک HTTP↔NATS روی
// آن‌هاست، دقیقاً همان الگویی که NEEDS.md پیشنهاد داده بود («جمع‌بندی پیشنهادی»).
// فقط Code/Folder در این دور — Backup/ForceJoinChannel در آینده.

// resolveOwnedUploaderInstance مالکیت (یا نقش admin/owner) و نوعِ instance را چک می‌کند —
// مشترک بین همه‌ی endpoint های زیر.
func (h *Handler) resolveOwnedUploaderInstance(c *gin.Context) (*models.BotInstance, bool) {
	ctx := c.Request.Context()
	uid, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid user")
		return nil, false
	}
	inst, err := h.store.FindInstance(ctx, c.Param("id"))
	if err != nil {
		fail(c, http.StatusInternalServerError, "db error")
		return nil, false
	}
	if inst == nil {
		fail(c, http.StatusNotFound, "instance not found")
		return nil, false
	}
	role := c.GetString("role")
	isAdmin := role == "admin" || role == "owner"
	if inst.OwnerID != uid && !isAdmin {
		fail(c, http.StatusForbidden, "not your instance")
		return nil, false
	}
	tmpl, err := h.store.FindTemplate(ctx, inst.TemplateID.String())
	if err != nil || tmpl == nil {
		fail(c, http.StatusInternalServerError, "template not found")
		return nil, false
	}
	if tmpl.Type != "uploader" {
		fail(c, http.StatusBadRequest, "this endpoint is only available for uploader-type bots")
		return nil, false
	}
	return inst, true
}

// natsRequestJSON یک NATS request/reply انجام می‌دهد و پاسخِ خطای Client.Respond (که
// {"error":"..."} برمی‌گرداند، رجوع shared/pkg/adapters/nats.Client.Respond) را قبل از
// parse کردنِ نتیجه‌ی موفق چک می‌کند.
func (h *Handler) natsRequestJSON(ctx context.Context, subject string, payload any, out any) error {
	var raw json.RawMessage
	if err := h.nc.Request(ctx, subject, payload, &raw, 5*time.Second); err != nil {
		return err
	}
	var errCheck struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &errCheck) == nil && errCheck.Error != "" {
		return fmt.Errorf("%s", errCheck.Error)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// GET /api/v1/instances/:id/uploader/codes?folder_id=&page=&limit=
func (h *Handler) ListUploaderCodes(c *gin.Context) {
	inst, okInst := h.resolveOwnedUploaderInstance(c)
	if !okInst {
		return
	}
	payload := gin.H{"folder_id": c.Query("folder_id")}
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			payload["page"] = n
		}
	}
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			payload["limit"] = n
		}
	}
	var result any
	subject := fmt.Sprintf("uploader.codes.list.%d", inst.BotID)
	if err := h.natsRequestJSON(c.Request.Context(), subject, payload, &result); err != nil {
		fail(c, http.StatusBadGateway, "uploader-bot request failed: "+err.Error())
		return
	}
	ok(c, result)
}

// DELETE /api/v1/instances/:id/uploader/codes/:codeId
func (h *Handler) DeleteUploaderCode(c *gin.Context) {
	inst, okInst := h.resolveOwnedUploaderInstance(c)
	if !okInst {
		return
	}
	var result any
	subject := fmt.Sprintf("uploader.codes.delete.%d", inst.BotID)
	if err := h.natsRequestJSON(c.Request.Context(), subject, gin.H{"id": c.Param("codeId")}, &result); err != nil {
		fail(c, http.StatusBadGateway, "uploader-bot request failed: "+err.Error())
		return
	}
	ok(c, result)
}

// GET /api/v1/instances/:id/uploader/folders?parent_id=
func (h *Handler) ListUploaderFolders(c *gin.Context) {
	inst, okInst := h.resolveOwnedUploaderInstance(c)
	if !okInst {
		return
	}
	var result any
	subject := fmt.Sprintf("uploader.folders.list.%d", inst.BotID)
	if err := h.natsRequestJSON(c.Request.Context(), subject, gin.H{"parent_id": c.Query("parent_id")}, &result); err != nil {
		fail(c, http.StatusBadGateway, "uploader-bot request failed: "+err.Error())
		return
	}
	ok(c, result)
}

// POST /api/v1/instances/:id/uploader/folders
func (h *Handler) CreateUploaderFolder(c *gin.Context) {
	inst, okInst := h.resolveOwnedUploaderInstance(c)
	if !okInst {
		return
	}
	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID string `json:"parent_id"`
		Icon     string `json:"icon"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var result any
	subject := fmt.Sprintf("uploader.folders.create.%d", inst.BotID)
	if err := h.natsRequestJSON(c.Request.Context(), subject, req, &result); err != nil {
		fail(c, http.StatusBadGateway, "uploader-bot request failed: "+err.Error())
		return
	}
	ok(c, result)
}

// DELETE /api/v1/instances/:id/uploader/folders/:folderId
func (h *Handler) DeleteUploaderFolder(c *gin.Context) {
	inst, okInst := h.resolveOwnedUploaderInstance(c)
	if !okInst {
		return
	}
	var result any
	subject := fmt.Sprintf("uploader.folders.delete.%d", inst.BotID)
	if err := h.natsRequestJSON(c.Request.Context(), subject, gin.H{"id": c.Param("folderId")}, &result); err != nil {
		fail(c, http.StatusBadGateway, "uploader-bot request failed: "+err.Error())
		return
	}
	ok(c, result)
}
