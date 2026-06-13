package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// onText routing اصلی — کد دریافتی یا state فعال.
func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	// ── state فعال ────────────────────────────────────────
	st := h.getState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	// ── دکمه‌های ادمین ────────────────────────────────────
	if h.isAdmin(c) {
		switch text {
		case btnNewCode:
			return h.adminNewCode(c)
		case btnCodeList:
			return h.adminCodeList(c)
		case btnUsers:
			return h.adminUsers(c)
		case btnStats:
			return h.adminStats(c)
		case btnSettings:
			return h.adminSettings(c)
		case btnBroadcast:
			return h.adminBroadcast(c)
		case "❓ راهنما":
			return h.onHelp(c)
		case btnCancel, btnBack:
			h.clearState(ctx, uid)
			h.albumClear(ctx, uid)
			return c.Send("لغو شد.", kbAdmin())
		}
	} else {
		switch text {
		case "❓ راهنما":
			return h.onHelp(c)
		case btnCancel:
			return c.Send("لغو شد.", kbUser())
		}
	}

	// ── کاربر: جستجوی کد ─────────────────────────────────
	if !h.isAdmin(c) {
		return h.userReceiveCode(ctx, c, text)
	}

	return nil
}

// userReceiveCode کد دریافتی را پردازش و فایل‌ها را ارسال می‌کند.
func (h *Handler) userReceiveCode(ctx context.Context, c tele.Context, codeStr string) error {
	uid := c.Sender().ID

	// بررسی بلاک
	user, _ := h.users.FindByTelegramID(ctx, uid)
	if user != nil && user.IsBlocked {
		return c.Send(h.setting(ctx, "blocked_text", "⛔️ دسترسی شما محدود شده است."))
	}

	// بررسی عضویت
	if err := h.checkMembership(ctx, c); err != nil {
		return err
	}

	// پیدا کردن کد
	code, err := h.codes.FindByCode(ctx, codeStr)
	if err != nil {
		h.log().Error("userReceiveCode: find", ports.F("err", err))
		return c.Send("❌ خطای سرور. لطفاً دوباره تلاش کنید.")
	}
	if code == nil || !h.codes.IsValid(code) {
		notFound := h.setting(ctx, "not_found_text", "❌ کد یافت نشد یا منقضی شده است.")
		return c.Send(notFound)
	}

	// دریافت فایل‌ها
	files, err := h.files.FindByIDs(ctx, code.FileIDs)
	if err != nil || len(files) == 0 {
		return c.Send("❌ فایل‌های این کد یافت نشد.")
	}

	// ارسال فایل‌ها
	for _, f := range files {
		if err := sendFile(c, f); err != nil {
			h.log().Error("userReceiveCode: send",
				ports.F("file", f.ID.Hex()), ports.F("err", err))
		}
	}

	// ثبت استفاده و آمار
	h.codes.IncrementUse(ctx, code.ID.Hex())
	h.eng.Stats.IncrementDaily(ctx, "total_uses", 1)
	h.eng.Stats.IncrementDaily(ctx, "unique_users", 1)

	return nil
}

// onMedia فایل ارسالی از ادمین را ذخیره می‌کند.
func (h *Handler) onMedia(c tele.Context) error {
	if !h.isAdmin(c) {
		// کاربر عادی — فایل قبول نمی‌کنیم
		return nil
	}
	ctx := context.Background()
	uid := c.Sender().ID

	fi := extractFile(c)
	if fi == nil {
		return nil
	}

	st := h.getState(ctx, uid)

	// اگه در حالت آلبوم هستیم → اضافه کن
	if st.Step == stepCodeFiles {
		ids := h.albumAdd(ctx, uid, fi.ID)
		return c.Send(
			"✅ فایل اضافه شد. مجموع: "+string(rune('0'+len(ids)))+"\nبرای پایان «تمام» بزنید.",
			kbAlbumDone(),
		)
	}

	// حالت عادی → ذخیره فایل و نمایش ID
	fileOID := h.files.CreateFromInfo(ctx, fi.ID, fi.Type, fi.Caption, uid)
	if fileOID == nil {
		return c.Send("❌ خطا در ذخیره فایل.")
	}

	return c.Send(
		"✅ فایل ذخیره شد.\n"+
			fileTypeIcon(fi.Type)+" نوع: "+fi.Type+"\n"+
			"🆔 ID: <code>"+fileOID.Hex()+"</code>\n\n"+
			"برای ساخت کد: /newcode",
		tele.ModeHTML, kbAdmin(),
	)
}

// onCallback callback های inline keyboard.
func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	data := strings.TrimPrefix(c.Callback().Data, "\f")
	defer c.Respond()

	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 2 {
		return nil
	}

	switch parts[0] {
	case "del_code":
		if len(parts) < 2 {
			return nil
		}
		code := parts[1]
		if err := h.codes.DeleteByCode(ctx, code); err != nil {
			return c.Edit("❌ خطا در حذف کد.")
		}
		return c.Edit("🗑 کد <code>" + code + "</code> حذف شد.")
	}
	return nil
}
