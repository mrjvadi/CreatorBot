package tgbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/payclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ════════════════════════════════════════════════════════════
// ربات‌های من
// ════════════════════════════════════════════════════════════

func (h *Handler) userBotsList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	u, err := h.getOrCreateUser(ctx, c)
	if err != nil || u == nil {
		return c.Send(h.t(ctx, uid, i18n.KeyError))
	}
	if u.IsBlocked {
		return c.Send(h.t(ctx, uid, i18n.KeyBlocked))
	}

	instances, _ := h.store.ListInstancesByOwner(ctx, u.ID)
	sub, _ := h.store.GetActiveSubscription(ctx, u.ID)

	// کاربر هیچ ربات و هیچ اشتراکی ندارد
	if len(instances) == 0 && sub == nil {
		return h.userShowWelcome(ctx, c, u)
	}

	// کاربر اشتراک دارد ولی ربات ندارد
	if len(instances) == 0 {
		plan, _ := h.store.FindPlan(ctx, sub.PlanID.String())
		planName := ""
		if plan != nil {
			planName = plan.Name
		}
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.t(ctx, c.Sender().ID, i18n.KeyBuildWithLink), "how_to_build")))
		return c.Send(
			h.t(ctx, c.Sender().ID, i18n.KeySubActiveNoBot, planName),
			tele.ModeHTML, kb,
		)
	}

	// نمایش ربات‌ها
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🤖 ربات‌های شما</b> (%d عدد)\n\n", len(instances)))

	for i, inst := range instances {
		icon := statusIcon(inst.Status)
		statusLabel := statusLabel(inst.Status)
		sb.WriteString(fmt.Sprintf(
			"%s <b>%s</b>\n"+
				"   وضعیت: %s\n",
			icon, inst.ContainerName, statusLabel,
		))
		if inst.ExpiresAt != nil {
			rem := time.Until(*inst.ExpiresAt)
			if rem < 0 {
				sb.WriteString("   ⏰ منقضی شده\n")
			} else if rem < 72*time.Hour {
				sb.WriteString(fmt.Sprintf("   ⚠️ %d ساعت تا انقضا\n", int(rem.Hours())))
			} else {
				sb.WriteString(fmt.Sprintf("   ⏰ %d روز مانده\n", int(rem.Hours()/24)))
			}
		}
		if i < len(instances)-1 {
			sb.WriteString("\n")
		}
	}

	// نمایش وضعیت اشتراک
	if sub != nil {
		plan, _ := h.store.FindPlan(ctx, sub.PlanID.String())
		if plan != nil {
			sb.WriteString("\n──────────────\n")
			remaining := "♾ ابدی"
			if sub.ExpiresAt != nil {
				rem := time.Until(*sub.ExpiresAt)
				if rem < 0 {
					remaining = "❌ منقضی شده"
				} else {
					remaining = fmt.Sprintf("⏰ %d روز مانده", int(rem.Hours()/24))
				}
			}
			sb.WriteString(fmt.Sprintf(
				"📋 پلن: <b>%s</b>\n🤖 %d از %d ربات\n%s",
				plan.Name, sub.BotCount, plan.MaxBots, remaining,
			))
		}
	}

	// نمایش موجودی از botpay
	if h.pay != nil {
		bal, err := h.pay.Balance(ctx, u.TelegramID)
		if err == nil && bal.Total > 0 {
			sb.WriteString(fmt.Sprintf("\n💳 موجودی کیف پول: <b>%.4f TON</b>", bal.Total))
		}
	}

	return c.Send(sb.String(), tele.ModeHTML, h.kbUserFull(ctx, uid, sub))
}

// userShowWelcome برای کاربرانی که هیچ چیز ندارند.
func (h *Handler) userShowWelcome(ctx context.Context, c tele.Context, u *models.User) error {

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🆓 شروع رایگان", "start_free")),
		kb.Row(kb.Data("💎 مشاهده پلن‌ها", "show_plans")),
	)

	return c.Send(
		fmt.Sprintf(
			"👋 سلام <b>%s</b>!\n\n"+
				"با CreatorBot می‌توانید ربات تلگرام اختصاصی بسازید.\n\n"+
				"🆓 <b>پلن رایگان:</b>\n"+
				"یک ربات رایگان برای همیشه\n\n"+
				"💎 <b>پلن‌های پولی:</b>\n"+
				"چند ربات — با امکانات بیشتر\n\n"+
				"برای شروع روی «شروع رایگان» کلیک کنید:",
			c.Sender().FirstName,
		),
		tele.ModeHTML, kb,
	)
}

// ════════════════════════════════════════════════════════════
// نمایش پلن‌ها — با inline button، نه UUID
// ════════════════════════════════════════════════════════════

func (h *Handler) userPlans(ctx context.Context, c tele.Context) error {
	plans, _ := h.store.ListActivePlans(ctx)

	if len(plans) == 0 {
		return c.Send("در حال حاضر پلنی موجود نیست. بعداً دوباره بررسی کنید.", h.kbUser(ctx, c.Sender().ID))
	}

	// ── نمایش هر پلن با دکمه مخصوص خودش ──
	msg := "<b>💎 پلن‌های موجود</b>\n\n"

	for _, p := range plans {
		priceText := fmt.Sprintf("%.2f TON", p.Price)
		if p.IsFree {
			priceText = "🆓 رایگان"
		}
		dur := fmt.Sprintf("%d روز", p.DurationDay)
		if p.DurationDay == 0 {
			dur = "برای همیشه"
		}
		botWord := "ربات"
		if p.MaxBots > 1 {
			botWord = "ربات"
		}

		msg += fmt.Sprintf(
			"<b>%s</b>\n"+
				"💰 %s  |  🤖 %d %s  |  ⏳ %s\n\n",
			p.Name, priceText, p.MaxBots, botWord, dur,
		)
	}

	msg += "برای خرید روی پلن مورد نظر کلیک کنید:"

	// ── inline keyboard — یه دکمه برای هر پلن ──
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := p.Name
		if p.IsFree {
			label = "🆓 " + p.Name + " — رایگان"
		} else {
			label = fmt.Sprintf("💎 %s — %.2f TON", p.Name, p.Price)
		}
		rows = append(rows, kb.Row(kb.Data(label, "select_plan:"+p.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data("❌ بستن", "cancel")))
	kb.Inline(rows...)

	return c.Send(msg, tele.ModeHTML, kb)
}

// userSelectPlan — کاربر پلن را انتخاب کرد
func (h *Handler) userSelectPlan(ctx context.Context, c tele.Context, planID string) error {
	defer c.Respond()
	uid := c.Sender().ID

	plan, err := h.store.FindPlan(ctx, planID)
	if err != nil || plan == nil || !plan.IsActive {
		return c.Edit("این پلن دیگر در دسترس نیست.")
	}

	u, _ := h.getOrCreateUser(ctx, c)
	if u == nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
	}

	// اشتراک موجود
	existing, _ := h.store.GetActiveSubscription(ctx, u.ID)
	if existing != nil {
		existPlan, _ := h.store.FindPlan(ctx, existing.PlanID.String())
		existName := ""
		if existPlan != nil {
			existName = existPlan.Name
		}
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("❌ بستن", "cancel")))
		return c.Edit(fmt.Sprintf(
			"شما در حال حاضر پلن <b>%s</b> فعال دارید.\n\nبرای ارتقا با پشتیبانی تماس بگیرید.",
			existName,
		), tele.ModeHTML, kb)
	}

	// پلن رایگان
	if plan.IsFree {
		return h.activateFreePlanInline(ctx, c, u, plan)
	}

	// پلن پولی — نمایش جزئیات و دکمه خرید
	return h.showPlanDetail(ctx, c, u, plan)
}

// showPlanDetail جزئیات پلن + وضعیت موجودی + دکمه‌های مناسب
func (h *Handler) showPlanDetail(ctx context.Context, c tele.Context, u *models.User, plan *models.Plan) error {
	uid := c.Sender().ID

	dur := fmt.Sprintf("%d روز", plan.DurationDay)
	if plan.DurationDay == 0 {
		dur = "برای همیشه"
	}

	msg := fmt.Sprintf(
		"<b>💎 %s</b>\n\n"+
			"🤖 تعداد ربات: %d عدد\n"+
			"⏳ مدت: %s\n"+
			"💰 قیمت: <b>%.2f TON</b>\n\n",
		plan.Name, plan.MaxBots, dur, plan.Price,
	)

	kb := &tele.ReplyMarkup{}

	// بررسی موجودی
	if h.pay != nil {
		bal, err := h.pay.Balance(ctx, u.TelegramID)
		if err == nil {
			msg += fmt.Sprintf("💳 موجودی کیف پول شما: <b>%.4f TON</b>\n", bal.Total)

			if bal.Total >= plan.Price {
				// موجودی کافیه
				msg += "\n✅ موجودی شما کافی است!"
				kb.Inline(
					kb.Row(kb.Data(fmt.Sprintf("✅ خرید با %.2f TON", plan.Price), "buy_plan:"+plan.ID.String())),
					kb.Row(kb.Data("🔙 بازگشت به پلن‌ها", "show_plans")),
				)
			} else {
				// موجودی کافی نیست
				needed := plan.Price - bal.Total
				msg += fmt.Sprintf("\n⚠️ کمبود موجودی: <b>%.4f TON</b>\n\n", needed)
				msg += "برای شارژ کیف پول به ربات @BotPayBot مراجعه کنید."

				inv, err := h.pay.CreateInvoice(ctx, u.TelegramID, needed, plan.ID.String())
				if err == nil {
					kb.Inline(
						kb.Row(kb.URL("💳 شارژ کیف پول", inv.PayURL)),
						kb.Row(kb.Data("🔄 بررسی پرداخت", "check_plan:"+plan.ID.String()+":"+inv.InvoiceCode)),
						kb.Row(kb.Data("🔙 بازگشت", "show_plans")),
					)
				} else {
					kb.Inline(kb.Row(kb.Data("🔙 بازگشت", "show_plans")))
				}
			}
		} else {
			// سرویس پرداخت در دسترس نیست
			msg += "\n⚠️ سرویس پرداخت موقتاً در دسترس نیست."
			kb.Inline(kb.Row(kb.Data("🔙 بازگشت", "show_plans")))
		}
	} else {
		kb.Inline(kb.Row(kb.Data("🔙 بازگشت", "show_plans")))
	}

	_ = uid
	return c.Edit(msg, tele.ModeHTML, kb)
}

// activateFreePlanInline پلن رایگان رو activate کن
func (h *Handler) activateFreePlanInline(ctx context.Context, c tele.Context, u *models.User, plan *models.Plan) error {
	var expiresAt *time.Time
	if plan.DurationDay > 0 {
		t := time.Now().AddDate(0, 0, plan.DurationDay)
		expiresAt = &t
	}
	sub := &models.Subscription{
		UserID: u.ID, PlanID: plan.ID,
		StartedAt: time.Now(), ExpiresAt: expiresAt, IsActive: true,
	}
	if err := h.store.CreateSubscription(ctx, sub); err != nil {
		return c.Edit("❌ خطا در فعال‌سازی. دوباره تلاش کنید.")
	}

	dur := fmt.Sprintf("%d روز", plan.DurationDay)
	if plan.DurationDay == 0 {
		dur = "برای همیشه"
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("🤖 ربات‌های من", "my_bots")))

	return c.Edit(fmt.Sprintf(
		"🎉 <b>پلن رایگان فعال شد!</b>\n\n"+
			"✅ %d ربات — %s\n\n"+
			"حالا برای ساخت ربات از ادمین لینک دعوت بگیرید.",
		plan.MaxBots, dur,
	), tele.ModeHTML, kb)
}

// ════════════════════════════════════════════════════════════
// خرید پلن — اجرای نهایی
// ════════════════════════════════════════════════════════════

func (h *Handler) userBuyPlan(ctx context.Context, c tele.Context, planID string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	// redirect به select plan
	return h.userSelectPlan(ctx, c, planID)
}

func (h *Handler) executePlanPurchase(ctx context.Context, c tele.Context, planID string) error {
	defer c.Respond()
	uid := c.Sender().ID

	plan, _ := h.store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Edit("❌ پلن یافت نشد.")
	}

	u, _ := h.getOrCreateUser(ctx, c)
	if u == nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
	}

	// کسر از botpay
	_, err := h.pay.Deduct(ctx, u.TelegramID, plan.Price, plan.ID.String(), "خرید پلن "+plan.Name)
	if err != nil {
		if payclient.IsInsufficientBalance(err) {
			kb := &tele.ReplyMarkup{}
			kb.Inline(kb.Row(kb.Data("💎 شارژ کیف پول", "show_plans")))
			return c.Edit("❌ موجودی کافی نیست.\n\nکیف پول خود را شارژ کنید.", kb)
		}
		h.log.Error("executePlanPurchase", ports.F("err", err))
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
	}

	// فعال‌سازی اشتراک
	var expiresAt *time.Time
	if plan.DurationDay > 0 {
		t := time.Now().AddDate(0, 0, plan.DurationDay)
		expiresAt = &t
	}
	h.store.CreateSubscription(ctx, &models.Subscription{
		UserID: u.ID, PlanID: plan.ID,
		StartedAt: time.Now(), ExpiresAt: expiresAt, IsActive: true,
	})

	h.log.Info("plan purchased", ports.F("user", u.TelegramID), ports.F("plan", plan.Name))

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("🤖 ربات‌های من", "my_bots")))

	return c.Edit(fmt.Sprintf(
		"🎉 <b>خرید موفق!</b>\n\n"+
			"✅ پلن <b>%s</b> فعال شد\n"+
			"🤖 %d ربات در اختیار شماست\n\n"+
			"برای ساخت ربات از ادمین لینک دعوت بگیرید.",
		plan.Name, plan.MaxBots,
	), tele.ModeHTML, kb)
}

func (h *Handler) checkPlanAfterDeposit(ctx context.Context, c tele.Context, planID, invoiceCode string) error {
	defer c.Respond()
	
	plan, _ := h.store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Edit("❌ پلن یافت نشد.")
	}

	u, _ := h.getOrCreateUser(ctx, c)
	bal, err := h.pay.Balance(ctx, u.TelegramID)
	if err != nil {
		return c.Edit("❌ خطا در بررسی موجودی. دوباره تلاش کنید.")
	}

	if bal.Total < plan.Price {
		needed := plan.Price - bal.Total
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.Data("🔄 بررسی مجدد", "check_plan:"+planID+":"+invoiceCode)),
			kb.Row(kb.Data("🔙 بازگشت", "show_plans")),
		)
		return c.Edit(fmt.Sprintf(
			"⏳ پرداخت هنوز تأیید نشده.\n\n"+
				"💳 موجودی فعلی: <b>%.4f TON</b>\n"+
				"💰 نیاز: <b>%.2f TON</b>\n"+
				"⚠️ کمبود: %.4f TON\n\n"+
				"چند دقیقه صبر کنید و دوباره بررسی کنید.",
			bal.Total, plan.Price, needed,
		), tele.ModeHTML, kb)
	}

	return h.executePlanPurchase(ctx, c, planID)
}

// ════════════════════════════════════════════════════════════
// بررسی ظرفیت
// ════════════════════════════════════════════════════════════

func (h *Handler) checkBuildCapacity(ctx context.Context, c tele.Context) (bool, error) {
	uid := c.Sender().ID
	u, _ := h.getOrCreateUser(ctx, c)
	if u == nil {
		return false, c.Send(h.t(ctx, uid, i18n.KeyError))
	}

	sub, _ := h.store.GetActiveSubscription(ctx, u.ID)
	if sub == nil {
		// کاربر هیچ اشتراکی ندارد
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.Data("🆓 شروع رایگان", "start_free")),
			kb.Row(kb.Data("💎 مشاهده پلن‌ها", "show_plans")),
		)
		return false, c.Send(
			"برای ساخت ربات باید ابتدا یک پلن داشته باشید.\n\nیک ربات رایگان می‌توانید داشته باشید:",
			kb,
		)
	}

	plan, _ := h.store.FindPlan(ctx, sub.PlanID.String())
	if plan == nil {
		return false, nil
	}

	if !sub.HasCapacity(plan.MaxBots) {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("💎 ارتقای پلن", "show_plans")))
		return false, c.Send(fmt.Sprintf(
			"❌ به حداکثر ربات رسیده‌اید.\n\n"+
				"🤖 %d از %d ربات استفاده شده\n\n"+
				"برای ساخت ربات بیشتر پلن خود را ارتقا دهید.",
			sub.BotCount, plan.MaxBots,
		), kb)
	}

	return true, nil
}

// ════════════════════════════════════════════════════════════
// Helpers
// ════════════════════════════════════════════════════════════

func statusIcon(s models.InstanceStatus) string {
	switch s {
	case models.StatusRunning:
		return "🟢"
	case models.StatusStopped:
		return "🔴"
	case models.StatusPending:
		return "🟡"
	case models.StatusError:
		return "⚠️"
	}
	return "⚪️"
}

func statusLabel(s models.InstanceStatus) string {
	switch s {
	case models.StatusRunning:
		return "در حال اجرا"
	case models.StatusStopped:
		return "متوقف"
	case models.StatusPending:
		return "در حال راه‌اندازی..."
	case models.StatusError:
		return "خطا — با پشتیبانی تماس بگیرید"
	}
	return string(s)
}

func fmtInstanceUser(inst models.BotInstance) string {
	return fmt.Sprintf("%s <b>%s</b> — %s",
		statusIcon(inst.Status), inst.ContainerName, statusLabel(inst.Status))
}

func (h *Handler) userSupport(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	return c.Send(h.t(ctx, uid, i18n.KeySupportText), tele.ModeHTML, h.kbUser(ctx, uid))
}

func (h *Handler) kbUserFull(ctx context.Context, uid int64, sub *models.Subscription) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	hasSub := sub != nil && sub.IsActive
	if hasSub {
		kb.Reply(
			kb.Row(kb.Text(h.btn(ctx, uid, i18n.KeyMenuMyBots))),
			kb.Row(kb.Text(h.btn(ctx, uid, i18n.KeyMenuHelp)), kb.Text(h.btn(ctx, uid, i18n.KeyMenuSupport))),
		)
	} else {
		kb.Reply(
			kb.Row(kb.Text(h.btn(ctx, uid, i18n.KeyMenuMyBots))),
			kb.Row(kb.Text("💎 خرید پلن")),
			kb.Row(kb.Text(h.btn(ctx, uid, i18n.KeyMenuHelp)), kb.Text(h.btn(ctx, uid, i18n.KeyMenuSupport))),
		)
	}
	return kb
}
