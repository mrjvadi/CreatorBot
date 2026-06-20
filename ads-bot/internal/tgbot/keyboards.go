package tgbot

import (
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
)

const (
	btnNewCampaign  = "➕ کمپین جدید"
	btnMyCampaigns  = "📊 کمپین‌های من"
	btnMyChannels   = "📢 کانال‌های من"
	btnAddChannel   = "➕ افزودن کانال"
	btnBalance      = "💰 موجودی"
	btnHelp         = "❓ راهنما"
	btnCancel       = "❌ انصراف"
	btnBack         = "🔙 بازگشت"
	btnSkip         = "⏭ رد کردن"
)

func kbMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnNewCampaign), kb.Text(btnMyCampaigns)),
		kb.Row(kb.Text(btnMyChannels), kb.Text(btnAddChannel)),
		kb.Row(kb.Text(btnBalance), kb.Text(btnHelp)),
	)
	return kb
}

func kbCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnCancel)))
	return kb
}

func kbSkipCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnSkip), kb.Text(btnCancel)))
	return kb
}

func kbAdminMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text("📊 آمار"), kb.Text("⏳ در انتظار تأیید")),
		kb.Row(kb.Text("📢 کانال‌ها"), kb.Text("📣 broadcast")),
		kb.Row(kb.Text(btnBack)),
	)
	return kb
}

func kbCampaignActions(campID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("⏸ توقف", "camp_pause:"+campID)),
		kb.Row(kb.Data("🗑 حذف", "camp_del:"+campID)),
	)
	return kb
}

func kbReview(campID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("✅ تأیید", "approve:"+campID),
			kb.Data("❌ رد", "reject:"+campID),
		),
	)
	return kb
}

func kbRentReview(rentalID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("✅ تأیید اجاره", "rent_approve:"+rentalID),
			kb.Data("❌ رد", "rent_reject:"+rentalID),
		),
	)
	return kb
}

func statusLabel(s store.CampaignStatus) string {
	m := map[store.CampaignStatus]string{
		store.CampaignDraft:    "📝 پیش‌نویس",
		store.CampaignPending:  "⏳ منتظر تأیید",
		store.CampaignActive:   "🟢 فعال",
		store.CampaignPaused:   "⏸ متوقف",
		store.CampaignDone:     "✅ تمام",
		store.CampaignRejected: "❌ رد شده",
	}
	if l, ok := m[s]; ok { return l }
	return string(s)
}

func fmtCampaign(c store.Campaign) string {
	return fmt.Sprintf(
		"%s <b>%s</b>\n"+
			"💰 بودجه: %.2f TON | خرج: %.2f TON\n"+
			"👥 عضو: %d | CPJ: %.2f TON\n"+
			"🆔 <code>%s</code>",
		statusLabel(c.Status), c.Name,
		c.Budget, c.Spent,
		c.TotalJoins, c.CPJ,
		c.ID,
	)
}
