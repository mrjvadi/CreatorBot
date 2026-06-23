package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Admin) AdminPlansList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	plans, _ := h.Store.ListPlans(ctx)
	templates, _ := h.Store.ListTemplates(ctx)

	lines := []string{h.T(ctx, uid, i18n.KeyPlansTitle), ""}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row

	if len(plans) == 0 {
		lines = append(lines, h.T(ctx, uid, i18n.KeyPlansEmpty))
	} else {
		for _, p := range plans {
			priceStr := fmt.Sprintf("%.2f", p.Price)
			if p.IsFree {
				priceStr = h.T(ctx, uid, i18n.KeyFree)
			}
			status := "✅"
			if !p.IsActive {
				status = "⛔"
			}
			lines = append(lines, h.T(ctx, uid, i18n.KeyAdminPlanRow,
				status, p.Name, priceStr, p.DurationDay, p.MaxBots))
			rows = append(rows, kb.Row(kb.Data(
				h.T(ctx, uid, i18n.KeyBtnEditPlan, p.Name), "plan_edit:"+p.ID.String())))
		}
	}
	lines = append(lines, "")

	if len(templates) > 0 {
		rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnNewPlan), "start_plan_add")))
	} else {
		lines = append(lines, h.T(ctx, uid, i18n.KeyPlansNoTemplate))
	}
	kb.Inline(rows...)
	return c.Send(format.JoinLines(lines), tele.ModeHTML, kb)
}

// ── پنل ویرایش پلن با دکمه‌های ➕➖ ──────────────────────────────

// adminPlanEdit پنل ویرایش پلن با دکمه‌های inline.
func (h *Admin) AdminPlanEdit(ctx context.Context, c tele.Context, planIDStr string) error {
	uid := c.Sender().ID
	plan, err := h.Store.FindPlanWithLimits(ctx, planIDStr)
	if err != nil || plan == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyPlanNotFound))
	}

	priceStr := fmt.Sprintf("%.2f TON", plan.Price)
	if plan.IsFree {
		priceStr = "🆓 " + h.T(ctx, uid, i18n.KeyFree)
	}
	dur := h.T(ctx, uid, i18n.KeyDaysCount, plan.DurationDay)
	if plan.DurationDay == 0 {
		dur = h.T(ctx, uid, i18n.KeyForever)
	}
	status := h.T(ctx, uid, i18n.KeyStatusActive)
	if !plan.IsActive {
		status = h.T(ctx, uid, i18n.KeyStatusInactive)
	}

	msg := h.T(ctx, uid, i18n.KeyPlanEditTitle, plan.Name, priceStr, dur, status)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row

	// ردیف‌های +/- برای هر نوع سرویس (پویا از DB)
	types, _ := h.Store.ListServiceTypes(ctx)
	for _, bt := range types {
		current := plan.LimitFor(bt)
		label := fmt.Sprintf("%s %s:  %d", format.BotTypeEmoji(models.BotType(bt)), bt, current)
		rows = append(rows, kb.Row(
			kb.Data("➖", "plim_dn:"+planIDStr+":"+bt),
			kb.Data(label, "plan_edit:"+planIDStr), // نمایش — کلیک refresh
			kb.Data("➕", "plim_up:"+planIDStr+":"+bt),
		))
	}

	// ردیف سقف کلی
	rows = append(rows, kb.Row(
		kb.Data("➖", "pmb_dn:"+planIDStr),
		kb.Data(h.T(ctx, uid, i18n.KeyBtnTotalCap, plan.MaxBots), "plan_edit:"+planIDStr),
		kb.Data("➕", "pmb_up:"+planIDStr),
	))

	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBackToPlans), "admin_plans_back")))
	kb.Inline(rows...)

	return c.Edit(msg, tele.ModeHTML, kb)
}

// adminPlanLimitChange مقدار limit یک نوع ربات را ۱ واحد تغییر می‌دهد.
func (h *Admin) AdminPlanLimitChange(ctx context.Context, c tele.Context, planIDStr, botType string, delta int) error {
	uid := c.Sender().ID
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyErrShort)})
	}

	plan, _ := h.Store.FindPlanWithLimits(ctx, planIDStr)
	if plan == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyPlanNotFound)})
	}

	current := plan.LimitFor(botType)
	newVal := current + delta
	if newVal < 0 {
		newVal = 0
	}

	if err := h.Store.SetPlanLimit(ctx, planID, botType, newVal); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyErrSave)})
	}

	return h.AdminPlanEdit(ctx, c, planIDStr)
}

// adminPlanMaxBotsChange سقف کلی پلن را ۱ واحد تغییر می‌دهد.
func (h *Admin) AdminPlanMaxBotsChange(ctx context.Context, c tele.Context, planIDStr string, delta int) error {
	uid := c.Sender().ID
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyErrShort)})
	}

	plan, _ := h.Store.FindPlanWithLimits(ctx, planIDStr)
	if plan == nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyPlanNotFound)})
	}

	newVal := plan.MaxBots + delta
	if newVal < 0 {
		newVal = 0
	}

	if err := h.Store.UpdatePlanMaxBots(ctx, planID, newVal); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: h.T(ctx, uid, i18n.KeyErrSave)})
	}

	return h.AdminPlanEdit(ctx, c, planIDStr)
}

// adminStartAddPlan شروع فرآیند افزودن پلن جدید — از طریق دکمه‌ی inline.
func (h *Admin) AdminStartAddPlan(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	templates, _ := h.Store.ListTemplates(ctx)
	if len(templates) == 0 {
		return c.Edit(h.T(ctx, uid, i18n.KeyPlansNoTemplate))
	}

	lines := []string{h.T(ctx, uid, i18n.KeyAvailableTemplates), ""}
	for _, t := range templates {
		free := ""
		if t.IsFree {
			free = " 🆓"
		}
		lines = append(lines, fmt.Sprintf("  %s %s%s — ID: <code>%s</code>",
			format.BotTypeEmoji(models.BotType(t.Type)), t.Name, free, t.ID))
	}
	lines = append(lines, "")
	lines = append(lines, h.T(ctx, uid, i18n.KeyPlanAskTemplate))

	h.SetStep(ctx, uid, state.StepPlanTmpl)
	return c.Edit(format.JoinLines(lines), tele.ModeHTML, h.KbBackCancel(ctx, uid))
}

func (h *Admin) AdminPlanStepName(ctx context.Context, c tele.Context, tmplID string) error {
	uid := c.Sender().ID
	tmpl, err := h.Store.FindTemplate(ctx, tmplID)
	if err != nil || tmpl == nil {
		return c.Send(h.T(ctx, uid, i18n.KeyPlanTmplNotFound), h.KbBackCancel(ctx, uid))
	}
	h.SetStep(ctx, uid, state.StepPlanName, "tmpl_id", tmplID)
	return c.Send(
		h.T(ctx, uid, i18n.KeyPlanTmplChosen, tmpl.Name, h.T(ctx, uid, i18n.KeyPlanAskName)),
		tele.ModeHTML, h.KbBackCancel(ctx, uid),
	)
}

func (h *Admin) AdminPlanAdd(ctx context.Context, c tele.Context, tmplID, name string, days int, price float64) error {
	uid := c.Sender().ID

	tmpl, _ := h.Store.FindTemplate(ctx, tmplID)
	if tmpl == nil {
		h.ClearState(ctx, uid)
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyNotFound))
	}

	isFree := price == 0
	plan := &models.Plan{
		TemplateID:  &tmpl.ID,
		Name:        name,
		DurationDay: days,
		Price:       price,
		IsFree:      isFree,
		MaxBots:     1,
		IsActive:    true,
	}
	if err := h.Store.CreatePlan(ctx, plan); err != nil {
		h.ClearState(ctx, uid)
		h.Log.Error("adminPlanAdd", h.F("err", err))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyPlanAddError))
	}

	// config.updated — bot های در حال اجرا را آپدیت کن
	if h.NC != nil {
		_ = h.NC.PublishCore("config.updated", map[string]any{
			"type": "plan", "id": plan.ID.String(),
		})
	}

	// مرحله بعد: limit به تفکیک نوع ربات (انواع پویا از DB)
	h.SetStep(ctx, uid, state.StepPlanLimits, "plan_id", plan.ID.String())
	types, _ := h.Store.ListServiceTypes(ctx)
	return c.Send(
		h.T(ctx, uid, i18n.KeyPlanLimitsPrompt, strings.Join(types, ", ")),
		tele.ModeHTML, h.KbBackCancel(ctx, uid),
	)
}

// adminPlanSetLimits ورودی limit ها را پردازش می‌کند.
func (h *Admin) AdminPlanSetLimits(ctx context.Context, c tele.Context, planIDStr, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyError))
	}

	// انواع معتبر پویا از روی سرویس‌های موجود در DB
	types, _ := h.Store.ListServiceTypes(ctx)
	validTypes := make(map[string]bool, len(types))
	for _, t := range types {
		validTypes[t] = true
	}
	limits := map[string]int{}

	text = strings.TrimSpace(text)
	if n, err := strconv.Atoi(text); err == nil && n >= 0 {
		// یک عدد → همه انواع
		for t := range validTypes {
			limits[t] = n
		}
	} else {
		// فرمت type=N,type=N
		for _, part := range strings.Split(text, ",") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) != 2 {
				continue
			}
			bt := strings.ToLower(strings.TrimSpace(kv[0]))
			n, err := strconv.Atoi(strings.TrimSpace(kv[1]))
			if err != nil || !validTypes[bt] || n < 0 {
				continue
			}
			limits[bt] = n
		}
	}

	if len(limits) == 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyPlanLimitsInvalid),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	}

	total := 0
	var lines []string
	for bt, n := range limits {
		if err := h.Store.SetPlanLimit(ctx, planID, bt, n); err != nil {
			h.Log.Error("SetPlanLimit", h.F("err", err))
			continue
		}
		if n > 0 {
			lines = append(lines, fmt.Sprintf("  %s %s: %d", format.BotTypeEmoji(models.BotType(bt)), bt, n))
			total += n
		}
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyPlanLimitsSaved, strings.Join(lines, "\n"), total),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}
