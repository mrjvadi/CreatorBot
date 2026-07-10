package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// lockList فهرست قفل‌ها + دکمه‌های مدیریت.
func (h *Handler) lockList(ctx context.Context, c tele.Context) error {
	locks, err := h.Store.ListForceJoinChannels(ctx)
	h.LogErr("lockList", err)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	mandatoryCount := 0
	for i := range locks {
		l := locks[i]
		if l.IsMandatory() {
			mandatoryCount++
		}
		icon := "🔓"
		if l.IsMandatory() {
			icon = "🔒"
		}
		title := lockTitle(&l)
		cap := ""
		if l.MemberCap > 0 {
			cap = fmt.Sprintf(" (%d/%d)", l.JoinedCount, l.MemberCap)
		}
		rows = append(rows, kb.Row(kb.Data(icon+" "+title+cap, "lk:"+l.ID)))
	}
	leave := "🔔 گزارش لفت: خاموش"
	if h.Store.GetSetting(ctx, models.SettingLeaveReport) == "true" {
		leave = "🔔 گزارش لفت: روشن"
	}
	rows = append(rows,
		kb.Row(kb.Data("➕ افزودن قفل", "lk_add")),
		kb.Row(kb.Data(leave, "lk_leave")),
		kb.Row(kb.Data(btnPanelLabel, "p:home")),
	)
	kb.Inline(rows...)

	head := "🔐 <b>قفل‌ها</b>\n"
	if len(locks) == 0 {
		head += "هیچ قفلی ثبت نشده."
	} else if mandatoryCount == 0 {
		head += "⚠️ هیچ قفل اجباری‌ای ندارید؛ ربات درخواست عضویت نمی‌دهد."
	} else {
		head += fmt.Sprintf("قفل اجباری: %d", mandatoryCount)
	}
	return sendOrEdit(c, head, tele.ModeHTML, kb)
}

// sendOrEdit اگر از callback آمده باشد ویرایش، وگرنه ارسال می‌کند.
func sendOrEdit(c tele.Context, what string, opts ...any) error {
	if c.Callback() != nil {
		return c.Edit(what, opts...)
	}
	return c.Send(what, opts...)
}

func lockTitle(l *models.ForceJoinChannel) string {
	if l.Title != "" {
		return l.Title
	}
	if l.BotUsername != "" {
		return "@" + l.BotUsername
	}
	if l.Username != "" {
		return "@" + l.Username
	}
	if l.InviteURL != "" {
		return l.InviteURL
	}
	return strconv.FormatInt(l.ChatID, 10)
}

// lockDetail جزئیات و کنترل یک قفل.
func (h *Handler) lockDetail(ctx context.Context, c tele.Context, id string) error {
	l, err := h.Store.FindForceJoinChannel(ctx, id)
	h.LogErr("lockDetail", err)
	if l == nil {
		return c.Edit("❌ قفل یافت نشد.", kbBackHome())
	}
	mode := "🔓 اختیاری"
	if l.IsMandatory() {
		mode = "🔒 اجباری"
	}
	capTxt := "نامحدود"
	if l.MemberCap > 0 {
		capTxt = fmt.Sprintf("%d (تاکنون %d)", l.MemberCap, l.JoinedCount)
	}
	text := fmt.Sprintf("🔐 <b>%s</b>\nنوع: %s\nحالت: %s\nحد عضو: %s\n🔗 %s",
		lockTitle(l), lockKindLabel(l.Kind), mode, capTxt, l.LinkURL())

	kb := &tele.ReplyMarkup{}
	toggleLbl := "🔒 اجباری کن"
	if l.IsMandatory() {
		toggleLbl = "🔓 اختیاری کن"
	}
	kb.Inline(
		kb.Row(kb.Data(toggleLbl, "lk_mode:"+id)),
		kb.Row(kb.Data("👥 تنظیم حد عضو", "lk_cap:"+id)),
		kb.Row(kb.Data("🗑 حذف قفل", "lk_del:"+id)),
		kb.Row(kb.Data(btnBackLabel, "p:fjoin")),
	)
	return c.Edit(text, tele.ModeHTML, kb)
}

func lockKindLabel(k string) string {
	switch k {
	case models.LockBot:
		return "ربات"
	case models.LockGroup:
		return "گروه"
	case models.LockLink:
		return "لینک"
	default:
		return "کانال"
	}
}

func (h *Handler) lockToggleMode(ctx context.Context, c tele.Context, id string) error {
	l, err := h.Store.FindForceJoinChannel(ctx, id)
	h.LogErr("lockToggleMode: find", err)
	if l == nil {
		return c.Edit(msgNotFound)
	}
	if l.IsMandatory() {
		l.Mode = models.LockOptional
	} else {
		l.Mode = models.LockMandatory
	}
	if err := h.Store.UpdateForceJoinChannel(ctx, l); err != nil {
		h.LogErr("lockToggleMode: update", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد"})
	}
	return h.lockDetail(ctx, c, id)
}

func (h *Handler) lockDelete(ctx context.Context, c tele.Context, id string) error {
	if err := h.Store.RemoveForceJoinChannel(ctx, id); err != nil {
		h.LogErr("lockDelete", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ حذف ناموفق بود"})
	}
	return h.lockList(ctx, c)
}

func (h *Handler) lockToggleLeave(ctx context.Context, c tele.Context) error {
	h.adminToggleSettingRaw(ctx, models.SettingLeaveReport)
	return h.lockList(ctx, c)
}

// lockAskAdd از ادمین ورودی قفل را می‌پرسد.
func (h *Handler) lockAskAdd(ctx context.Context, c tele.Context) error {
	h.SetStep(ctx, c.Sender().ID, stepAddLock)
	msg := "🔐 قفل جدید را بفرستید:\n" +
		"• کانال/گروه: @username یا آیدی -100...\n" +
		"• لینک دعوت: https://t.me/...\n" +
		"• ربات: یوزرنیم ربات (@xbot) یا توکن ربات\n\n" +
		"پیش‌فرض کانال/گروه «اجباری» و لینک/ربات «اختیاری» ثبت می‌شود؛ بعداً قابل تغییر است."
	return c.Send(msg, kbCancelOnly())
}

// lockSaveAdd ورودی را تحلیل و قفل را ذخیره می‌کند.
func (h *Handler) lockSaveAdd(ctx context.Context, c tele.Context, text string) error {
	h.ClearState(ctx, c.Sender().ID)
	text = strings.TrimSpace(text)
	l := &models.ForceJoinChannel{IsActive: true, Mode: models.LockMandatory, Kind: models.LockChannel}

	switch {
	case strings.Contains(text, ":") && !strings.HasPrefix(text, "http"):
		// توکن ربات → قفل ربات اجباری
		// امنیت: این توکن قبلاً به‌صورت متن‌خام در Mongo ذخیره می‌شد (برخلاف
		// BotInstance.BotToken و member-bot's CheckBot.Token که هر دو
		// AES-256-GCM رمزنگاری‌شده‌اند) — رجوع کنید به گزارش امنیتی. حالا
		// قبل از ذخیره رمزنگاری می‌شود.
		l.Kind = models.LockBot
		l.Mode = models.LockMandatory
		if h.EncryptKey != "" {
			if enc, err := auth.Encrypt(text, h.EncryptKey); err == nil {
				l.BotToken = enc
			} else {
				h.Log.Error("encrypt lock bot token failed", ports.F("err", err))
				return c.Send("❌ خطا در ذخیره‌ی امن توکن. دوباره تلاش کنید.", kbAdmin())
			}
		} else {
			h.Log.Error("ENCRYPTION_KEY not configured — refusing to store bot token in plaintext")
			return c.Send("❌ سرویس برای ذخیره‌ی امن توکن تنظیم نشده (ENCRYPTION_KEY). به ادمین پلتفرم اطلاع دهید.", kbAdmin())
		}
		l.Title = "ربات (با توکن)"
	case strings.HasPrefix(text, "http"):
		l.Kind = models.LockLink
		l.Mode = models.LockOptional
		l.InviteURL = text
		l.Title = text
	case strings.HasSuffix(strings.ToLower(strings.TrimPrefix(text, "@")), "bot"):
		l.Kind = models.LockBot
		l.Mode = models.LockOptional
		l.BotUsername = strings.TrimPrefix(text, "@")
		l.Title = "@" + l.BotUsername
	case strings.HasPrefix(text, "@"):
		l.Kind = models.LockChannel
		l.Username = strings.TrimPrefix(text, "@")
		l.Title = text
	default:
		if id, err := strconv.ParseInt(text, 10, 64); err == nil {
			l.ChatID = id
			l.Title = text
		} else {
			l.Username = text
			l.Title = "@" + text
		}
	}
	if err := h.Store.AddForceJoinChannel(ctx, l); err != nil {
		h.LogErr("lockSaveAdd", err)
		return c.Send("❌ ثبت قفل با خطا مواجه شد. دوباره امتحان کنید.", kbAdmin())
	}
	return c.Send("✅ قفل اضافه شد. از «قفل‌ها» می‌توانید حالت و حد عضو را تنظیم کنید.", kbAdmin())
}

// lockAskCap حد عضو را می‌پرسد.
func (h *Handler) lockAskCap(ctx context.Context, c tele.Context, id string) error {
	h.SetStepData(ctx, c.Sender().ID, stepLockCap, "lock_id", id)
	return c.Send("👥 حد عضو را به‌صورت عدد بفرستید (0 = نامحدود):", kbCancelOnly())
}

func (h *Handler) lockSaveCap(ctx context.Context, c tele.Context, st userState, text string) error {
	h.ClearState(ctx, c.Sender().ID)
	l, err := h.Store.FindForceJoinChannel(ctx, st.Data["lock_id"])
	h.LogErr("lockSaveCap: find", err)
	if l == nil {
		return c.Send("❌ قفل یافت نشد.", kbAdmin())
	}
	n, _ := strconv.Atoi(strings.TrimSpace(text)) // نامعتبر → 0 (نامحدود)
	l.MemberCap = n
	if err := h.Store.UpdateForceJoinChannel(ctx, l); err != nil {
		h.LogErr("lockSaveCap: update", err)
		return c.Send("❌ ذخیره‌ی حد عضو با خطا مواجه شد.", kbAdmin())
	}
	return c.Send("✅ حد عضو تنظیم شد.", kbAdmin())
}
