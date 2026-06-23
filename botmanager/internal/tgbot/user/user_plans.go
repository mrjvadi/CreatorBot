package user

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ════════════════════════════════════════════════════════════
// نمایش پلن‌ها — با inline button، نه UUID
// ════════════════════════════════════════════════════════════

func (h *User) UserPlans(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	plans, _ := h.Store.ListActivePlans(ctx)

	if len(plans) == 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyPlansUnavailable), h.KbUser(ctx, uid))
	}

	// ── اشتراک فعال فعلی کاربر ──
	msg := h.T(ctx, uid, i18n.KeyPlansAvailableTitle)
	u, _ := h.GetOrCreateUser(ctx, c)
	if u != nil {
		if sub, _ := h.Store.GetActiveSubscription(ctx, u.ID); sub != nil {
			if curPlan, _ := h.Store.FindPlan(ctx, sub.PlanID.String()); curPlan != nil {
				rem := ""
				if sub.ExpiresAt != nil {
					d := int(time.Until(*sub.ExpiresAt).Hours() / 24)
					if d > 0 {
						rem = h.T(ctx, uid, i18n.KeyPlanRemDays, d)
					} else {
						rem = h.T(ctx, uid, i18n.KeyPlanExpiredShort)
					}
				}
				msg += h.T(ctx, uid, i18n.KeyPlanActiveYours, curPlan.Name, rem)
			}
		}
	}

	for _, p := range plans {
		priceText := fmt.Sprintf("%.2f TON", p.Price)
		if p.IsFree {
			priceText = "🆓 " + h.T(ctx, uid, i18n.KeyFree)
		}
		dur := h.T(ctx, uid, i18n.KeyDaysCount, p.DurationDay)
		if p.DurationDay == 0 {
			dur = h.T(ctx, uid, i18n.KeyDurationForever)
		}
		msg += h.T(ctx, uid, i18n.KeyPlanRow, p.Name, priceText, p.MaxBots, dur)
	}

	msg += h.T(ctx, uid, i18n.KeyPlansClickToBuy)

	// ── inline keyboard — یه دکمه برای هر پلن ──
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		var label string
		if p.IsFree {
			label = h.T(ctx, uid, i18n.KeyPlanLabelFree, p.Name)
		} else {
			label = h.T(ctx, uid, i18n.KeyPlanLabelPaid, p.Name, p.Price)
		}
		rows = append(rows, kb.Row(kb.Data(label, "select_plan:"+p.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnClose), "cancel")))
	kb.Inline(rows...)

	return c.Send(msg, tele.ModeHTML, kb)
}

// userSelectPlan — کاربر پلن را انتخاب کرد
func (h *User) UserSelectPlan(ctx context.Context, c tele.Context, planID string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID

	plan, err := h.Store.FindPlan(ctx, planID)
	if err != nil || plan == nil || !plan.IsActive {
		return c.Edit(h.T(ctx, uid, i18n.KeyNotFound))
	}

	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	// اشتراک موجود — بررسی اینکه همین پلن است یا می‌خواهد ارتقا دهد
	existing, _ := h.Store.GetActiveSubscription(ctx, u.ID)
	if existing != nil && existing.PlanID.String() == planID {
		// همین پلن فعال است
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "show_plans")))
		return c.Edit(h.T(ctx, uid, i18n.KeyPlanAlreadyActive), kb)
	}

	// پلن رایگان
	if plan.IsFree {
		return h.ActivateFreePlanInline(ctx, c, u, plan)
	}

	// پلن پولی — نمایش جزئیات و دکمه خرید (حتی اگر پلن قبلی داشته باشد)
	return h.ShowPlanDetail(ctx, c, u, plan)
}

// showPlanDetail جزئیات پلن + وضعیت موجودی + دکمه‌های مناسب
func (h *User) ShowPlanDetail(ctx context.Context, c tele.Context, u *models.User, plan *models.Plan) error {
	uid := c.Sender().ID

	dur := h.T(ctx, uid, i18n.KeyDaysCount, plan.DurationDay)
	if plan.DurationDay == 0 {
		dur = h.T(ctx, uid, i18n.KeyDurationForever)
	}

	msg := h.T(ctx, uid, i18n.KeyPlanDetail, plan.Name, plan.MaxBots, dur, plan.Price)

	kb := &tele.ReplyMarkup{}

	// بررسی موجودی
	if h.Pay != nil {
		bal, err := h.Pay.Balance(ctx, u.TelegramID)
		if err == nil {
			msg += h.T(ctx, uid, i18n.KeyWalletBalanceLine, bal.Total)

			if bal.Total >= plan.Price {
				// موجودی کافیه
				msg += h.T(ctx, uid, i18n.KeyBalanceEnough)
				kb.Inline(
					kb.Row(kb.Data(h.T(ctx, uid, i18n.KeyBtnBuyWith, plan.Price), "buy_plan:"+plan.ID.String())),
					kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBackToPlans), "show_plans")),
				)
			} else {
				// موجودی کافی نیست
				needed := plan.Price - bal.Total
				msg += h.T(ctx, uid, i18n.KeyBalanceShortfall, needed)

				inv, err := h.Pay.CreateInvoice(ctx, u.TelegramID, needed, plan.ID.String())
				if err == nil {
					msg += h.T(ctx, uid, i18n.KeyDepositAddrCode, inv.MasterAddress, inv.Code)
					kb.Inline(
						kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCheckPayment), "check_plan:"+plan.ID.String()+":"+inv.Code)),
						kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "show_plans")),
					)
				} else {
					kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "show_plans")))
				}
			}
		} else {
			// سرویس پرداخت در دسترس نیست
			msg += h.T(ctx, uid, i18n.KeyPayServiceUnavailable)
			kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "show_plans")))
		}
	} else {
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "show_plans")))
	}

	return c.Edit(msg, tele.ModeHTML, kb)
}

// findFreePlan اولین پلن رایگانِ فعال را برمی‌گرداند (nil اگر موجود نبود).
func (h *User) FindFreePlan(ctx context.Context) *models.Plan {
	plans, _ := h.Store.ListActivePlans(ctx)
	for i := range plans {
		if plans[i].IsFree {
			return &plans[i]
		}
	}
	return nil
}

// userStartFree دکمه «شروع رایگان» را هندل می‌کند.
// اگر planID خالی باشد (دکمه‌ی منوی خوش‌آمد) پلن رایگانِ فعال را پیدا می‌کند؛
// در غیر این صورت همان پلن را فعال می‌کند. این از کوئریِ FindPlan با UUID صفر
// (که قبلاً «record not found» تولید می‌کرد) جلوگیری می‌کند.
func (h *User) UserStartFree(ctx context.Context, c tele.Context, planID string) error {
	uid := c.Sender().ID
	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	var plan *models.Plan
	if planID != "" {
		plan, _ = h.Store.FindPlan(ctx, planID)
	} else {
		plan = h.FindFreePlan(ctx)
	}
	if plan == nil || !plan.IsFree {
		return c.Edit(h.T(ctx, uid, i18n.KeyNoFreePlan))
	}

	return h.ActivateFreePlanInline(ctx, c, u, plan)
}

// activateFreePlanInline پلن رایگان رو activate کن
func (h *User) ActivateFreePlanInline(ctx context.Context, c tele.Context, u *models.User, plan *models.Plan) error {
	uid := c.Sender().ID
	var expiresAt *time.Time
	if plan.DurationDay > 0 {
		t := time.Now().AddDate(0, 0, plan.DurationDay)
		expiresAt = &t
	}
	sub := &models.Subscription{
		UserID: u.ID, PlanID: plan.ID,
		StartedAt: time.Now(), ExpiresAt: expiresAt, IsActive: true,
	}
	if err := h.Store.CreateSubscription(ctx, sub); err != nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	dur := h.T(ctx, uid, i18n.KeyDaysCount, plan.DurationDay)
	if plan.DurationDay == 0 {
		dur = h.T(ctx, uid, i18n.KeyDurationForever)
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnMyBots), "my_bots")))

	return c.Edit(
		h.T(ctx, uid, i18n.KeyFreePlanActivated, plan.MaxBots, dur),
		tele.ModeHTML, kb,
	)
}

// ════════════════════════════════════════════════════════════
// خرید پلن — اجرای نهایی
// ════════════════════════════════════════════════════════════

func (h *User) ExecutePlanPurchase(ctx context.Context, c tele.Context, planID string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID

	plan, _ := h.Store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyNotFound))
	}

	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	if h.Pay == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	// کسر از botpay
	_, err := h.Pay.Deduct(ctx, u.TelegramID, plan.Price, plan.ID.String(),
		h.T(ctx, uid, i18n.KeyPlanPurchaseDesc, plan.Name))
	if err != nil {
		if natspayclient.IsInsufficientBalance(err) {
			kb := &tele.ReplyMarkup{}
			kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnTopupWallet), "show_plans")))
			return c.Edit(h.T(ctx, uid, i18n.KeyWizardLowBalance), kb)
		}
		h.Log.Error("executePlanPurchase", ports.F("err", err))
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	// فعال‌سازی اشتراک
	var expiresAt *time.Time
	if plan.DurationDay > 0 {
		t := time.Now().AddDate(0, 0, plan.DurationDay)
		expiresAt = &t
	}
	newSub := &models.Subscription{
		UserID: u.ID, PlanID: plan.ID,
		StartedAt: time.Now(), ExpiresAt: expiresAt, IsActive: true,
	}
	_ = h.Store.CreateSubscription(ctx, newSub)

	// ── reset quota cache ─────────────────────────────────
	// plan.upgraded → سرویس‌ها quota را ریست می‌کنند
	if h.NC != nil {
		_ = h.NC.PublishCore("plan.upgraded", map[string]any{
			"user_id":     u.ID,
			"telegram_id": u.TelegramID,
			"plan_id":     plan.ID,
			"plan_name":   plan.Name,
			"max_bots":    plan.MaxBots,
		})
	}

	h.Log.Info("plan purchased", ports.F("user", u.TelegramID), ports.F("plan", plan.Name))
	metrics.IncPlanPurchase(plan.Name, "success")
	h.AuditLog(ctx, u.ID, string(u.Role), plan.ID.String(), "plan", models.AuditBuyPlan,
		plan.Name)

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnMyBots), "my_bots")))

	return c.Edit(
		h.T(ctx, uid, i18n.KeyPurchaseSuccess, plan.Name, plan.MaxBots),
		tele.ModeHTML, kb,
	)
}

func (h *User) CheckPlanAfterDeposit(ctx context.Context, c tele.Context, planID, invoiceCode string) error {
	uid := c.Sender().ID

	plan, _ := h.Store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyNotFound), ShowAlert: true})
	}
	if h.Pay == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyError), ShowAlert: true})
	}

	// وضعیتِ خودِ فاکتورِ واریز را استعلام کن (نه موجودی)
	st, err := h.Pay.InvoiceStatus(ctx, uid, invoiceCode)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyTxCheckFailed), ShowAlert: true})
	}

	// تراکنش دریافت شد → خرید را نهایی کن (پیام به موفقیت ویرایش می‌شود)
	if st.Status == protocol.InvoiceStatusPaid {
		return h.ExecutePlanPurchase(ctx, c, planID)
	}

	// هنوز دریافت نشده → فقط اعلانِ وضعیت؛ پیام و دکمه دست‌نخورده می‌ماند
	return c.Respond(h.TxStatusAlert(ctx, uid, st))
}
