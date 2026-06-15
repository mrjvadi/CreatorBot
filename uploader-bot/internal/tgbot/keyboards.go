package tgbot

import (
	"fmt"

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
	btnBack      = "🔙 بازگشت"
	btnSearch    = "🔍 جستجو"
	btnHelp      = "❓ راهنما"
	btnSupport   = "💬 پشتیبانی"
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
		kb.Row(kb.Text(btnAdmins)),
	)
	return kb
}

// ── User Keyboard ─────────────────────────────────────────────

func kbUser() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnSearch)),
		kb.Row(kb.Text(btnHelp), kb.Text(btnSupport)),
	)
	return kb
}

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

func kbCodeSettings(codeID string, forwardLock, autoDelete bool, hasPassword bool, limit int) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	fwdLabel := "🔓 قفل فوروارد: خاموش"
	if forwardLock {
		fwdLabel = "🔒 قفل فوروارد: روشن"
	}
	adLabel := "⏱ ضدفیلتر: خاموش"
	if autoDelete {
		adLabel = "⏱ ضدفیلتر: روشن"
	}
	pwLabel := "🔐 رمز عبور: ندارد"
	if hasPassword {
		pwLabel = "🔐 رمز عبور: دارد"
	}
	limLabel := "📥 محدودیت: نامحدود"
	if limit > 0 {
		limLabel = fmt.Sprintf("📥 محدودیت: %d بار", limit)
	}

	kb.Inline(
		kb.Row(kb.Data(fwdLabel, "code_toggle_forward:"+codeID)),
		kb.Row(kb.Data(adLabel, "code_toggle_antidl:"+codeID)),
		kb.Row(kb.Data(pwLabel, "code_set_password:"+codeID)),
		kb.Row(kb.Data(limLabel, "code_set_limit:"+codeID)),
		kb.Row(
			kb.Data("✏️ ویرایش کپشن", "code_edit_caption:"+codeID),
			kb.Data("📤 پیش‌نمایش", "code_send_preview:"+codeID),
		),
		kb.Row(kb.Data("🗑 حذف", "code_delete:"+codeID)),
		kb.Row(kb.Data("🔙 بازگشت", "code_list")),
	)
	return kb
}

func kbFolderList(folders []folderItem) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, f := range folders {
		rows = append(rows, kb.Row(
			kb.Data("📁 "+f.Name, "folder_open:"+f.ID),
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

func kbSettings(settings map[string]string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	toggle := func(key, label string) tele.Row {
		val := settings[key]
		icon := "🔴"
		if val == "true" || val == "1" {
			icon = "🟢"
		}
		return kb.Row(kb.Data(icon+" "+label, "toggle_setting:"+key))
	}

	kb.Inline(
		toggle(models_SettingBotActive, "ربات فعال"),
		toggle(models_SettingSubRequired, "اشتراک اجباری"),
		toggle(models_SettingUserUpload, "آپلود کاربر"),
		toggle(models_SettingShowSearch, "جستجو"),
		toggle(models_SettingForwardLockDefault, "قفل فوروارد پیش‌فرض"),
		kb.Row(kb.Data("✏️ تعداد دانلود رایگان", "set_setting:"+models_SettingFreeDownloads)),
		kb.Row(kb.Data("✏️ زمان ضدفیلتر (ثانیه)", "set_setting:"+models_SettingAutoDeleteDefault)),
		kb.Row(kb.Data("✏️ امضا", "set_setting:"+models_SettingSignature)),
		kb.Row(kb.Data("✏️ متن خوش‌آمد", "set_setting:welcome_text")),
		kb.Row(kb.Data("🔙 بازگشت", "admin_main")),
	)
	return kb
}

// constants برای avoid circular import
const (
	models_SettingBotActive          = "bot_active"
	models_SettingSubRequired        = "sub_required"
	models_SettingUserUpload         = "user_upload"
	models_SettingShowSearch         = "show_search"
	models_SettingForwardLockDefault = "forward_lock_default"
	models_SettingFreeDownloads      = "free_downloads"
	models_SettingAutoDeleteDefault  = "auto_delete_default"
	models_SettingSignature          = "signature"
)
