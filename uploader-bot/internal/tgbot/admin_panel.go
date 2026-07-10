package tgbot

import (
	"context"
	"fmt"
	"regexp"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// btnDone متن دکمه‌ی پایان آپلود آلبوم.
const btnDone = "✅ تمام شد"

// linkRegex برای حذف لینک‌ها از کپشن (در صورت فعال‌بودن تنظیم remove_links).
var linkRegex = regexp.MustCompile(`(?i)\b((https?://|www\.|t\.me/|@)\S+)`)

// maybeStripLinks اگر تنظیم حذف لینک فعال باشد، لینک‌ها را از متن پاک می‌کند.
func (h *Handler) maybeStripLinks(ctx context.Context, text string) string {
	if h.Store.GetSetting(ctx, models.SettingRemoveLinks) == "true" {
		return linkRegex.ReplaceAllString(text, "")
	}
	return text
}

// ── نهایی‌سازی آپلود رسانه (آلبوم/تک‌فایل) ────────────────────

// finishUpload از فایل‌های بافرشده یک کد رسانه می‌سازد.
func (h *Handler) finishUpload(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	fileIDs := h.AlbumDrain(ctx, uid)
	if len(fileIDs) == 0 {
		h.ClearState(ctx, uid)
		return c.Send("❌ هیچ فایلی ارسال نشد.", kbAdmin())
	}

	code := &models.Code{
		Code:       h.Store.GenerateUniqueCode(ctx),
		Type:       models.CodeUnlimited,
		IsAlbum:    len(fileIDs) > 1,
		FileIDs:    fileIDs,
		UploaderID: uid,
	}
	h.applyUploadDefaults(ctx, code)
	if err := h.Store.CreateCode(ctx, code); err != nil {
		h.ClearState(ctx, uid)
		h.AlbumClear(ctx, uid)
		return c.Send("❌ خطا در ساخت کد رسانه.", kbAdmin())
	}
	h.Store.RefreshCodeTypes(ctx, code.ID)

	h.AlbumClear(ctx, uid)
	h.ClearState(ctx, uid)

	link := h.deepLink(code.Code)
	msg := fmt.Sprintf("✅ رسانه با %d فایل ذخیره شد.\n\n🔑 کد: <code>%s</code>\n🔗 لینک: %s",
		len(fileIDs), code.Code, link)
	return c.Send(msg, tele.ModeHTML, kbAdmin())
}

// applyUploadDefaults تنظیمات پیش‌فرض آپلود (قفل فوروارد، ضدفیلتر/حذف خودکار)
// را روی یک کد تازه اعمال می‌کند تا کاربر مجبور نباشد دستی روی هر کد فعال کند.
func (h *Handler) applyUploadDefaults(ctx context.Context, code *models.Code) {
	code.ForwardLock = h.Store.GetSetting(ctx, models.SettingForwardLockDefault) == "true"
	if h.Store.GetSetting(ctx, models.SettingAntiFilterDefault) == "true" {
		sec := h.GetSettingInt(ctx, models.SettingAutoDeleteDefault, 30)
		if sec <= 0 {
			sec = 30
		}
		code.AutoDelete = sec
		code.AntiFilter = true
	}
}

// deepLink لینک دریافت مستقیم یک کد را می‌سازد.
func (h *Handler) deepLink(code string) string {
	if h.Bot != nil && h.Bot.Me != nil && h.Bot.Me.Username != "" {
		return fmt.Sprintf("https://t.me/%s?start=%s", h.Bot.Me.Username, code)
	}
	return code
}

// ── تنظیمات (toggle) ──────────────────────────────────────────
//
// نکته: adminToggleSetting/adminAskSetting (نسخه‌ی قدیمی، مسطح) این‌جا حذف
// شدند — هیچ دکمه‌ای صدایشان نمی‌زد. مسیر زنده‌ی تغییر تنظیمات، پنل
// دسته‌بندی‌شده در admin_menu.go («pt:»/«pv:») است که مستقیماً
// adminToggleSettingRaw و stepEditSetting را صدا می‌زند.

// adminToggleSettingRaw فقط یک تنظیم بولی را برعکس می‌کند (بدون رفرش UI).
func (h *Handler) adminToggleSettingRaw(ctx context.Context, key string) {
	cur := h.Store.GetSetting(ctx, key)
	next := "true"
	if cur == "true" || cur == "1" {
		next = "false"
	}
	h.LogErr("adminToggleSettingRaw", h.Store.SetSetting(ctx, key, next))
}

// onPanel دستور /panel — باز کردن پنل مدیریت.
func (h *Handler) onPanel(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	return h.OpenPanel(context.Background(), c)
}

// ── پوشه‌ی جدید ───────────────────────────────────────────────

func (h *Handler) adminNewFolder(ctx context.Context, c tele.Context) error {
	h.SetStep(ctx, c.Sender().ID, stepNewFolder)
	return c.Send("📁 نام پوشه‌ی جدید را بفرستید:", kbCancelOnly())
}

// ── پیش‌نمایش رسانه در کانال‌های پیش‌نمایش ────────────────────

func (h *Handler) adminSendPreview(ctx context.Context, c tele.Context, codeID string) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminSendPreview: find code", err)
	if code == nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ یافت نشد"})
	}
	channels, err := h.Store.ListPreviewChannels(ctx)
	h.LogErr("adminSendPreview: list channels", err)
	if len(channels) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "هیچ کانال پیش‌نمایشی تنظیم نشده"})
	}
	files, err := h.Store.GetFilesForCode(ctx, code.ID)
	h.LogErr("adminSendPreview: get files", err)
	sent := 0
	for _, ch := range channels {
		for _, f := range files {
			if _, err := sendToChat(h.Bot, ch.ChatID, f, code.Caption); err == nil {
				sent++
			}
		}
	}
	return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("✅ در %d کانال ارسال شد", len(channels))})
}

// ── بررسی عضویت (دکمه «✅ عضو شدم») ───────────────────────────

func (h *Handler) onCheckJoin(ctx context.Context, c tele.Context) error {
	notJoined := h.checkForceJoin(ctx, c)
	if len(notJoined) > 0 {
		return c.Respond(&tele.CallbackResponse{Text: "هنوز همه‌ی عضویت‌ها کامل نشده 🙈"})
	}
	h.countLockJoins(ctx, c) // شمارش برای حد عضو
	h.LogErr("onCheckJoin: respond", c.Respond(&tele.CallbackResponse{Text: "✅ عالی، تایید شد!"}))
	return c.Edit("🎉 عضویتت تایید شد! حالا کد رسانه رو برام بفرست.")
}

// countLockJoins برای قفل‌های اجباریِ دارای حد عضو، هر کاربرِ یکتا را یک‌بار
// می‌شمارد و با رسیدن به حد، قفل خودکار غیرفعال می‌شود.
func (h *Handler) countLockJoins(ctx context.Context, c tele.Context) {
	if h.Cache == nil {
		return
	}
	uid := c.Sender().ID
	locks, err := h.Store.ListForceJoinChannels(ctx)
	h.LogErr("countLockJoins: list", err)
	for i := range locks {
		l := locks[i]
		if !l.IsMandatory() || l.MemberCap <= 0 {
			continue
		}
		key := fmt.Sprintf("lkjoin:%s:%s:%d", h.InstanceID, l.ID, uid)
		ok, err := h.Cache.SetNX(ctx, key, "1", 720*time.Hour)
		h.LogErr("countLockJoins: setnx", err)
		if !ok {
			continue // قبلاً شمرده شده
		}
		if deactivated := h.Store.IncrLockJoined(ctx, l.ID); deactivated && h.OwnerID != 0 {
			if _, sendErr := h.Bot.Send(&tele.User{ID: h.OwnerID},
				fmt.Sprintf("✅ قفل «%s» به حد عضو %d رسید و خودکار غیرفعال شد.", lockTitle(&l), l.MemberCap)); sendErr != nil {
				h.LogErr("countLockJoins: notify owner", sendErr)
			}
		}
	}
}
