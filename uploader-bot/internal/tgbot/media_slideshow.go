package tgbot

import (
	"context"
	"fmt"
	"strconv"

	tele "gopkg.in/telebot.v4"
)

// adminSlideshow نمایش اسلایدی رسانه‌ها — یک کد در هر صفحه با ناوبری.
func (h *Handler) adminSlideshow(ctx context.Context, c tele.Context, idxStr string) error {
	idx, _ := strconv.Atoi(idxStr) // از دکمه‌ی ◀️/▶️ خودمان می‌آید
	codes, _, err := h.Store.ListCodes(ctx, "", 1, 100)
	h.LogErr("adminSlideshow: list", err)
	if len(codes) == 0 {
		return c.Edit("📭 هیچ رسانه‌ای ثبت نشده.", kbBackHome())
	}
	if idx < 0 {
		idx = len(codes) - 1
	}
	if idx >= len(codes) {
		idx = 0
	}
	code := codes[idx]
	files, err := h.Store.GetFilesForCode(ctx, code.ID)
	h.LogErr("adminSlideshow: files", err)

	caption := code.Caption
	if len(caption) > 120 {
		caption = caption[:120] + "…"
	}
	if caption == "" {
		caption = "—"
	}
	text := fmt.Sprintf(
		"🎞 <b>%d / %d</b>\n\n🔑 کد: <code>%s</code>\n📦 فایل‌ها: %d\n📥 دریافت: %d\n📝 کپشن: %s",
		idx+1, len(codes), code.Code, len(files), code.UsedCount, caption)

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("◀️", "slide:"+strconv.Itoa(idx-1)),
			kb.Data(fmt.Sprintf("%d/%d", idx+1, len(codes)), "slide:"+strconv.Itoa(idx)),
			kb.Data("▶️", "slide:"+strconv.Itoa(idx+1)),
		),
		kb.Row(kb.Data("⚙️ تنظیمات این رسانه", "admin_code_edit:"+code.ID)),
		kb.Row(kb.Data("👁 ارسال برای خودم", "code_resend:"+code.Code)),
		kb.Row(kb.Data(btnBackLabel, "p:home")),
	)
	return c.Edit(text, tele.ModeHTML, kb)
}
