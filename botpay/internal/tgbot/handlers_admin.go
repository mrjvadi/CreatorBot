package tgbot

import (
	"context"
	"strconv"
	"strings"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botpay/internal/i18n"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
)

// ════════════════════════════════════════════════════════════
// پنل تعاملی (دکمه‌محور)
// ════════════════════════════════════════════════════════════

// onAdmin پنل ادمین را با آمار کلی و منوی دکمه‌ای نمایش می‌دهد.
func (h *Handler) onAdmin(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	lang := h.langOf(ctx, c)
	return c.Send(h.adminStatsText(ctx, lang), tele.ModeHTML, kbAdminMenu(lang))
}

// adminStatsText متن کارت آمار را می‌سازد.
func (h *Handler) adminStatsText(ctx context.Context, lang i18n.Lang) string {
	stats, _ := h.st.GetStats(ctx)
	pending, _ := h.st.ListPendingWithdrawals(ctx)
	return i18n.T(lang, i18n.KAdminStats,
		stats.TotalWallets, stats.TotalDeposits, stats.TotalPayments, len(pending))
}

// onAdminHome (callback) صفحه‌ی اصلی پنل را به‌روزرسانی می‌کند.
func (h *Handler) onAdminHome(ctx context.Context, c tele.Context) error {
	lang := h.langOf(ctx, c)
	return c.Edit(h.adminStatsText(ctx, lang), tele.ModeHTML, kbAdminMenu(lang))
}

// onAdminWithdrawsList (callback) فهرست برداشت‌های منتظر را نمایش می‌دهد.
func (h *Handler) onAdminWithdrawsList(ctx context.Context, c tele.Context) error {
	lang := h.langOf(ctx, c)
	reqs, _ := h.st.ListPendingWithdrawals(ctx)
	if len(reqs) == 0 {
		return c.Edit(i18n.T(lang, i18n.KAdminNoWithdraws), tele.ModeHTML, kbAdminMenu(lang))
	}

	var b strings.Builder
	b.WriteString(i18n.T(lang, i18n.KAdminWithdrawsTitle, len(reqs)))
	for _, r := range reqs {
		b.WriteString("\n")
		b.WriteString(adminWithdrawText(lang, r))
		b.WriteString("\n")
	}
	return c.Edit(b.String(), tele.ModeHTML, kbWithdrawList(lang, reqs))
}

// onAdminCreditStart (callback) جریان افزودن اعتبار را آغاز می‌کند.
func (h *Handler) onAdminCreditStart(ctx context.Context, c tele.Context) error {
	lang := h.langOf(ctx, c)
	h.states.set(c.Sender().ID, convState{kind: kindAdminCredit, step: "user_id"})
	return c.Send(i18n.T(lang, i18n.KAdminAskCreditUserID), tele.ModeHTML, kbCancelOnly(lang))
}

// onAdminApproveStart (callback) از ادمین txhash می‌خواهد.
func (h *Handler) onAdminApproveStart(ctx context.Context, c tele.Context, id string) error {
	lang := h.langOf(ctx, c)
	if _, err := uuid.Parse(id); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: i18n.T(lang, i18n.KAdminBadID)})
	}
	h.states.set(c.Sender().ID, convState{kind: kindAdminApprove, arg: id})
	return c.Send(i18n.T(lang, i18n.KAdminAskTxHash, id), tele.ModeHTML, kbCancelOnly(lang))
}

// onAdminRejectStart (callback) از ادمین دلیل رد می‌خواهد.
func (h *Handler) onAdminRejectStart(ctx context.Context, c tele.Context, id string) error {
	lang := h.langOf(ctx, c)
	if _, err := uuid.Parse(id); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: i18n.T(lang, i18n.KAdminBadID)})
	}
	h.states.set(c.Sender().ID, convState{kind: kindAdminReject, arg: id})
	return c.Send(i18n.T(lang, i18n.KAdminAskReason, id), tele.ModeHTML, kbCancelOnly(lang))
}

// ── جریان‌های گفتگوی ادمین ─────────────────────────────────

func (h *Handler) handleAdminApprove(ctx context.Context, c tele.Context, lang i18n.Lang, st convState, text string) error {
	uid := c.Sender().ID
	if isCancel(text) {
		h.states.del(uid)
		return c.Send(i18n.T(lang, i18n.KCancelled), kbMain(lang))
	}

	withdrawID, err := uuid.Parse(st.arg)
	if err != nil {
		h.states.del(uid)
		return c.Send(i18n.T(lang, i18n.KAdminBadID), kbMain(lang))
	}
	h.states.del(uid)

	if err := h.st.CompleteWithdraw(ctx, withdrawID, strings.TrimSpace(text)); err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminError, err.Error()), kbMain(lang))
	}
	return c.Send(i18n.T(lang, i18n.KAdminApproved, st.arg), tele.ModeHTML, kbMain(lang))
}

func (h *Handler) handleAdminReject(ctx context.Context, c tele.Context, lang i18n.Lang, st convState, text string) error {
	uid := c.Sender().ID
	if isCancel(text) {
		h.states.del(uid)
		return c.Send(i18n.T(lang, i18n.KCancelled), kbMain(lang))
	}

	withdrawID, err := uuid.Parse(st.arg)
	if err != nil {
		h.states.del(uid)
		return c.Send(i18n.T(lang, i18n.KAdminBadID), kbMain(lang))
	}
	h.states.del(uid)

	if err := h.st.RejectWithdraw(ctx, withdrawID, strings.TrimSpace(text)); err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminError, err.Error()), kbMain(lang))
	}
	return c.Send(i18n.T(lang, i18n.KAdminRejected, st.arg), tele.ModeHTML, kbMain(lang))
}

func (h *Handler) handleAdminCredit(ctx context.Context, c tele.Context, lang i18n.Lang, st convState, text string) error {
	uid := c.Sender().ID
	if isCancel(text) {
		h.states.del(uid)
		return c.Send(i18n.T(lang, i18n.KCancelled), kbMain(lang))
	}

	switch st.step {
	case "user_id":
		toID, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil || toID == 0 {
			return c.Send(i18n.T(lang, i18n.KAdminBadParam), kbCancelOnly(lang))
		}
		st.toID = toID
		st.step = "amount"
		h.states.set(uid, st)
		return c.Send(i18n.T(lang, i18n.KAdminAskCreditAmount, toID), tele.ModeHTML, kbCancelOnly(lang))

	case "amount":
		amt, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || amt <= 0 {
			return c.Send(i18n.T(lang, i18n.KAdminBadParam), kbCancelOnly(lang))
		}
		toID := st.toID
		h.states.del(uid)

		w, err := h.wallet.GetOrCreate(ctx, toID)
		if err != nil {
			return c.Send(i18n.T(lang, i18n.KAdminWalletNotFound), kbMain(lang))
		}
		if err := h.st.AddCredit(ctx, w.ID, wallet.TONToNano(amt), ""); err != nil {
			return c.Send(i18n.T(lang, i18n.KAdminError, err.Error()), kbMain(lang))
		}
		return c.Send(i18n.T(lang, i18n.KAdminCreditAdded, amt, toID), tele.ModeHTML, kbMain(lang))
	}
	return nil
}

// ════════════════════════════════════════════════════════════
// دستورات متنی (سازگاری — همچنان کار می‌کنند)
// ════════════════════════════════════════════════════════════

// onAdminWithdraws فهرست برداشت‌های منتظر (دستور /withdraws).
func (h *Handler) onAdminWithdraws(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	reqs, _ := h.st.ListPendingWithdrawals(ctx)
	if len(reqs) == 0 {
		return c.Send(i18n.T(lang, i18n.KAdminNoWithdraws))
	}

	var b strings.Builder
	b.WriteString(i18n.T(lang, i18n.KAdminWithdrawsTitle, len(reqs)))
	for _, r := range reqs {
		b.WriteString("\n")
		b.WriteString(adminWithdrawText(lang, r))
		b.WriteString("\n")
	}
	return c.Send(b.String(), tele.ModeHTML, kbWithdrawList(lang, reqs))
}

// onApproveWithdraw تأیید برداشت: /approve <id> <txhash>
func (h *Handler) onApproveWithdraw(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	parts := strings.Fields(c.Message().Payload)
	if len(parts) < 2 {
		return c.Send(i18n.T(lang, i18n.KAdminApproveUsage), tele.ModeHTML)
	}
	withdrawID, err := uuid.Parse(parts[0])
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminBadID))
	}
	if err := h.st.CompleteWithdraw(ctx, withdrawID, parts[1]); err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminError, err.Error()))
	}
	return c.Send(i18n.T(lang, i18n.KAdminApproved, parts[0]), tele.ModeHTML)
}

// onRejectWithdraw رد برداشت: /reject <id> <reason>
func (h *Handler) onRejectWithdraw(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	payload := strings.TrimSpace(c.Message().Payload)
	idStr, reason, found := strings.Cut(payload, " ")
	if !found || idStr == "" {
		return c.Send(i18n.T(lang, i18n.KAdminRejectUsage), tele.ModeHTML)
	}
	withdrawID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminBadID))
	}
	if err := h.st.RejectWithdraw(ctx, withdrawID, strings.TrimSpace(reason)); err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminError, err.Error()))
	}
	return c.Send(i18n.T(lang, i18n.KAdminRejected, idStr), tele.ModeHTML)
}

// onAddCredit افزودن اعتبار: /addcredit <telegram_id> <amount_ton> [desc]
func (h *Handler) onAddCredit(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	lang := h.langOf(ctx, c)

	parts := strings.Fields(c.Message().Payload)
	if len(parts) < 2 {
		return c.Send(i18n.T(lang, i18n.KAdminAddCreditUsage), tele.ModeHTML)
	}
	telegramID, errID := strconv.ParseInt(parts[0], 10, 64)
	amt, errAmt := strconv.ParseFloat(parts[1], 64)
	if errID != nil || errAmt != nil || telegramID == 0 || amt <= 0 {
		return c.Send(i18n.T(lang, i18n.KAdminBadParam))
	}

	w, err := h.wallet.GetOrCreate(ctx, telegramID)
	if err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminWalletNotFound))
	}

	desc := ""
	if len(parts) > 2 {
		desc = strings.Join(parts[2:], " ")
	}
	if err := h.st.AddCredit(ctx, w.ID, wallet.TONToNano(amt), desc); err != nil {
		return c.Send(i18n.T(lang, i18n.KAdminError, err.Error()))
	}
	return c.Send(i18n.T(lang, i18n.KAdminCreditAdded, amt, telegramID), tele.ModeHTML)
}
