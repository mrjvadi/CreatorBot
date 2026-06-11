package tgbot

import (
	"context"
	"fmt"

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
			free := ""
			if p.IsFree {
				free = h.t(ctx, uid, i18n.KeyAdminPlanFree)
			}
			lines = append(lines, h.t(ctx, uid, i18n.KeyAdminPlanLine,
				p.Name, free, p.DurationDay, p.Price, p.MaxBots, p.ID))
		}
	}
	lines = append(lines, "")

	if len(templates) == 0 {
		lines = append(lines, h.t(ctx, uid, i18n.KeyPlansNoTemplate))
		return c.Send(joinLines(lines), tele.ModeHTML, h.kbAdmin(ctx, uid))
	}

	lines = append(lines, h.t(ctx, uid, i18n.KeyAdminTemplates))
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
	h.clearState(ctx, uid)

	tmpl, _ := h.store.FindTemplate(ctx, tmplID)
	if tmpl == nil {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyNotFound))
	}

	isFree := price == 0
	maxBots := 1
	if price > 0 {
		maxBots = 3
	}

	plan := &models.Plan{
		TemplateID:  tmpl.ID,
		Name:        name,
		DurationDay: days,
		Price:       price,
		IsFree:      isFree,
		MaxBots:     maxBots,
		IsActive:    true,
	}
	if err := h.store.CreatePlan(ctx, plan); err != nil {
		h.log.Error("adminPlanAdd", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyPlanAddError))
	}

	free := ""
	if isFree {
		free = h.t(ctx, uid, i18n.KeyAdminPlanFree)
	}

	return c.Send(
		h.t(ctx, uid, i18n.KeyAdminPlanAdded,
			plan.Name, free, tmpl.Name, plan.DurationDay, plan.Price, plan.MaxBots, plan.ID),
		tele.ModeHTML, h.kbAdmin(ctx, uid),
	)
}
