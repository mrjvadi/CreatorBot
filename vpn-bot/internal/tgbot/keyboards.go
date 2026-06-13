package tgbot

import (
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
)

// ── دکمه‌ها ────────────────────────────────────────────────

const (
	btnBuy      = "🛒 خرید اشتراک"
	btnMyVPN    = "🔑 اشتراک من"
	btnWallet   = "💳 کیف پول"
	btnSupport  = "🆘 پشتیبانی"
	btnHelp     = "❓ راهنما"
	btnCancel   = "❌ انصراف"
	btnBack     = "🔙 بازگشت"
	btnRenew    = "🔄 تمدید"
	btnGetLink  = "🔗 دریافت لینک"
	btnGetQR    = "📱 QR Code"
)

func kbMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnBuy), kb.Text(btnMyVPN)),
		kb.Row(kb.Text(btnWallet), kb.Text(btnSupport)),
		kb.Row(kb.Text(btnHelp)),
	)
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

// kbPlans inline keyboard برای لیست پلن‌ها.
func kbPlans(plans []models.Plan) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("📦 %s — %s — %.0f تومان",
			p.Name, fmtDuration(p.DurationDay), p.Price)
		rows = append(rows, kb.Row(kb.Data(label, "plan:"+p.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data(btnCancel, "cancel")))
	kb.Inline(rows...)
	return kb
}

// kbSubscription inline keyboard برای مدیریت اشتراک.
func kbSubscription(subID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(btnGetLink, "link:"+subID)),
		kb.Row(kb.Data(btnGetQR, "qr:"+subID)),
		kb.Row(kb.Data(btnRenew, "renew:"+subID)),
	)
	return kb
}

// kbPaymentGateway inline keyboard برای انتخاب روش پرداخت.
func kbPaymentGateway() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("💳 کارت به کارت", "gw:card")),
		kb.Row(kb.Data("🌐 زرین‌پال", "gw:zarinpal")),
		kb.Row(kb.Data("₿ NowPayments", "gw:nowpayments")),
		kb.Row(kb.Data(btnCancel, "cancel")),
	)
	return kb
}

// kbAdminMain panel ادمین.
func kbAdminMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text("📊 آمار"), kb.Text("👥 کاربران")),
		kb.Row(kb.Text("💰 پلن‌ها"), kb.Text("🖥 پنل‌ها")),
		kb.Row(kb.Text("💸 پرداخت‌ها"), kb.Text("📣 broadcast")),
		kb.Row(kb.Text(btnBack)),
	)
	return kb
}

// ── helpers ────────────────────────────────────────────────

func fmtDuration(days int) string {
	switch {
	case days < 30:
		return fmt.Sprintf("%d روزه", days)
	case days == 30:
		return "یک ماهه"
	case days == 60:
		return "دو ماهه"
	case days == 90:
		return "سه ماهه"
	default:
		return fmt.Sprintf("%d روزه", days)
	}
}
