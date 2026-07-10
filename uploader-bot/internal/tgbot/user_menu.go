package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// userBtn یک دکمه‌ی کاربری: کلیدِ برچسب (قابل تغییر)، برچسب پیش‌فرض،
// کلیدِ نمایش (on/off)، و اکشن.
type userBtn struct {
	lblKey  string
	def     string
	showKey string // "" یعنی همیشه نمایش داده شود
	act     string
}

func userButtons() []userBtn {
	return []userBtn{
		{models.SettingLblNewest, btnNewest, models.SettingBtnNewest, "newest"},
		{models.SettingLblPopular, btnPopular, models.SettingBtnPopular, "popular"},
		{models.SettingLblTop, btnTop, models.SettingBtnTop, "top"},
		{models.SettingLblSearch, btnSearch, models.SettingShowSearch, "search"},
		{models.SettingLblUpload, btnUploadU, models.SettingUserUpload, "upload"},
		{models.SettingLblHelp, btnHelp, "", "help"},
		{models.SettingLblSupport, btnSupport, "", "support"},
	}
}

// btnLabel برچسب فعلی (تنظیم‌شده یا پیش‌فرض) یک دکمه.
func (h *Handler) btnLabel(ctx context.Context, b userBtn) string {
	return h.GetSetting(ctx, b.lblKey, b.def)
}

// kbUserMenu کیبورد کاربری را با برچسب‌های قابل‌تغییر و بر اساس تنظیمات نمایش می‌سازد.
func (h *Handler) kbUserMenu(ctx context.Context) *tele.ReplyMarkup {
	s := h.Store.GetAllSettings(ctx)
	on := func(k string) bool { return k == "" || s[k] == "true" }
	lbl := func(b userBtn) string {
		if v := s[b.lblKey]; v != "" {
			return v
		}
		return b.def
	}

	btns := userButtons()
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	var rows []tele.Row

	// ردیف اول: ناوبری محتوا (جدید/پربازدید/محبوب)
	var nav []tele.Btn
	for _, b := range btns {
		if (b.act == "newest" || b.act == "popular" || b.act == "top") && on(b.showKey) {
			nav = append(nav, kb.Text(lbl(b)))
		}
	}
	if len(nav) > 0 {
		rows = append(rows, kb.Row(nav...))
	}

	// جستجو و آپلود (هرکدام در ردیف خودش)
	for _, b := range btns {
		if (b.act == "search" || b.act == "upload") && on(b.showKey) {
			rows = append(rows, kb.Row(kb.Text(lbl(b))))
		}
	}

	// راهنما + پشتیبانی
	var help, support userBtn
	for _, b := range btns {
		if b.act == "help" {
			help = b
		}
		if b.act == "support" {
			support = b
		}
	}
	rows = append(rows, kb.Row(kb.Text(lbl(help)), kb.Text(lbl(support))))

	kb.Reply(rows...)
	return kb
}

// userOnText پیام متنی کاربر: ابتدا دکمه‌های منو (با برچسب فعلی)، سپس دریافت با کد.
func (h *Handler) userOnText(ctx context.Context, c tele.Context, text string) error {
	// نگاشت برچسب فعلی → اکشن
	act := ""
	for _, b := range userButtons() {
		if h.btnLabel(ctx, b) == text {
			act = b.act
			break
		}
	}

	switch act {
	case "search":
		h.SetStep(ctx, c.Sender().ID, stepSearch)
		return c.Send("🔎 عبارت جستجو را بفرستید:", kbCancelOnly())
	case "help":
		return c.Send(h.GetSetting(ctx, models.SettingHelpText, "ℹ️ کد رسانه را بفرستید تا فایل را دریافت کنید."), h.kbUserMenu(ctx))
	case "support":
		return c.Send(h.GetSetting(ctx, models.SettingSupportText, "💬 برای پشتیبانی با ادمین در تماس باشید."), h.kbUserMenu(ctx))
	case "newest":
		return h.userShowList(ctx, c, "created_at", "🆕 جدیدترین رسانه‌ها")
	case "popular":
		return h.userShowList(ctx, c, "fake_views", "🔥 پربازدیدترین رسانه‌ها")
	case "top":
		return h.userShowList(ctx, c, "used_count", "⭐️ محبوب‌ترین رسانه‌ها")
	case "upload":
		return c.Send("📤 فایل خود را بفرستید.", h.kbUserMenu(ctx))
	}

	if !h.spamOK(ctx, c) {
		return c.Send("⏳ کمی صبر کنید و دوباره تلاش کنید.")
	}
	user, err := h.Store.GetUser(ctx, c.Sender().ID)
	h.LogErr("userOnText: get user", err)
	return h.userDeliverCode(ctx, c, user, text)
}

// userShowList فهرستی از کدها را با دکمه‌ی دریافت نشان می‌دهد.
func (h *Handler) userShowList(ctx context.Context, c tele.Context, field, title string) error {
	codes, err := h.Store.ListCodesSorted(ctx, field, 10)
	h.LogErr("userShowList", err)
	if len(codes) == 0 {
		return c.Send("📭 موردی برای نمایش نیست.", h.kbUserMenu(ctx))
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, code := range codes {
		label := code.Caption
		if strings.TrimSpace(label) == "" {
			label = "📄 " + code.Code
		}
		if len(label) > 40 {
			label = label[:40] + "…"
		}
		rows = append(rows, kb.Row(kb.Data(label, "code_resend:"+code.Code)))
	}
	kb.Inline(rows...)
	return c.Send(fmt.Sprintf("%s:", title), kb)
}
