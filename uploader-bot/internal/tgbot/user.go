package tgbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ── /start ────────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	// ثبت یا بروزرسانی کاربر
	user, _ := h.store.GetOrCreateUser(ctx, uid,
		c.Sender().Username, c.Sender().FirstName)
	if user == nil {
		return c.Send("❌ خطای سرور.")
	}
	if user.IsBlocked {
		return c.Send(h.getSetting(ctx, models.SettingBotActive, "⛔️ دسترسی شما محدود شده است."))
	}

	// ربات خاموش؟
	if h.getSetting(ctx, models.SettingBotActive, "true") == "false" && !h.isAdmin(c) {
		return c.Send("⚠️ ربات موقتاً غیرفعال است. لطفاً بعداً تلاش کنید.")
	}

	// deep link — کد رسانه در start
	args := c.Message().Payload
	if args != "" {
		return h.deliverCode(ctx, c, user, args)
	}

	// منوی اصلی
	if h.isAdmin(c) {
		return c.Send(h.getSetting(ctx, "welcome_text_admin", "👑 پنل مدیریت"), kbAdmin())
	}
	return c.Send(h.getSetting(ctx, "welcome_text",
		"👋 خوش آمدید!\n\nکد رسانه را ارسال کنید:"), kbUser())
}

// ── onText ────────────────────────────────────────────────────

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	st := h.getState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	if h.isAdmin(c) {
		return h.routeAdmin(ctx, c, text)
	}
	return h.routeUser(ctx, c, text)
}

func (h *Handler) routeAdmin(ctx context.Context, c tele.Context, text string) error {
	switch text {
	case btnNewCode:
		return h.adminNewCode(c)
	case btnCodeList:
		return h.adminCodeList(c)
	case btnFolders:
		return h.adminFolderList(c)
	case btnUsers:
		return h.adminUsers(c)
	case btnStats:
		return h.adminStats(c)
	case btnSettings:
		return h.adminSettings(c)
	case btnBroadcast:
		return h.adminBroadcastStart(c)
	case btnBackup:
		return h.adminBackup(c)
	case btnChannels:
		return h.adminChannelList(c)
	case btnSubPlans:
		return h.adminSubPlans(c)
	case btnAdmins:
		return h.adminAdminList(c)
	case btnCancel, btnBack:
		h.clearState(ctx, c.Sender().ID)
		return c.Send("لغو شد.", kbAdmin())
	}
	return nil
}

func (h *Handler) routeUser(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID

	// بررسی بلاک
	user, _ := h.store.GetUser(ctx, uid)
	if user != nil && user.IsBlocked {
		return c.Send(h.getSetting(ctx, "blocked_text", "⛔️ دسترسی شما محدود شده است."))
	}

	switch text {
	case btnSearch:
		h.setStep(ctx, uid, stepSearch)
		return c.Send("🔍 متن جستجو را وارد کنید:", kbCancelOnly())
	case btnHelp:
		return c.Send(h.getSetting(ctx, "help_text", "❓ راهنما:\n\nکد رسانه را وارد کنید."))
	case btnSupport:
		return c.Send(h.getSetting(ctx, "support_text", "💬 پشتیبانی: @support"))
	case btnCancel:
		h.clearState(ctx, uid)
		return c.Send("لغو شد.", kbUser())
	default:
		// کد رسانه
		if user != nil {
			return h.deliverCode(ctx, c, user, text)
		}
	}
	return nil
}

// ── deliverCode — ارسال فایل‌ها ────────────────────────────

func (h *Handler) deliverCode(ctx context.Context, c tele.Context, user *models.User, codeStr string) error {
	uid := c.Sender().ID

	code, err := h.store.FindCode(ctx, codeStr)
	if err != nil {
		h.log.Error("deliverCode", ports.F("err", err))
		return c.Send("❌ خطای سرور.")
	}
	if code == nil {
		return c.Send(h.getSetting(ctx, "not_found_text", "❌ کد یافت نشد."))
	}

	// بررسی انقضا
	if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
		return c.Send("⏰ این رسانه منقضی شده است.")
	}

	// بررسی جوین اجباری
	if code.ChannelLock {
		if notJoined, err := h.checkMembership(ctx, c); err != nil || len(notJoined) > 0 {
			return h.sendJoinRequired(c, notJoined)
		}
	}

	// بررسی اشتراک
	if code.SubRequired || h.getSetting(ctx, models.SettingSubRequired, "false") == "true" {
		if !user.HasActiveSub() {
			freeLimit := h.getSettingInt(ctx, models.SettingFreeDownloads, 3)
			if user.FreeDownloads >= freeLimit {
				return h.sendSubRequired(c)
			}
		}
	}

	// بررسی محدودیت دانلود per code
	if code.DownloadLimit > 0 {
		count := h.store.GetDownloadCount(ctx, user.ID, code.ID)
		if count >= code.DownloadLimit {
			return c.Send("❌ شما به حداکثر دانلود این رسانه رسیده‌اید.")
		}
	}

	// رمز عبور
	if code.Password != "" {
		st := h.getState(ctx, uid)
		if st.Data["verified_code"] != code.Code {
			h.setStepData(ctx, uid, stepPassword, "code", code.Code)
			return c.Send("🔐 رمز عبور رسانه را وارد کنید:", kbCancelOnly())
		}
	}

	// ارسال فایل‌ها
	return h.sendCodeFiles(ctx, c, user, code)
}

func (h *Handler) sendCodeFiles(ctx context.Context, c tele.Context, user *models.User, code *models.Code) error {
	files, err := h.store.GetFilesForCode(ctx, code.ID)
	if err != nil || len(files) == 0 {
		return c.Send("❌ فایل‌های این کد یافت نشد.")
	}

	// امضا
	sig := h.getSetting(ctx, models.SettingSignature, "")

	autoDelete := code.AutoDelete
	if autoDelete == 0 {
		autoDelete = h.getSettingInt(ctx, models.SettingAutoDeleteDefault, 0)
	}

	var sentMsgIDs []int

	if code.IsAlbum && len(files) > 1 {
		// ارسال به صورت آلبوم
		var album tele.Album
		for _, f := range files {
			media := fileToMedia(f, sig)
			if media != nil {
				album = append(album, media)
			}
		}
		if len(album) > 0 {
			msgs, err := c.Bot().SendAlbum(c.Recipient(), album,
				tele.Silent, noForwardOpt(code.ForwardLock))
			if err == nil {
				for _, m := range msgs {
					sentMsgIDs = append(sentMsgIDs, m.ID)
				}
			}
		}
	} else {
		// ارسال تکی
		for _, f := range files {
			msg, err := sendFileWithSig(c, f, sig, code.ForwardLock)
			if err != nil {
				h.log.Error("send file", ports.F("file", f.ID), ports.F("err", err))
				continue
			}
			if msg != nil {
				sentMsgIDs = append(sentMsgIDs, msg.ID)
			}
		}
	}

	// تایمر حذف خودکار
	if autoDelete > 0 && len(sentMsgIDs) > 0 {
		go func() {
			time.Sleep(time.Duration(autoDelete) * time.Second)
			for _, msgID := range sentMsgIDs {
				c.Bot().Delete(&tele.Message{
					ID:   msgID,
					Chat: c.Chat(),
				})
			}
		}()
		c.Send(fmt.Sprintf("⏱ این رسانه بعد از %d ثانیه حذف می‌شود.", autoDelete))
	}

	// لاگ دانلود
	h.store.LogDownload(ctx, user.ID, code.ID)
	h.store.IncrementCodeUse(ctx, code.ID)

	// آپدیت free downloads
	if !user.HasActiveSub() {
		user.FreeDownloads++
		h.store.UpdateUser(ctx, user)
	}

	return nil
}

// ── Force Join ────────────────────────────────────────────────

func (h *Handler) checkMembership(ctx context.Context, c tele.Context) ([]models.ForceJoinChannel, error) {
	channels, _ := h.store.ListForceJoinChannels(ctx)
	var notJoined []models.ForceJoinChannel

	for _, ch := range channels {
		member, err := c.Bot().ChatMemberOf(
			&tele.Chat{ID: ch.ChatID},
			c.Sender(),
		)
		if err != nil || member == nil ||
			member.Role == tele.Left || member.Role == tele.Kicked {
			notJoined = append(notJoined, ch)
		}
	}
	return notJoined, nil
}

func (h *Handler) sendJoinRequired(c tele.Context, channels []models.ForceJoinChannel) error {
	text := h.getSetting(context.Background(), "not_member_text",
		"⚠️ برای دسترسی به این رسانه باید در کانال‌های زیر عضو شوید:")

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ch := range channels {
		url := ch.InviteURL
		if ch.Username != "" {
			url = "https://t.me/" + ch.Username
		}
		rows = append(rows, kb.Row(kb.URL("📢 "+ch.Title, url)))
	}
	rows = append(rows, kb.Row(kb.Data("✅ عضو شدم", "check_join")))
	kb.Inline(rows...)

	return c.Send(text, kb)
}

func (h *Handler) sendSubRequired(c tele.Context) error {
	ctx := context.Background()
	plans, _ := h.store.ListSubPlans(ctx)

	text := h.getSetting(ctx, models.SettingSubRequiredText,
		"💎 برای دسترسی به این رسانه باید اشتراک تهیه کنید:")

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("💎 %s — %.0f تومان (%d روز)", p.Name, p.Price, p.Days)
		rows = append(rows, kb.Row(kb.Data(label, fmt.Sprintf("buy_plan:%s", p.ID))))
	}
	kb.Inline(rows...)
	return c.Send(text, kb)
}

// ── Inline Query ─────────────────────────────────────────────

func (h *Handler) onInlineQuery(c tele.Context) error {
	ctx := context.Background()
	query := strings.TrimSpace(c.Query().Text)

	// بررسی جستجو فعال است؟
	if h.getSetting(ctx, models.SettingShowSearch, "true") != "true" {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	if len(query) < 2 {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	codes, err := h.store.SearchCodes(ctx, query)
	if err != nil || len(codes) == 0 {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	var results tele.Results
	for i, code := range codes {
		desc := fmt.Sprintf("کد: %s | %d فایل", code.Code, len(code.Files))
		r := &tele.ArticleResult{
			Title:       code.Caption,
			Description: desc,
			Text:        code.Code,
		}
		r.SetResultID(fmt.Sprintf("%d", i))
		results = append(results, r)
	}

	return c.Answer(&tele.QueryResponse{
		Results:   results,
		CacheTime: 30,
	})
}

// ── onCallback ────────────────────────────────────────────────

func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	data := strings.TrimPrefix(c.Callback().Data, "\f")
	defer c.Respond()

	parts := strings.SplitN(data, ":", 2)
	action := parts[0]
	arg := ""
	if len(parts) == 2 {
		arg = parts[1]
	}

	switch action {

	// ── کاربر ────────────────────────────────────────────
	case "check_join":
		user, _ := h.store.GetUser(ctx, c.Sender().ID)
		if user == nil {
			return c.Edit("❌ خطا.")
		}
		notJoined, _ := h.checkMembership(ctx, c)
		if len(notJoined) > 0 {
			return c.Edit("❌ هنوز در همه کانال‌ها عضو نشده‌اید.")
		}
		return c.Edit("✅ عضویت تأیید شد. کد رسانه را مجدداً ارسال کنید.")

	case "buy_plan":
		return h.userBuyPlan(ctx, c, arg)

	// ── کد رسانه ─────────────────────────────────────────
	case "code_settings":
		return h.adminCodeSettings(ctx, c, arg)
	case "code_delete":
		return h.adminCodeDelete(ctx, c, arg)
	case "code_edit_caption":
		h.setStepData(ctx, c.Sender().ID, stepEditCaption, "code_id", arg)
		return c.Edit("✏️ کپشن جدید را وارد کنید:")
	case "code_toggle_forward":
		return h.adminToggleForward(ctx, c, arg)
	case "code_toggle_antidl":
		return h.adminToggleAntiDelete(ctx, c, arg)
	case "code_set_password":
		h.setStepData(ctx, c.Sender().ID, stepSetPassword, "code_id", arg)
		return c.Edit("🔐 رمز عبور جدید را وارد کنید (یا 0 برای حذف):")
	case "code_set_limit":
		h.setStepData(ctx, c.Sender().ID, stepSetLimit, "code_id", arg)
		return c.Edit("🔢 حداکثر تعداد دانلود را وارد کنید (0=نامحدود):")
	case "code_send_preview":
		return h.adminSendPreview(ctx, c, arg)

	// ── پوشه ─────────────────────────────────────────────
	case "folder_list":
		return h.adminFolderList(c)
	case "folder_new":
		h.setStep(ctx, c.Sender().ID, stepNewFolder)
		return c.Edit("📁 نام پوشه جدید را وارد کنید:")
	case "folder_delete":
		return h.adminFolderDelete(ctx, c, arg)

	// ── کانال جوین اجباری ────────────────────────────────
	case "channel_delete":
		return h.adminChannelDelete(ctx, c, arg)

	// ── اشتراک ───────────────────────────────────────────
	case "plan_delete":
		return h.adminPlanDelete(ctx, c, arg)

	// ── تنظیمات ──────────────────────────────────────────
	case "toggle_setting":
		return h.adminToggleSetting(ctx, c, arg)
	case "set_setting":
		h.setStepData(ctx, c.Sender().ID, stepEditSetting, "key", arg)
		return c.Edit("✏️ مقدار جدید را وارد کنید:")
	}
	return nil
}

// ── handleStep ────────────────────────────────────────────────

func (h *Handler) handleStep(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID

	// لغو
	if text == btnCancel || text == btnBack {
		h.clearState(ctx, uid)
		kb := kbUser()
		if h.isAdmin(c) {
			kb = kbAdmin()
		}
		return c.Send("لغو شد.", kb)
	}

	switch st.Step {

	// ── رمز عبور ─────────────────────────────────────────
	case stepPassword:
		codeStr := st.Data["code"]
		code, _ := h.store.FindCode(ctx, codeStr)
		if code == nil {
			h.clearState(ctx, uid)
			return c.Send("❌ کد منقضی شده.")
		}
		if text != code.Password {
			return c.Send("❌ رمز عبور اشتباه است.")
		}
		// تأیید و ارسال
		h.setStepData(ctx, uid, stepIdle, "verified_code", codeStr)
		user, _ := h.store.GetUser(ctx, uid)
		if user == nil {
			h.clearState(ctx, uid)
			return c.Send("❌ خطا.")
		}
		h.clearState(ctx, uid)
		return h.sendCodeFiles(ctx, c, user, code)

	// ── جستجو ────────────────────────────────────────────
	case stepSearch:
		h.clearState(ctx, uid)
		codes, _ := h.store.SearchCodes(ctx, text)
		if len(codes) == 0 {
			return c.Send("❌ نتیجه‌ای یافت نشد.", kbUser())
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🔍 %d نتیجه:\n\n", len(codes)))
		for _, code := range codes {
			sb.WriteString(fmt.Sprintf("📦 <code>%s</code>", code.Code))
			if code.Caption != "" {
				sb.WriteString(" — " + code.Caption[:min(50, len(code.Caption))])
			}
			sb.WriteString("\n")
		}
		return c.Send(sb.String(), tele.ModeHTML, kbUser())

	// ── ادمین steps ──────────────────────────────────────
	case stepNewFolder:
		return h.adminNewFolderSave(ctx, c, text)
	case stepEditCaption:
		return h.adminEditCaptionSave(ctx, c, st.Data["code_id"], text)
	case stepSetPassword:
		return h.adminSetPasswordSave(ctx, c, st.Data["code_id"], text)
	case stepSetLimit:
		return h.adminSetLimitSave(ctx, c, st.Data["code_id"], text)
	case stepEditSetting:
		return h.adminSetSettingSave(ctx, c, st.Data["key"], text)
	case stepAddChannel:
		return h.adminAddChannelSave(ctx, c, text)
	case stepNewPlan:
		return h.adminNewPlanStep(ctx, c, st, text)
	case stepBroadcast:
		return h.adminBroadcastSend(ctx, c, text)
	case stepAddAdmin:
		return h.adminAddAdminSave(ctx, c, text)
	case stepSearchUser:
		return h.adminSearchUserResult(ctx, c, text)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────

func (h *Handler) getSetting(ctx context.Context, key, defaultVal string) string {
	val := h.store.GetSetting(ctx, key)
	if val == "" {
		return defaultVal
	}
	return val
}

func (h *Handler) getSettingInt(ctx context.Context, key string, defaultVal int) int {
	val := h.getSetting(ctx, key, "")
	if val == "" {
		return defaultVal
	}
	var n int
	fmt.Sscan(val, &n)
	return n
}

func noForwardOpt(noForward bool) tele.Option {
	if noForward {
		return tele.Silent
	}
	return nil
}

func fileToMedia(f models.File, sig string) tele.InputMedia {
	caption := f.Caption
	if sig != "" {
		caption += "\n" + sig
	}
	switch f.FileType {
	case "video":
		v := &tele.Video{File: tele.File{FileID: f.FileID}, Caption: caption}
		if f.Thumbnail != "" {
			v.Thumbnail = &tele.Photo{File: tele.File{FileID: f.Thumbnail}}
		}
		return v
	case "photo":
		return &tele.Photo{File: tele.File{FileID: f.FileID}, Caption: caption}
	case "audio":
		return &tele.Audio{File: tele.File{FileID: f.FileID}, Caption: caption}
	default:
		return &tele.Document{File: tele.File{FileID: f.FileID}, Caption: caption}
	}
}

func sendFileWithSig(c tele.Context, f models.File, sig string, noFwd bool) (*tele.Message, error) {
	caption := f.Caption
	if sig != "" {
		caption += "\n" + sig
	}
	opts := []any{tele.ModeHTML}
	if noFwd {
		opts = append(opts, tele.Silent)
	}

	switch f.FileType {
	case "video":
		v := &tele.Video{File: tele.File{FileID: f.FileID}, Caption: caption}
		if f.Thumbnail != "" {
			v.Thumbnail = &tele.Photo{File: tele.File{FileID: f.Thumbnail}}
		}
		return c.Bot().Send(c.Recipient(), v, opts...)
	case "photo":
		return c.Bot().Send(c.Recipient(),
			&tele.Photo{File: tele.File{FileID: f.FileID}, Caption: caption}, opts...)
	case "audio":
		return c.Bot().Send(c.Recipient(),
			&tele.Audio{File: tele.File{FileID: f.FileID}, Caption: caption}, opts...)
	case "animation":
		return c.Bot().Send(c.Recipient(),
			&tele.Animation{File: tele.File{FileID: f.FileID}, Caption: caption}, opts...)
	case "voice":
		return c.Bot().Send(c.Recipient(),
			&tele.Voice{File: tele.File{FileID: f.FileID}, Caption: caption}, opts...)
	default:
		return c.Bot().Send(c.Recipient(),
			&tele.Document{File: tele.File{FileID: f.FileID}, Caption: caption}, opts...)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
