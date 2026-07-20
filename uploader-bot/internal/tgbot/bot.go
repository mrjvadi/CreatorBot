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

	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/joinevents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/core"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/store"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/util"
)

// Handler اصلی — وابستگی‌های مشترک را از core.App می‌گیرد (امبد).
type Handler struct {
	*core.App
	bcJobs chan bcJobMsg // صف ارسال همگانی پس‌زمینه
}

// Deps وابستگی‌های Handler — هماهنگ با الگوی سایر bot های فرعی.
type Deps struct {
	Engine        *engine.Engine
	Bot           *tele.Bot
	OwnerID       int64
	ChannelID     int64
	EncryptKey    string // برای رمزنگاری BotToken قفل‌های نوع «ربات» قبل از ذخیره
	RentalStatus  *memberclient.RentalStatus
	JoinPublisher *joinevents.Publisher
}

// NewHandler سازنده‌ی هماهنگ با Deps. از engine، DB/Cache را می‌سازد
// تا main.go مجبور به ساخت دستی store نباشد.
func NewHandler(d Deps) *Handler {
	st := store.New(d.Engine.Mongo, d.Engine.InstanceID, d.Engine.Cache, d.Engine.Log)
	h := &Handler{
		App: &core.App{
			Bot:           d.Bot,
			Store:         st,
			Cache:         d.Engine.Cache,
			Log:           d.Engine.Log,
			OwnerID:       d.OwnerID,
			ChannelID:     d.ChannelID,
			InstanceID:    d.Engine.InstanceID,
			Eng:           d.Engine,
			EncryptKey:    d.EncryptKey,
			RentalStatus:  d.RentalStatus,
			JoinPublisher: d.JoinPublisher,
		},
		bcJobs: make(chan bcJobMsg, 50),
	}
	go h.broadcastWorker()                      // یک worker تکی → نرخ کل کنترل‌شده
	go h.broadcastSweeper(context.Background()) // حذف خودکار پیام‌های همگانی
	h.startNATS()                               // دریافت آپدیت تنظیمات/متن‌ها از NATS
	h.startQueryNATS()                          // مدیریتِ محتوا (کدها/پوشه‌ها) از پنل وب apimanager
	return h
}

// New سازنده‌ی قدیمی — برای سازگاری با کد موجود نگه داشته شده.
func New(st *store.Store, bot *tele.Bot, cache ports.Cache, log ports.Logger, ownerID int64, instanceID string, eng *engine.Engine) *Handler {
	return &Handler{App: &core.App{
		Bot: bot, Store: st, Cache: cache, Log: log,
		OwnerID: ownerID, InstanceID: instanceID, Eng: eng,
	}}
}

// EnsureDefaults مقادیر پیش‌فرض تنظیمات را در صورت نبود ست می‌کند.
func (h *Handler) EnsureDefaults(ctx context.Context) { h.Store.SeedDefaults(ctx) }

// Register همه handler ها را ثبت می‌کند. b همچنان *tele.Bot خام است چون
// دریافت update (برخلاف ارسال پاسخ) انتزاع نشده — این الگوی رایج پلتفرم است.
func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", h.onStart)
	b.Handle("/panel", h.onPanel)
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
	b.Handle(tele.OnMyChatMember, h.onMyChatMember)
	b.Handle(tele.OnChatMember, h.onChatMember) // تشخیص لفت برای گزارش لفت + join/leave رایگان‌ها
	if h.JoinPublisher != nil {
		b.Handle(tele.OnUserJoined, h.JoinPublisher.HandleUserJoined)
		b.Handle(tele.OnUserLeft, h.JoinPublisher.HandleUserLeft)
	}
}

// ── /start ────────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	// بررسی bot active
	if h.Store.GetSetting(ctx, models.SettingBotActive) == "false" && !h.isAdmin(c) {
		return c.Send("😴 ربات فعلاً خاموشه، کمی بعد دوباره سر بزن.")
	}

	// ثبت/آپدیت کاربر
	user, err := h.Store.GetOrCreateUser(ctx, uid,
		c.Sender().Username, c.Sender().FirstName)
	h.LogErr("onStart: get/create user", err)

	if user != nil && user.IsBlocked {
		return c.Send("⛔️ دسترسی شما به این ربات محدود شده. اگه فکر می‌کنی اشتباهیه، با پشتیبانی در تماس باش.")
	}

	// بررسی deep link — /start CODE
	args := c.Message().Payload
	if args != "" {
		if !h.spamOK(ctx, c) {
			return c.Send("⏳ یه‌کم آروم‌تر! چند لحظه صبر کن و دوباره امتحان کن.")
		}
		return h.userDeliverCode(ctx, c, user, args)
	}

	// منو اصلی
	if h.isAdmin(c) {
		return h.OpenPanel(ctx, c)
	}

	welcome := h.Store.GetSetting(ctx, models.SettingWelcomeText)
	if welcome == "" {
		welcome = fmt.Sprintf("👋 سلام %s، خوش اومدی!\n\n📮 فقط کافیه کد رسانه‌ای که می‌خوای رو برام بفرستی تا سریع تحویلت بدم.", c.Sender().FirstName)
	}
	if err := c.Send(welcome, tele.ModeHTML, h.kbUserMenu(ctx)); err != nil {
		return err
	}
	// دکمه‌های شروع پیشرفته (در صورت تنظیم)
	if sb := h.startButtons(ctx); sb != nil {
		return c.Send("👇", sb)
	}
	return nil
}

// ── onText ────────────────────────────────────────────────────

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	// state فعال
	st := h.GetState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	// cancel
	if text == btnCancel || text == btnBack {
		h.ClearState(ctx, uid)
		if h.isAdmin(c) {
			return c.Send(msgDone, kbAdmin())
		}
		return c.Send(msgDone, h.kbUserMenu(ctx))
	}

	// ── ادمین routing ────────────────────────────────────────
	if h.isAdmin(c) {
		return h.adminOnText(ctx, c, text)
	}

	// ── کاربر: منو یا دریافت کد ──────────────────────────────
	return h.userOnText(ctx, c, text)
}

// ── onMedia ───────────────────────────────────────────────────

func (h *Handler) onMedia(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	if !h.isAdmin(c) {
		// بررسی user upload
		if h.Store.GetSetting(ctx, models.SettingUserUpload) != "true" {
			return nil
		}
		return h.userUploadMedia(ctx, c)
	}

	return h.adminHandleMedia(ctx, c, uid)
}

// ── Inline Query ──────────────────────────────────────────────

func (h *Handler) onInlineQuery(c tele.Context) error {
	ctx := context.Background()
	query := strings.TrimSpace(c.Query().Text)

	if h.Store.GetSetting(ctx, models.SettingShowSearch) != "true" {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	if len(query) < 2 {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	codes, err := h.Store.SearchCodes(ctx, query)
	h.LogErr("onInlineQuery", err)
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
		var err error
		user, err = h.Store.GetOrCreateUser(ctx, uid, c.Sender().Username, c.Sender().FirstName)
		h.LogErr("userDeliverCode: get/create user", err)
	}
	if user != nil && user.IsBlocked {
		return c.Send("⛔️ دسترسی محدود شده است.")
	}

	// پیدا کردن کد + فایل‌ها (با کش برای کاهش درخواست به دیتابیس)
	code, deliverFiles, err := h.Store.FindCodeForDelivery(ctx, codeStr)
	h.LogErr("userDeliverCode: find code", err)
	if code == nil {
		return c.Send(h.Store.GetSetting(ctx, "not_found_text") + msgCodeNF)
	}

	// در انتظار تایید
	if code.Pending {
		return c.Send("⏳ این رسانه هنوز منتظر تایید ادمینه؛ یه‌کم صبر کن و بعد دوباره امتحان کن.")
	}

	// انقضا
	if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
		return c.Send("⏰ مهلت این کد تموم شده. از فرستنده بخواه یه کد تازه برات بفرسته.")
	}

	// محدودیت استفاده
	if code.Type == models.CodeLimited && code.UsedCount >= code.MaxUse {
		return c.Send("🚫 ظرفیت استفاده از این کد پر شده.")
	}

	// جوین اجباری — قفل‌های اجباری (سراسری) بررسی می‌شوند
	if notJoined := h.checkForceJoin(ctx, c); len(notJoined) > 0 {
		return h.sendJoinRequest(c, notJoined)
	}

	// رمز عبور
	if code.Password != "" {
		st := h.GetState(ctx, uid)
		if st.Data["pwd_verified"] != code.Code {
			h.SetStepData(ctx, uid, stepPassword, "code_id", code.ID)
			return c.Send(h.Store.GetSetting(ctx, models.SettingPasswordText) + "🔐 این رسانه رمز داره؛ رمزش رو برام بفرست:")
		}
	}

	// اشتراک
	if code.SubRequired || h.Store.GetSetting(ctx, models.SettingSubRequired) == "true" {
		if user == nil || !user.HasActiveSub() {
			// بررسی دانلود رایگان
			freeLimit := 0
			fmt.Sscan(h.Store.GetSetting(ctx, models.SettingFreeDownloads), &freeLimit)
			if freeLimit > 0 && user != nil && user.FreeDownloads < freeLimit {
				// هنوز دانلود رایگان دارد — کل سند کاربر را حفظ می‌کنیم
				user.FreeDownloads++
				h.LogErr("userDeliverCode: update free downloads", h.Store.UpdateUser(ctx, user))
			} else {
				return h.sendSubRequired(ctx, c)
			}
		}
	}

	// محدودیت دانلود per user
	if code.DownloadLimit > 0 && user != nil {
		count := h.Store.GetDownloadCount(ctx, user.ID, code.ID)
		if count >= code.DownloadLimit {
			return c.Send("🚫 سهمیه‌ی دانلود شما برای این رسانه به پایان رسیده.")
		}
	}

	// گیت سین/ری‌اکشن اجباری فیک
	if (code.ForceSeen || code.ForceReact) && !h.gatePassed(ctx, uid, code.Code) {
		return h.sendGate(c, code)
	}

	// ارسال فایل‌ها (از کش تحویل بالا آمده‌اند)
	files := deliverFiles
	if len(files) == 0 {
		return c.Send("😕 فایلی برای این کد پیدا نکردم.")
	}

	// signature — فقط اگر فعال باشد
	sig := ""
	if h.Store.GetSetting(ctx, models.SettingSignatureEnabled) == "true" {
		sig = h.Store.GetSetting(ctx, models.SettingSignature)
	}

	autoDelete := h.computeAutoDelete(ctx, code)
	warnWillSend := h.warnWillSend(ctx, code)
	adWillSend := h.adWillSend(ctx)
	showResend := h.Store.GetSetting(ctx, models.SettingShowResendButton) == "true"

	// رای‌ها + گزارش (+ ارسال مجدد فقط اگر روی فایل باشد) → زیر خودِ فایل‌ها
	fileKb, _ := h.buildFileKb(ctx, code)
	msgIDs := h.sendFiles(ctx, c, code, files, sig, fileKb)

	// ثبت دانلود
	if user != nil {
		h.LogErr("userDeliverCode: log download", h.Store.LogDownload(ctx, user.ID, code.ID))
	}
	h.LogErr("userDeliverCode: increment use", h.Store.IncrementCodeUse(ctx, code.ID))

	// تبلیغ حین نمایش — اگر هشدار نباشد، «ارسال مجدد» اینجا می‌آید.
	if adWillSend {
		var adRows []tele.Row
		if showResend && !warnWillSend {
			adRows = []tele.Row{resendRow(code)}
		}
		h.sendActiveAd(ctx, c, adRows)
	}

	// ضد فیلتر — هشدار حذف خودکار. «ارسال مجدد» روی پیام هشدار می‌نشیند.
	if autoDelete > 0 && len(msgIDs) > 0 {
		if warnWillSend {
			warnText := h.GetSetting(ctx, models.SettingAutoDeleteWarn,
				"⏳ این فایل‌ها تا {sec} ثانیه‌ی دیگه خودکار پاک می‌شن؛ اگه می‌خوایشون، همین الان ذخیره یا فوروارد کن 🙏")
			warnText = strings.ReplaceAll(warnText, "{sec}", fmt.Sprintf("%d", autoDelete))
			opts := []any{tele.ModeHTML}
			if rk := h.resendKb(ctx, code); rk != nil {
				opts = append(opts, rk)
			}
			if wm, err := c.Bot().Send(c.Recipient(), warnText, opts...); err == nil && wm != nil {
				if h.Store.GetSetting(ctx, models.SettingAutoDeleteWarnKeep) != "true" {
					msgIDs = append(msgIDs, wm.ID)
				}
			}
		}
		go h.scheduleDelete(ctx, c.Chat().ID, msgIDs, autoDelete)
	}

	return nil
}

// ── Helper ────────────────────────────────────────────────────

func (h *Handler) isAdmin(c tele.Context) bool {
	uid := c.Sender().ID
	if uid == h.OwnerID {
		return true
	}
	return h.Store.IsAdmin(context.Background(), uid)
}

func (h *Handler) showSearch(ctx context.Context) bool {
	return h.Store.GetSetting(ctx, models.SettingShowSearch) == "true"
}

func (h *Handler) sendSubRequired(ctx context.Context, c tele.Context) error {
	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("sendSubRequired", err)
	if len(plans) == 0 {
		return c.Send("💎 این رسانه فقط برای مشترک‌هاست، ولی فعلاً پلنی برای فروش تعریف نشده. با ادمین هماهنگ کن.")
	}

	msg := h.Store.GetSetting(ctx, models.SettingSubRequiredText)
	if msg == "" {
		msg = "💎 <b>این رسانه مخصوص مشترک‌هاست.</b>\nیکی از پلن‌های زیر رو انتخاب کن تا بهت دسترسی بدیم:"
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("💎 %s — %g تومان (%d روز)", p.Name, p.Price, p.Days)
		rows = append(rows, kb.Row(kb.Data(label, "sub_buy:"+p.ID)))
	}
	kb.Inline(rows...)

	return c.Send(msg, tele.ModeHTML, kb)
}

func (h *Handler) scheduleDelete(ctx context.Context, chatID int64, msgIDs []int, delaySec int) {
	time.Sleep(time.Duration(delaySec) * time.Second)
	for _, msgID := range msgIDs {
		if err := h.Bot.Delete(&tele.Message{ID: msgID, Chat: &tele.Chat{ID: chatID}}); err != nil {
			h.Log.Error("scheduleDelete", ports.F("err", err))
		}
	}
}

// computeAutoDelete زمان حذف خودکار یک کد (تکی یا پیش‌فرض) را برمی‌گرداند.
func (h *Handler) computeAutoDelete(ctx context.Context, code *models.Code) int {
	ad := code.AutoDelete
	if ad == 0 {
		fmt.Sscan(h.Store.GetSetting(ctx, models.SettingAutoDeleteDefault), &ad)
	}
	return ad
}

func (h *Handler) warnWillSend(ctx context.Context, code *models.Code) bool {
	return h.computeAutoDelete(ctx, code) > 0 &&
		h.Store.GetSetting(ctx, models.SettingAutoDeleteWarnOff) != "true"
}

func (h *Handler) adWillSend(ctx context.Context) bool { return h.adsCount(ctx) > 0 }

// resendOnFile مشخص می‌کند ارسال مجدد باید روی خود فایل باشد یا روی پیام دنباله.
func (h *Handler) resendOnFile(ctx context.Context, code *models.Code) bool {
	if h.Store.GetSetting(ctx, models.SettingShowResendButton) != "true" {
		return false
	}
	return !h.warnWillSend(ctx, code) && !h.adWillSend(ctx)
}

// buildFileKb کیبورد زیر خودِ فایل: رای‌ها + گزارش (+ ارسال مجدد اگر روی فایل باشد).
func (h *Handler) buildFileKb(ctx context.Context, code *models.Code) (*tele.ReplyMarkup, bool) {
	showLikes := h.Store.GetSetting(ctx, models.SettingShowLikesButtons) == "true"
	showReport := h.Store.GetSetting(ctx, models.SettingShowReportButton) == "true"
	var rLikes, rDislikes int64
	if showLikes {
		rLikes, rDislikes = h.Store.CountReactions(ctx, code.Code)
	}
	rows := reactReportRows(code, showLikes, showReport, rLikes, rDislikes)
	if h.resendOnFile(ctx, code) {
		rows = append(rows, resendRow(code))
	}
	if len(rows) == 0 {
		return nil, false
	}
	kb := &tele.ReplyMarkup{}
	kb.Inline(rows...)
	return kb, true
}

// resendKb کیبوردِ تنها‌شاملِ «ارسال مجدد» (برای پیام هشدار/تبلیغ).
func (h *Handler) resendKb(ctx context.Context, code *models.Code) *tele.ReplyMarkup {
	if h.Store.GetSetting(ctx, models.SettingShowResendButton) != "true" {
		return nil
	}
	kb := &tele.ReplyMarkup{}
	kb.Inline(resendRow(code))
	return kb
}

// sendFiles فایل‌ها را ارسال می‌کند. lastKb (در صورت غیر nil) به آخرین پیام می‌چسبد.
func (h *Handler) sendFiles(ctx context.Context, c tele.Context,
	code *models.Code, files []models.File, signature string, lastKb *tele.ReplyMarkup) []int {

	var msgIDs []int

	// پسوندِ آخرین پیام: بازدید فیک + امضا (به انتها اضافه می‌شود تا offset
	// قالب‌بندی‌ها به‌هم نریزد).
	suffix := ""
	if code.FakeViews > 0 {
		suffix += fmt.Sprintf("\n\n👁 %d بازدید", code.FakeViews)
	}
	if signature != "" {
		sig := strings.ReplaceAll(signature, "{code}", code.Code)
		sig = strings.ReplaceAll(sig, "{count}", fmt.Sprintf("%d", len(files)))
		sig = strings.ReplaceAll(sig, "{downloads}", fmt.Sprintf("%d", code.UsedCount))
		suffix += "\n\n" + sig
	}

	protected := code.ForwardLock

	// کپشن مؤثرِ هر فایل: اگر ادمین کپشنِ کد را ویرایش کرده، همان (متنی)؛
	// وگرنه کپشن و قالب‌بندیِ اصلیِ خودِ فایل.
	effCaption := func(f models.File) (string, []models.Entity) {
		if code.Caption != "" {
			return code.Caption, nil
		}
		return f.Caption, f.CaptionEntities
	}

	// آلبوم — فقط وقتی همه‌ی فایل‌ها با آلبوم سازگارند (ویس/استیکر/ویدیونوت نه)
	if code.IsAlbum && len(files) > 1 && albumCompatible(files) {
		var album tele.Album
		used := files[:0:0]
		for _, f := range files {
			inp := fileToInput(f)
			if inp == nil {
				continue
			}
			// هر آیتم با کپشن HTMLِ خودش (حفظ قالب‌بندی)
			setInputCaption(inp, util.EntitiesToHTML(f.Caption, f.CaptionEntities))
			album = append(album, inp)
			used = append(used, f)
		}
		// کپشن آیتم اول = کپشن مؤثرِ فایل اول + پسوند (به HTML)
		if len(album) > 0 && len(used) > 0 {
			var cap0 string
			if code.Caption != "" {
				cap0 = code.Caption // override ادمین (ممکن است HTML باشد)
			} else {
				cap0 = util.EntitiesToHTML(used[0].Caption, used[0].CaptionEntities)
			}
			setInputCaption(album[0], cap0+util.EscapeHTML(suffix))
		}
		opts := []any{tele.ModeHTML}
		if protected {
			opts = append(opts, tele.Protected)
		}
		msgs, err := c.Bot().SendAlbum(c.Recipient(), album, opts...)
		if err == nil {
			for _, m := range msgs {
				msgIDs = append(msgIDs, m.ID)
			}
			if lastKb != nil {
				if bm, e := c.Bot().Send(c.Recipient(), "⬆️ رسانه بالا", lastKb); e == nil {
					msgIDs = append(msgIDs, bm.ID)
				}
			}
		}
		return msgIDs
	}

	// تک یا چند فایل غیرآلبوم — هر فایل با کپشن/قالب‌بندیِ خودش
	for i, f := range files {
		cap, ents := effCaption(f)
		so := &tele.SendOptions{}
		if protected {
			so.Protected = true
		}
		if i == len(files)-1 {
			cap += suffix
			if lastKb != nil {
				so.ReplyMarkup = lastKb
			}
		}
		msg, err := sendMedia(c, f, cap, ents, so)
		if err != nil && f.StorageMsgID != 0 {
			// file_id کار نکرد (مثلاً تغییر توکن) → کپی از کانال ذخیره‌سازی
			msg, err = h.sendFromStorage(c, f, so)
		}
		if err != nil {
			h.Log.Error("sendFiles", ports.F("err", err))
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
	autoApprove := h.Store.GetSetting(ctx, models.SettingAutoApproveFiles) == "true"

	fi := extractFileInfo(c)
	if fi == nil {
		return nil
	}

	// ذخیره فایل (با کپشن و قالب‌بندی اصلی)
	f := &models.File{
		FileID:          fi.fileID,
		FileType:        fi.fileType,
		Caption:         c.Message().Caption,
		CaptionEntities: util.ToModelEntities(c.Message().CaptionEntities),
		UploaderID:      c.Sender().ID,
	}
	if err := h.Store.CreateFile(ctx, f); err != nil {
		h.LogErr("userUploadMedia: create file", err)
		return c.Send("⚠️ نشد فایل رو ذخیره کنم؛ یه بار دیگه امتحان کن.")
	}
	if cid, mid, ok := h.archiveToStorage(ctx, c); ok {
		h.LogErr("userUploadMedia: set storage", h.Store.SetFileStorage(ctx, f.ID, cid, mid))
		f.StorageChatID, f.StorageMsgID = cid, mid
	}

	if autoApprove {
		// ساخت کد خودکار با اعمال تنظیمات پیش‌فرض آپلود
		code := &models.Code{
			Code:       h.Store.GenerateUniqueCode(ctx),
			Type:       models.CodeUnlimited,
			UploaderID: c.Sender().ID,
			FileIDs:    []string{f.ID},
		}
		h.applyUploadDefaults(ctx, code)
		if err := h.Store.CreateCode(ctx, code); err != nil {
			h.LogErr("userUploadMedia: create code", err)
			return c.Send("⚠️ فایلت ذخیره شد ولی ساخت کد به مشکل خورد؛ لطفاً با پشتیبانی تماس بگیر.")
		}
		return c.Send(fmt.Sprintf("🎉 آپلود شد!\n🔑 کد رسانه‌ات: <code>%s</code>\nهر وقت خواستی همین کد رو بفرست تا دوباره برات بفرستمش.", code.Code), tele.ModeHTML)
	}

	// ساخت کد در حالت انتظار + اطلاع به ادمین‌ها
	code := &models.Code{
		Code:       h.Store.GenerateUniqueCode(ctx),
		Type:       models.CodeUnlimited,
		UploaderID: c.Sender().ID,
		FileIDs:    []string{f.ID},
		MediaTypes: []string{fi.fileType},
		Pending:    true,
	}
	h.applyUploadDefaults(ctx, code)
	if err := h.Store.CreateCode(ctx, code); err != nil {
		h.LogErr("userUploadMedia: create pending code", err)
		return c.Send("⚠️ فایلت ذخیره شد ولی ساخت کد به مشکل خورد؛ لطفاً با پشتیبانی تماس بگیر.")
	}
	h.notifyAdminsPending(ctx, code, c.Sender())
	return c.Send("📬 فایلت رسید! به‌محض تایید ادمین در دسترس قرار می‌گیره و بهت خبر می‌دیم.")
}
