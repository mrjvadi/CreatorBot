package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/util"
)

// ── handleStep — مسیریابی state machine ──────────────────────

func (h *Handler) handleStep(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID

	// لغو سراسری در هر مرحله‌ای — قبل از ذخیره‌ی مقدار.
	if text == btnCancel || text == btnBack {
		h.ClearState(ctx, uid)
		h.AlbumClear(ctx, uid)
		if h.isAdmin(c) {
			return c.Send(msgDone, kbAdmin())
		}
		return c.Send(msgDone, h.kbUserMenu(ctx))
	}

	switch st.Step {
	case stepCodeFiles:
		if text == btnDone {
			return h.finishUpload(ctx, c)
		}
		if text == btnCancel || text == btnBack {
			h.AlbumClear(ctx, uid)
			h.ClearState(ctx, uid)
			return c.Send(msgDone, kbAdmin())
		}
		return c.Send("📤 فایل بفرستید یا «✅ تمام شد» را بزنید.")
	case stepPassword:
		return h.userCheckPassword(ctx, c, st, text)
	case stepSearch:
		h.ClearState(ctx, uid)
		return h.userSearch(ctx, c, text)
	case stepNewFolder:
		h.ClearState(ctx, uid)
		if err := h.Store.CreateFolder(ctx, &models.Folder{Name: text, IsActive: true}); err != nil {
			return c.Send("❌ خطا در ساخت پوشه.", kbAdmin())
		}
		return c.Send("✅ پوشه ساخته شد.", kbAdmin())
	case stepEditCaption:
		return h.adminSaveCaption(ctx, c, st, text)
	case stepSetPassword:
		return h.adminSavePassword(ctx, c, st, text)
	case stepSetLimit:
		return h.adminSaveLimit(ctx, c, st, text)
	case stepEditSetting:
		return h.adminSaveSetting(ctx, c, st, text)
	case stepAddChannel:
		return h.adminSaveChannel(ctx, c, text)
	case stepNewPlan:
		return h.adminSavePlan(ctx, c, st, text)
	case stepBroadcast:
		return h.finalizeBroadcast(ctx, c)
	case stepAddAdmin:
		return h.adminSaveAdmin(ctx, c, text)
	case stepSearchUser:
		h.ClearState(ctx, uid)
		return h.adminShowUser(ctx, c, text)
	case stepAddPreview:
		return h.adminSavePreview(ctx, c, text)
	case stepAddAd:
		return h.adminSaveAd(ctx, c, text)
	case stepSetLikes:
		return h.adminCodeSaveFake(ctx, c, st, "likes", text)
	case stepSetDownloads:
		return h.adminCodeSaveFake(ctx, c, st, "downloads", text)
	case stepSetViews:
		return h.adminCodeSaveFake(ctx, c, st, "views", text)
	case stepNewSubfolder:
		return h.adminSaveSubfolder(ctx, c, st, text)
	case stepAddLock:
		return h.lockSaveAdd(ctx, c, text)
	case stepLockCap:
		return h.lockSaveCap(ctx, c, st, text)
	}
	h.ClearState(ctx, uid)
	return nil
}

// ══════════════════════════════════════════════════════════════
// متدهای کاربر
// ══════════════════════════════════════════════════════════════

// حداکثر تلاش ناموفق برای رمز یک کد قبل از قفل موقت — رمزها معمولاً PIN
// عددی کوتاه هستند و بدون این محدودیت به‌راحتی قابل brute-force بودند.
const (
	maxPasswordAttempts = 5
	passwordLockout     = 5 * time.Minute
)

func (h *Handler) userCheckPassword(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	codeID := st.Data["code_id"]

	if lockedUntil := st.Data["pwd_locked_until"]; lockedUntil != "" {
		if t, err := time.Parse(time.RFC3339, lockedUntil); err == nil && time.Now().Before(t) {
			return c.Send(fmt.Sprintf("⏳ به‌دلیل تلاش ناموفق زیاد، %d ثانیه دیگر دوباره امتحان کنید.",
				int(time.Until(t).Seconds())))
		}
	}

	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("userCheckPassword: find code", err)
	if code == nil {
		h.ClearState(ctx, uid)
		return c.Send(msgCodeNF, h.kbUserMenu(ctx))
	}
	if text != code.Password {
		attempts := 0
		fmt.Sscan(st.Data["pwd_attempts"], &attempts)
		attempts++

		cur := h.GetState(ctx, uid)
		if cur.Data == nil {
			cur.Data = map[string]string{}
		}
		cur.Step = stepPassword
		cur.Data["code_id"] = codeID

		if attempts >= maxPasswordAttempts {
			cur.Data["pwd_attempts"] = "0"
			cur.Data["pwd_locked_until"] = time.Now().Add(passwordLockout).Format(time.RFC3339)
			h.SetState(ctx, uid, cur)
			return c.Send("⛔️ تلاش بیش از حد مجاز. ۵ دقیقه دیگر دوباره امتحان کنید.")
		}
		cur.Data["pwd_attempts"] = fmt.Sprintf("%d", attempts)
		h.SetState(ctx, uid, cur)
		return c.Send(fmt.Sprintf("❌ رمز اشتباه است. (%d از %d تلاش) دوباره وارد کنید:", attempts, maxPasswordAttempts))
	}
	// رمز درست — علامت‌گذاری و رفتن به مسیر کامل تحویل (دکمه‌ها/تبلیغ/حذف خودکار).
	h.SetStepData(ctx, uid, stepIdle, "pwd_verified", code.Code)
	user, err := h.Store.GetUser(ctx, uid)
	h.LogErr("userCheckPassword: get user", err)
	return h.userDeliverCode(ctx, c, user, code.Code)
}

func (h *Handler) userSearch(ctx context.Context, c tele.Context, query string) error {
	codes, err := h.Store.SearchCodes(ctx, query)
	h.LogErr("userSearch", err)
	if len(codes) == 0 {
		return c.Send("🔍 چیزی با این عبارت پیدا نکردم؛ یه کلمه‌ی دیگه رو امتحان کن.", h.kbUserMenu(ctx))
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 %d نتیجه پیدا کردم:\n\n", len(codes)))
	for _, code := range codes {
		cap := code.Caption
		if len(cap) > 40 {
			cap = cap[:40] + "…"
		}
		sb.WriteString(fmt.Sprintf("📄 %s\n/get_%s\n\n", cap, code.Code))
	}
	return c.Send(sb.String(), h.kbUserMenu(ctx))
}

func (h *Handler) userBuySubPlan(ctx context.Context, c tele.Context, arg string) error {
	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("userBuySubPlan", err)
	if len(plans) == 0 {
		return c.Edit("❌ پلنی موجود نیست.")
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("%s — %.0f تومان (%d روز)", p.Name, p.Price, p.Days)
		rows = append(rows, kb.Row(kb.Data(label, "sub_pay:"+p.ID)))
	}
	kb.Inline(rows...)
	return c.Edit("💎 یک پلن انتخاب کنید:", kb)
}

func (h *Handler) userPaySub(ctx context.Context, c tele.Context, planID, gateway string) error {
	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("userPaySub: list plans", err)
	var plan *models.SubPlan
	for i := range plans {
		if plans[i].ID == planID {
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		return c.Edit(msgPlanNF)
	}
	// روش‌های پرداخت (بر اساس روش‌های فعال در تنظیمات)
	if gateway == "" {
		kb := &tele.ReplyMarkup{}
		var rows []tele.Row
		if h.Store.GetSetting(ctx, models.SettingPaymentCard) == "true" {
			rows = append(rows, kb.Row(kb.Data("💳 کارت به کارت", "sub_pay:"+planID+":card")))
		}
		if h.Store.GetSetting(ctx, models.SettingPaymentZarinpal) == "true" {
			rows = append(rows, kb.Row(kb.Data("🌐 زرین‌پال", "sub_pay:"+planID+":zarinpal")))
		}
		if h.Store.GetSetting(ctx, models.SettingPaymentZibal) == "true" {
			rows = append(rows, kb.Row(kb.Data("🌐 زیبال", "sub_pay:"+planID+":zibal")))
		}
		if h.Store.GetSetting(ctx, models.SettingPaymentTON) == "true" {
			rows = append(rows, kb.Row(kb.Data("💎 تون (TON)", "sub_pay:"+planID+":ton")))
		}
		if len(rows) == 0 {
			// اگر هیچ روشی فعال نشده، حداقل کارت‌به‌کارت
			rows = append(rows, kb.Row(kb.Data("💳 کارت به کارت", "sub_pay:"+planID+":card")))
		}
		kb.Inline(rows...)
		return c.Edit(fmt.Sprintf("روش پرداخت برای «%s»:", plan.Name), kb)
	}

	// درگاه‌های آنلاین واقعی
	if gateway == "zarinpal" || gateway == "zibal" {
		return h.startOnlinePayment(ctx, c, plan, gateway)
	}

	// کارت/تون/ترون → ثبت در حالت انتظار تایید ادمین
	uid := c.Sender().ID
	user, err := h.Store.GetUser(ctx, uid)
	h.LogErr("userPaySub: get user", err)
	userID := ""
	if user != nil {
		userID = user.ID
	}
	pay := &models.Payment{
		UserID: userID, TelegramID: uid, PlanID: plan.ID,
		Gateway: models.PaymentGateway(gateway),
		Amount:  plan.Price, Status: models.PaymentPending,
	}
	if err := h.Store.CreatePayment(ctx, pay); err != nil {
		h.LogErr("userPaySub: create payment", err)
		return c.Edit("❌ ثبت درخواست پرداخت با خطا مواجه شد. دوباره امتحان کنید.")
	}
	// بدون این، درخواست هرگز به دست هیچ ادمینی نمی‌رسید — دکمه‌ی تایید/رد اصلاً
	// جایی ارسال نمی‌شد و پرداخت برای همیشه در حالت pending می‌ماند.
	h.notifyAdminsPayment(ctx, pay, c.Sender(), plan)
	if gateway == "card" {
		card := h.Store.GetSetting(ctx, models.SettingCardNumber)
		holder := h.Store.GetSetting(ctx, models.SettingCardHolder)
		return c.Edit(fmt.Sprintf("💳 مبلغ %.0f تومان را به کارت زیر واریز و سپس رسید را برای ادمین بفرستید:\n\n<code>%s</code>\n%s", plan.Price, card, holder), tele.ModeHTML)
	}
	return c.Edit("✅ درخواست پرداخت ثبت شد.\nپس از تأیید ادمین اشتراک فعال می‌شود.")
}

// notifyAdminsPayment ادمین‌ها را از یک درخواست پرداخت دستی (کارت/TON/TRON) که
// نیاز به تایید دارد آگاه می‌کند، با دکمه‌های «تایید»/«رد» مستقیم.
func (h *Handler) notifyAdminsPayment(ctx context.Context, pay *models.Payment, from *tele.User, plan *models.SubPlan) {
	uname := "—"
	if from.Username != "" {
		uname = "@" + from.Username
	}
	msg := fmt.Sprintf(
		"💳 <b>درخواست پرداخت جدید</b>\n\n👤 از: %d (%s)\n💎 پلن: %s\n💰 مبلغ: %.0f تومان\n🧭 روش: %s",
		from.ID, uname, plan.Name, pay.Amount, gatewayLabel(pay.Gateway))
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(
		kb.Data("✅ تایید و فعال‌سازی", "admin_pay_confirm:"+pay.ID),
		kb.Data("🗑 رد", "admin_pay_reject:"+pay.ID),
	))
	admins, err := h.Store.ListAdmins(ctx)
	h.LogErr("notifyAdminsPayment: list admins", err)
	recipients := map[int64]bool{}
	for _, a := range admins {
		recipients[a.TelegramID] = true
	}
	if h.OwnerID != 0 {
		recipients[h.OwnerID] = true
	}
	for id := range recipients {
		if _, sendErr := h.Bot.Send(&tele.User{ID: id}, msg, tele.ModeHTML, kb); sendErr != nil {
			h.LogErr("notifyAdminsPayment: send", sendErr)
		}
	}
}

func gatewayLabel(g models.PaymentGateway) string {
	switch g {
	case models.GatewayCard:
		return "کارت به کارت"
	case models.GatewayTON:
		return "ولت TON"
	case models.GatewayTRON:
		return "ولت TRON"
	case models.GatewayZarinpal:
		return "زرین‌پال"
	case models.GatewayZibal:
		return "زیبال"
	case models.GatewayStars:
		return "استارز"
	}
	return string(g)
}

func (h *Handler) userOpenFolder(ctx context.Context, c tele.Context, folderID string) error {
	parentID := folderID
	if parentID == "root" {
		parentID = ""
	}
	folders, err := h.Store.ListFolders(ctx, parentID)
	h.LogErr("userOpenFolder: list folders", err)
	codes, _, err := h.Store.ListCodes(ctx, parentID, 1, 50)
	h.LogErr("userOpenFolder: list codes", err)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, f := range folders {
		rows = append(rows, kb.Row(kb.Data("📁 "+f.Name, "folder_open:"+f.ID)))
	}
	for _, code := range codes {
		label := code.Caption
		if label == "" {
			label = code.Code
		}
		rows = append(rows, kb.Row(kb.Data("📄 "+label, "code_resend:"+code.Code)))
	}
	kb.Inline(rows...)
	return c.Edit("📂 محتوا:", kb)
}

// ══════════════════════════════════════════════════════════════
// متدهای ادمین
// ══════════════════════════════════════════════════════════════

func (h *Handler) adminOnText(ctx context.Context, c tele.Context, text string) error {
	switch text {
	case btnNewCode:
		h.SetStep(ctx, c.Sender().ID, stepCodeFiles)
		return c.Send("📤 فایل‌ها را ارسال کنید. پس از پایان «✅ تمام شد» بزنید.", kbAlbumDone())
	case btnCodeList:
		return h.adminListCodes(ctx, c)
	case btnFolders:
		return h.adminListFolders(ctx, c)
	case btnUsers:
		h.SetStep(ctx, c.Sender().ID, stepSearchUser)
		return c.Send("👤 آیدی عددی یا یوزرنیم کاربر را بفرستید:", kbCancelOnly())
	case btnStats:
		return h.adminShowStats(ctx, c)
	case btnSettings:
		return c.Send("⚙️ <b>تنظیمات</b> — یک دسته را انتخاب کنید:", tele.ModeHTML, kbSettingsHome())
	case btnSubPlans:
		return h.adminListPlans(ctx, c)
	case btnChannels:
		return h.lockList(ctx, c)
	case btnBroadcast:
		return h.adminBroadcastMenu(ctx, c)
	case btnBackup:
		return c.Send("💾 بکاپ و بازیابی:", kbBackupMenu())
	case btnAdmins:
		return h.adminListAdmins(ctx, c)
	case btnPreview:
		return h.adminListPreview(ctx, c)
	case btnAds:
		return h.adminListAds(ctx, c)
	case btnReset:
		return h.adminResetDownloads(ctx, c)
	}
	return nil
}

func (h *Handler) adminHandleMedia(ctx context.Context, c tele.Context, uid int64) error {
	st := h.GetState(ctx, uid)
	if st.Step == stepSetCover {
		return h.adminSaveCover(ctx, c)
	}
	if st.Step == stepRestore {
		return h.adminBackupRestore(ctx, c)
	}
	if st.Step == stepBroadcast {
		return h.finalizeBroadcast(ctx, c)
	}
	if st.Step != stepCodeFiles {
		return nil
	}
	fi := extractFileInfo(c)
	if fi == nil {
		return c.Send("❌ نوع فایل پشتیبانی نمی‌شود.")
	}
	// ذخیره‌ی سند فایل (با کپشن و قالب‌بندیِ اصلی) و نگه‌داری شناسه‌اش در بافر آلبوم
	caption := c.Message().Caption
	entities := util.ToModelEntities(c.Message().CaptionEntities)
	if h.Store.GetSetting(ctx, models.SettingRemoveLinks) == "true" {
		caption = h.maybeStripLinks(ctx, caption)
		entities = nil // با حذف لینک، offsetها بی‌اعتبار می‌شوند
	}
	f := &models.File{
		FileID:          fi.fileID,
		FileType:        fi.fileType,
		Caption:         caption,
		CaptionEntities: entities,
		UploaderID:      uid,
	}
	if err := h.Store.CreateFile(ctx, f); err != nil {
		return c.Send("❌ خطا در ذخیره فایل.")
	}
	if cid, mid, ok := h.archiveToStorage(ctx, c); ok {
		h.LogErr("adminHandleMedia: set storage", h.Store.SetFileStorage(ctx, f.ID, cid, mid))
	}
	h.AlbumAdd(ctx, uid, f.ID)
	// در حالت آلبوم (media group) برای هر آیتم پاسخ نمی‌دهیم تا اسپم نشود.
	if c.Message().AlbumID != "" {
		return nil
	}
	return c.Send("✅ فایل اضافه شد. ادامه دهید یا «✅ تمام شد».")
}

func (h *Handler) adminListCodes(ctx context.Context, c tele.Context) error {
	codes, total, err := h.Store.ListCodes(ctx, "", 1, 20)
	h.LogErr("adminListCodes", err)
	if len(codes) == 0 {
		return c.Send("📭 هیچ رسانه‌ای ثبت نشده.", kbAdmin())
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 رسانه‌ها (کل: %d):\n\n", total))
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	rows = append(rows, kb.Row(kb.Data("🎞 نمایش اسلایدی", "slide:0")))
	for _, code := range codes {
		label := code.Caption
		if label == "" {
			label = code.Code
		}
		rows = append(rows, kb.Row(kb.Data("⚙️ "+label, "admin_code_edit:"+code.ID)))
	}
	kb.Inline(rows...)
	return c.Send(sb.String(), kb)
}

func (h *Handler) adminEditCodeMenu(ctx context.Context, c tele.Context, codeID string) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminEditCodeMenu: find", err)
	if code == nil {
		return c.Edit(msgNotFound)
	}
	files, err := h.Store.GetFilesForCode(ctx, code.ID)
	h.LogErr("adminEditCodeMenu: files", err)
	info := fmt.Sprintf("⚙️ <b>تنظیمات رسانه</b>\n\n🔑 کد: <code>%s</code>\n📦 فایل‌ها: %d\n📥 دریافت: %d\n🔗 %s",
		code.Code, len(files), code.UsedCount, h.deepLink(code.Code))
	return c.Edit(info, tele.ModeHTML, kbCodeAdvanced(code))
}

func (h *Handler) adminDeleteCode(ctx context.Context, c tele.Context, codeID string) error {
	if err := h.Store.DeleteCode(ctx, codeID); err != nil {
		return c.Edit("❌ خطا در حذف.")
	}
	return c.Edit("🗑 رسانه حذف شد.")
}

func (h *Handler) adminToggleCodeProp(ctx context.Context, c tele.Context, codeID, prop string) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminToggleCodeProp: find", err)
	if code == nil {
		return c.Edit(msgNotFound)
	}
	switch prop {
	case "forward_lock":
		code.ForwardLock = !code.ForwardLock
	case "anti_filter":
		code.AntiFilter = !code.AntiFilter
	case "sub_required":
		code.SubRequired = !code.SubRequired
	case "channel_lock":
		code.ChannelLock = !code.ChannelLock
	case "force_seen":
		code.ForceSeen = !code.ForceSeen
	case "force_react":
		code.ForceReact = !code.ForceReact
	}
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminToggleCodeProp: update", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد"})
	}
	return h.adminEditCodeMenu(ctx, c, codeID)
}

// adminToggleAutoDelete حذف خودکار (ضدفیلتر) رسانه از پیوی را روشن/خاموش می‌کند.
func (h *Handler) adminToggleAutoDelete(ctx context.Context, c tele.Context, codeID string) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminToggleAutoDelete: find", err)
	if code == nil {
		return c.Edit(msgNotFound)
	}
	if code.AutoDelete > 0 {
		code.AutoDelete = 0
	} else {
		sec := h.GetSettingInt(ctx, models.SettingAutoDeleteDefault, 30)
		if sec <= 0 {
			sec = 30
		}
		code.AutoDelete = sec
	}
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminToggleAutoDelete: update", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد"})
	}
	return h.adminEditCodeMenu(ctx, c, codeID)
}

func (h *Handler) adminSetAutoDelete(ctx context.Context, c tele.Context, codeID, sec string) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminSetAutoDelete: find", err)
	if code == nil {
		return c.Edit(msgNotFound)
	}
	n, _ := strconv.Atoi(sec) // ورودی نامعتبر → 0 (بی‌خطر: یعنی خاموش)
	code.AutoDelete = n
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminSetAutoDelete: update", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد"})
	}
	return h.adminEditCodeMenu(ctx, c, codeID)
}

func (h *Handler) adminSaveCaption(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	code, err := h.Store.FindCodeByID(ctx, st.Data["code_id"])
	h.LogErr("adminSaveCaption: find", err)
	if code == nil {
		return c.Send(msgNotFound, kbAdmin())
	}
	code.Caption = h.maybeStripLinks(ctx, text)
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminSaveCaption: update", err)
		return c.Send("❌ ذخیره‌ی کپشن با خطا مواجه شد.", kbAdmin())
	}
	return c.Send("✅ کپشن بروزرسانی شد.", kbAdmin())
}

func (h *Handler) adminSavePassword(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	code, err := h.Store.FindCodeByID(ctx, st.Data["code_id"])
	h.LogErr("adminSavePassword: find", err)
	if code == nil {
		return c.Send(msgNotFound, kbAdmin())
	}
	if text == "0" || text == "حذف" {
		code.Password = ""
	} else {
		code.Password = text
	}
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminSavePassword: update", err)
		return c.Send("❌ ذخیره‌ی رمز با خطا مواجه شد.", kbAdmin())
	}
	return c.Send("✅ رمز بروزرسانی شد.", kbAdmin())
}

func (h *Handler) adminSaveLimit(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	code, err := h.Store.FindCodeByID(ctx, st.Data["code_id"])
	h.LogErr("adminSaveLimit: find", err)
	if code == nil {
		return c.Send(msgNotFound, kbAdmin())
	}
	n, _ := strconv.Atoi(text) // نامعتبر → 0 (نامحدود)
	code.DownloadLimit = n
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminSaveLimit: update", err)
		return c.Send("❌ ذخیره‌ی محدودیت با خطا مواجه شد.", kbAdmin())
	}
	return c.Send("✅ محدودیت دانلود تنظیم شد.", kbAdmin())
}

func (h *Handler) adminListFolders(ctx context.Context, c tele.Context) error {
	folders, err := h.Store.ListFolders(ctx, "")
	h.LogErr("adminListFolders", err)
	items := make([]folderItem, 0, len(folders))
	for _, f := range folders {
		items = append(items, folderItem{ID: f.ID, Name: f.Name})
	}
	return c.Send("📁 پوشه‌ها:", kbFolderList(items))
}

func (h *Handler) adminFolderOpen(ctx context.Context, c tele.Context, folderID string) error {
	codes, _, err := h.Store.ListCodes(ctx, folderID, 1, 50)
	h.LogErr("adminFolderOpen", err)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 محتوای پوشه (%d رسانه):\n\n", len(codes)))
	for _, code := range codes {
		sb.WriteString(fmt.Sprintf("📄 %s\n", code.Code))
	}
	return c.Edit(sb.String())
}

func (h *Handler) adminFolderDelete(ctx context.Context, c tele.Context, folderID string) error {
	if err := h.Store.DeleteFolder(ctx, folderID); err != nil {
		h.LogErr("adminFolderDelete", err)
		return c.Edit("❌ حذف پوشه با خطا مواجه شد.")
	}
	return c.Edit("🗑 پوشه حذف شد.")
}

func (h *Handler) adminForceJoinDelete(ctx context.Context, c tele.Context, chID string) error {
	if err := h.Store.RemoveForceJoinChannel(ctx, chID); err != nil {
		h.LogErr("adminForceJoinDelete", err)
		return c.Edit("❌ حذف کانال با خطا مواجه شد.")
	}
	return c.Edit("🗑 کانال حذف شد.")
}

func (h *Handler) adminListChannels(ctx context.Context, c tele.Context) error {
	channels, err := h.Store.ListForceJoinChannels(ctx)
	h.LogErr("adminListChannels", err)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ch := range channels {
		title := ch.Title
		if title == "" {
			if ch.Username != "" {
				title = "@" + ch.Username
			} else {
				title = ch.InviteURL
			}
		}
		rows = append(rows, kb.Row(kb.Data("🗑 "+title, "admin_ch_del:"+ch.ID)))
	}
	rows = append(rows, kb.Row(kb.Data("➕ افزودن کانال", "ch_add")))
	rows = append(rows, kb.Row(kb.Data(btnPanelLabel, "p:home")))
	kb.Inline(rows...)
	head := "📡 کانال‌های جوین اجباری:"
	if len(channels) == 0 {
		head = "📭 کانال جوین اجباری ثبت نشده."
	}
	return c.Send(head, kb)
}

// adminAskChannel از ادمین کانال جوین اجباری را می‌پرسد.
func (h *Handler) adminAskChannel(ctx context.Context, c tele.Context) error {
	h.SetStep(ctx, c.Sender().ID, stepAddChannel)
	return c.Send("📡 کانال را بفرستید:\n• یوزرنیم: @channel\n• آیدی عددی: -100...\n• یا لینک دعوت: https://t.me/...", kbCancelOnly())
}

func (h *Handler) adminSaveChannel(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	text = strings.TrimSpace(text)
	ch := &models.ForceJoinChannel{IsActive: true, Title: text}
	switch {
	case strings.HasPrefix(text, "http"):
		ch.InviteURL = text
	case strings.HasPrefix(text, "@"):
		ch.Username = strings.TrimPrefix(text, "@")
		ch.Title = text
	default:
		if id, err := strconv.ParseInt(text, 10, 64); err == nil {
			ch.ChatID = id
		} else {
			ch.Username = text
			ch.Title = "@" + text
		}
	}
	if err := h.Store.AddForceJoinChannel(ctx, ch); err != nil {
		h.LogErr("adminSaveChannel", err)
		return c.Send("❌ ثبت کانال با خطا مواجه شد. دوباره امتحان کنید.", kbAdmin())
	}
	return c.Send("✅ کانال اضافه شد.", kbAdmin())
}

func (h *Handler) adminListPlans(ctx context.Context, c tele.Context) error {
	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("adminListPlans", err)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("🗑 %s (%.0f ت | %d روز)", p.Name, p.Price, p.Days)
		rows = append(rows, kb.Row(kb.Data(label, "admin_sub_del:"+p.ID)))
	}
	rows = append(rows, kb.Row(kb.Data("➕ افزودن اشتراک", "plan_add")))
	rows = append(rows, kb.Row(kb.Data(btnPanelLabel, "p:home")))
	kb.Inline(rows...)
	head := "💎 پلن‌های اشتراک:"
	if len(plans) == 0 {
		head = "📭 هیچ پلنی ثبت نشده."
	}
	return c.Send(head, kb)
}

// adminAskPlan از ادمین مشخصات پلن جدید را می‌پرسد.
func (h *Handler) adminAskPlan(ctx context.Context, c tele.Context) error {
	h.SetStep(ctx, c.Sender().ID, stepNewPlan)
	return c.Send("💎 پلن را با این قالب بفرستید:\n<code>نام:قیمت:روز</code>\nمثال: <code>طلایی:50000:30</code>", tele.ModeHTML, kbCancelOnly())
}

func (h *Handler) adminSubPlanDelete(ctx context.Context, c tele.Context, planID string) error {
	if err := h.Store.DeleteSubPlan(ctx, planID); err != nil {
		h.LogErr("adminSubPlanDelete", err)
		return c.Edit("❌ حذف پلن با خطا مواجه شد.")
	}
	return c.Edit("🗑 پلن حذف شد.")
}

func (h *Handler) adminSavePlan(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	// فرمت: نام:قیمت:روز
	parts := strings.Split(text, ":")
	if len(parts) != 3 {
		return c.Send("❌ فرمت: نام:قیمت:روز")
	}
	price, priceErr := strconv.ParseFloat(parts[1], 64)
	days, daysErr := strconv.Atoi(parts[2])
	if priceErr != nil || daysErr != nil || price <= 0 || days <= 0 {
		// قبلاً خطای پارس بی‌صدا نادیده گرفته می‌شد و یک پلن رایگان/صفرروزه‌ی
		// خراب ساخته می‌شد؛ حالا صریحاً رد می‌شود.
		return c.Send("❌ قیمت و روز باید عدد مثبت باشند. فرمت: نام:قیمت:روز")
	}
	h.ClearState(ctx, uid)
	if err := h.Store.CreateSubPlan(ctx, &models.SubPlan{
		Name: parts[0], Price: price, Days: days, IsActive: true,
	}); err != nil {
		h.LogErr("adminSavePlan", err)
		return c.Send("❌ ثبت پلن با خطا مواجه شد. دوباره امتحان کنید.", kbAdmin())
	}
	return c.Send("✅ پلن اضافه شد.", kbAdmin())
}

func (h *Handler) adminShowStats(ctx context.Context, c tele.Context) error {
	s := h.Store.GetStats(ctx)
	msg := fmt.Sprintf(
		"📊 آمار ربات\n\n👥 کاربران: %d\n📄 رسانه‌ها: %d\n📁 فایل‌ها: %d\n🆕 امروز: %d\n💎 اشتراک فعال: %d",
		s.TotalUsers, s.TotalCodes, s.TotalFiles, s.TodayUsers, s.ActiveSubs,
	)
	return c.Send(msg, kbAdmin())
}

func (h *Handler) adminSaveSetting(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	key := st.Data["key"]
	page := st.Data["page"]
	if text == "حذف" {
		text = ""
	}
	if err := h.Store.SetSetting(ctx, key, text); err != nil {
		h.LogErr("adminSaveSetting", err)
		return c.Send("❌ ذخیره‌ی تنظیم با خطا مواجه شد.", kbAdmin())
	}
	if page != "" {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("🔙 بازگشت به تنظیمات", "ps:"+page)))
		return c.Send(msgSaved, kb)
	}
	return c.Send("✅ تنظیم ذخیره شد.", kbAdmin())
}

func (h *Handler) adminShowUser(ctx context.Context, c tele.Context, query string) error {
	user, err := h.Store.SearchUser(ctx, query)
	h.LogErr("adminShowUser", err)
	if user == nil {
		return c.Send("❌ کاربر یافت نشد.", kbAdmin())
	}
	status := "✅ فعال"
	if user.IsBlocked {
		status = "🚫 مسدود"
	}
	msg := fmt.Sprintf("👤 کاربر\n\n🆔 %d\n📛 %s\nوضعیت: %s\n📥 دانلود رایگان: %d",
		user.TelegramID, user.FirstName, status, user.FreeDownloads)
	tgStr := strconv.FormatInt(user.TelegramID, 10)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	if user.IsBlocked {
		rows = append(rows, kb.Row(kb.Data("✅ رفع مسدودی", "admin_user_unblock:"+tgStr)))
	} else {
		rows = append(rows, kb.Row(kb.Data("🚫 مسدود کردن", "admin_user_block:"+tgStr)))
	}
	rows = append(rows,
		kb.Row(kb.Data("💎 تغییر اشتراک", "admin_user_subm:"+tgStr)),
		kb.Row(kb.Data("♻️ ریست دانلودهای کاربر", "admin_user_reset:"+tgStr)),
	)
	kb.Inline(rows...)
	return c.Send(msg, kb)
}

func (h *Handler) adminToggleBlock(ctx context.Context, c tele.Context, tgIDStr string, block bool) error {
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return c.Edit(msgInvalid)
	}
	if err := h.Store.BlockUser(ctx, tgID, block); err != nil {
		h.LogErr("adminToggleBlock", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد"})
	}
	if block {
		return c.Edit("🚫 کاربر مسدود شد.")
	}
	return c.Edit("✅ مسدودی رفع شد.")
}

// adminConfirmPayment یک پرداخت دستی (کارت/TON/TRON) را تایید می‌کند: وضعیت
// پرداخت را «تاییدشده» می‌کند، پلن مربوطه را پیدا و اشتراک کاربر را واقعاً
// فعال می‌کند (قبلاً این تابع فقط وضعیت را ست می‌کرد و هیچ اشتراکی فعال
// نمی‌شد)، و به خودِ کاربر هم اطلاع می‌دهد.
func (h *Handler) adminConfirmPayment(ctx context.Context, c tele.Context, payID string) error {
	pay, err := h.Store.FindPayment(ctx, payID)
	if err != nil {
		h.LogErr("adminConfirmPayment: find", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در خواندن پرداخت"})
	}
	if pay == nil {
		return c.Edit("❌ این پرداخت یافت نشد (شاید قبلاً حذف شده).")
	}
	if pay.Status == models.PaymentConfirmed {
		return c.Respond(&tele.CallbackResponse{Text: "قبلاً تایید شده"})
	}

	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("adminConfirmPayment: list plans", err)
	var plan *models.SubPlan
	for i := range plans {
		if plans[i].ID == pay.PlanID {
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		return c.Edit("❌ پلن این پرداخت دیگر وجود ندارد؛ اشتراک فعال نشد. با کاربر هماهنگ کنید.")
	}

	if err := h.Store.SetUserSub(ctx, pay.TelegramID, pay.PlanID, plan.Days); err != nil {
		h.LogErr("adminConfirmPayment: set sub", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ فعال‌سازی اشتراک با خطا مواجه شد"})
	}
	if err := h.Store.ConfirmPayment(ctx, payID); err != nil {
		h.LogErr("adminConfirmPayment: mark confirmed", err)
	}
	if _, err := h.Bot.Send(&tele.User{ID: pay.TelegramID},
		fmt.Sprintf("✅ پرداخت شما تایید شد!\n💎 اشتراک «%s» به مدت %d روز فعال شد.", plan.Name, plan.Days)); err != nil {
		h.LogErr("adminConfirmPayment: notify user", err)
	}
	return c.Edit(fmt.Sprintf("✅ پرداخت تأیید شد و اشتراک «%s» (%d روز) برای کاربر فعال شد.", plan.Name, plan.Days))
}

// adminRejectPayment یک پرداخت دستی را رد می‌کند و به کاربر اطلاع می‌دهد
// (قبلاً این تابع هیچ‌چیزی در دیتابیس تغییر نمی‌داد — فقط پیام «رد شد» را
// به ادمین نشان می‌داد و پرداخت برای همیشه در وضعیت pending می‌ماند).
func (h *Handler) adminRejectPayment(ctx context.Context, c tele.Context, payID string) error {
	pay, err := h.Store.FindPayment(ctx, payID)
	h.LogErr("adminRejectPayment: find", err)
	if err := h.Store.RejectPayment(ctx, payID); err != nil {
		h.LogErr("adminRejectPayment: reject", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در ثبت رد پرداخت"})
	}
	if pay != nil {
		if _, err := h.Bot.Send(&tele.User{ID: pay.TelegramID},
			"❌ متاسفانه پرداخت شما تایید نشد. برای پیگیری با پشتیبانی تماس بگیرید."); err != nil {
			h.LogErr("adminRejectPayment: notify user", err)
		}
	}
	return c.Edit("🗑 پرداخت رد شد و به کاربر اطلاع داده شد.")
}

func (h *Handler) adminDoBackup(ctx context.Context, c tele.Context) error {
	s := h.Store.GetStats(ctx)
	if err := h.Store.CreateBackup(ctx, &models.Backup{
		TotalCodes: int(s.TotalCodes), TotalFiles: int(s.TotalFiles),
		CreatedBy: c.Sender().ID,
	}); err != nil {
		h.LogErr("adminDoBackup", err)
		return c.Send("❌ ساخت بکاپ با خطا مواجه شد.", kbAdmin())
	}
	return c.Send(fmt.Sprintf("💾 بکاپ ساخته شد.\n📄 %d رسانه | 📁 %d فایل", s.TotalCodes, s.TotalFiles), kbAdmin())
}

func (h *Handler) adminListAdmins(ctx context.Context, c tele.Context) error {
	admins, err := h.Store.ListAdmins(ctx)
	h.LogErr("adminListAdmins", err)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, a := range admins {
		label := fmt.Sprintf("👤 %d", a.TelegramID)
		if a.IsOwner {
			label += " (مالک)"
			rows = append(rows, kb.Row(kb.Data(label, "noop")))
			continue
		}
		rows = append(rows, kb.Row(
			kb.Data(label+" — 🔑 دسترسی‌ها", "aperm:"+strconv.FormatInt(a.TelegramID, 10)),
			kb.Data("🗑", "admin_del:"+strconv.FormatInt(a.TelegramID, 10)),
		))
	}
	rows = append(rows, kb.Row(kb.Data("➕ افزودن ادمین", "admin_add")))
	rows = append(rows, kb.Row(kb.Data(btnPanelLabel, "p:home")))
	kb.Inline(rows...)
	return sendOrEdit(c, fmt.Sprintf("👑 <b>ادمین‌ها</b> (%d):", len(admins)), tele.ModeHTML, kb)
}

// adminAskAdmin افزودن ادمین جدید.
func (h *Handler) adminAskAdmin(ctx context.Context, c tele.Context) error {
	h.SetStep(ctx, c.Sender().ID, stepAddAdmin)
	return c.Send("👑 آیدی عددی ادمین جدید را بفرستید:", kbCancelOnly())
}

// adminRemoveAdmin حذف یک ادمین.
func (h *Handler) adminRemoveAdmin(ctx context.Context, c tele.Context, tgIDStr string) error {
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return c.Edit(msgInvalid)
	}
	if err := h.Store.RemoveAdmin(ctx, tgID); err != nil {
		h.LogErr("adminRemoveAdmin", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ حذف ناموفق بود"})
	}
	return h.adminListAdmins(ctx, c)
}

func (h *Handler) adminSaveAdmin(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	tgID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return c.Send("❌ آیدی باید عددی باشد.", kbAdmin())
	}
	if err := h.Store.AddAdmin(ctx, tgID, ""); err != nil {
		h.LogErr("adminSaveAdmin", err)
		return c.Send("❌ افزودن ادمین با خطا مواجه شد.", kbAdmin())
	}
	return c.Send("✅ ادمین اضافه شد.", kbAdmin())
}

// notifyAdminsReport گزارش کاربر روی یک رسانه را با جزئیات کامل به ادمین‌ها می‌فرستد.
func (h *Handler) notifyAdminsReport(ctx context.Context, c tele.Context, codeStr string) {
	sender := c.Sender()
	code, err := h.Store.FindCode(ctx, codeStr)
	h.LogErr("notifyAdminsReport: find code", err)

	caption := "—"
	fileCount := 0
	uploader := int64(0)
	if code != nil {
		if code.Caption != "" {
			caption = code.Caption
			if len(caption) > 100 {
				caption = caption[:100] + "…"
			}
		}
		fileCount = len(code.FileIDs)
		uploader = code.UploaderID
	}

	uname := "—"
	if sender.Username != "" {
		uname = "@" + sender.Username
	}
	msg := fmt.Sprintf(
		"⚠️ <b>گزارش رسانه</b>\n\n"+
			"🔑 کد: <code>%s</code>\n"+
			"📦 تعداد فایل: %d\n"+
			"📝 کپشن: %s\n"+
			"⬆️ آپلودکننده: <code>%d</code>\n\n"+
			"👤 گزارش‌دهنده: %s (نام: %s)\n🆔 <code>%d</code>\n🔗 %s",
		codeStr, fileCount, caption, uploader,
		uname, sender.FirstName, sender.ID, h.deepLink(codeStr))

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	rows = append(rows, kb.Row(kb.Data("👁 مشاهده رسانه", "code_resend:"+codeStr)))
	if code != nil {
		rows = append(rows, kb.Row(kb.Data("🗑 حذف رسانه", "code_delete:"+code.ID)))
	}
	rows = append(rows, kb.Row(kb.Data("🚫 بلاک گزارش‌دهنده", "admin_user_block:"+strconv.FormatInt(sender.ID, 10))))
	kb.Inline(rows...)

	admins, err := h.Store.ListAdmins(ctx)
	h.LogErr("notifyAdminsReport: list admins", err)
	recipients := map[int64]bool{}
	for _, a := range admins {
		recipients[a.TelegramID] = true
	}
	if h.OwnerID != 0 {
		recipients[h.OwnerID] = true
	}
	for id := range recipients {
		if _, sendErr := h.Bot.Send(&tele.User{ID: id}, msg, tele.ModeHTML, kb); sendErr != nil {
			h.LogErr("notifyAdminsReport: send", sendErr)
		}
	}
}
