// Package tgbot — uploader-bot handler.
// ادمین: آپلود، کد، پوشه، اشتراک، تنظیمات، آمار، بکاپ
// کاربر: دریافت فایل با کد، جوین اجباری، اشتراک، جستجو
package tgbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/store"
)

// Handler اصلی.
type Handler struct {
	store      *store.Store
	bot        *tele.Bot
	log        ports.Logger
	ownerID    int64
	instanceID string // برای cache key
	cache      ports.Cache
}

func New(st *store.Store, bot *tele.Bot, cache ports.Cache, log ports.Logger, ownerID int64, instanceID string) *Handler {
	return &Handler{
		store:      st,
		bot:        bot,
		cache:      cache,
		log:        log,
		ownerID:    ownerID,
		instanceID: instanceID,
	}
}

// Register همه handler ها را ثبت می‌کند.
func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", h.onStart)
	b.Handle(tele.OnText, h.onText)
	b.Handle(tele.OnCallback, h.onCallback)
	b.Handle(tele.OnPhoto, h.onMedia)
	b.Handle(tele.OnVideo, h.onMedia)
	b.Handle(tele.OnDocument, h.onMedia)
	b.Handle(tele.OnAudio, h.onMedia)
	b.Handle(tele.OnAnimation, h.onMedia)
	b.Handle(tele.OnVoice, h.onMedia)
	b.Handle(tele.OnSticker, h.onMedia)
	b.Handle(tele.OnQuery, h.onInlineQuery)
}

// ── /start ────────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	// بررسی bot active
	if h.store.GetSetting(ctx, models.SettingBotActive) == "false" && !h.isAdmin(c) {
		return c.Send("ربات در حال حاضر در دسترس نیست.")
	}

	// ثبت/آپدیت کاربر
	user, _ := h.store.GetOrCreateUser(ctx, uid,
		c.Sender().Username, c.Sender().FirstName)

	if user != nil && user.IsBlocked {
		return c.Send("⛔️ دسترسی شما محدود شده است.")
	}

	// بررسی deep link — /start CODE
	args := c.Message().Payload
	if args != "" {
		return h.userDeliverCode(ctx, c, user, args)
	}

	// منو اصلی
	if h.isAdmin(c) {
		welcome := h.store.GetSetting(ctx, "admin_welcome")
		if welcome == "" {
			welcome = fmt.Sprintf("👑 پنل مدیریت\n\nربات: @%s", c.Bot().Me.Username)
		}
		return c.Send(welcome, kbAdmin())
	}

	welcome := h.store.GetSetting(ctx, models.SettingWelcomeText)
	if welcome == "" {
		welcome = fmt.Sprintf("👋 سلام %s!\n\nکد رسانه خود را ارسال کنید.", c.Sender().FirstName)
	}
	return c.Send(welcome, tele.ModeHTML, kbUser(h.showSearch(ctx)))
}

// ── onText ────────────────────────────────────────────────────

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	// state فعال
	st := h.getState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	// cancel
	if text == btnCancel || text == btnBack {
		h.clearState(ctx, uid)
		if h.isAdmin(c) {
			return c.Send("لغو شد.", kbAdmin())
		}
		return c.Send("لغو شد.", kbUser(h.showSearch(ctx)))
	}

	// ── ادمین routing ────────────────────────────────────────
	if h.isAdmin(c) {
		return h.adminOnText(ctx, c, text)
	}

	// ── کاربر: دریافت کد ────────────────────────────────────
	user, _ := h.store.GetUser(ctx, uid)
	return h.userDeliverCode(ctx, c, user, text)
}

// ── onMedia ───────────────────────────────────────────────────

func (h *Handler) onMedia(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	if !h.isAdmin(c) {
		// بررسی user upload
		if h.store.GetSetting(ctx, models.SettingUserUpload) != "true" {
			return nil
		}
		return h.userUploadMedia(ctx, c)
	}

	return h.adminHandleMedia(ctx, c, uid)
}

// ── onCallback ────────────────────────────────────────────────

func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	data := strings.TrimPrefix(c.Callback().Data, "\f")
	defer c.Respond()

	parts := strings.SplitN(data, ":", 3)
	action := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}
	arg2 := ""
	if len(parts) > 2 {
		arg2 = parts[2]
	}

	switch action {
	// ── ادمین ────────────────────────────────────────────────
	case "admin_code_del":
		return h.adminDeleteCode(ctx, c, arg)
	case "admin_code_edit":
		return h.adminEditCodeMenu(ctx, c, arg)
	case "admin_code_set_forward":
		return h.adminToggleCodeProp(ctx, c, arg, "forward_lock")
	case "admin_code_set_delete":
		return h.adminSetAutoDelete(ctx, c, arg, arg2)
	case "admin_code_set_sub":
		return h.adminToggleCodeProp(ctx, c, arg, "sub_required")
	case "admin_code_set_channel":
		return h.adminToggleCodeProp(ctx, c, arg, "channel_lock")
	case "admin_folder_open":
		return h.adminFolderOpen(ctx, c, arg)
	case "admin_folder_del":
		return h.adminFolderDelete(ctx, c, arg)
	case "admin_sub_del":
		return h.adminSubPlanDelete(ctx, c, arg)
	case "admin_ch_del":
		return h.adminForceJoinDelete(ctx, c, arg)
	case "admin_backup_restore":
		return h.adminBackupRestore(ctx, c, arg)
	case "admin_user_block":
		return h.adminToggleBlock(ctx, c, arg, true)
	case "admin_user_unblock":
		return h.adminToggleBlock(ctx, c, arg, false)
	case "admin_pay_confirm":
		return h.adminConfirmPayment(ctx, c, arg)
	case "admin_pay_reject":
		return h.adminRejectPayment(ctx, c, arg)

	// ── کاربر ─────────────────────────────────────────────────
	case "sub_buy":
		return h.userBuySubPlan(ctx, c, arg)
	case "sub_pay":
		return h.userPaySub(ctx, c, arg, arg2) // planID:gateway
	case "folder_open":
		return h.userOpenFolder(ctx, c, arg)
	case "code_resend":
		user, _ := h.store.GetUser(ctx, c.Sender().ID)
		return h.userDeliverCode(ctx, c, user, arg)
	}

	return nil
}

// ── Inline Query ──────────────────────────────────────────────

func (h *Handler) onInlineQuery(c tele.Context) error {
	ctx := context.Background()
	query := strings.TrimSpace(c.Query().Text)

	if h.store.GetSetting(ctx, models.SettingShowSearch) != "true" {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	if len(query) < 2 {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	codes, _ := h.store.SearchCodes(ctx, query)
	var results tele.Results
	for i, code := range codes {
		title := code.Code
		desc := code.Caption
		if len(desc) > 80 {
			desc = desc[:80]
		}
		r := &tele.ArticleResult{
			Title:       "📦 " + title,
			Description: desc,
			Text:        code.Code,
		}
		r.SetResultID(fmt.Sprintf("%d", i))
		results = append(results, r)
	}

	return c.Answer(&tele.QueryResponse{
		Results:   results,
		CacheTime: 10,
	})
}

// ── User Deliver Code ─────────────────────────────────────────

func (h *Handler) userDeliverCode(ctx context.Context, c tele.Context, user *models.User, codeStr string) error {
	uid := c.Sender().ID

	// ثبت کاربر
	if user == nil {
		user, _ = h.store.GetOrCreateUser(ctx, uid, c.Sender().Username, c.Sender().FirstName)
	}
	if user != nil && user.IsBlocked {
		return c.Send("⛔️ دسترسی محدود شده است.")
	}

	// پیدا کردن کد
	code, _ := h.store.FindCode(ctx, codeStr)
	if code == nil {
		return c.Send(h.store.GetSetting(ctx, "not_found_text") + "❌ کد یافت نشد.")
	}

	// انقضا
	if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
		return c.Send("⏰ این کد منقضی شده است.")
	}

	// محدودیت استفاده
	if code.Type == models.CodeLimited && code.UsedCount >= code.MaxUse {
		return c.Send("⚠️ ظرفیت این کد تکمیل شده است.")
	}

	// جوین اجباری
	if code.ChannelLock || h.store.GetSetting(ctx, models.SettingBotActive) != "" {
		if notJoined := h.checkForceJoin(ctx, c); len(notJoined) > 0 {
			return h.sendJoinRequest(c, notJoined)
		}
	}

	// رمز عبور
	if code.Password != "" {
		st := h.getState(ctx, uid)
		if st.Data["pwd_verified"] != code.Code {
			h.setStep(ctx, uid, stepPassword, "code", codeStr)
			return c.Send(h.store.GetSetting(ctx, models.SettingPasswordText) + "🔐 رمز عبور را وارد کنید:")
		}
	}

	// اشتراک
	if code.SubRequired || h.store.GetSetting(ctx, models.SettingSubRequired) == "true" {
		if user == nil || !user.HasActiveSub() {
			// بررسی دانلود رایگان
			freeLimit := 0
			fmt.Sscan(h.store.GetSetting(ctx, models.SettingFreeDownloads), &freeLimit)
			if freeLimit > 0 && user != nil && user.FreeDownloads < freeLimit {
				// هنوز دانلود رایگان دارد
				h.store.UpdateUser(ctx, &models.User{
					Base:          user.Base,
					FreeDownloads: user.FreeDownloads + 1,
				})
			} else {
				return h.sendSubRequired(ctx, c)
			}
		}
	}

	// محدودیت دانلود per user
	if code.DownloadLimit > 0 && user != nil {
		count := h.store.GetDownloadCount(ctx, user.ID, code.ID)
		if count >= code.DownloadLimit {
			return c.Send("⚠️ محدودیت دانلود شما برای این رسانه تکمیل شده است.")
		}
	}

	// ارسال فایل‌ها
	files, _ := h.store.GetFilesForCode(ctx, code.ID)
	if len(files) == 0 {
		return c.Send("❌ فایلی یافت نشد.")
	}

	// signature
	sig := h.store.GetSetting(ctx, models.SettingSignature)

	msgIDs := h.sendFiles(ctx, c, code, files, sig)

	// ثبت دانلود
	if user != nil {
		h.store.LogDownload(ctx, user.ID, code.ID)
	}
	h.store.IncrementCodeUse(ctx, code.ID)

	// ضد فیلتر — حذف خودکار
	autoDelete := code.AutoDelete
	if autoDelete == 0 {
		fmt.Sscan(h.store.GetSetting(ctx, models.SettingAutoDeleteDefault), &autoDelete)
	}
	if autoDelete > 0 && len(msgIDs) > 0 {
		go h.scheduleDelete(ctx, c.Chat().ID, msgIDs, autoDelete)
	}

	// دکمه ارسال مجدد
	if h.store.GetSetting(ctx, "show_resend") == "true" {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("🔁 ارسال مجدد", "code_resend:"+codeStr)))
		c.Send(" ", kb)
	}

	return nil
}

// ── Helper ────────────────────────────────────────────────────

func (h *Handler) isAdmin(c tele.Context) bool {
	uid := c.Sender().ID
	if uid == h.ownerID {
		return true
	}
	return h.store.IsAdmin(context.Background(), uid)
}

func (h *Handler) showSearch(ctx context.Context) bool {
	return h.store.GetSetting(ctx, models.SettingShowSearch) == "true"
}

func (h *Handler) sendSubRequired(ctx context.Context, c tele.Context) error {
	plans, _ := h.store.ListSubPlans(ctx)
	if len(plans) == 0 {
		return c.Send("💎 برای دسترسی به این رسانه اشتراک لازم است.")
	}

	msg := h.store.GetSetting(ctx, models.SettingSubRequiredText)
	if msg == "" {
		msg = "💎 <b>برای دسترسی نیاز به اشتراک دارید:</b>"
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("💎 %s — %g تومان (%d روز)", p.Name, p.Price, p.Days)
		rows = append(rows, kb.Row(kb.Data(label, "sub_buy:"+p.ID.String())))
	}
	kb.Inline(rows...)

	return c.Send(msg, tele.ModeHTML, kb)
}

func (h *Handler) scheduleDelete(ctx context.Context, chatID int64, msgIDs []int, delaySec int) {
	time.Sleep(time.Duration(delaySec) * time.Second)
	for _, msgID := range msgIDs {
		h.bot.Delete(&tele.Message{ID: msgID, Chat: &tele.Chat{ID: chatID}})
	}
}

// sendFiles فایل‌ها را ارسال می‌کند و ID پیام‌ها را برمی‌گرداند.
func (h *Handler) sendFiles(ctx context.Context, c tele.Context,
	code *models.Code, files []models.File, signature string) []int {

	var msgIDs []int

	// caption آخرین فایل + امضا
	caption := code.Caption
	if signature != "" {
		caption += "\n\n" + signature
	}

	// آلبوم
	if code.IsAlbum && len(files) > 1 {
		var album tele.Album
		for i, f := range files {
			inp := fileToInput(f)
			if inp == nil {
				continue
			}
			if i == len(files)-1 {
				inp.(tele.Inputtable).SetCaption(caption)
			}
			album = append(album, inp)
		}
		msgs, err := c.Bot().SendAlbum(c.Recipient(), album, tele.ModeHTML)
		if err == nil {
			for _, m := range msgs {
				msgIDs = append(msgIDs, m.ID)
			}
		}
		return msgIDs
	}

	// تک فایل
	for i, f := range files {
		var cap string
		if i == len(files)-1 {
			cap = caption
		}

		opts := []any{tele.ModeHTML}
		if code.ForwardLock {
			opts = append(opts, tele.Silent)
		}

		msg, err := sendSingleFile(c, f, cap)
		if err != nil {
			h.log.Error("sendFiles", ports.F("err", err))
			continue
		}
		if msg != nil {
			msgIDs = append(msgIDs, msg.ID)
		}
	}

	return msgIDs
}

// userUploadMedia کاربر فایل آپلود می‌کند.
func (h *Handler) userUploadMedia(ctx context.Context, c tele.Context) error {
	autoApprove := h.store.GetSetting(ctx, models.SettingAutoApproveFiles) == "true"

	fi := extractFileInfo(c)
	if fi == nil {
		return nil
	}

	// ذخیره فایل
	f := &models.File{
		FileID:   fi.fileID,
		FileType: fi.fileType,
		Caption:  c.Message().Caption,
	}
	if err := h.store.CreateFile(ctx, f); err != nil {
		return c.Send("❌ خطا در ذخیره فایل.")
	}

	if autoApprove {
		// ساخت کد خودکار
		code := &models.Code{
			Code:       h.store.GenerateUniqueCode(ctx),
			Type:       models.CodeUnlimited,
			UploaderID: c.Sender().ID,
		}
		h.store.CreateCode(ctx, code)
		h.store.AddFileToCode(ctx, code.ID, f.ID, 0)
		return c.Send(fmt.Sprintf("✅ فایل آپلود شد!\n🔑 کد: <code>%s</code>", code.Code), tele.ModeHTML)
	}

	// ارسال به ادمین برای تأیید
	return c.Send("✅ فایل دریافت شد. در انتظار تأیید ادمین...")
}
