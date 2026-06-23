package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botpay/internal/i18n"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
)

// onStart صفحه‌ی خوش‌آمد و موجودی کاربر.
func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KErrGeneric))
	}
	return c.Send(
		i18n.T(lang, i18n.KStart, c.Sender().FirstName, w.BalanceTON(), w.CreditTON(), w.TotalTON()),
		tele.ModeHTML, kbMain(lang),
	)
}

// onWallet نمایش جزئیات کیف پول و آدرس واریز.
func (h *Handler) onWallet(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KErrGeneric), kbMain(lang))
	}

	frozen := ""
	if w.Frozen > 0 {
		frozen = i18n.T(lang, i18n.KWalletFrozen, wallet.NanoToTON(w.Frozen))
	}

	return c.Send(
		i18n.T(lang, i18n.KWallet, w.BalanceTON(), w.CreditTON(), w.TotalTON(), frozen, w.TONAddress, w.PayHandle),
		tele.ModeHTML, kbMain(lang),
	)
}

// onDeposit ساخت فاکتور واریز و نمایش دستورالعمل.
func (h *Handler) onDeposit(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	code, payURL, err := h.wallet.DepositInstructions(ctx, c.Sender().ID, 0, "manual", "direct")
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KDepositErr), kbMain(lang))
	}

	return c.Send(
		i18n.T(lang, i18n.KDepositBody, code),
		tele.ModeHTML, kbDeposit(lang, payURL, code),
	)
}

// onWithdraw شروع گفتگوی برداشت.
func (h *Handler) onWithdraw(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KErrGeneric), kbMain(lang))
	}

	minTON := wallet.NanoToTON(wallet.MinWithdrawNano)
	feeTON := wallet.NanoToTON(wallet.NetworkFeeNano)

	if w.TotalTON() < minTON+feeTON {
		return c.Send(i18n.T(lang, i18n.KWithdrawInsufficient, minTON, feeTON), tele.ModeHTML, kbMain(lang))
	}

	h.states.set(c.Sender().ID, convState{kind: kindWithdraw, step: "addr"})
	return c.Send(
		i18n.T(lang, i18n.KWithdrawAskAddr, w.TotalTON(), feeTON),
		tele.ModeHTML, kbCancelOnly(lang),
	)
}

// onHistory نمایش ۱۵ تراکنش اخیر.
func (h *Handler) onHistory(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KErrGeneric), kbMain(lang))
	}

	txs, err := h.st.ListTransactions(ctx, w.ID, 15)
	if err != nil || len(txs) == 0 {
		return c.Send(i18n.T(lang, i18n.KHistoryEmpty), kbMain(lang))
	}

	var b strings.Builder
	b.WriteString(i18n.T(lang, i18n.KHistoryTitle))
	for _, tx := range txs {
		icon, labelKey := txMeta(tx.Type)
		amt := wallet.NanoToTON(tx.Amount)
		sign := "+"
		if amt < 0 {
			sign = ""
			amt = -amt
		}
		status := ""
		if tx.Status == store.TxPending {
			status = " ⏳"
		}
		desc := tx.Description
		if desc == "" {
			desc = i18n.T(lang, labelKey)
		}
		desc = truncate(desc, 25)
		b.WriteString("\n")
		b.WriteString(i18n.T(lang, i18n.KHistoryLine,
			icon, sign, amt, status, tx.CreatedAt.Format("01/02 15:04"), desc))
	}

	return c.Send(b.String(), tele.ModeHTML, kbMain(lang))
}

// onHelp راهنمای ربات.
func (h *Handler) onHelp(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)
	return c.Send(i18n.T(lang, i18n.KHelp), tele.ModeHTML, kbMain(lang))
}

// onTransfer شروع گفتگوی انتقال داخلی.
func (h *Handler) onTransfer(c tele.Context) error {
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	w, err := h.wallet.GetOrCreate(ctx, c.Sender().ID)
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KErrGeneric), kbMain(lang))
	}
	if w.TotalTON() <= 0 {
		return c.Send(i18n.T(lang, i18n.KTransferInsufficient), kbMain(lang))
	}

	h.states.set(c.Sender().ID, convState{kind: kindTransfer, step: "to_id"})
	return c.Send(i18n.T(lang, i18n.KTransferAskID), tele.ModeHTML, kbCancelOnly(lang))
}

// txMeta آیکن و کلید برچسب نوع تراکنش را برمی‌گرداند.
func txMeta(t store.TxType) (icon, labelKey string) {
	switch t {
	case store.TxDeposit:
		return "📥", i18n.KTxDeposit
	case store.TxWithdraw:
		return "📤", i18n.KTxWithdraw
	case store.TxCreditAdd:
		return "🎁", i18n.KTxCreditAdd
	case store.TxPayment:
		return "💸", i18n.KTxPayment
	case store.TxRefund:
		return "↩️", i18n.KTxRefund
	}
	return "💰", i18n.KTxPayment
}

// truncate رشته را به حداکثر n کاراکتر (با درنظرگرفتن rune) کوتاه می‌کند.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
