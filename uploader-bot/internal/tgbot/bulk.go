package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// kbTools منوی ابزارهای انبوه.
func kbTools() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🔒 قفل فوروارد همه", "tools_fwd_on"), kb.Data("🔓 برداشتن همه", "tools_fwd_off")),
		kb.Row(kb.Data("⏱ ضدفیلتر همه", "tools_ad_on"), kb.Data("⏱ خاموش همه", "tools_ad_off")),
		kb.Row(kb.Data("🗑 حذف همه‌ی رسانه‌ها", "tools_delall")),
		kb.Row(kb.Data(btnBackLabel, "p:home")),
	)
	return kb
}

func (h *Handler) panelTools(ctx context.Context, c tele.Context) error {
	return c.Edit("🧰 <b>ابزارهای انبوه</b>\n\nاعمال روی همه‌ی رسانه‌ها:", tele.ModeHTML, kbTools())
}

func (h *Handler) toolsForwardAll(ctx context.Context, c tele.Context, on bool) error {
	n, err := h.Store.SetForwardLockAll(ctx, on)
	if err != nil {
		h.LogErr("toolsForwardAll", err)
		return c.Edit("❌ اعمال قفل فوروارد روی همه‌ی رسانه‌ها با خطا مواجه شد.", kbTools())
	}
	state := "فعال"
	if !on {
		state = "غیرفعال"
	}
	return c.Edit(fmt.Sprintf("✅ قفل فوروارد روی %d رسانه %s شد.", n, state), kbTools())
}

func (h *Handler) toolsAutoDeleteAll(ctx context.Context, c tele.Context, on bool) error {
	sec := 0
	if on {
		sec = h.GetSettingInt(ctx, models.SettingAutoDeleteDefault, 30)
		if sec <= 0 {
			sec = 30
		}
	}
	n, err := h.Store.SetAutoDeleteAll(ctx, sec)
	if err != nil {
		h.LogErr("toolsAutoDeleteAll", err)
		return c.Edit("❌ اعمال ضدفیلتر روی همه‌ی رسانه‌ها با خطا مواجه شد.", kbTools())
	}
	if on {
		return c.Edit(fmt.Sprintf("✅ ضدفیلتر (%d ثانیه) روی %d رسانه فعال شد.", sec, n), kbTools())
	}
	return c.Edit(fmt.Sprintf("✅ ضدفیلتر روی %d رسانه خاموش شد.", n), kbTools())
}

func (h *Handler) toolsDeleteAllConfirm(ctx context.Context, c tele.Context) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("⚠️ بله، همه حذف شود", "tools_delall_yes")),
		kb.Row(kb.Data("🔙 انصراف", "p:tools")),
	)
	return c.Edit("⚠️ <b>هشدار:</b> همه‌ی رسانه‌ها و فایل‌ها برای همیشه حذف می‌شوند. مطمئنید؟", tele.ModeHTML, kb)
}

func (h *Handler) toolsDeleteAll(ctx context.Context, c tele.Context) error {
	if err := h.Store.DeleteAllCodes(ctx); err != nil {
		return c.Edit("❌ خطا در حذف.")
	}
	return c.Edit("🗑 همه‌ی رسانه‌ها حذف شدند.", kbTools())
}
