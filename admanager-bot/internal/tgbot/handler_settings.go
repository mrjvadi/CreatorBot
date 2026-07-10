// handler_settings.go — تنظیمات کلی ربات.
//
// نسخه‌ی فعلی فقط مهلتِ یادآوریِ پیش از ارسال را قابل‌تغییر می‌کند
// (بخش ۲.۶ سند نیازمندی). بقیه‌ی BotSettings (پیام خوش‌آمد/راهنما) هنوز
// از طریق منو در دسترس نیست.
package tgbot

import (
	"context"
	"fmt"
	"strconv"

	tele "gopkg.in/telebot.v4"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (h *Handler) settingsHome(c tele.Context) error {
	ctx := context.Background()
	st, err := h.store.GetSettings(ctx)
	if err != nil || st == nil {
		return c.Send("❌ خطا در خواندن تنظیمات.", kbAdminMain())
	}
	text := fmt.Sprintf(
		"⚙️ <b>تنظیمات</b>\n\n"+
			"⏰ مهلتِ یادآوری پیش از ارسال: %s\n"+
			"(هر بار ۱۰ دقیقه قبل از ارسال واقعیِ یک تبلیغ، پیام یادآوری در همین چت دریافت می‌کنید. با ۰ خاموش می‌شود.)",
		reminderLeadLabel(st.ReminderMinutesBefore),
	)
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(cbBtn(kb, "✏️ تغییر مهلتِ یادآوری", "set_reminder")))
	return c.Send(text, tele.ModeHTML, kb)
}

func reminderLeadLabel(n int) string {
	if n <= 0 {
		return "خاموش"
	}
	return fmt.Sprintf("%d دقیقه", n)
}

func (h *Handler) settingsReminderStart(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepSettingsReminder)
	return c.Edit("⏰ چند دقیقه پیش از ارسال یادآوری بفرستم؟ (۰ = خاموش)", kbCancelOnly())
}

func (h *Handler) handleSettingsReminder(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	n, err := strconv.Atoi(trimText(models.NormalizeDigits(text)))
	if err != nil || n < 0 {
		return c.Send("❌ یک عدد معتبر (۰ یا بیشتر) بفرستید:", kbCancelOnly())
	}
	h.clearState(ctx, uid)
	if err := h.store.UpdateSettings(ctx, bson.D{{Key: "reminder_minutes_before", Value: n}}); err != nil {
		h.log.Error("update reminder setting", portsF("err", err))
		return c.Send("❌ خطا در ذخیره‌ی تنظیمات.", kbAdminMain())
	}
	_ = c.Send("✅ ذخیره شد.", kbAdminMain())
	return h.settingsHome(c)
}
