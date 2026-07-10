package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ── ریست دانلودها (همه‌ی کاربران) ─────────────────────────────

func (h *Handler) adminResetDownloads(ctx context.Context, c tele.Context) error {
	if err := h.Store.ResetDownloadCounts(ctx); err != nil {
		return c.Send("❌ خطا در ریست دانلودها.", kbAdmin())
	}
	return c.Send("♻️ تعداد دانلود همه‌ی کاربران صفر شد.", kbAdmin())
}

// ── کانال‌های پیش‌نمایش ───────────────────────────────────────

func (h *Handler) adminListPreview(ctx context.Context, c tele.Context) error {
	channels, err := h.Store.ListPreviewChannels(ctx)
	h.LogErr("adminListPreview", err)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ch := range channels {
		title := ch.Title
		if title == "" {
			title = strconv.FormatInt(ch.ChatID, 10)
		}
		rows = append(rows, kb.Row(kb.Data("🗑 "+title, "admin_prev_del:"+ch.ID)))
	}
	kb.Inline(rows...)
	h.SetStep(ctx, c.Sender().ID, stepAddPreview)
	msg := "🖼 کانال‌های پیش‌نمایش:\n\nبرای افزودن، آیدی عددی کانال را بفرستید (اختیاری با عنوان: `-100123:عنوان`)."
	if len(channels) == 0 {
		msg = "🖼 هیچ کانال پیش‌نمایشی ثبت نشده.\n\nبرای افزودن، آیدی عددی کانال را بفرستید."
	}
	return c.Send(msg, kb)
}

func (h *Handler) adminSavePreview(ctx context.Context, c tele.Context, text string) error {
	h.ClearState(ctx, c.Sender().ID)
	idPart, title := splitIDTitle(text)
	chatID, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return c.Send("❌ آیدی کانال باید عددی باشد.", kbAdmin())
	}
	if err := h.Store.AddPreviewChannel(ctx, &models.PreviewChannel{ChatID: chatID, Title: title, IsActive: true}); err != nil {
		return c.Send("❌ ثبت کانال پیش‌نمایش با خطا مواجه شد. دوباره امتحان کنید.", kbAdmin())
	}
	return c.Send("✅ کانال پیش‌نمایش اضافه شد.", kbAdmin())
}

func (h *Handler) adminPreviewDelete(ctx context.Context, c tele.Context, id string) error {
	if err := h.Store.RemovePreviewChannel(ctx, id); err != nil {
		h.LogErr("adminPreviewDelete", err)
		return c.Edit("❌ حذف کانال پیش‌نمایش با خطا مواجه شد.")
	}
	return c.Edit("🗑 کانال پیش‌نمایش حذف شد.")
}

// ── تبلیغات ───────────────────────────────────────────────────

func (h *Handler) adminListAds(ctx context.Context, c tele.Context) error {
	ads, err := h.Store.ListAds(ctx)
	h.LogErr("adminListAds", err)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ad := range ads {
		rows = append(rows, kb.Row(kb.Data("🗑 "+ad.Title, "admin_ad_del:"+ad.ID)))
	}
	kb.Inline(rows...)
	h.SetStep(ctx, c.Sender().ID, stepAddAd)
	msg := "📣 تبلیغات:\n\nبرای افزودن با این قالب بفرستید:\n`عنوان | متن | متن دکمه | لینک دکمه`"
	if len(ads) == 0 {
		msg = "📣 هیچ تبلیغی ثبت نشده.\n\nبرای افزودن:\n`عنوان | متن | متن دکمه | لینک دکمه`"
	}
	return c.Send(msg, kb)
}

func (h *Handler) adminSaveAd(ctx context.Context, c tele.Context, text string) error {
	h.ClearState(ctx, c.Sender().ID)
	parts := strings.Split(text, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) < 2 || parts[0] == "" {
		return c.Send("❌ قالب نادرست. حداقل: عنوان | متن", kbAdmin())
	}
	ad := &models.Ad{Title: parts[0], Text: parts[1], IsActive: true}
	if len(parts) >= 3 {
		ad.ButtonText = parts[2]
	}
	if len(parts) >= 4 {
		ad.ButtonURL = parts[3]
	}
	if err := h.Store.AddAd(ctx, ad); err != nil {
		return c.Send("❌ ثبت تبلیغ با خطا مواجه شد. دوباره امتحان کنید.", kbAdmin())
	}
	return c.Send("✅ تبلیغ اضافه شد.", kbAdmin())
}

func (h *Handler) adminAdDelete(ctx context.Context, c tele.Context, id string) error {
	if err := h.Store.RemoveAd(ctx, id); err != nil {
		h.LogErr("adminAdDelete", err)
		return c.Edit("❌ حذف تبلیغ با خطا مواجه شد.")
	}
	return c.Edit("🗑 تبلیغ حذف شد.")
}

// ── تغییر اشتراک کاربر ────────────────────────────────────────

// adminUserSubMenu پلن‌ها را برای تغییر اشتراک یک کاربر نشان می‌دهد.
func (h *Handler) adminUserSubMenu(ctx context.Context, c tele.Context, tgIDStr string) error {
	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("adminUserSubMenu", err)
	if len(plans) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "هیچ پلنی موجود نیست"})
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("%s (%d روز)", p.Name, p.Days)
		rows = append(rows, kb.Row(kb.Data(label, "admin_setsub:"+tgIDStr+":"+p.ID)))
	}
	kb.Inline(rows...)
	return c.Edit("💎 پلن جدید برای این کاربر را انتخاب کنید:", kb)
}

func (h *Handler) adminSetUserSub(ctx context.Context, c tele.Context, tgIDStr, planID string) error {
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return c.Edit(msgInvalid)
	}
	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("adminSetUserSub: list plans", err)
	days := 0
	name := ""
	for _, p := range plans {
		if p.ID == planID {
			days = p.Days
			name = p.Name
			break
		}
	}
	if days == 0 {
		return c.Edit(msgPlanNF)
	}
	if err := h.Store.SetUserSub(ctx, tgID, planID, days); err != nil {
		return c.Edit("❌ خطا در فعال‌سازی اشتراک.")
	}
	return c.Edit(fmt.Sprintf("✅ اشتراک «%s» برای کاربر فعال شد (%d روز).", name, days))
}

func (h *Handler) adminResetUserDownloads(ctx context.Context, c tele.Context, tgIDStr string) error {
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return c.Edit(msgInvalid)
	}
	user, err := h.Store.GetUser(ctx, tgID)
	h.LogErr("adminResetUserDownloads: get user", err)
	if user == nil {
		return c.Edit("❌ کاربر یافت نشد.")
	}
	user.FreeDownloads = 0
	if err := h.Store.UpdateUser(ctx, user); err != nil {
		h.LogErr("adminResetUserDownloads: update", err)
		return c.Edit("❌ بازنشانی با خطا مواجه شد.")
	}
	if err := h.Store.DeleteUserDownloads(ctx, user.ID); err != nil {
		h.LogErr("adminResetUserDownloads: delete logs", err)
	}
	return c.Edit("♻️ دانلودهای این کاربر ریست شد.")
}

// ── helper ────────────────────────────────────────────────────

// splitIDTitle ورودی "id:title" یا فقط "id" را جدا می‌کند.
func splitIDTitle(text string) (id, title string) {
	text = strings.TrimSpace(text)
	if i := strings.Index(text, ":"); i >= 0 {
		return strings.TrimSpace(text[:i]), strings.TrimSpace(text[i+1:])
	}
	return text, ""
}
