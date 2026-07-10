package tgbot

import (
	"context"
	"fmt"
	"strconv"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// kbCodeAdvanced منوی کامل تنظیمات یک کد رسانه.
func kbCodeAdvanced(code *models.Code) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	on := func(b bool) string {
		if b {
			return "🟢"
		}
		return "🔴"
	}
	id := code.ID
	kb.Inline(
		kb.Row(
			kb.Data(on(code.ForwardLock)+" قفل فوروارد", "code_toggle_forward:"+id),
			kb.Data(on(code.AutoDelete > 0)+" ضدفیلتر", "code_toggle_antidl:"+id),
		),
		kb.Row(
			kb.Data(on(code.ChannelLock)+" قفل کانال", "code_toggle_channel:"+id),
			kb.Data(on(code.SubRequired)+" اشتراک اجباری", "code_toggle_sub:"+id),
		),
		kb.Row(
			kb.Data(on(code.ForceSeen)+" سین اجباری", "code_toggle_seen:"+id),
			kb.Data(on(code.ForceReact)+" ری‌اکشن اجباری", "code_toggle_react:"+id),
		),
		kb.Row(
			kb.Data(fmt.Sprintf("👍 لایک فیک: %d", code.FakeLikes), "code_set_likes:"+id),
			kb.Data(fmt.Sprintf("📥 دانلود فیک: %d", code.FakeDownloads), "code_set_downloads:"+id),
		),
		kb.Row(
			kb.Data(fmt.Sprintf("👁 بازدید فیک: %d", code.FakeViews), "code_set_views:"+id),
			kb.Data(fmt.Sprintf("📥 محدودیت: %s", limitText(code.DownloadLimit)), "code_set_limit:"+id),
		),
		kb.Row(
			kb.Data(passwordText(code.Password), "code_set_password:"+id),
			kb.Data("✏️ کپشن", "code_edit_caption:"+id),
		),
		kb.Row(
			kb.Data("🖼 کاور ویدیو", "code_set_cover:"+id),
			kb.Data("📂 انتقال به پوشه", "code_move:"+id),
		),
		kb.Row(
			kb.Data("🔀 ترتیب فایل‌ها", "code_order:"+id),
		),
		kb.Row(
			kb.Data("📤 پیش‌نمایش", "code_send_preview:"+id),
			kb.Data("🗑 حذف", "code_delete:"+id),
		),
		kb.Row(kb.Data("🔙 لیست رسانه‌ها", "code_list")),
	)
	return kb
}

func limitText(n int) string {
	if n <= 0 {
		return "نامحدود"
	}
	return strconv.Itoa(n)
}

func passwordText(p string) string {
	if p == "" {
		return "🔓 رمز: ندارد"
	}
	return "🔐 رمز: دارد"
}

// adminCodeAskFake از ادمین مقدار یک آمار فیک را می‌پرسد.
func (h *Handler) adminCodeAskFake(ctx context.Context, c tele.Context, codeID string, st step, label string) error {
	h.SetStepData(ctx, c.Sender().ID, st, "code_id", codeID)
	return c.Send("🔢 "+label+" را به‌صورت عدد بفرستید:", kbCancelOnly())
}

// adminCodeSaveFake مقدار آمار فیک را ذخیره می‌کند.
func (h *Handler) adminCodeSaveFake(ctx context.Context, c tele.Context, us userState, kind, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	code, err := h.Store.FindCodeByID(ctx, us.Data["code_id"])
	h.LogErr("adminCodeSaveFake: find", err)
	if code == nil {
		return c.Send(msgNotFound, kbAdmin())
	}
	n, _ := strconv.Atoi(text) // نامعتبر → 0
	switch kind {
	case "likes":
		code.FakeLikes = n
	case "downloads":
		code.FakeDownloads = n
	case "views":
		code.FakeViews = n
	}
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminCodeSaveFake: update", err)
		return c.Send("❌ ذخیره‌سازی با خطا مواجه شد.", kbAdmin())
	}
	return c.Send(msgSaved, kbAdmin())
}
