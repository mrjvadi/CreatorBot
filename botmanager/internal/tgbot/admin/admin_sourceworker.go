package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

// این فایل پنلِ ادمین برای مدیریتِ source-service workerها را پیاده می‌کند —
// سمتِ ذخیره‌سازیِ قراردادِ source.worker.* که پاسخ‌دهیِ NATS آن در
// internal/sourceworker (پکیجِ جدا، چون Telegram Context لازم ندارد) است.
// اینجا فقط CRUD روی shared-core/models.SourceWorkerConfig است.

// AdminSourceWorkersList لیستِ workerها را نشان می‌دهد.
func (h *Admin) AdminSourceWorkersList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	list, _ := h.Store.ListSourceWorkerConfigs(ctx)

	lines := []string{h.T(ctx, uid, i18n.KeySWTitle), ""}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	if len(list) == 0 {
		lines = append(lines, h.T(ctx, uid, i18n.KeySWEmpty))
	} else {
		for _, sw := range list {
			lines = append(lines, format.FmtSourceWorker(sw))
			rows = append(rows, kb.Row(
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnToggleSW), "admin_sw_toggle:"+sw.ID.String()),
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDeleteSW), "admin_sw_del:"+sw.ID.String()),
			))
		}
	}
	lines = append(lines, "")
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnAddSourceWorker), "admin_sw_add")))
	kb.Inline(rows...)
	return c.Send(format.JoinLines(lines), tele.ModeHTML, kb)
}

// AdminSourceWorkerStart ویزاردِ چندمرحله‌ای ساختِ worker را شروع می‌کند.
func (h *Admin) AdminSourceWorkerStart(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	h.SetStep(ctx, uid, state.StepSWAppID)
	return c.Edit(h.T(ctx, uid, i18n.KeySWAskAppID), tele.ModeHTML, h.KbBackCancel(ctx, uid))
}

// AdminSourceWorkerAdd آخرین مرحله — رکورد را می‌سازد و LicenseKey/WorkerID/
// SessionKey تازه‌تولیدشده را (فقط همین یک‌بار) کامل به ادمین نشان می‌دهد.
func (h *Admin) AdminSourceWorkerAdd(ctx context.Context, c tele.Context, appIDStr, appHash, phone, label string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	appID, err := strconv.Atoi(strings.TrimSpace(appIDStr))
	if err != nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyError))
	}
	appHash = strings.TrimSpace(appHash)
	phone = strings.TrimSpace(phone)
	label = strings.TrimSpace(label)
	if label == "-" {
		label = ""
	}
	if appHash == "" || phone == "" {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyError))
	}

	sessionKeyHex, err := randomHex(32)
	if err != nil {
		h.Log.Error("adminSourceWorkerAdd: session key gen", h.F("err", err))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeySWCreateError))
	}

	encAppHash, err1 := auth.Encrypt(appHash, h.EncryptKey)
	encSessionKey, err2 := auth.Encrypt(sessionKeyHex, h.EncryptKey)
	if err1 != nil || err2 != nil {
		h.Log.Error("adminSourceWorkerAdd: encrypt", h.F("err1", err1), h.F("err2", err2))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeySWCreateError))
	}

	licenseKey := uuid.New().String()
	workerID := "sw_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	cfg := &models.SourceWorkerConfig{
		Label:      label,
		LicenseKey: licenseKey,
		WorkerID:   workerID,
		AppID:      appID,
		AppHash:    encAppHash,
		Phone:      phone,
		SessionKey: encSessionKey,
		IsActive:   true,
	}
	if err := h.Store.CreateSourceWorkerConfig(ctx, cfg); err != nil {
		h.Log.Error("adminSourceWorkerAdd: create", h.F("err", err))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeySWCreateError))
	}

	displayLabel := label
	if displayLabel == "" {
		displayLabel = "-"
	}
	return c.Send(
		h.T(ctx, uid, i18n.KeySWCreated, displayLabel, workerID, licenseKey),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}

// AdminSourceWorkerDeleteConfirm تأییدِ حذف را نشان می‌دهد.
func (h *Admin) AdminSourceWorkerDeleteConfirm(ctx context.Context, c tele.Context, id string) error {
	uid := c.Sender().ID
	cfg, err := h.Store.FindSourceWorkerConfig(ctx, id)
	if err != nil || cfg == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeySWNotFound))
	}
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnConfirmDelete), "admin_sw_del_do:"+id),
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel"),
		),
	)
	return c.Edit(h.T(ctx, uid, i18n.KeySWDeleteConfirm), tele.ModeHTML, kb)
}

// AdminSourceWorkerDelete رکورد را حذف می‌کند.
func (h *Admin) AdminSourceWorkerDelete(ctx context.Context, c tele.Context, id string) error {
	uid := c.Sender().ID
	if err := h.Store.DeleteSourceWorkerConfig(ctx, id); err != nil {
		h.Log.Error("adminSourceWorkerDelete", h.F("err", err))
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}
	return c.Edit(h.T(ctx, uid, i18n.KeySWDeleted))
}

// AdminSourceWorkerToggle فعال/غیرفعال می‌کند.
func (h *Admin) AdminSourceWorkerToggle(ctx context.Context, c tele.Context, id string) error {
	uid := c.Sender().ID
	cfg, err := h.Store.FindSourceWorkerConfig(ctx, id)
	if err != nil || cfg == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeySWNotFound), ShowAlert: true})
	}
	newActive := !cfg.IsActive
	if err := h.Store.SetSourceWorkerConfigActive(ctx, id, newActive); err != nil {
		h.Log.Error("adminSourceWorkerToggle", h.F("err", err))
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyError), ShowAlert: true})
	}
	key := i18n.KeySWToggledOff
	if newActive {
		key = i18n.KeySWToggledOn
	}
	return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, key), ShowAlert: true})
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
