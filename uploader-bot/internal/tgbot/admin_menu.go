package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// پنل مدیریت inline و حرفه‌ای — منوی شیشه‌ای تو‌در‌تو.
// همه‌ی callbackها با پیشوندهای زیر مدیریت می‌شوند:
//   p:<section>            باز کردن یک بخش اصلی
//   ps:<page>             باز کردن یک صفحه‌ی تنظیمات
//   pt:<page>:<key>       برعکس‌کردن یک تنظیم بولی و رفرش همان صفحه
//   pv:<page>:<key>       پرسیدن مقدار جدید یک تنظیم متنی/عددی

// ── منوی اصلی ─────────────────────────────────────────────────

func (h *Handler) panelHomeText(ctx context.Context) string {
	st := h.Store.GetStats(ctx)
	status := "🟢 روشن"
	if h.Store.GetSetting(ctx, models.SettingBotActive) == "false" {
		status = "🔴 خاموش"
	}
	prefix := h.Store.GetSetting(ctx, models.SettingCodePrefix)
	return fmt.Sprintf(
		"👑 <b>پنل مدیریت</b>\n\n"+
			"وضعیت ربات: %s\nپیشوند کدها: <code>%s_</code>\n\n"+
			"👥 کاربران: <b>%d</b>   📄 رسانه‌ها: <b>%d</b>\n"+
			"📁 فایل‌ها: <b>%d</b>   💎 اشتراک فعال: <b>%d</b>",
		status, prefix, st.TotalUsers, st.TotalCodes, st.TotalFiles, st.ActiveSubs)
}

func kbPanelHome(botActive bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	toggleLabel := "🔴 خاموش‌کردن ربات"
	if !botActive {
		toggleLabel = "🟢 روشن‌کردن ربات"
	}
	kb.Inline(
		kb.Row(kb.Data("📤 آپلود رسانه", "p:upload"), kb.Data("📋 رسانه‌ها", "p:codes")),
		kb.Row(kb.Data("📁 پوشه‌ها", "p:folders"), kb.Data("🔎 جستجو", "p:search")),
		kb.Row(kb.Data("👥 کاربران", "p:users"), kb.Data("📊 آمار", "p:stats")),
		kb.Row(kb.Data("📡 جوین اجباری", "p:fjoin"), kb.Data("🖼 پیش‌نمایش", "p:preview")),
		kb.Row(kb.Data("💎 اشتراک‌ها", "p:plans"), kb.Data("📣 تبلیغات", "p:ads")),
		kb.Row(kb.Data("📢 ارسال همگانی", "p:bc"), kb.Data("⏳ صف تایید", "p:pending")),
		kb.Row(kb.Data("♻️ ریست دانلودها", "p:reset"), kb.Data("📊 وضعیت همگانی", "p:bcstat")),
		kb.Row(kb.Data("💾 بکاپ", "p:backup"), kb.Data("👑 ادمین‌ها", "p:admins")),
		kb.Row(kb.Data("⚙️ تنظیمات", "p:set"), kb.Data("🧰 ابزارها", "p:tools")),
		kb.Row(kb.Data(toggleLabel, "p:togglebot")),
	)
	return kb
}

// OpenPanel پنل اصلی را به‌صورت پیام جدید نشان می‌دهد (برای /start و /panel).
func (h *Handler) OpenPanel(ctx context.Context, c tele.Context) error {
	active := h.Store.GetSetting(ctx, models.SettingBotActive) != "false"
	return c.Send(h.panelHomeText(ctx), tele.ModeHTML, kbPanelHome(active))
}

func (h *Handler) editPanelHome(ctx context.Context, c tele.Context) error {
	active := h.Store.GetSetting(ctx, models.SettingBotActive) != "false"
	return c.Edit(h.panelHomeText(ctx), tele.ModeHTML, kbPanelHome(active))
}

// ── صفحات تنظیمات ─────────────────────────────────────────────

type settingItem struct {
	key    string
	label  string
	toggle bool // true=بولی، false=مقدار متنی/عددی
}

type settingsPage struct {
	title string
	items []settingItem
}

func settingsPages() map[string]settingsPage {
	return map[string]settingsPage{
		"content": {"📦 آپلود و محتوا", []settingItem{
			{models.SettingForwardLockDefault, "قفل فوروارد پیش‌فرض", true},
			{models.SettingAntiFilterDefault, "ضدفیلتر (حذف خودکار) پیش‌فرض", true},
			{models.SettingRemoveLinks, "حذف لینک از کپشن", true},
			{models.SettingSignatureEnabled, "نمایش امضا", true},
			{models.SettingSignature, "متن امضا", false},
			{models.SettingThumbUploadDefault, "آپلود تامبنیل هنگام آپلود", true},
			{models.SettingVideoThumbDefault, "کاور پیش‌فرض ویدیو", true},
		}},
		"antidel": {"⏱ ضدفیلتر / حذف خودکار", []settingItem{
			{models.SettingAutoDeleteDefault, "زمان حذف پیش‌فرض (ثانیه)", false},
			{models.SettingAutoDeleteWarnOff, "خاموش‌کردن پیام هشدار", true},
			{models.SettingAutoDeleteWarnKeep, "هشدار پاک نشود", true},
			{models.SettingAutoDeleteWarn, "متن هشدار ({sec}=ثانیه)", false},
		}},
		"display": {"🎛 نمایش و دکمه‌ها", []settingItem{
			{models.SettingShowSearch, "دکمه جستجو", true},
			{models.SettingInlineSearch, "جستجوی اینلاین", true},
			{models.SettingShowLikesButtons, "لایک/دیسلایک", true},
			{models.SettingShowReportButton, "دکمه گزارش", true},
			{models.SettingShowResendButton, "دکمه ارسال مجدد", true},
			{models.SettingShowComment, "دکمه نظر", true},
			{models.SettingBtnNewest, "دکمه جدیدترین‌ها", true},
			{models.SettingBtnPopular, "دکمه پربازدیدها", true},
			{models.SettingBtnTop, "دکمه محبوب‌ترین‌ها", true},
		}},
		"access": {"🔐 دسترسی و آپلود کاربر", []settingItem{
			{models.SettingBotActive, "ربات فعال", true},
			{models.SettingUserUpload, "آپلود توسط کاربر", true},
			{models.SettingAutoApproveFiles, "تایید خودکار فایل کاربر", true},
			{models.SettingSubRequired, "اشتراک اجباری", true},
			{models.SettingFreeDownloads, "تعداد دانلود رایگان", false},
			{models.SettingSpamDelay, "زمان اسپم (ثانیه)", false},
			{models.SettingStorageChannel, "کانال ذخیره‌سازی (آیدی -100…)", false},
		}},
		"labels": {"🔤 نام دکمه‌ها", []settingItem{
			{models.SettingLblNewest, "دکمه جدیدترین‌ها", false},
			{models.SettingLblPopular, "دکمه پربازدیدها", false},
			{models.SettingLblTop, "دکمه محبوب‌ترین‌ها", false},
			{models.SettingLblSearch, "دکمه جستجو", false},
			{models.SettingLblUpload, "دکمه آپلود", false},
			{models.SettingLblHelp, "دکمه راهنما", false},
			{models.SettingLblSupport, "دکمه پشتیبانی", false},
		}},
		"texts": {"📝 متن‌ها", []settingItem{
			{models.SettingWelcomeText, "متن خوش‌آمد", false},
			{models.SettingNotMemberText, "متن جوین اجباری", false},
			{models.SettingPasswordText, "متن درخواست رمز", false},
			{models.SettingSubRequiredText, "متن نیاز به اشتراک", false},
			{models.SettingNotFoundText, "متن کد یافت نشد", false},
			{models.SettingSupportText, "متن پشتیبانی", false},
			{models.SettingHelpText, "متن راهنما", false},
			{models.SettingStartButtons, "دکمه‌های شروع (برچسب|لینک)", false},
		}},
		"bc": {"📢 ارسال همگانی", []settingItem{
			{models.SettingBroadcastPin, "پین پیام همگانی", true},
			{models.SettingBroadcastAutoDelete, "حذف خودکار پیام همگانی", true},
			{models.SettingBroadcastDeleteHours, "زمان حذف (ساعت)", false},
			{models.SettingBroadcastDelayMS, "فاصله بین ارسال‌ها (میلی‌ثانیه)", false},
		}},
		"pay": {"💳 پرداخت", []settingItem{
			{models.SettingPaymentCard, "کارت‌به‌کارت", true},
			{models.SettingPaymentZarinpal, "زرین‌پال", true},
			{models.SettingPaymentZibal, "زیبال", true},
			{models.SettingPaymentTON, "TON", true},
			{models.SettingPaymentTRON, "TRON", true},
			{models.SettingPaymentStars, "Stars", true},
			{models.SettingActiveGateway, "درگاه فعال (zarinpal/zibal)", false},
			{models.SettingZarinpalMerchant, "مرچنت زرین‌پال", false},
			{models.SettingZibalMerchant, "مرچنت زیبال", false},
			{models.SettingCardNumber, "شماره کارت", false},
			{models.SettingCardHolder, "نام صاحب کارت", false},
			{models.SettingTONWallet, "ولت TON", false},
			{models.SettingTRONWallet, "ولت TRON", false},
		}},
	}
}

// kbSettingsHome دسته‌بندی تنظیمات.
func kbSettingsHome() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("📦 آپلود و محتوا", "ps:content"), kb.Data("⏱ ضدفیلتر", "ps:antidel")),
		kb.Row(kb.Data("🎛 نمایش و دکمه‌ها", "ps:display"), kb.Data("🔐 دسترسی", "ps:access")),
		kb.Row(kb.Data("📝 متن‌ها", "ps:texts"), kb.Data("🔤 نام دکمه‌ها", "ps:labels")),
		kb.Row(kb.Data("📢 همگانی", "ps:bc"), kb.Data("💳 پرداخت", "ps:pay")),
		kb.Row(kb.Data(btnBackLabel, "p:home")),
	)
	return kb
}

// kbSettingsPage کیبورد یک صفحه‌ی تنظیمات با وضعیت زنده.
func (h *Handler) kbSettingsPage(ctx context.Context, page string) (*tele.ReplyMarkup, string) {
	p, ok := settingsPages()[page]
	if !ok {
		return kbSettingsHome(), "⚙️ تنظیمات"
	}
	all := h.Store.GetAllSettings(ctx)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, it := range p.items {
		if it.toggle {
			icon := "🔴"
			if all[it.key] == "true" || all[it.key] == "1" {
				icon = "🟢"
			}
			rows = append(rows, kb.Row(kb.Data(icon+" "+it.label, "pt:"+page+":"+it.key)))
		} else {
			val := all[it.key]
			if len(val) > 18 {
				val = val[:18] + "…"
			}
			lbl := "✏️ " + it.label
			if val != "" {
				lbl += " : " + val
			}
			rows = append(rows, kb.Row(kb.Data(lbl, "pv:"+page+":"+it.key)))
		}
	}
	rows = append(rows, kb.Row(kb.Data(btnBackLabel, "p:set")))
	kb.Inline(rows...)
	return kb, p.title
}

// ── Dispatcher ────────────────────────────────────────────────

// handlePanel همه‌ی callbackهای پنل (p / ps / pt / pv) را مدیریت می‌کند.
// خروجی true یعنی این callback متعلق به پنل بود و پردازش شد.
func (h *Handler) handlePanel(ctx context.Context, c tele.Context, action, arg, arg2 string) (bool, error) {
	if !h.isAdmin(c) {
		return true, c.Respond(&tele.CallbackResponse{Text: "⛔️ دسترسی ندارید"})
	}

	switch action {
	case "p":
		return true, h.panelSection(ctx, c, arg)
	case "ps":
		kb, title := h.kbSettingsPage(ctx, arg)
		return true, c.Edit("⚙️ "+title, kb)
	case "pt":
		h.adminToggleSettingRaw(ctx, arg2)
		kb, title := h.kbSettingsPage(ctx, arg)
		return true, c.Edit("⚙️ "+title, kb)
	case "pv":
		h.SetStepData(ctx, c.Sender().ID, stepEditSetting, "key", arg2)
		h.SetStepData(ctx, c.Sender().ID, stepEditSetting, "page", arg)
		return true, c.Send("✏️ مقدار جدید را بفرستید (برای خالی‌کردن «حذف» بفرستید):", kbCancelOnly())
	}
	return false, nil
}

// panelSection بخش‌های اصلی منو.
func (h *Handler) panelSection(ctx context.Context, c tele.Context, sec string) error {
	// کنترل دسترسی بر اساس نقش ادمین
	if p := sectionPerm(sec); p != "" && !h.adminCan(ctx, c, p) {
		return c.Respond(&tele.CallbackResponse{Text: "⛔️ دسترسی به این بخش را ندارید", ShowAlert: true})
	}
	switch sec {
	case "home":
		return h.editPanelHome(ctx, c)
	case "set":
		return c.Edit("⚙️ <b>تنظیمات</b> — یک دسته را انتخاب کنید:", tele.ModeHTML, kbSettingsHome())
	case "togglebot":
		cur := h.Store.GetSetting(ctx, models.SettingBotActive)
		next := "true"
		if cur != "false" {
			next = "false"
		}
		if err := h.Store.SetSetting(ctx, models.SettingBotActive, next); err != nil {
			h.LogErr("panelSection: togglebot", err)
			return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد"})
		}
		return h.editPanelHome(ctx, c)
	case "upload":
		h.SetStep(ctx, c.Sender().ID, stepCodeFiles)
		return c.Send("📤 فایل‌ها را بفرستید، سپس «✅ تمام شد» را بزنید.", kbAlbumDone())
	case "codes":
		return h.adminListCodes(ctx, c)
	case "folders":
		return h.adminListFolders(ctx, c)
	case "search":
		h.SetStep(ctx, c.Sender().ID, stepSearch)
		return c.Send("🔎 عبارت جستجو (کد یا کپشن) را بفرستید:", kbCancelOnly())
	case "users":
		h.SetStep(ctx, c.Sender().ID, stepSearchUser)
		return c.Send("👤 آیدی عددی یا یوزرنیم کاربر را بفرستید:", kbCancelOnly())
	case "stats":
		return h.adminShowStats(ctx, c)
	case "fjoin":
		return h.lockList(ctx, c)
	case "preview":
		return h.adminListPreview(ctx, c)
	case "plans":
		return h.adminListPlans(ctx, c)
	case "ads":
		return h.adminListAds(ctx, c)
	case "bc":
		return h.adminBroadcastMenu(ctx, c)
	case "backup":
		return h.adminBackupMenu(ctx, c)
	case "admins":
		return h.adminListAdmins(ctx, c)
	case "reset":
		return h.adminResetDownloads(ctx, c)
	case "tools":
		return h.panelTools(ctx, c)
	case "pending":
		return h.adminPendingList(ctx, c)
	case "bcstat":
		return h.adminBroadcastStatus(ctx, c)
	}
	return c.Respond()
}
