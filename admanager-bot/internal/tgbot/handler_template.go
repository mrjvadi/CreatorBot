// handler_template.go — قالب‌های کمپین (فاز ۷).
package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (h *Handler) renderTemplates(ctx context.Context) (string, *tele.ReplyMarkup) {
	list, _ := h.store.ListTemplates(ctx)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, t := range list {
		rows = append(rows, kb.Row(cbBtn(kb, "🧩 "+t.Name, "tpl_view:"+t.ID)))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "➕ قالب جدید", "tpl_new")))
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 منوی اصلی", "home")))
	kb.Inline(rows...)

	text := "🧩 <b>قالب‌های کمپین</b>\nتنظیمات پیش‌فرضِ زمان‌بندی را یک‌بار در قالب ذخیره کنید تا هر بار کمپین جدید را سریع‌تر بسازید."
	if len(list) == 0 {
		text += "\n\nهنوز قالبی نساخته‌اید."
	}
	return text, kb
}

func (h *Handler) templatesHome(c tele.Context) error {
	text, kb := h.renderTemplates(context.Background())
	return c.Send(text, tele.ModeHTML, kb)
}

func (h *Handler) templatesList(c tele.Context) error {
	text, kb := h.renderTemplates(context.Background())
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) templateView(c tele.Context, id string) error {
	ctx := context.Background()
	t, err := h.store.FindTemplate(ctx, id)
	if err != nil || t == nil {
		return c.Edit("قالب پیدا نشد.")
	}
	text := fmt.Sprintf(
		"🧩 <b>%s</b>\n\nبازه‌ی روزانه: %s\nفاصله پست‌ها: %d دقیقه\nعمر کل چرخه: %s\nچرخش: %s\nحداقل اعضا: %d\n🆔 <code>%s</code>",
		t.Name, dailyWindowLabel(t.StartHour, t.StartMinute, t.EndHour, t.EndMinute), t.IntervalMinutes,
		minutesLabel(t.DeleteAfterMinutes), minutesLabel(t.RotationMinutes), t.MinMemberCount, t.ID,
	)
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(cbBtn(kb, "🚀 ساخت کمپین از این قالب", "tpl_use:"+id)),
		kb.Row(cbBtn(kb, "🗑 حذف", "tpl_del:"+id)),
		kb.Row(cbBtn(kb, "🔙 بازگشت", "tpl_list")),
	)
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) templateDelete(c tele.Context, id string) error {
	ctx := context.Background()
	if err := h.store.DeleteTemplate(ctx, id); err != nil {
		h.log.Error("delete template", portsF("err", err))
	}
	return h.templatesList(c)
}

func (h *Handler) templateUse(c tele.Context, id string) error {
	ctx := context.Background()
	t, err := h.store.FindTemplate(ctx, id)
	if err != nil || t == nil {
		return c.Respond(&tele.CallbackResponse{Text: "قالب پیدا نشد"})
	}
	cm := &models.Campaign{
		Name:               t.Name + " (کپی)",
		StartHour:          t.StartHour,
		StartMinute:        t.StartMinute,
		EndHour:            t.EndHour,
		EndMinute:          t.EndMinute,
		IntervalMinutes:    t.IntervalMinutes,
		DeleteAfterMinutes: t.DeleteAfterMinutes,
		RotationMinutes:    t.RotationMinutes,
		TargetTagIDs:       t.TargetTagIDs,
		MinMemberCount:     t.MinMemberCount,
	}
	if err := h.store.CreateCampaign(ctx, cm); err != nil {
		h.log.Error("create campaign from template", portsF("err", err))
		return c.Respond(&tele.CallbackResponse{Text: "خطا", ShowAlert: true})
	}
	h.audit(ctx, c, models.AuditCampaignCreate, "campaign", cm.ID, "ساخت کمپین از قالب")
	return h.campaignView(c, cm.ID)
}

// ── wizard ───────────────────────────────────────────────────────

func (h *Handler) templateNewStart(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepTemplateName)
	return c.Edit("یک نام برای قالبِ جدید بفرستید:")
}

func (h *Handler) handleTemplateName(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	name := strings.TrimSpace(text)
	if name == "" {
		return c.Send("نام خالی است.", kbAdminMain())
	}
	t := &models.CampaignTemplate{
		Name:            name,
		StartHour:       models.DefaultStartHour,
		StartMinute:     0,
		EndHour:         models.DefaultEndHour,
		EndMinute:       0,
		IntervalMinutes: models.DefaultIntervalMinutes,
	}
	if err := h.store.CreateTemplate(ctx, t); err != nil {
		h.log.Error("create template", portsF("err", err))
		return c.Send("❌ خطا در ساخت قالب.", kbAdminMain())
	}
	return c.Send(
		fmt.Sprintf("✅ قالب «%s» ساخته شد (بازه %02d:00 ← %02d:00، فاصله %d دقیقه). بعداً می‌توانید کمپین را از آن بسازید و تنظیمش کنید.",
			name, models.DefaultStartHour, models.DefaultEndHour, models.DefaultIntervalMinutes),
		kbAdminMain(),
	)
}
