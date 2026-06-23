package tgbot

import (
	"context"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botpay/internal/i18n"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// onLanguage منوی انتخاب زبان را نمایش می‌دهد.
func (h *Handler) onLanguage(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)
	return c.Send(i18n.T(lang, i18n.KLanguageAsk), tele.ModeHTML, kbLanguage())
}

// onSetLang زبان انتخاب‌شده را ذخیره و رابط را به زبان جدید به‌روزرسانی می‌کند.
func (h *Handler) onSetLang(ctx context.Context, c tele.Context, code string) error {
	lang, ok := i18n.Parse(code)
	if !ok {
		return nil
	}

	// اطمینان از وجود کیف پول، سپس ذخیره‌ی زبان.
	w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: i18n.T(lang, i18n.KErrGeneric)})
	}
	if err := h.st.SetWalletLang(ctx, c.Sender().ID, string(lang)); err != nil {
		h.log.Error("set lang failed", ports.F("err", err))
		return c.Respond(&tele.CallbackResponse{Text: i18n.T(lang, i18n.KErrGeneric)})
	}

	// پیام منوی زبان را به تأیید تبدیل کن، سپس منوی اصلی را با کیبورد جدید بفرست.
	_ = c.Edit(i18n.T(lang, i18n.KLanguageChanged), tele.ModeHTML)
	return c.Send(
		i18n.T(lang, i18n.KStart, c.Sender().FirstName, w.BalanceTON(), w.CreditTON(), w.TotalTON()),
		tele.ModeHTML, kbMain(lang),
	)
}
