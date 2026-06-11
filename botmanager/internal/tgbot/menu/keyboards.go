// Package menu همه keyboard های ربات botmanager را تعریف می‌کند.
// از ReplyKeyboard (دکمه عادی) استفاده می‌شود نه InlineKeyboard.
package menu

import tele "gopkg.in/telebot.v4"

// ── دکمه‌های ثابت ───────────────────────────────────────────

// ادمین - منوی اصلی
const (
	BtnServers   = "🖥 سرورها"
	BtnTemplates = "📦 تمپلیت‌ها"
	BtnPlans     = "💰 پلن‌ها"
	BtnBots      = "🤖 ربات‌ها"
	BtnLinks     = "🔗 لینک‌های دعوت"
	BtnUsers     = "👥 کاربران"
	BtnStats     = "📊 آمار"
	BtnBack      = "🔙 بازگشت"
	BtnCancel    = "❌ انصراف"
	BtnConfirm   = "✅ تأیید"
)

// کاربر - منوی اصلی
const (
	BtnMyBots   = "🤖 ربات‌های من"
	BtnMyPlan   = "💳 اشتراک من"
	BtnHelp     = "❓ راهنما"
	BtnContact  = "📞 پشتیبانی"
)

// ── Keyboards ───────────────────────────────────────────────

// MainAdmin منوی اصلی ادمین.
func MainAdmin() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(BtnBots), kb.Text(BtnLinks)),
		kb.Row(kb.Text(BtnServers), kb.Text(BtnTemplates)),
		kb.Row(kb.Text(BtnPlans), kb.Text(BtnUsers)),
		kb.Row(kb.Text(BtnStats)),
	)
	return kb
}

// MainUser منوی اصلی کاربر عادی.
func MainUser() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(BtnMyBots)),
		kb.Row(kb.Text(BtnHelp), kb.Text(BtnContact)),
	)
	return kb
}

// BackOnly فقط دکمه بازگشت.
func BackOnly() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(BtnBack)))
	return kb
}

// BackCancel دکمه بازگشت + انصراف.
func BackCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(BtnBack), kb.Text(BtnCancel)))
	return kb
}

// ConfirmCancel تأیید + انصراف.
func ConfirmCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(BtnConfirm), kb.Text(BtnCancel)))
	return kb
}

// Remove keyboard را حذف می‌کند.
func Remove() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{RemoveKeyboard: true}
}

// BotTypeSelect انتخاب نوع ربات.
func BotTypeSelect() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text("📤 Uploader"), kb.Text("🔒 VPN")),
		kb.Row(kb.Text("📂 Archive"), kb.Text("👥 Member")),
		kb.Row(kb.Text(BtnCancel)),
	)
	return kb
}

// LinkLimitSelect انتخاب محدودیت لینک.
func LinkLimitSelect() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text("1️⃣ یک‌بار"), kb.Text("3️⃣ سه‌بار")),
		kb.Row(kb.Text("5️⃣ پنج‌بار"), kb.Text("♾ نامحدود")),
		kb.Row(kb.Text(BtnCancel)),
	)
	return kb
}

// UserRoleSelect انتخاب نقش کاربر.
func UserRoleSelect() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text("👤 User"), kb.Text("🛡 Admin")),
		kb.Row(kb.Text(BtnCancel)),
	)
	return kb
}
