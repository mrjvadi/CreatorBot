package tgbot

import (
	tele "gopkg.in/telebot.v4"
)

// ── دکمه‌های ثابت ────────────────────────────────────────────
const (
	btnNewCode   = "📤 آپلود رسانه"
	btnCodeList  = "📋 لیست رسانه‌ها"
	btnFolders   = "📁 پوشه‌ها"
	btnUsers     = "👥 کاربران"
	btnStats     = "📊 آمار"
	btnSettings  = "⚙️ تنظیمات"
	btnBroadcast = "📢 ارسال همگانی"
	btnBackup    = "💾 بکاپ"
	btnChannels  = "📡 کانال‌ها"
	btnSubPlans  = "💎 اشتراک‌ها"
	btnAdmins    = "👑 ادمین‌ها"
	btnCancel    = "❌ لغو"
	btnBack      = btnBackLabel
	btnSearch    = "🔍 جستجو"
	btnHelp      = "❓ راهنما"
	btnSupport   = "💬 پشتیبانی"
	btnPreview   = "🖼 کانال پیش‌نمایش"
	btnAds       = "📣 تبلیغات"
	btnReset     = "♻️ ریست دانلودها"
	btnPopular   = "🔥 پربازدیدها"
	btnNewest    = "🆕 جدیدترین‌ها"
	btnTop       = "⭐️ محبوب‌ترین‌ها"
	btnUploadU   = "📤 آپلود فایل"
)

// ── Admin Keyboard ────────────────────────────────────────────

func kbAdmin() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnNewCode), kb.Text(btnCodeList)),
		kb.Row(kb.Text(btnFolders), kb.Text(btnChannels)),
		kb.Row(kb.Text(btnUsers), kb.Text(btnStats)),
		kb.Row(kb.Text(btnSettings), kb.Text(btnBroadcast)),
		kb.Row(kb.Text(btnBackup), kb.Text(btnSubPlans)),
		kb.Row(kb.Text(btnPreview), kb.Text(btnAds)),
		kb.Row(kb.Text(btnReset), kb.Text(btnAdmins)),
	)
	return kb
}

// ── User Keyboard ─────────────────────────────────────────────
//
// نکته: منوی واقعیِ کاربر h.kbUserMenu(ctx) در user_menu.go است (دکمه‌ها و
// برچسب‌ها را از تنظیمات می‌خواند). این تابع ساده‌ی kbUser(bool) قبلاً جایگزین
// شده بود ولی هیچ‌جا صدا زده نمی‌شد؛ برای جلوگیری از سردرگمی حذف شد.

func kbCancelOnly() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnCancel)))
	return kb
}

// ── Album ─────────────────────────────────────────────────────

func kbAlbumDone() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text("✅ تمام شد")),
		kb.Row(kb.Text(btnCancel)),
	)
	return kb
}

// ── Code Settings Inline ──────────────────────────────────────
//
// نکته: منوی واقعیِ تنظیمات یک کد kbCodeAdvanced (در media_edit.go) است که
// کامل‌تر است (قفل کانال، اشتراک اجباری، سین/ری‌اکشن، آمار فیک، کاور، جابه‌جایی
// فایل‌ها، انتقال پوشه). این نسخه‌ی ساده‌تر قبلاً جایگزین شده بود ولی هیچ
// دکمه‌ای در کل ربات به آن اشاره نمی‌کرد؛ حذف شد تا برای توسعه‌دهنده‌ی بعدی
// گمراه‌کننده نباشد.

func kbFolderList(folders []folderItem) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, f := range folders {
		rows = append(rows, kb.Row(
			kb.Data("📁 "+f.Name, "afolder:"+f.ID),
			kb.Data("🗑", "folder_delete:"+f.ID),
		))
	}
	rows = append(rows, kb.Row(kb.Data("➕ پوشه جدید", "folder_new")))
	kb.Inline(rows...)
	return kb
}

type folderItem struct {
	ID   string
	Name string
}

// نکته: یک منوی تنظیمات مسطح (kbSettings) + adminToggleSetting/adminAskSetting
// این‌جا وجود داشت که با «toggle_setting:»/«set_setting:»/«admin_main» کار
// می‌کرد. بررسی کامل نشان داد هیچ دکمه‌ای در کل ربات این callbackها را صادر
// نمی‌کرد — منوی واقعی تنظیمات همان پنل دسته‌بندی‌شده‌ی kbSettingsHome/
// kbSettingsPage در admin_menu.go است («ps:»/«pt:»/«pv:»). این خوشه‌ی کد مرده
// (تابع کیبورد + دو هندلر + سه شاخه‌ی callback) حذف شد.
