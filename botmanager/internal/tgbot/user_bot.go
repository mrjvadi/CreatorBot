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
	"github.com/mrjvadi/creatorbot/shared/pkg/metrics"
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

	// بدون ربات و اشتراک
	if len(instances) == 0 && sub == nil {
		return h.userShowWelcome(ctx, c, u)
	}

	// اشتراک دارد ولی ربات ندارد
	if len(instances) == 0 {
		plan, _ := h.store.FindPlan(ctx, sub.PlanID.String())
		planName := ""
		if plan != nil {
			planName = plan.Name
		}
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("➕ ایجاد سرویس", "svc_create")))
		return c.Send(
			h.t(ctx, uid, i18n.KeySubActiveNoBot, planName),
			tele.ModeHTML, kb,
		)
	}

	// ── ارسال هر سرویس به صورت جداگانه ───────────────────
	// اول خلاصه
	header := fmt.Sprintf("<b>🤖 سرویس‌های من</b> (%d سرویس)\n", len(instances))
	c.Send(header, tele.ModeHTML)

	for _, inst := range instances {
		// نوع سرویس از template
		serviceType := "سرویس"
		serviceIcon := "🤖"
		if tmpl, _ := h.store.FindTemplate(ctx, inst.TemplateID); tmpl != nil {
			switch tmpl.Type {
			case "vpn":
				serviceType = "VPN"
				serviceIcon = "🌐"
			case "uploader":
				serviceType = "آپلودر"
				serviceIcon = "📤"
			case "member":
				serviceType = "قفل ممبرشیپ"
				serviceIcon = "🔒"
			case "archive":
				serviceType = "آرشیو"
				serviceIcon = "📦"
			}
		}

		icon := statusIcon(inst.Status)
		statusLbl := statusLabel(inst.Status)

		// متن کارت سرویس
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s <b>%s</b>\n", serviceIcon, serviceType))
		sb.WriteString(fmt.Sprintf("📛 نام: <code>%s</code>\n", inst.ContainerName))
		sb.WriteString(fmt.Sprintf("%s وضعیت: <b>%s</b>\n", icon, statusLbl))
		if inst.ExpiresAt != nil {
			rem := time.Until(*inst.ExpiresAt)
			if rem < 0 {
				sb.WriteString("⏰ <b>منقضی شده</b>\n")
			} else if rem < 72*time.Hour {
				sb.WriteString(fmt.Sprintf("⚠️ %d ساعت تا انقضا\n", int(rem.Hours())))
			} else {
				sb.WriteString(fmt.Sprintf("⏰ %d روز مانده\n", int(rem.Hours()/24)))
			}
		}

		// دکمه‌های مخصوص این سرویس
		id := inst.ID.String()
		kb := &tele.ReplyMarkup{}

		switch inst.Status {
		case "running":
			kb.Inline(
				kb.Row(
					kb.Data("📊 آمار", "svc_stats:"+id),
					kb.Data("⚙️ تنظیمات", "svc_settings:"+id),
				),
				kb.Row(
					kb.Data("🔄 ری‌استارت", "bot_restart:"+id),
					kb.Data("⏸ توقف", "bot_stop:"+id),
				),
				kb.Row(kb.Data("🗑 حذف سرویس", "bot_delete:"+id)),
			)
		case "stopped":
			kb.Inline(
				kb.Row(
					kb.Data("▶️ شروع", "bot_start:"+id),
					kb.Data("🗑 حذف", "bot_delete:"+id),
				),
			)
		case "pending", "provisioning":
			kb.Inline(
				kb.Row(kb.Data("🔄 بررسی وضعیت", "svc_status:"+id)),
			)
		case "failed":
			kb.Inline(
				kb.Row(
					kb.Data("🔄 تلاش مجدد", "bot_restart:"+id),
					kb.Data("🗑 حذف", "bot_delete:"+id),
				),
			)
		default:
			kb.Inline(kb.Row(kb.Data("🗑 حذف", "bot_delete:"+id)))
		}

		c.Send(sb.String(), tele.ModeHTML, kb)
	}

	// دکمه ایجاد سرویس جدید در آخر
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("➕ ایجاد سرویس جدید", "svc_create")),
		kb.Row(kb.Data("🔙 بازگشت", "back_main")),
	)
	return c.Send("─────────────────────", kb)
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
		return c.Edit(h.t(ctx, uid, i18n.KeyNotFound))
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
	if err := h.store.CreateSubscription(ctx, sub); err != nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
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
		return c.Edit(h.t(ctx, uid, i18n.KeyNotFound))
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
	newSub := &models.Subscription{
		UserID: u.ID, PlanID: plan.ID,
		StartedAt: time.Now(), ExpiresAt: expiresAt, IsActive: true,
	}
	h.store.CreateSubscription(ctx, newSub)

	// ── reset quota cache ─────────────────────────────────
	// plan.upgraded → سرویس‌ها quota را ریست می‌کنند
	if h.nc != nil {
		h.nc.PublishCore("plan.upgraded", map[string]any{
			"user_id":     u.ID,
			"telegram_id": u.TelegramID,
			"plan_id":     plan.ID,
			"plan_name":   plan.Name,
			"max_bots":    plan.MaxBots,
		})
	}

	h.log.Info("plan purchased", ports.F("user", u.TelegramID), ports.F("plan", plan.Name))
	metrics.IncPlanPurchase(plan.Name, "success")
	h.auditLog(ctx, u.ID, string(u.Role), plan.ID.String(), "plan", models.AuditBuyPlan,
		plan.Name)

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
	uid := c.Sender().ID
	defer c.Respond()

	plan, _ := h.store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyNotFound))
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

// checkBuildCapacityForType بررسی ظرفیت به تفکیک نوع ربات (Capacity Engine).
func (h *Handler) checkBuildCapacityForType(ctx context.Context, c tele.Context, botType string) (bool, error) {
	uid := c.Sender().ID
	u, _ := h.getOrCreateUser(ctx, c)
	if u == nil {
		return false, c.Send(h.t(ctx, uid, i18n.KeyError))
	}

	ok, current, limit, err := h.store.CanCreateInstance(ctx, u.ID, botType)
	if err != nil {
		h.log.Error("capacity check", ports.F("err", err))
		return false, c.Send(h.t(ctx, uid, i18n.KeyError))
	}

	if ok {
		return true, nil
	}

	// دلیل رد را تشخیص بده
	sub, _ := h.store.GetActiveSubscription(ctx, u.ID)
	if sub == nil {
		// اشتراکی ندارد → پیشنهاد پلن
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.Data("🆓 شروع رایگان", "start_free")),
			kb.Row(kb.Data("💎 مشاهده پلن‌ها", "show_plans")),
		)
		return false, c.Send(h.t(ctx, uid, i18n.KeyNoPlan), kb)
	}

	if limit <= 0 {
		// این نوع ربات در پلن مجاز نیست
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("💎 ارتقای پلن", "show_plans")))
		return false, c.Send(fmt.Sprintf(
			"❌ پلن فعلی شما اجازه ساخت ربات <b>%s</b> را نمی‌دهد.\n\nبرای دسترسی، پلن خود را ارتقا دهید.",
			botType,
		), tele.ModeHTML, kb)
	}

	// به سقف رسیده
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("💎 ارتقای پلن", "show_plans")))
	return false, c.Send(fmt.Sprintf(
		"❌ به حداکثر ربات <b>%s</b> رسیده‌اید (%d از %d).\n\nبرای ساخت بیشتر، پلن خود را ارتقا دهید.",
		botType, current, limit,
	), tele.ModeHTML, kb)
}

// instanceAction Stop/Start/Restart/Delete یک instance را از طریق apimanager انجام می‌دهد.
func (h *Handler) instanceAction(ctx context.Context, c tele.Context, instIDStr, action string) error {
	defer c.Respond()
	uid := c.Sender().ID

	inst, err := h.store.FindInstance(ctx, instIDStr)
	if err != nil || inst == nil {
		return c.Edit("❌ ربات یافت نشد.")
	}

	// بررسی owner بودن
	u, _ := h.getOrCreateUser(ctx, c)
	if u == nil || inst.OwnerID != u.ID {
		return c.Edit("❌ دسترسی ندارید.")
	}

	// publish command به apimanager از طریق NATS
	subject := fmt.Sprintf("instance.%s", action)
	if h.nc != nil {
		h.nc.PublishCore(subject, map[string]any{
			"instance_id":    inst.ID,
			"container_name": inst.ContainerName,
			"server_id":      inst.ServerID,
			"action":         action,
		})
	}

	labels := map[string]string{
		"stop":    "⏹ دستور توقف ارسال شد",
		"start":   "▶️ دستور شروع ارسال شد",
		"restart": "🔄 دستور ری‌استارت ارسال شد",
		"delete":  "🗑 دستور حذف ارسال شد",
	}

	h.log.Info("instance action",
		ports.F("instance", inst.ID),
		ports.F("action", action),
		ports.F("user", uid))

	return c.Edit(labels[action] + "\n\n" + inst.ContainerName)
}

// bot_stop referenced in router.go

// ── Wallet Handlers ──────────────────────────────────────

// ── Communities Handlers ─────────────────────────────────

// ── Ads Handlers ─────────────────────────────────────────

// ── Settings Handlers ─────────────────────────────────────

// sendWalletHome نمایش صفحه کیف پول با موجودی.

// instanceStatus وضعیت فعلی سرویس را نمایش می‌دهد.
func (h *Handler) instanceStatus(ctx context.Context, c tele.Context, idStr string) error {
	defer c.Respond()
	inst, err := h.store.FindInstance(ctx, idStr)
	if err != nil || inst == nil {
		return c.Edit("❌ سرویس یافت نشد.")
	}
	icon := statusIcon(inst.Status)
	lbl := statusLabel(inst.Status)
	return c.Edit(fmt.Sprintf("%s وضعیت سرویس: <b>%s</b>", icon, lbl), tele.ModeHTML)
}

// instanceStats آمار ساده یک سرویس.
func (h *Handler) instanceStats(ctx context.Context, c tele.Context, idStr string) error {
	defer c.Respond()
	inst, err := h.store.FindInstance(ctx, idStr)
	if err != nil || inst == nil {
		return c.Edit("❌ سرویس یافت نشد.")
	}
	msg := fmt.Sprintf(
		"📊 <b>آمار سرویس</b>\n\n"+
			"📛 نام: <code>%s</code>\n"+
			"%s وضعیت: <b>%s</b>",
		inst.ContainerName,
		statusIcon(inst.Status),
		statusLabel(inst.Status),
	)
	return c.Edit(msg, tele.ModeHTML)
}

// adminUserAction اعمال action روی کاربر.
func (h *Handler) adminUserAction(ctx context.Context, c tele.Context, idStr, action string) error {
	defer c.Respond()
	return h.adminUserHandleAction(ctx, c, idStr, action)
}
