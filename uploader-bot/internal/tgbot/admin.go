package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/documents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ════════════════════════════════════════════════════════════
// پنل مدیریت
// ════════════════════════════════════════════════════════════

func (h *Handler) adminPanel(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	return c.Send("پنل مدیریت:", kbAdmin())
}

// ════════════════════════════════════════════════════════════
// ساخت کد جدید
// ════════════════════════════════════════════════════════════

func (h *Handler) adminNewCode(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepCodeType)
	return c.Send(
		"نوع کد را انتخاب کنید:\n\n"+
			"1️⃣ <b>یک‌بار</b> — فقط یک نفر می‌تواند استفاده کند\n"+
			"🔢 <b>محدود</b> — تعداد مشخص استفاده\n"+
			"♾ <b>نامحدود</b> — بی‌نهایت استفاده\n"+
			"⏰ <b>زمان‌دار</b> — تا تاریخ مشخص",
		tele.ModeHTML, kbCodeType(),
	)
}

// handleStep state machine ادمین برای ساخت کد.
func (h *Handler) handleStep(ctx context.Context, c tele.Context, st state, text string) error {
	uid := c.Sender().ID

	// لغو همیشه ممکنه
	if text == btnCancel || text == btnBack {
		h.clearState(ctx, uid)
		h.albumClear(ctx, uid)
		return c.Send("لغو شد.", kbAdmin())
	}

	switch st.Step {

	// ── نوع کد ───────────────────────────────────────────
	case stepCodeType:
		ct := parseCodeType(text)
		if ct == "" {
			return c.Send("نوع کد را از دکمه‌های بالا انتخاب کنید.", kbCodeType())
		}
		h.setStep(ctx, uid, stepCodeLimit, "type", string(ct))

		switch ct {
		case documents.CodeLimited:
			return c.Send("تعداد مجاز استفاده را وارد کنید:\nمثال: 5", kbCancel())
		case documents.CodeExpiry:
			return c.Send(
				"مدت اعتبار کد را وارد کنید:\n\n"+
					"مثال: <code>7d</code> (هفت روز)\n"+
					"<code>24h</code> (بیست و چهار ساعت)\n"+
					"<code>30</code> (سی روز)",
				tele.ModeHTML, kbCancel(),
			)
		default:
			// یک‌بار یا نامحدود → برو به انتخاب فایل
			h.setStep(ctx, uid, stepCodeFiles)
			return c.Send(
				"فایل یا فایل‌های مورد نظر را ارسال کنید.\n\n"+
					"برای <b>آلبوم</b>: چند فایل پشت سر هم بفرستید، سپس «تمام» بزنید.\n"+
					"برای <b>فایل تکی</b>: یک فایل بفرستید.",
				tele.ModeHTML, kbAlbumDone(),
			)
		}

	// ── محدودیت تعداد ─────────────────────────────────────
	case stepCodeLimit:
		ct := documents.CodeType(st.Data["type"])
		if ct == documents.CodeExpiry {
			// این مرحله برای expiry date هست
			exp, err := parseDuration(text)
			if err != nil {
				return c.Send("فرمت نامعتبر. مثال: 7d یا 24h یا 30", kbCancel())
			}
			h.setStep(ctx, uid, stepCodeFiles,
				"expiry", exp.Format(time.RFC3339))
		} else {
			// limited count
			n, err := strconv.Atoi(text)
			if err != nil || n <= 0 {
				return c.Send("عدد صحیح مثبت وارد کنید.", kbCancel())
			}
			h.setStep(ctx, uid, stepCodeFiles, "max_use", strconv.Itoa(n))
		}
		return c.Send(
			"فایل یا فایل‌های مورد نظر را ارسال کنید.\n\n"+
				"برای آلبوم: چند فایل پشت سر هم بفرستید، سپس «تمام» بزنید.",
			kbAlbumDone(),
		)

	// ── جمع‌آوری فایل‌ها ──────────────────────────────────
	case stepCodeFiles:
		if text == btnDone {
			ids := h.albumGet(ctx, uid)
			if len(ids) == 0 {
				return c.Send("هیچ فایلی ارسال نشده. ابتدا فایل بفرستید.", kbAlbumDone())
			}
			return h.finishCodeCreation(ctx, c, st, ids)
		}
		// متن در این مرحله معنایی ندارد
		return c.Send("فایل بفرستید یا «تمام» بزنید.", kbAlbumDone())

	// ── تأیید نهایی ───────────────────────────────────────
	case stepCodeConfirm:
		if text == btnConfirm {
			return h.createCodeFromState(ctx, c, st)
		}
		h.clearState(ctx, uid)
		h.albumClear(ctx, uid)
		return c.Send("لغو شد.", kbAdmin())

	// ── تنظیمات ───────────────────────────────────────────
	case stepSettingKey:
		h.setStep(ctx, uid, stepSettingValue, "key", settingKeyFromBtn(text))
		return c.Send("مقدار جدید را وارد کنید:", kbCancel())

	case stepSettingValue:
		key := st.Data["key"]
		if key == "" {
			h.clearState(ctx, uid)
			return c.Send("خطا.", kbAdmin())
		}
		if err := h.eng.Settings.Set(ctx, key, text); err != nil {
			return c.Send("❌ خطا در ذخیره تنظیم.")
		}
		// آپدیت config MongoDB برای hot-reload
		h.eng.Config.Update(ctx, key, text)
		// اطلاع به همه instance های این bot
		configstore.PublishConfigUpdated(h.eng.Nats, h.eng.InstanceID, "uploader")
		h.clearState(ctx, uid)
		return c.Send("✅ تنظیم ذخیره شد.", kbAdmin())

	// ── broadcast ─────────────────────────────────────────
	case stepBroadcast:
		h.clearState(ctx, uid)
		return h.doBroadcast(ctx, c, text)
	}

	return nil
}

// finishCodeCreation پیش‌نمایش کد و درخواست تأیید.
func (h *Handler) finishCodeCreation(ctx context.Context, c tele.Context, st state, fileIDs []string) error {
	uid := c.Sender().ID

	ct := documents.CodeType(st.Data["type"])
	maxUse := 0
	if n, err := strconv.Atoi(st.Data["max_use"]); err == nil {
		maxUse = n
	}

	// ذخیره fileIDs در state
	idsStr := strings.Join(fileIDs, ",")
	h.setStep(ctx, uid, stepCodeConfirm,
		"type", string(ct),
		"max_use", strconv.Itoa(maxUse),
		"expiry", st.Data["expiry"],
		"file_ids", idsStr,
	)
	h.albumClear(ctx, uid)

	limitText := codeTypeLabel(ct)
	if ct == documents.CodeLimited {
		limitText = fmt.Sprintf("محدود — %d بار", maxUse)
	}
	expiryText := "ندارد"
	if st.Data["expiry"] != "" {
		if exp, err := time.Parse(time.RFC3339, st.Data["expiry"]); err == nil {
			expiryText = exp.Format("2006/01/02 15:04")
		}
	}

	return c.Send(
		fmt.Sprintf(
			"<b>پیش‌نمایش کد:</b>\n\n"+
				"📁 تعداد فایل: %d\n"+
				"🔑 نوع: %s\n"+
				"⏰ انقضا: %s\n\n"+
				"تأیید می‌کنید؟",
			len(fileIDs), limitText, expiryText,
		),
		tele.ModeHTML, kbConfirmCancel(),
	)
}

// createCodeFromState کد نهایی را می‌سازد.
func (h *Handler) createCodeFromState(ctx context.Context, c tele.Context, st state) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	ct := documents.CodeType(st.Data["type"])
	maxUse, _ := strconv.Atoi(st.Data["max_use"])
	fileIDs := strings.Split(st.Data["file_ids"], ",")

	var expiresAt *time.Time
	if st.Data["expiry"] != "" {
		if exp, err := time.Parse(time.RFC3339, st.Data["expiry"]); err == nil {
			expiresAt = &exp
		}
	}

	code := &documents.Code{
		Code:      genAlphaCode(),
		Type:      ct,
		MaxUse:    maxUse,
		ExpiresAt: expiresAt,
		IsAlbum:   len(fileIDs) > 1,
		FileIDs:   fileIDs,
	}

	if err := h.codes.Create(ctx, code); err != nil {
		h.log().Error("createCode", ports.F("err", err))
		return c.Send("❌ خطا در ساخت کد.", kbAdmin())
	}

	limitText := codeTypeLabel(ct)
	if ct == documents.CodeLimited {
		limitText = fmt.Sprintf("محدود — %d بار", maxUse)
	}
	expiryText := "ندارد"
	if expiresAt != nil {
		expiryText = expiresAt.Format("2006/01/02 15:04")
	}

	return c.Send(
		fmt.Sprintf(
			"✅ <b>کد ساخته شد</b>\n\n"+
				"🔑 کد: <code>%s</code>\n"+
				"📁 فایل‌ها: %d عدد\n"+
				"📋 نوع: %s\n"+
				"⏰ انقضا: %s\n\n"+
				"این کد را برای کاربر ارسال کنید.",
			code.Code, len(fileIDs), limitText, expiryText,
		),
		tele.ModeHTML, kbAdmin(),
	)
}

// ════════════════════════════════════════════════════════════
// لیست کدها
// ════════════════════════════════════════════════════════════

func (h *Handler) adminCodeList(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()

	codes, err := h.codes.List(ctx, 20)
	if err != nil {
		return c.Send("❌ خطا در دریافت کدها.")
	}
	if len(codes) == 0 {
		return c.Send("هیچ کدی وجود ندارد.", kbAdmin())
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📋 کدها (%d)</b>\n\n", len(codes)))

	for _, code := range codes {
		valid := "✅"
		if !h.codes.IsValid(&code) {
			valid = "❌"
		}
		limitText := codeTypeLabel(code.Type)
		if code.Type == documents.CodeLimited {
			limitText = fmt.Sprintf("%d/%d", code.UsedCount, code.MaxUse)
		} else if code.Type == documents.CodeOnce {
			if code.UsedCount > 0 {
				limitText = "استفاده شده"
			} else {
				limitText = "استفاده نشده"
			}
		}
		expiryText := ""
		if code.ExpiresAt != nil {
			if time.Now().After(*code.ExpiresAt) {
				expiryText = " 🔴منقضی"
			} else {
				expiryText = fmt.Sprintf(" ⏰%s", code.ExpiresAt.Format("01/02"))
			}
		}
		sb.WriteString(fmt.Sprintf(
			"%s <code>%s</code> — %d فایل — %s%s\n",
			valid, code.Code, len(code.FileIDs), limitText, expiryText,
		))
	}

	sb.WriteString("\nبرای حذف: /delcode &lt;code&gt;")

	return c.Send(sb.String(), tele.ModeHTML, kbAdmin())
}

// /delcode <code>
func (h *Handler) adminDelCode(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	codeStr := strings.TrimSpace(c.Message().Payload)
	if codeStr == "" {
		return c.Send("استفاده: /delcode <code>")
	}

	ctx := context.Background()
	code, _ := h.codes.FindByCode(ctx, codeStr)
	if code == nil {
		return c.Send("❌ کد یافت نشد.")
	}

	// inline keyboard برای تأیید حذف
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🗑 حذف شود", "del_code:"+codeStr),
			kb.Data("❌ انصراف", "cancel"),
		),
	)

	return c.Send(
		fmt.Sprintf("کد <code>%s</code> حذف شود؟\n(%d فایل)",
			codeStr, len(code.FileIDs)),
		tele.ModeHTML, kb,
	)
}

// ════════════════════════════════════════════════════════════
// آمار
// ════════════════════════════════════════════════════════════

func (h *Handler) adminStats(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()

	totalUsers, _ := h.users.Count(ctx)
	totalCodes, _ := h.codes.Count(ctx)
	activeCodes, _ := h.codes.CountActive(ctx)
	totalFiles, _ := h.files.Count(ctx)

	return c.Send(
		fmt.Sprintf(
			"<b>📊 آمار ربات</b>\n\n"+
				"👥 کل کاربران: <b>%d</b>\n"+
				"📋 کل کدها: <b>%d</b> (فعال: %d)\n"+
				"📁 کل فایل‌ها: <b>%d</b>\n\n"+
				"⏰ آخرین به‌روزرسانی: %s",
			totalUsers, totalCodes, activeCodes, totalFiles,
			time.Now().Format("15:04:05"),
		),
		tele.ModeHTML, kbAdmin(),
	)
}

// ════════════════════════════════════════════════════════════
// کاربران
// ════════════════════════════════════════════════════════════

func (h *Handler) adminUsers(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()

	users, err := h.users.List(ctx, 30)
	if err != nil {
		return c.Send("❌ خطا.")
	}
	if len(users) == 0 {
		return c.Send("هیچ کاربری ثبت‌نام نکرده.", kbAdmin())
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>👥 کاربران (%d)</b>\n\n", len(users)))
	for _, u := range users {
		blocked := ""
		if u.IsBlocked {
			blocked = " 🚫"
		}
		uname := ""
		if u.Username != "" {
			uname = " @" + u.Username
		}
		sb.WriteString(fmt.Sprintf("• <b>%s</b>%s%s — <code>%d</code>\n",
			u.FirstName, uname, blocked, u.TelegramID))
	}
	sb.WriteString("\n/block &lt;tid&gt; — بلاک\n/unblock &lt;tid&gt; — آنبلاک")

	return c.Send(sb.String(), tele.ModeHTML, kbAdmin())
}

// /block <telegram_id>
func (h *Handler) adminBlock(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	var tid int64
	if _, err := fmt.Sscanf(c.Message().Payload, "%d", &tid); err != nil || tid == 0 {
		return c.Send("استفاده: /block <telegram_id>")
	}
	ctx := context.Background()
	if err := h.users.SetBlocked(ctx, tid, true); err != nil {
		return c.Send("❌ خطا.")
	}
	return c.Send(fmt.Sprintf("🚫 کاربر <code>%d</code> بلاک شد.", tid), tele.ModeHTML, kbAdmin())
}

// /unblock <telegram_id>
func (h *Handler) adminUnblock(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	var tid int64
	if _, err := fmt.Sscanf(c.Message().Payload, "%d", &tid); err != nil || tid == 0 {
		return c.Send("استفاده: /unblock <telegram_id>")
	}
	ctx := context.Background()
	if err := h.users.SetBlocked(ctx, tid, false); err != nil {
		return c.Send("❌ خطا.")
	}
	return c.Send(fmt.Sprintf("✅ کاربر <code>%d</code> آنبلاک شد.", tid), tele.ModeHTML, kbAdmin())
}

// ════════════════════════════════════════════════════════════
// تنظیمات
// ════════════════════════════════════════════════════════════

func (h *Handler) adminSettings(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	uid := c.Sender().ID

	welcome, _ := h.eng.Settings.Get(ctx, "welcome_text")
	notMember, _ := h.eng.Settings.Get(ctx, "not_member_text")
	channel, _ := h.eng.Settings.Get(ctx, "channel_username")

	if welcome == "" {
		welcome = "(پیش‌فرض)"
	}
	if notMember == "" {
		notMember = "(پیش‌فرض)"
	}
	if channel == "" {
		channel = "(تنظیم نشده)"
	}

	text := fmt.Sprintf(
		"<b>⚙️ تنظیمات</b>\n\n"+
			"👋 متن خوشامد:\n%s\n\n"+
			"🚫 متن عدم عضویت:\n%s\n\n"+
			"📢 کانال اجباری: %s\n\n"+
			"یکی از دکمه‌های زیر را انتخاب کنید:",
		welcome, notMember, channel,
	)

	h.setStep(ctx, uid, stepSettingKey)
	return c.Send(text, tele.ModeHTML, kbSettings())
}

func settingKeyFromBtn(text string) string {
	m := map[string]string{
		btnSetWelcome:   "welcome_text",
		btnSetNotMember: "not_member_text",
		btnSetNotFound:  "not_found_text",
		btnSetBlocked:   "blocked_text",
		btnSetChannel:   "channel_username",
	}
	return m[text]
}

// ════════════════════════════════════════════════════════════
// پیام همگانی
// ════════════════════════════════════════════════════════════

func (h *Handler) adminBroadcast(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepBroadcast)
	return c.Send("پیام مورد نظر را ارسال کنید:\n(متن، عکس، فایل — هر نوع پیام)", kbCancel())
}

func (h *Handler) doBroadcast(ctx context.Context, c tele.Context, text string) error {
	users, err := h.users.List(ctx, 0) // همه کاربران
	if err != nil {
		return c.Send("❌ خطا در دریافت کاربران.")
	}

	sent, failed := 0, 0
	for _, u := range users {
		if u.IsBlocked {
			continue
		}
		if err := h.sender.Send(ctx, u.TelegramID, text); err != nil {
			failed++
		} else {
			sent++
		}
	}

	return c.Send(
		fmt.Sprintf("📣 پیام همگانی ارسال شد.\n\n✅ موفق: %d\n❌ ناموفق: %d", sent, failed),
		kbAdmin(),
	)
}
