// Package tgbot - promo.go
// Redeem کدهای پروموشن توسط کاربر. مدل/store در shared-core،
// مدیریت/ساختِ کدها در admin/admin_promo.go.
package user

import (
	"context"
	"errors"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/store"
)

// PromoRedeemStart دکمه‌ی «کد پروموشن دارم؟» را هندل می‌کند.
func (h *User) PromoRedeemStart(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()
	h.SetStep(ctx, uid, state.StepPromoRedeem)
	return c.EditOrSend(h.T(ctx, uid, i18n.KeyPromoAsk), tele.ModeHTML, h.KbCancel(ctx, uid))
}

// PromoRedeemSubmit کدِ واردشده را اعتبارسنجی و اعمال می‌کند.
func (h *User) PromoRedeemSubmit(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	code := strings.ToUpper(strings.TrimSpace(text))
	if code == "" {
		return c.Send(h.T(ctx, uid, i18n.KeyPromoNotFound), h.KbUser(ctx, uid))
	}

	u, err := h.GetOrCreateUser(ctx, c)
	if err != nil || u == nil {
		return c.Send(h.T(ctx, uid, i18n.KeyError), h.KbUser(ctx, uid))
	}

	promo, err := h.Store.FindPromoCodeByCode(ctx, code)
	if err != nil {
		h.Log.Error("promo lookup failed", h.F("err", err), h.F("code", code))
		return c.Send(h.T(ctx, uid, i18n.KeyError), h.KbUser(ctx, uid))
	}
	if promo == nil {
		return c.Send(h.T(ctx, uid, i18n.KeyPromoNotFound), h.KbUser(ctx, uid))
	}

	if redeemErr := h.Store.RedeemPromoCode(ctx, promo.ID, u.ID); redeemErr != nil {
		switch {
		case errors.Is(redeemErr, store.ErrPromoAlreadyRedeemed):
			return c.Send(h.T(ctx, uid, i18n.KeyPromoAlreadyUsed), h.KbUser(ctx, uid))
		case errors.Is(redeemErr, store.ErrPromoNotRedeemable):
			return c.Send(h.T(ctx, uid, i18n.KeyPromoExpiredOrFull), h.KbUser(ctx, uid))
		default:
			h.Log.Error("promo redeem claim failed", h.F("err", redeemErr), h.F("code", code), h.F("user", uid))
			return c.Send(h.T(ctx, uid, i18n.KeyError), h.KbUser(ctx, uid))
		}
	}

	// claim سمتِ DB موفق بود — حالا اعتبار را در کیف پول (botpay) اعطا کن.
	// اگر این مرحله شکست بخورد، کاربر پولی از دست نداده (فقط هنوز شارژ
	// نشده)، پس خطا را واضح لاگ می‌کنیم تا دستی رسیدگی شود، نه اینکه گم شود.
	if h.Pay != nil {
		if credErr := h.Pay.Credit(ctx, uid, promo.AmountTON, "promo:"+code, ""); credErr != nil {
			h.Log.Error("promo credit grant failed after successful claim — needs manual reconciliation",
				h.F("err", credErr), h.F("code", code), h.F("user", uid), h.F("amount", promo.AmountTON))
			return c.Send(h.T(ctx, uid, i18n.KeyPromoCreditFailed, code), h.KbUser(ctx, uid))
		}
	}

	h.AuditLog(ctx, u.ID, string(u.Role), promo.ID.String(), "promo", models.AuditAdminAction, "redeem:"+code)
	h.Log.Info("promo redeemed", h.F("code", code), h.F("user", uid), h.F("amount", promo.AmountTON))

	return c.Send(h.T(ctx, uid, i18n.KeyPromoRedeemed, promo.AmountTON), tele.ModeHTML, h.KbUser(ctx, uid))
}
