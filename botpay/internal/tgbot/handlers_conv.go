package tgbot

import (
	"context"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botpay/internal/i18n"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
)

// onText ورودی متنی را مسیریابی می‌کند: ابتدا گفتگوهای فعال، سپس دکمه‌های منو.
func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())
	lang := h.langOf(ctx, c)

	if st, ok := h.states.get(uid); ok {
		switch st.kind {
		case kindWithdraw:
			return h.handleWithdrawState(ctx, c, lang, st, text)
		case kindTransfer:
			return h.handleTransferState(ctx, c, lang, st, text)
		case kindAdminApprove:
			return h.handleAdminApprove(ctx, c, lang, st, text)
		case kindAdminReject:
			return h.handleAdminReject(ctx, c, lang, st, text)
		case kindAdminCredit:
			return h.handleAdminCredit(ctx, c, lang, st, text)
		}
	}

	switch text {
	case i18n.T(lang, i18n.KBtnWallet):
		return h.onWallet(c)
	case i18n.T(lang, i18n.KBtnDeposit):
		return h.onDeposit(c)
	case i18n.T(lang, i18n.KBtnWithdraw):
		return h.onWithdraw(c)
	case i18n.T(lang, i18n.KBtnTransfer):
		return h.onTransfer(c)
	case i18n.T(lang, i18n.KBtnHistory):
		return h.onHistory(c)
	case i18n.T(lang, i18n.KBtnHelp):
		return h.onHelp(c)
	case i18n.T(lang, i18n.KBtnLanguage):
		return h.onLanguage(c)
	}
	return nil
}

// handleWithdrawState مراحل گفتگوی برداشت (آدرس → مبلغ) را مدیریت می‌کند.
func (h *Handler) handleWithdrawState(ctx context.Context, c tele.Context, lang i18n.Lang, st convState, text string) error {
	uid := c.Sender().ID

	if isCancel(text) {
		h.states.del(uid)
		return c.Send(i18n.T(lang, i18n.KCancelled), kbMain(lang))
	}

	switch st.step {
	case "addr":
		if !wallet.IsValidTONAddress(text) {
			return c.Send(i18n.T(lang, i18n.KWithdrawBadAddr), tele.ModeHTML, kbCancelOnly(lang))
		}
		st.addr = text
		st.step = "amount"
		h.states.set(uid, st)

		w, err := h.wallet.GetOrCreate(ctx, uid)
		if err != nil {
			h.states.del(uid)
			return c.Send(i18n.T(lang, i18n.KErrGeneric), kbMain(lang))
		}
		feeTON := wallet.NanoToTON(wallet.NetworkFeeNano)
		maxAmount := w.TotalTON() - feeTON

		return c.Send(
			i18n.T(lang, i18n.KWithdrawAskAmount, text[:6], text[len(text)-4:], maxAmount, feeTON),
			tele.ModeHTML, kbCancelOnly(lang),
		)

	case "amount":
		amt, err := strconv.ParseFloat(text, 64)
		if err != nil || amt <= 0 {
			return c.Send(i18n.T(lang, i18n.KWithdrawBadAmount), kbCancelOnly(lang))
		}

		h.states.del(uid)
		req, err := h.wallet.RequestWithdraw(ctx, uid, st.addr, wallet.TONToNano(amt), "")
		if err != nil {
			return c.Send(i18n.T(lang, i18n.KWithdrawError, err.Error()), kbMain(lang))
		}

		return c.Send(
			i18n.T(lang, i18n.KWithdrawSubmitted, amt, wallet.NanoToTON(wallet.NetworkFeeNano), req.ToAddress, req.ID),
			tele.ModeHTML, kbMain(lang),
		)
	}
	return nil
}

// handleTransferState مراحل گفتگوی انتقال (شناسه گیرنده → مبلغ) را مدیریت می‌کند.
func (h *Handler) handleTransferState(ctx context.Context, c tele.Context, lang i18n.Lang, st convState, text string) error {
	uid := c.Sender().ID

	if isCancel(text) {
		h.states.del(uid)
		return c.Send(i18n.T(lang, i18n.KCancelled), kbMain(lang))
	}

	switch st.step {
	case "to_id":
		toID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil || toID <= 0 || toID == uid {
			return c.Send(i18n.T(lang, i18n.KTransferBadID), kbCancelOnly(lang))
		}

		toWallet, err := h.wallet.Store().GetWallet(ctx, toID)
		if err != nil || toWallet == nil {
			return c.Send(i18n.T(lang, i18n.KTransferNoRecipient), kbCancelOnly(lang))
		}

		st.toID = toID
		st.step = "amount"
		h.states.set(uid, st)

		fromWallet, err := h.wallet.GetOrCreate(ctx, uid)
		if err != nil {
			h.states.del(uid)
			return c.Send(i18n.T(lang, i18n.KErrGeneric), kbMain(lang))
		}
		return c.Send(
			i18n.T(lang, i18n.KTransferAskAmount, toID, fromWallet.TotalTON()),
			tele.ModeHTML, kbCancelOnly(lang),
		)

	case "amount":
		amt, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || amt <= 0 {
			return c.Send(i18n.T(lang, i18n.KTransferBadAmount), kbCancelOnly(lang))
		}

		toID := st.toID
		h.states.del(uid)

		if err := h.wallet.Transfer(ctx, uid, toID, wallet.TONToNano(amt), ""); err != nil {
			return c.Send(i18n.T(lang, i18n.KTransferError, err.Error()), kbMain(lang))
		}

		return c.Send(
			i18n.T(lang, i18n.KTransferDone, amt, toID),
			tele.ModeHTML, kbMain(lang),
		)
	}
	return nil
}
