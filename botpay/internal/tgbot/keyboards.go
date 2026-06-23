package tgbot

import (
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botpay/internal/i18n"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
)

// kbMain کیبورد اصلی reply را با برچسب‌های ترجمه‌شده می‌سازد.
func kbMain(lang i18n.Lang) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(i18n.T(lang, i18n.KBtnWallet))),
		kb.Row(kb.Text(i18n.T(lang, i18n.KBtnDeposit)), kb.Text(i18n.T(lang, i18n.KBtnWithdraw))),
		kb.Row(kb.Text(i18n.T(lang, i18n.KBtnTransfer)), kb.Text(i18n.T(lang, i18n.KBtnHistory))),
		kb.Row(kb.Text(i18n.T(lang, i18n.KBtnHelp)), kb.Text(i18n.T(lang, i18n.KBtnLanguage))),
	)
	return kb
}

// kbCancelOnly کیبوردی فقط با دکمه‌ی انصراف.
func kbCancelOnly(lang i18n.Lang) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(i18n.T(lang, i18n.KBtnCancel))))
	return kb
}

// kbLanguage کیبورد inline انتخاب زبان — هر زبان با نام بومی خودش.
// قالب callback: "set_lang:<code>" (سازگار با پارسر onCallback).
func kbLanguage() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, l := range i18n.Supported() {
		rows = append(rows, kb.Row(kb.Data(i18n.Name(l), "set_lang:"+string(l))))
	}
	kb.Inline(rows...)
	return kb
}

// kbDeposit کیبورد inline صفحه‌ی واریز.
// قالب callback بررسی: "check_deposit:<code>".
func kbDeposit(lang i18n.Lang, payURL, code string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.URL(i18n.T(lang, i18n.KBtnOpenWallet), payURL)),
		kb.Row(kb.Data(i18n.T(lang, i18n.KBtnCheckDeposit), "check_deposit:"+code)),
	)
	return kb
}

// kbAdminMenu منوی اصلی پنل ادمین (inline).
func kbAdminMenu(lang i18n.Lang) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(i18n.T(lang, i18n.KBtnAdminWithdraws), "adm:withdraws")),
		kb.Row(
			kb.Data(i18n.T(lang, i18n.KBtnAdminCredit), "adm:credit"),
			kb.Data(i18n.T(lang, i18n.KBtnAdminRefresh), "adm:stats"),
		),
	)
	return kb
}

// kbWithdrawList فهرست برداشت‌های منتظر را با دکمه‌های تأیید/رد برای هر مورد می‌سازد.
func kbWithdrawList(lang i18n.Lang, reqs []store.WithdrawRequest) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, r := range reqs {
		id := r.ID.String()
		short := id
		if len(short) > 6 {
			short = short[:6]
		}
		rows = append(rows, kb.Row(
			kb.Data("✅ "+short, "wd:approve:"+id),
			kb.Data("🚫 "+short, "wd:reject:"+id),
		))
	}
	rows = append(rows, kb.Row(kb.Data(i18n.T(lang, i18n.KBtnAdminBack), "adm:menu")))
	kb.Inline(rows...)
	return kb
}

// adminWithdrawText متن یک مورد برداشت را برای فهرست ادمین قالب‌بندی می‌کند.
func adminWithdrawText(lang i18n.Lang, r store.WithdrawRequest) string {
	return i18n.T(lang, i18n.KAdminWithdrawItem,
		r.ID, wallet.NanoToTON(r.Amount), wallet.NanoToTON(r.Fee),
		r.ToAddress, r.CreatedAt.Format("01/02 15:04"))
}

// isCancel بررسی می‌کند متن ورودی، دستور انصراف باشد (در همه‌ی زبان‌ها).
func isCancel(text string) bool {
	if text == "/cancel" {
		return true
	}
	for _, l := range i18n.Supported() {
		if text == i18n.T(l, i18n.KBtnCancel) {
			return true
		}
	}
	return false
}
