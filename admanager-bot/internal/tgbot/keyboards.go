package tgbot

import (
	tele "gopkg.in/telebot.v4"
)

// ── دکمه‌های ثابت منوی اصلی ─────────────────────────────────────
const (
	btnChannels    = "📡 کانال‌ها"
	btnCampaigns   = "📋 کمپین‌ها"
	btnNewCampaign = "➕ کمپین جدید"
	btnSchedule    = "🗓 زمان‌بندی"
	btnStats       = "📈 آمار"
	btnTemplates   = "🧩 قالب‌ها"
	btnSettings    = "⚙️ تنظیمات"
	btnHelp        = "❓ راهنما"

	btnCancel = "❌ لغو"
	btnBack   = "🔙 بازگشت"
)

// kbAdminMain منوی اصلی ادمین (Reply Keyboard).
func kbAdminMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnChannels), kb.Text(btnCampaigns)),
		kb.Row(kb.Text(btnNewCampaign), kb.Text(btnSchedule)),
		kb.Row(kb.Text(btnStats), kb.Text(btnTemplates)),
		kb.Row(kb.Text(btnSettings), kb.Text(btnHelp)),
	)
	return kb
}

// isMainMenuButton مشخص می‌کند متن یکی از دکمه‌های منوی اصلی است.
// برای اینکه دکمه‌های منو همیشه از هر wizard خارج شوند.
func isMainMenuButton(t string) bool {
	switch t {
	case btnChannels, btnCampaigns, btnNewCampaign, btnSchedule, btnStats, btnTemplates, btnSettings, btnHelp:
		return true
	}
	return false
}

// kbCancelOnly کیبورد تنها با دکمه‌ی لغو (هنگام مراحل state machine).
func kbCancelOnly() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnCancel)))
	return kb
}

// ── کمک‌کننده‌های inline ─────────────────────────────────────────

// inlineRows یک ReplyMarkup inline از ردیف‌های آماده می‌سازد.
func inlineRows(rows ...tele.Row) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(rows...)
	return kb
}

// cbBtn یک دکمه‌ی callback می‌سازد. کل payload به شکل "action:arg" در
// فیلد unique قرار می‌گیرد (هم‌خوان با parser در router.go).
func cbBtn(kb *tele.ReplyMarkup, text, payload string) tele.Btn {
	return kb.Data(text, payload)
}
