package tgbot

import tele "gopkg.in/telebot.v4"

const (
	btnCancel    = "❌ انصراف"
	btnBack      = "🔙 بازگشت"
	btnDone      = "✅ تمام"
	btnConfirm   = "✅ تأیید"

	// کد - نوع
	btnOnce      = "1️⃣ یک‌بار"
	btnLimited   = "🔢 محدود"
	btnUnlimited = "♾ نامحدود"
	btnExpiry    = "⏰ زمان‌دار"

	// آلبوم
	btnAlbumMode = "📁 حالت آلبوم"
	btnSingleMode= "📄 فایل تکی"

	// تنظیمات
	btnSetWelcome    = "👋 متن خوشامد"
	btnSetNotMember  = "🚫 متن عدم عضویت"
	btnSetNotFound   = "❓ متن کد نیافت"
	btnSetBlocked    = "⛔️ متن بلاک"
	btnSetChannel    = "📢 کانال اجباری"

	// پنل ادمین
	btnNewCode   = "➕ کد جدید"
	btnCodeList  = "📋 لیست کدها"
	btnUsers     = "👥 کاربران"
	btnStats     = "📊 آمار"
	btnSettings  = "⚙️ تنظیمات"
	btnBroadcast = "📣 پیام همگانی"
)

func kbAdmin() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnNewCode), kb.Text(btnCodeList)),
		kb.Row(kb.Text(btnUsers), kb.Text(btnStats)),
		kb.Row(kb.Text(btnSettings), kb.Text(btnBroadcast)),
	)
	return kb
}

func kbUser() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text("❓ راهنما")))
	return kb
}

func kbCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnCancel)))
	return kb
}

func kbBackCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnBack), kb.Text(btnCancel)))
	return kb
}

func kbCodeType() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnOnce), kb.Text(btnUnlimited)),
		kb.Row(kb.Text(btnLimited), kb.Text(btnExpiry)),
		kb.Row(kb.Text(btnCancel)),
	)
	return kb
}

func kbAlbumDone() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnDone)),
		kb.Row(kb.Text(btnCancel)),
	)
	return kb
}

func kbSettings() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnSetWelcome), kb.Text(btnSetNotMember)),
		kb.Row(kb.Text(btnSetNotFound), kb.Text(btnSetBlocked)),
		kb.Row(kb.Text(btnSetChannel)),
		kb.Row(kb.Text(btnBack)),
	)
	return kb
}

func kbConfirmCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnConfirm), kb.Text(btnCancel)))
	return kb
}

// inline keyboard برای join channel
func kbJoinChannel(channelUsername, channelID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	if channelUsername != "" {
		kb.Inline(kb.Row(
			kb.URL("📢 عضویت در کانال", "https://t.me/"+channelUsername),
		))
	}
	_ = channelID
	return kb
}
