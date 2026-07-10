// Package admin - admin_promo.go
// مدیریتِ کدهای پروموشن: ساخت/لیست/فعال-غیرفعال/حذف. redeem سمتِ کاربر در
// internal/tgbot/user/promo.go است.
package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

// AdminPromoList لیستِ کدهای پروموشن را نشان می‌دهد.
func (h *Admin) AdminPromoList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	list, _ := h.Store.ListPromoCodes(ctx)

	lines := []string{h.T(ctx, uid, i18n.KeyPromoAdminTitle), ""}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	if len(list) == 0 {
		lines = append(lines, h.T(ctx, uid, i18n.KeyPromoAdminEmpty))
	} else {
		for _, p := range list {
			lines = append(lines, format.FmtPromoCode(p))
			rows = append(rows, kb.Row(
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnToggleSW), "admin_promo_toggle:"+p.ID.String()),
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDeleteSW), "admin_promo_del:"+p.ID.String()),
			))
		}
	}
	lines = append(lines, "")
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnAddPromo), "admin_promo_add")))
	kb.Inline(rows...)
	return c.Send(format.JoinLines(lines), tele.ModeHTML, kb)
}

// AdminPromoStart ویزاردِ ساختِ کد را شروع می‌کند.
func (h *Admin) AdminPromoStart(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	h.SetStep(ctx, uid, state.StepPromoAdminCode)
	return c.Edit(h.T(ctx, uid, i18n.KeyPromoAskCode), tele.ModeHTML, h.KbBackCancel(ctx, uid))
}

// AdminPromoAdd آخرین مرحله — رکورد را می‌سازد.
func (h *Admin) AdminPromoAdd(ctx context.Context, c tele.Context, code, amountStr, maxUsesStr, daysStr string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	code = strings.ToUpper(strings.TrimSpace(code))
	amount, err := strconv.ParseFloat(strings.TrimSpace(amountStr), 64)
	if err != nil || amount <= 0 || code == "" {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyPromoCreateError))
	}
	maxUses, err := strconv.Atoi(strings.TrimSpace(maxUsesStr))
	if err != nil || maxUses < 0 {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyPromoCreateError))
	}
	days, err := strconv.Atoi(strings.TrimSpace(daysStr))
	if err != nil || days < 0 {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyPromoCreateError))
	}

	var expiresAt *time.Time
	if days > 0 {
		t := time.Now().AddDate(0, 0, days)
		expiresAt = &t
	}

	promo := &models.PromoCode{
		Code:      code,
		AmountTON: amount,
		MaxUses:   maxUses,
		ExpiresAt: expiresAt,
		IsActive:  true,
		CreatedBy: uid,
	}
	if err := h.Store.CreatePromoCode(ctx, promo); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return h.SendMain(c, h.T(ctx, uid, i18n.KeyPromoDuplicate))
		}
		h.Log.Error("adminPromoAdd", h.F("err", err))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyPromoCreateError))
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyPromoCreated, code, amount, maxUses, days),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}

// AdminPromoDeleteConfirm تأییدِ حذف.
func (h *Admin) AdminPromoDeleteConfirm(ctx context.Context, c tele.Context, id string) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnConfirmDelete), "admin_promo_del_do:"+id),
			kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel"),
		),
	)
	_ = c.Respond()
	return c.Send(h.T(ctx, uid, i18n.KeyPromoDeleteConfirm), tele.ModeHTML, kb)
}

// AdminPromoDelete کد را حذف می‌کند.
func (h *Admin) AdminPromoDelete(ctx context.Context, c tele.Context, id string) error {
	uid := c.Sender().ID
	if err := h.Store.DeletePromoCode(ctx, id); err != nil {
		h.Log.Error("adminPromoDelete", h.F("err", err))
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}
	return c.Edit(h.T(ctx, uid, i18n.KeyPromoDeleted))
}

// AdminPromoToggle فعال/غیرفعال می‌کند.
func (h *Admin) AdminPromoToggle(ctx context.Context, c tele.Context, id string) error {
	uid := c.Sender().ID
	promo, err := h.Store.FindPromoCode(ctx, id)
	if err != nil || promo == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyNotFound), ShowAlert: true})
	}
	newActive := !promo.IsActive
	if err := h.Store.SetPromoCodeActive(ctx, id, newActive); err != nil {
		h.Log.Error("adminPromoToggle", h.F("err", err))
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyError), ShowAlert: true})
	}
	key := i18n.KeySWToggledOff
	if newActive {
		key = i18n.KeySWToggledOn
	}
	return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, key), ShowAlert: true})
}
