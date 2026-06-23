package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botpay/internal/i18n"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
)

// onCallback همه‌ی callbackهای inline را مسیریابی می‌کند.
// قالب data: "<action>[:<arg>[:<arg2>]]" (با حذف بایت کنترلی \f ابتدای آن).
func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	defer c.Respond()

	data := strings.TrimPrefix(c.Callback().Data, "\f")
	parts := strings.Split(data, ":")
	action := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch action {
	case "check_deposit":
		return h.onCheckDeposit(ctx, c, arg)
	case "set_lang":
		return h.onSetLang(ctx, c, arg)
	case "adm":
		return h.onAdminCallback(ctx, c, arg)
	case "wd":
		return h.onWithdrawCallback(ctx, c, arg, parts)
	}
	return nil
}

// onAdminCallback اکشن‌های منوی ادمین را مسیریابی می‌کند (فقط برای مالک).
func (h *Handler) onAdminCallback(ctx context.Context, c tele.Context, sub string) error {
	if !h.isAdmin(c) {
		return nil
	}
	switch sub {
	case "stats", "menu":
		return h.onAdminHome(ctx, c)
	case "withdraws":
		return h.onAdminWithdrawsList(ctx, c)
	case "credit":
		return h.onAdminCreditStart(ctx, c)
	}
	return nil
}

// onWithdrawCallback تأیید/رد یک برداشت از طریق دکمه را آغاز می‌کند.
func (h *Handler) onWithdrawCallback(ctx context.Context, c tele.Context, op string, parts []string) error {
	if !h.isAdmin(c) || len(parts) < 3 {
		return nil
	}
	id := parts[2]
	switch op {
	case "approve":
		return h.onAdminApproveStart(ctx, c, id)
	case "reject":
		return h.onAdminRejectStart(ctx, c, id)
	}
	return nil
}

// onCheckDeposit وضعیت یک invoice واریز را بررسی می‌کند.
func (h *Handler) onCheckDeposit(ctx context.Context, c tele.Context, code string) error {
	lang := h.langOf(ctx, c)
	if code == "" {
		return nil
	}

	inv, _ := h.st.FindInvoiceByCode(ctx, code)
	if inv == nil {
		return c.Edit(i18n.T(lang, i18n.KCheckPending), tele.ModeHTML)
	}
	if inv.Status == store.InvoicePaid {
		w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
		newBalance := wallet.NanoToTON(inv.Amount)
		if err == nil && w != nil {
			newBalance = w.TotalTON()
		}
		return c.Edit(
			i18n.T(lang, i18n.KCheckConfirmed, wallet.NanoToTON(inv.Amount), newBalance),
			tele.ModeHTML,
		)
	}
	return c.Edit(i18n.T(lang, i18n.KCheckUnconfirmed), tele.ModeHTML)
}
