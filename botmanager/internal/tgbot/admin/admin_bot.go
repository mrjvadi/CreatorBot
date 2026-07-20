package admin

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Admin) AdminBotsList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	instances, _ := h.Store.ListAllInstances(ctx)

	if len(instances) == 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyBotsEmpty), tele.ModeHTML, h.KbAdmin(ctx, uid))
	}

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

	lines := []string{
		h.T(ctx, uid, i18n.KeyBotsTitle, len(instances)),
		"",
		h.T(ctx, uid, i18n.KeyAdminBotSummary, running, stopped, pending, errored),
		"",
	}
	// دکمه‌ی عملیاتِ متناسب با وضعیت هر بات — قبلاً این لیست فقط متن بود و
	// AdminBotStop/Start/Delete هیچ‌جا صدا زده نمی‌شدند (کد مرده).
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, inst := range instances {
		lines = append(lines, format.FmtInstance(inst, true))
		id := inst.ID.String()
		switch inst.Status {
		case models.StatusRunning:
			rows = append(rows, kb.Row(
				kb.Data("⏹ "+inst.ContainerName, "admin_bot_stop:"+id),
				kb.Data("🔄", "admin_bot_migrate:"+id),
				kb.Data("🗑", "admin_bot_del:"+id),
			))
		case models.StatusStopped, models.StatusError:
			rows = append(rows, kb.Row(
				kb.Data("▶️ "+inst.ContainerName, "admin_bot_start:"+id),
				kb.Data("🔄", "admin_bot_migrate:"+id),
				kb.Data("🗑", "admin_bot_del:"+id),
			))
		default:
			rows = append(rows, kb.Row(kb.Data("🔄 "+inst.ContainerName, "admin_bot_migrate:"+id), kb.Data("🗑", "admin_bot_del:"+id)))
		}
	}
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBack), "back_main")))
	kb.Inline(rows...)

	return c.Send(format.JoinLines(lines), tele.ModeHTML, kb)
}

func (h *Admin) AdminBotStop(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, err := h.Store.FindInstance(ctx, instID)
	if err != nil || inst == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotNotFound), ShowAlert: true})
	}
	if dErr := h.Docker.Stop(ctx, inst.ServerID.String(), inst.ContainerID); dErr != nil {
		h.Log.Error("adminBotStop: docker stop failed", h.F("err", dErr), h.F("instance", instID))
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotActionFailed), ShowAlert: true})
	}
	if sErr := h.Store.UpdateInstanceStatus(ctx, instID, models.StatusStopped); sErr != nil {
		// خودِ Docker موفق بود، فقط رکورد وضعیت آپدیت نشد — این را جدا لاگ
		// می‌کنیم چون علتش با شکستِ Docker فرق دارد (ناسازگاریِ state تا
		// heartbeatِ بعدی که وضعیت را دوباره sync می‌کند).
		h.Log.Error("adminBotStop: docker stopped but status update failed",
			h.F("err", sErr), h.F("instance", instID))
	}
	_ = c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotStopped, inst.ContainerName)})
	return h.AdminBotsList(ctx, c)
}

func (h *Admin) AdminBotStart(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, err := h.Store.FindInstance(ctx, instID)
	if err != nil || inst == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotNotFound), ShowAlert: true})
	}
	if dErr := h.Docker.Start(ctx, inst.ServerID.String(), inst.ContainerID); dErr != nil {
		h.Log.Error("adminBotStart: docker start failed", h.F("err", dErr), h.F("instance", instID))
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotActionFailed), ShowAlert: true})
	}
	if sErr := h.Store.UpdateInstanceStatus(ctx, instID, models.StatusPending); sErr != nil {
		h.Log.Error("adminBotStart: docker started but status update failed",
			h.F("err", sErr), h.F("instance", instID))
	}
	_ = c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotStarted, inst.ContainerName)})
	return h.AdminBotsList(ctx, c)
}

// AdminBotDeleteConfirm تأییدِ حذف را قبل از عملیاتِ غیرقابل‌بازگشت نشان می‌دهد.
func (h *Admin) AdminBotDeleteConfirm(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, err := h.Store.FindInstance(ctx, instID)
	if err != nil || inst == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotNotFound), ShowAlert: true})
	}
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnConfirmDelete), "admin_bot_del_do:"+instID),
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel"),
		),
	)
	_ = c.Respond()
	return c.Send(h.T(ctx, uid, i18n.KeyDeleteConfirm, inst.ContainerName), tele.ModeHTML, kb)
}

func (h *Admin) AdminBotDelete(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, err := h.Store.FindInstance(ctx, instID)
	if err != nil || inst == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotNotFound), ShowAlert: true})
	}
	name := strings.TrimSpace(inst.ContainerName)
	if dErr := h.Docker.Remove(ctx, inst.ServerID.String(), inst.ContainerID); dErr != nil {
		// همچنان ادامه می‌دهیم و رکورد را حذف می‌کنیم — کانتینر ممکن است از
		// قبل روی سرور نبوده باشد (مثلاً سرور آفلاین/از قبل پاک شده)؛ ولی حتماً
		// لاگ می‌شود تا اگر واقعاً سرور در دسترس بوده و پاک نشده، قابل پیگیری باشد.
		h.Log.Error("adminBotDelete: docker remove failed (continuing to delete DB record)",
			h.F("err", dErr), h.F("instance", instID))
	}
	if err := h.Store.DeleteInstance(ctx, instID); err != nil {
		h.Log.Error("adminBotDelete: db delete failed", h.F("err", err), h.F("instance", instID))
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyBotActionFailed), ShowAlert: true})
	}
	return c.Edit(h.T(ctx, uid, i18n.KeyBotDeleted, name), tele.ModeHTML)
}
