package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/core"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
)

// ── حساب و موجودی کاربر ──────────────────────────────────────

func (h *User) UserAccount(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID

	status := "Standard"
	if user, _ := h.Store.FindUserByTelegramID(ctx, uid); user != nil {
		if sub, _ := h.Store.GetActiveSubscription(ctx, user.ID); sub != nil {
			status = "VIP"
		}
	}

	var tonBal, credit, total float64
	if h.Pay != nil {
		bal, err := h.Pay.Balance(ctx, uid)
		if err != nil {
			h.Log.Error("account: balance fetch", h.F("err", err))
			return c.Send(h.T(ctx, uid, i18n.KeyError), h.KbUser(ctx, uid))
		}
		tonBal, credit, total = bal.TONBalance, bal.Credit, bal.Total
	}

	text := fmt.Sprintf(h.T(ctx, uid, i18n.KeyAccountTitle),
		uid, tonBal, credit, total, status)

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(h.Btn(ctx, uid, i18n.KeyMenuWallet), "wallet_topup"),
			kb.Data("🎁", "redeem_promo"),
		),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBack), "back_main")),
	)
	return c.Send(text, tele.ModeHTML, kb)
}

// ── زبان ─────────────────────────────────────────────────────

func (h *User) UserLanguageMenu(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	return c.Send(h.T(ctx, uid, i18n.KeyLanguageSelect), tele.ModeHTML, core.KbLanguage())
}

func (h *User) SetLanguage(ctx context.Context, c tele.Context, lang string) error {
	uid := c.Sender().ID
	switch lang {
	case "fa":
		h.Tr.SetLang(ctx, uid, i18n.FA)
	case "en":
		h.Tr.SetLang(ctx, uid, i18n.EN)
	}
	_ = c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyLangChanged)})
	if h.IsAdmin(c) {
		return c.Send(h.T(ctx, uid, i18n.KeyLangChanged), h.KbAdmin(ctx, uid))
	}
	return c.Send(h.T(ctx, uid, i18n.KeyLangChanged), h.KbUser(ctx, uid))
}

// ── کیف پول ──────────────────────────────────────────────────

func (h *User) UserWallet(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID

	var tonBal, credit, total float64
	if h.Pay != nil {
		bal, err := h.Pay.Balance(ctx, uid)
		if err != nil {
			h.Log.Error("wallet: balance fetch", h.F("err", err))
			return c.Send(h.T(ctx, uid, i18n.KeyError), h.KbUser(ctx, uid))
		}
		tonBal, credit, total = bal.TONBalance, bal.Credit, bal.Total
	}

	text := h.T(ctx, uid, i18n.KeyWalletTitle, tonBal, credit, total)

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDepositTON), "wallet_topup"),
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnHistory), "wallet_history"),
		),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnRedeemPromo), "redeem_promo")),
	)
	return c.Send(text, tele.ModeHTML, kb)
}

// ── تنظیمات ──────────────────────────────────────────────────

func (h *User) UserSettings(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.T(ctx, uid, i18n.KeySettingsLanguage), "sys_lang")),
		kb.Row(kb.Data(h.T(ctx, uid, i18n.KeySettingsSupport), "user_support_inline")),
		kb.Row(kb.Data(h.T(ctx, uid, i18n.KeySettingsAbout), "about_platform")),
	)
	return c.Send(h.T(ctx, uid, i18n.KeySettingsHome), tele.ModeHTML, kb)
}

func (h *User) UserSupportInline(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBack), "cancel")))
	return c.Edit(h.T(ctx, uid, i18n.KeySupportInline), tele.ModeHTML, kb)
}

func (h *User) AboutPlatform(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBack), "cancel")))
	return c.Edit(h.T(ctx, uid, i18n.KeyAboutPlatform), tele.ModeHTML, kb)
}

// ── واریز کیف پول ─────────────────────────────────────────────

func (h *User) WalletTopupStart(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("1 TON", "topup_amt:1"),
			kb.Data("5 TON", "topup_amt:5"),
			kb.Data("10 TON", "topup_amt:10"),
		),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCustomAmount), "topup_custom")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel")),
	)
	return c.EditOrSend(h.T(ctx, uid, i18n.KeyWalletTopupAsk), tele.ModeHTML, kb)
}

func (h *User) WalletTopupCustom(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()
	h.SetStep(ctx, uid, state.StepWalletTopupAmount)
	return c.EditOrSend(h.T(ctx, uid, i18n.KeyWalletTopupAsk), tele.ModeHTML, h.KbCancel(ctx, uid))
}

func (h *User) WalletTopupAmount(ctx context.Context, c tele.Context, amountStr string) error {
	_ = c.Respond()
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		return c.Edit(h.T(ctx, c.Sender().ID, i18n.KeyWalletTopupInvalid))
	}
	return h.WalletCreateInvoice(ctx, c, amount)
}

func (h *User) WalletTopupProcess(ctx context.Context, c tele.Context, amountStr string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	amount, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
	if err != nil || amount <= 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyWalletTopupInvalid), h.KbUser(ctx, uid))
	}
	return h.WalletCreateInvoice(ctx, c, amount)
}

func (h *User) WalletCreateInvoice(ctx context.Context, c tele.Context, amount float64) error {
	uid := c.Sender().ID
	if h.Pay == nil {
		return c.EditOrSend(h.T(ctx, uid, i18n.KeyError), h.KbUser(ctx, uid))
	}

	inv, err := h.Pay.CreateInvoice(ctx, uid, amount, "topup")
	if err != nil {
		h.Log.Error("wallet topup invoice", h.F("err", err))
		return c.EditOrSend(h.T(ctx, uid, i18n.KeyError), h.KbUser(ctx, uid))
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCheckPayment), "topup_check:"+inv.Code)),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyMenuWallet), "wallet_home")),
	)
	return c.EditOrSend(
		h.T(ctx, uid, i18n.KeyWalletTopupInvoice, amount, inv.MasterAddress, inv.Code),
		tele.ModeHTML, kb,
	)
}

func (h *User) WalletHistory(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()

	var tonBal, credit, total float64
	if h.Pay != nil {
		if bal, err := h.Pay.Balance(ctx, uid); err == nil {
			tonBal, credit, total = bal.TONBalance, bal.Credit, bal.Total
		}
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnNewDeposit), "wallet_topup")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBack), "wallet_home")),
	)
	return c.Edit(
		fmt.Sprintf(h.T(ctx, uid, i18n.KeyWalletHistoryNote), tonBal, credit, total),
		tele.ModeHTML, kb,
	)
}
