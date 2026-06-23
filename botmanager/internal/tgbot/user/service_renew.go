// Package tgbot - service_renew.go
// تمدید سرویس: کسر هزینه از کیف پول (botpay از طریق NATS) → تمدید انقضا →
// اطمینان از روشن‌بودن سرویس (ارسال دستور start به agent در صورت خاموش‌بودن).
package user

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// instanceRenewConfirm کارت تأیید تمدید را نشان می‌دهد.
func (h *User) InstanceRenewConfirm(ctx context.Context, c tele.Context, idStr string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID

	inst, err := h.Store.FindInstance(ctx, idStr)
	if err != nil || inst == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNotFound))
	}
	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil || inst.OwnerID != u.ID {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNoAccess))
	}
	if inst.PlanID == nil {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnViewPlans), "show_plans")))
		return c.Edit(h.T(ctx, uid, i18n.KeyRenewNoPlan), tele.ModeHTML, kb)
	}

	plan, _ := h.Store.FindPlan(ctx, inst.PlanID.String())
	if plan == nil {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnViewPlans), "show_plans")))
		return c.Edit(h.T(ctx, uid, i18n.KeyRenewNoPlan), tele.ModeHTML, kb)
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnConfirmRenew), "svc_renew_do:"+idStr)),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "svc_settings:"+idStr)),
	)
	return c.Edit(
		h.T(ctx, uid, i18n.KeyRenewConfirm, inst.ContainerName, plan.Name, plan.Price),
		tele.ModeHTML, kb,
	)
}

// instanceRenewExecute تراکنش تمدید را اجرا می‌کند.
func (h *User) InstanceRenewExecute(ctx context.Context, c tele.Context, idStr string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID

	inst, err := h.Store.FindInstance(ctx, idStr)
	if err != nil || inst == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNotFound))
	}
	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil || inst.OwnerID != u.ID {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNoAccess))
	}
	if inst.PlanID == nil || h.Pay == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	plan, _ := h.Store.FindPlan(ctx, inst.PlanID.String())
	if plan == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyRenewNoPlan))
	}

	// کسر هزینه از کیف پول (botpay از طریق NATS)
	idem := fmt.Sprintf("renew:%s:%d", inst.ID.String(), time.Now().Unix())
	if _, err := h.Pay.Deduct(ctx, u.TelegramID, plan.Price, plan.ID.String(), idem); err != nil {
		if natspayclient.IsInsufficientBalance(err) {
			kb := &tele.ReplyMarkup{}
			kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnTopupWallet), "wallet_topup")))
			return c.Edit(h.T(ctx, uid, i18n.KeyWizardLowBalance), kb)
		}
		h.Log.Error("renew deduct", ports.F("err", err))
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	// تمدید انقضا: از max(الان، انقضای فعلی) به‌اندازه‌ی مدت پلن
	base := time.Now()
	if inst.ExpiresAt != nil && inst.ExpiresAt.After(base) {
		base = *inst.ExpiresAt
	}
	var newExp *time.Time
	durText := h.T(ctx, uid, i18n.KeyDurationForever)
	if plan.DurationDay > 0 {
		t := base.AddDate(0, 0, plan.DurationDay)
		newExp = &t
		durText = h.T(ctx, uid, i18n.KeyDaysCount, int(time.Until(t).Hours()/24))
	}
	if err := h.Store.UpdateInstanceExpiry(ctx, inst.ID, newExp); err != nil {
		h.Log.Error("renew update expiry", ports.F("err", err))
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	// اگر سرویس روشن نیست، دستور start به agent بفرست
	if inst.Status != models.StatusRunning && h.NC != nil {
		_ = h.NC.PublishCore("instance.start", map[string]any{
			"instance_id":    inst.ID,
			"container_name": inst.ContainerName,
			"server_id":      inst.ServerID,
			"action":         "start",
		})
		_ = h.Store.UpdateInstanceStatus(ctx, inst.ID.String(), models.StatusPending)
	}

	h.Log.Info("service renewed",
		ports.F("instance", inst.ID), ports.F("user", u.TelegramID), ports.F("plan", plan.Name))
	h.AuditLog(ctx, u.ID, string(u.Role), inst.ID.String(), "instance", models.AuditBuyPlan, "renew")

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnMyBots), "my_bots")))
	return c.Edit(
		h.T(ctx, uid, i18n.KeyRenewDone, inst.ContainerName, durText),
		tele.ModeHTML, kb,
	)
}
