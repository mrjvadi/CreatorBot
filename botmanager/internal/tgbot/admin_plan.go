package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminPlansList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	plans, _ := h.store.ListPlans(ctx)
	templates, _ := h.store.ListTemplates(ctx)

	lines := []string{h.t(ctx, uid, i18n.KeyPlansTitle), ""}
	if len(plans) == 0 {
		lines = append(lines, h.t(ctx, uid, i18n.KeyPlansEmpty))
	} else {
		for _, p := range plans {
			priceStr := fmt.Sprintf("%.2f", p.Price)
			if p.IsFree {
				priceStr = "رایگان"
			}
			lines = append(lines, fmt.Sprintf("💎 <b>%s</b> — %s TON | %d روز | %d ربات",
				p.Name, priceStr, p.DurationDay, p.MaxBots))
		}
	}
	lines = append(lines, "")

	if len(templates) == 0 {
		lines = append(lines, h.t(ctx, uid, i18n.KeyPlansNoTemplate))
		return c.Send(joinLines(lines), tele.ModeHTML, h.kbAdmin(ctx, uid))
	}

	lines = append(lines, "\n📦 <b>تمپلیت‌های موجود:</b>")
	for _, t := range templates {
		free := ""
		if t.IsFree {
			free = " 🆓"
		}
		lines = append(lines, fmt.Sprintf("  %s %s%s — ID: <code>%s</code>",
			botTypeEmoji(models.BotType(t.Type)), t.Name, free, t.ID))
	}
	lines = append(lines, "")
	lines = append(lines, h.t(ctx, uid, i18n.KeyPlanAskTemplate))

	h.setStep(ctx, uid, stepPlanTmpl)
	return c.Send(joinLines(lines), tele.ModeHTML, h.kbBackCancel(ctx, uid))
}

func (h *Handler) adminPlanStepName(ctx context.Context, c tele.Context, tmplID string) error {
	uid := c.Sender().ID
	tmpl, err := h.store.FindTemplate(ctx, tmplID)
	if err != nil || tmpl == nil {
		return c.Send(h.t(ctx, uid, i18n.KeyPlanTmplNotFound), h.kbBackCancel(ctx, uid))
	}
	h.setStep(ctx, uid, stepPlanName, "tmpl_id", tmplID)
	return c.Send(
		fmt.Sprintf("تمپلیت: <b>%s</b>\n\n%s", tmpl.Name, h.t(ctx, uid, i18n.KeyPlanAskName)),
		tele.ModeHTML, h.kbBackCancel(ctx, uid),
	)
}

func (h *Handler) adminPlanAdd(ctx context.Context, c tele.Context, tmplID, name string, days int, price float64) error {
	uid := c.Sender().ID

	tmpl, _ := h.store.FindTemplate(ctx, tmplID)
	if tmpl == nil {
		h.clearState(ctx, uid)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyNotFound))
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
	if err := h.store.CreatePlan(ctx, plan); err != nil {
		h.clearState(ctx, uid)
		h.log.Error("adminPlanAdd", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyPlanAddError))
	}

	// مرحله بعد: limit به تفکیک نوع ربات
	h.setStep(ctx, uid, stepPlanLimits, "plan_id", plan.ID.String())
	return c.Send(
		"حالا محدودیت هر نوع ربات را وارد کنید.\n\n"+
			"فرمت: <code>نوع=تعداد</code> جداشده با کاما\n"+
			"مثال: <code>uploader=2,vpn=1</code>\n\n"+
			"انواع: uploader, vpn, archive, member\n"+
			"یا فقط یک عدد بفرستید تا برای همه انواع اعمال شود.",
		tele.ModeHTML, h.kbBackCancel(ctx, uid),
	)
}

// adminPlanSetLimits ورودی limit ها را پردازش می‌کند.
func (h *Handler) adminPlanSetLimits(ctx context.Context, c tele.Context, planIDStr, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyError))
	}

	validTypes := map[string]bool{"uploader": true, "vpn": true, "archive": true, "member": true}
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
		return c.Send("فرمت نامعتبر. مثال: <code>uploader=2,vpn=1</code>",
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	}

	total := 0
	var lines []string
	for bt, n := range limits {
		if err := h.store.SetPlanLimit(ctx, planID, bt, n); err != nil {
			h.log.Error("SetPlanLimit", h.F("err", err))
			continue
		}
		if n > 0 {
			lines = append(lines, fmt.Sprintf("  %s %s: %d", botTypeEmoji(models.BotType(bt)), bt, n))
			total += n
		}
	}

	return c.Send(
		fmt.Sprintf("✅ <b>محدودیت‌ها ثبت شد</b>\n\n%s\n\nمجموع: %d ربات",
			strings.Join(lines, "\n"), total),
		tele.ModeHTML, h.kbAdmin(ctx, uid),
	)
}
