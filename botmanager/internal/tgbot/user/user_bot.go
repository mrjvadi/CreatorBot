package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ════════════════════════════════════════════════════════════
// ربات‌های من
// ════════════════════════════════════════════════════════════

func (h *User) UserBotsList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	u, err := h.GetOrCreateUser(ctx, c)
	if err != nil || u == nil {
		return c.Send(h.T(ctx, uid, i18n.KeyError))
	}
	if u.IsBlocked {
		return c.Send(h.T(ctx, uid, i18n.KeyBlocked))
	}

	instances, _ := h.Store.ListInstancesByOwner(ctx, u.ID)
	sub, _ := h.Store.GetActiveSubscription(ctx, u.ID)

	// بدون ربات و اشتراک
	if len(instances) == 0 && sub == nil {
		return h.UserShowWelcome(ctx, c, u)
	}

	// اشتراک دارد ولی ربات ندارد
	if len(instances) == 0 {
		plan, _ := h.Store.FindPlan(ctx, sub.PlanID.String())
		planName := ""
		if plan != nil {
			planName = plan.Name
		}
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCreateSvc), "svc_create")))
		return c.Send(
			h.T(ctx, uid, i18n.KeySubActiveNoBot, planName),
			tele.ModeHTML, kb,
		)
	}

	// ── ارسال هر سرویس به صورت جداگانه ───────────────────
	// اول خلاصه
	_ = c.Send(h.T(ctx, uid, i18n.KeyMyServicesHeader, len(instances)), tele.ModeHTML)

	for _, inst := range instances {
		// نوع سرویس از template — پویا (نوع از DB، آیکن graceful)
		serviceType := h.T(ctx, uid, i18n.KeyServiceGeneric)
		serviceIcon := "🤖"
		if tmpl, _ := h.Store.FindTemplate(ctx, inst.TemplateID); tmpl != nil {
			serviceType = tmpl.Type
			serviceIcon = format.BotTypeEmoji(models.BotType(tmpl.Type))
		}

		icon := format.StatusIcon(inst.Status)
		statusLbl := h.StatusLabel(ctx, uid, inst.Status)

		// متن کارت سرویس
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s <b>%s</b>\n", serviceIcon, serviceType))
		sb.WriteString(h.T(ctx, uid, i18n.KeySvcNameLine, inst.ContainerName))
		sb.WriteString(h.T(ctx, uid, i18n.KeySvcStatusLine, icon, statusLbl))
		if inst.ExpiresAt != nil {
			rem := time.Until(*inst.ExpiresAt)
			if rem < 0 {
				sb.WriteString(h.T(ctx, uid, i18n.KeySvcExpiredNL))
			} else if rem < 72*time.Hour {
				sb.WriteString(h.T(ctx, uid, i18n.KeySvcHoursLeft, int(rem.Hours())))
			} else {
				sb.WriteString(h.T(ctx, uid, i18n.KeySvcDaysLeft, int(rem.Hours()/24)))
			}
		}

		// دکمه‌های مخصوص این سرویس
		id := inst.ID.String()
		kb := &tele.ReplyMarkup{}

		switch inst.Status {
		case "running":
			kb.Inline(
				kb.Row(
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStats), "svc_stats:"+id),
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnSettings), "svc_settings:"+id),
				),
				kb.Row(
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnRestart), "bot_restart:"+id),
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStop), "bot_stop:"+id),
				),
				kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDeleteSvc), "bot_delete:"+id)),
			)
		case "stopped":
			kb.Inline(
				kb.Row(
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStart), "bot_start:"+id),
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDelete), "bot_delete:"+id),
				),
			)
		case "pending", "provisioning":
			kb.Inline(
				kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCheckStatus), "svc_status:"+id)),
			)
		case "failed":
			kb.Inline(
				kb.Row(
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnRetry), "bot_restart:"+id),
					kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDelete), "bot_delete:"+id),
				),
			)
		default:
			kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDelete), "bot_delete:"+id)))
		}

		_ = c.Send(sb.String(), tele.ModeHTML, kb)
	}

	// دکمه ایجاد سرویس جدید در آخر
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCreateNewSvc), "svc_create")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "back_main")),
	)
	return c.Send("─────────────────────", kb)
}

// userShowWelcome برای کاربرانی که هیچ چیز ندارند.
func (h *User) UserShowWelcome(ctx context.Context, c tele.Context, u *models.User) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStartFree), "start_free")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnViewPlans), "show_plans")),
	)

	return c.Send(
		h.T(ctx, uid, i18n.KeyWelcomeNoService, c.Sender().FirstName),
		tele.ModeHTML, kb,
	)
}

// ════════════════════════════════════════════════════════════
// بررسی ظرفیت
// ════════════════════════════════════════════════════════════

func (h *User) CheckBuildCapacity(ctx context.Context, c tele.Context) (bool, error) {
	uid := c.Sender().ID
	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil {
		return false, c.Send(h.T(ctx, uid, i18n.KeyError))
	}

	sub, _ := h.Store.GetActiveSubscription(ctx, u.ID)
	if sub == nil {
		// کاربر هیچ اشتراکی ندارد
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStartFree), "start_free")),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnViewPlans), "show_plans")),
		)
		return false, c.Send(h.T(ctx, uid, i18n.KeyNeedPlanFirst), kb)
	}

	plan, _ := h.Store.FindPlan(ctx, sub.PlanID.String())
	if plan == nil {
		return false, nil
	}

	if !sub.HasCapacity(plan.MaxBots) {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnUpgradePlan), "show_plans")))
		return false, c.Send(
			h.T(ctx, uid, i18n.KeyMaxBotsReached, sub.BotCount, plan.MaxBots), kb)
	}

	return true, nil
}

// ════════════════════════════════════════════════════════════
// Helpers
// ════════════════════════════════════════════════════════════

func (h *User) StatusLabel(ctx context.Context, uid int64, s models.InstanceStatus) string {
	switch s {
	case models.StatusRunning:
		return h.T(ctx, uid, i18n.KeyStatusRunning)
	case models.StatusStopped:
		return h.T(ctx, uid, i18n.KeyStatusStopped)
	case models.StatusPending:
		return h.T(ctx, uid, i18n.KeyStatusStarting)
	case models.StatusError:
		return h.T(ctx, uid, i18n.KeyStatusErrContact)
	}
	return string(s)
}

func (h *User) UserSupport(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	return c.Send(h.T(ctx, uid, i18n.KeySupportText), tele.ModeHTML, h.KbUser(ctx, uid))
}

// checkBuildCapacityForType بررسی ظرفیت به تفکیک نوع ربات (Capacity Engine).
func (h *User) CheckBuildCapacityForType(ctx context.Context, c tele.Context, botType string) (bool, error) {
	uid := c.Sender().ID
	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil {
		return false, c.Send(h.T(ctx, uid, i18n.KeyError))
	}

	ok, current, limit, err := h.Store.CanCreateInstance(ctx, u.ID, botType)
	if err != nil {
		h.Log.Error("capacity check", ports.F("err", err))
		return false, c.Send(h.T(ctx, uid, i18n.KeyError))
	}

	if ok {
		return true, nil
	}

	// دلیل رد را تشخیص بده
	sub, _ := h.Store.GetActiveSubscription(ctx, u.ID)
	if sub == nil {
		// اشتراکی ندارد → پیشنهاد پلن
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStartFree), "start_free")),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnViewPlans), "show_plans")),
		)
		return false, c.Send(h.T(ctx, uid, i18n.KeyNoPlan), kb)
	}

	if limit <= 0 {
		// این نوع ربات در پلن مجاز نیست
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnUpgradePlan), "show_plans")))
		return false, c.Send(
			h.T(ctx, uid, i18n.KeyTypeNotAllowed, botType), tele.ModeHTML, kb)
	}

	// به سقف رسیده
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnUpgradePlan), "show_plans")))
	return false, c.Send(
		h.T(ctx, uid, i18n.KeyMaxBotsReachedType, botType, current, limit), tele.ModeHTML, kb)
}

// instanceAction Stop/Start/Restart/Delete یک instance را از طریق apimanager انجام می‌دهد.
func (h *User) InstanceAction(ctx context.Context, c tele.Context, instIDStr, action string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID

	inst, err := h.Store.FindInstance(ctx, instIDStr)
	if err != nil || inst == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNotFound))
	}

	// بررسی owner بودن
	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil || inst.OwnerID != u.ID {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNoAccess))
	}

	// publish command به apimanager از طریق NATS
	subject := fmt.Sprintf("instance.%s", action)
	if h.NC != nil {
		_ = h.NC.PublishCore(subject, map[string]any{
			"instance_id":    inst.ID,
			"container_name": inst.ContainerName,
			"server_id":      inst.ServerID,
			"action":         action,
		})
	}

	labelKeys := map[string]i18n.Key{
		"stop":    i18n.KeyActionStopSent,
		"start":   i18n.KeyActionStartSent,
		"restart": i18n.KeyActionRestartSent,
		"delete":  i18n.KeyActionDeleteSent,
	}

	h.Log.Info("instance action",
		ports.F("instance", inst.ID),
		ports.F("action", action),
		ports.F("user", uid))

	return c.Edit(h.T(ctx, uid, labelKeys[action]) + "\n\n" + inst.ContainerName)
}

// bot_stop referenced in router.go

// ── Wallet Handlers ──────────────────────────────────────

// ── Communities Handlers ─────────────────────────────────

// ── Ads Handlers ─────────────────────────────────────────

// ── Settings Handlers ─────────────────────────────────────

// sendWalletHome نمایش صفحه کیف پول با موجودی.

// instanceStatus وضعیت فعلی سرویس را نمایش می‌دهد.
func (h *User) InstanceStatus(ctx context.Context, c tele.Context, idStr string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID
	inst, err := h.Store.FindInstance(ctx, idStr)
	if err != nil || inst == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNotFound))
	}
	icon := format.StatusIcon(inst.Status)
	lbl := h.StatusLabel(ctx, uid, inst.Status)
	return c.Edit(h.T(ctx, uid, i18n.KeySvcStatusShort, icon, lbl), tele.ModeHTML)
}

// instanceStats آمار ساده یک سرویس.
func (h *User) InstanceStats(ctx context.Context, c tele.Context, idStr string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID
	inst, err := h.Store.FindInstance(ctx, idStr)
	if err != nil || inst == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNotFound))
	}
	msg := h.T(ctx, uid, i18n.KeySvcStatsDetail,
		inst.ContainerName,
		format.StatusIcon(inst.Status),
		h.StatusLabel(ctx, uid, inst.Status),
	)
	return c.Edit(msg, tele.ModeHTML)
}

// instanceSettings نمایش تنظیمات و اطلاعات کامل یک سرویس.
func (h *User) InstanceSettings(ctx context.Context, c tele.Context, idStr string) error {
	defer func() { _ = c.Respond() }()
	uid := c.Sender().ID

	inst, err := h.Store.FindInstance(ctx, idStr)
	if err != nil || inst == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNotFound))
	}

	// بررسی مالکیت
	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil || inst.OwnerID != u.ID {
		return c.Edit(h.T(ctx, uid, i18n.KeyInstanceNoAccess))
	}

	// نوع سرویس از template — پویا
	serviceType := h.T(ctx, uid, i18n.KeyServiceGeneric)
	serviceIcon := "🤖"
	if tmpl, _ := h.Store.FindTemplate(ctx, inst.TemplateID); tmpl != nil {
		serviceType = tmpl.Type
		serviceIcon = format.BotTypeEmoji(models.BotType(tmpl.Type))
	}

	// اطلاعات سرور
	serverName := h.T(ctx, uid, i18n.KeyUnknown)
	if srv, _ := h.Store.FindServerByID(ctx, inst.ServerID.String()); srv != nil {
		serverName = srv.Name + " (" + srv.IP + ")"
	}

	// اطلاعات پلن و انقضا
	extraInfo := ""
	if inst.ExpiresAt != nil {
		rem := time.Until(*inst.ExpiresAt)
		if rem < 0 {
			extraInfo = h.T(ctx, uid, i18n.KeyExpiredLabel)
		} else {
			extraInfo = h.T(ctx, uid, i18n.KeyDaysUntilExpiry, int(rem.Hours()/24))
		}
	}
	if inst.PlanID != nil {
		if plan, _ := h.Store.FindPlan(ctx, inst.PlanID.String()); plan != nil {
			if extraInfo != "" {
				extraInfo += "\n"
			}
			extraInfo += h.T(ctx, uid, i18n.KeyPlanLine, plan.Name)
		}
	}

	msg := h.T(ctx, uid, i18n.KeySvcSettingsDetail,
		inst.ContainerName,
		serviceIcon+" "+serviceType,
		format.StatusIcon(inst.Status),
		h.StatusLabel(ctx, uid, inst.Status),
		serverName,
		extraInfo,
	)

	// keyboard بر اساس وضعیت
	kb := &tele.ReplyMarkup{}
	switch inst.Status {
	case "running":
		kb.Inline(
			kb.Row(
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnRestart), "bot_restart:"+idStr),
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStop), "bot_stop:"+idStr),
			),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnRenew), "svc_renew:"+idStr)),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDeleteSvc), "bot_delete:"+idStr)),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "my_bots")),
		)
	case "stopped":
		kb.Inline(
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnStart), "bot_start:"+idStr)),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnRenew), "svc_renew:"+idStr)),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDeleteSvc), "bot_delete:"+idStr)),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "my_bots")),
		)
	default:
		kb.Inline(
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnDeleteSvc), "bot_delete:"+idStr)),
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "my_bots")),
		)
	}

	return c.Edit(msg, tele.ModeHTML, kb)
}
