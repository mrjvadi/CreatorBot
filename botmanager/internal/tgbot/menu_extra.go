package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
)

// ── حساب و موجودی کاربر (داده‌ی واقعی از botpay) ──────────────

func (h *Handler) userAccount(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID

	status := "Standard"
	if user, _ := h.store.FindUserByTelegramID(ctx, uid); user != nil {
		if sub, _ := h.store.GetActiveSubscription(ctx, user.ID); sub != nil {
			status = "VIP"
		}
	}

	// موجودی واقعی از سرویس botpay
	var tonBal, credit, total float64
	if h.pay != nil {
		bal, err := h.pay.Balance(ctx, uid)
		if err != nil {
			h.log.Error("account: balance fetch", h.F("err", err))
			return c.Send(h.t(ctx, uid, i18n.KeyError), h.kbUser(ctx, uid))
		}
		tonBal, credit, total = bal.TONBalance, bal.Credit, bal.Total
	}

	text := fmt.Sprintf(h.t(ctx, uid, i18n.KeyAccountTitle),
		uid, tonBal, credit, total, status)

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(h.btn(ctx, uid, i18n.KeyMenuWallet), "wallet_topup"),
			kb.Data("🎁", "redeem_promo"),
		),
		kb.Row(kb.Data(h.btn(ctx, uid, i18n.KeyBack), "back_main")),
	)
	return c.Send(text, tele.ModeHTML, kb)
}

// ── منوی انتخاب زبان ─────────────────────────────────────────

func (h *Handler) userLanguageMenu(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	return c.Send(h.t(ctx, uid, i18n.KeyLanguageSelect), tele.ModeHTML, kbLanguage())
}

// setLanguage زبان کاربر را تغییر می‌دهد (callback: lang:fa | lang:en).
func (h *Handler) setLanguage(ctx context.Context, c tele.Context, lang string) error {
	uid := c.Sender().ID
	switch lang {
	case "fa":
		h.tr.SetLang(ctx, uid, i18n.FA)
	case "en":
		h.tr.SetLang(ctx, uid, i18n.EN)
	}
	_ = c.Respond(&tele.CallbackResponse{Text: h.t(ctx, uid, i18n.KeyLangChanged)})
	// منوی اصلی با زبان جدید
	if h.isAdmin(c) {
		return c.Send(h.t(ctx, uid, i18n.KeyLangChanged), h.kbAdmin(ctx, uid))
	}
	return c.Send(h.t(ctx, uid, i18n.KeyLangChanged), h.kbUser(ctx, uid))
}

// ── منوی ارسال همگانی ادمین ──────────────────────────────────

func (h *Handler) adminBroadcastMenu(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("💬 ارسال متنی هوشمند", "bc_text")),
		kb.Row(kb.Data("🔄 فوروارد همگانی", "bc_forward")),
		kb.Row(kb.Data("🎯 ارسال فیلترشده", "bc_filtered")),
		kb.Row(kb.Data(h.btn(ctx, uid, i18n.KeyBack), "back_main")),
	)
	return c.Send(h.t(ctx, uid, i18n.KeyBroadcastMenu), tele.ModeHTML, kb)
}

// ── منوی تنظیمات سیستم ادمین ─────────────────────────────────

func (h *Handler) adminSystemMenu(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.btn(ctx, uid, i18n.KeyMenuLanguage), "sys_lang")),
		kb.Row(kb.Data(h.btn(ctx, uid, i18n.KeyMenuNotifications), "sys_notif")),
		kb.Row(kb.Data(h.btn(ctx, uid, i18n.KeyBack), "back_main")),
	)
	return c.Send(h.t(ctx, uid, i18n.KeySystemMenu), tele.ModeHTML, kb)
}
