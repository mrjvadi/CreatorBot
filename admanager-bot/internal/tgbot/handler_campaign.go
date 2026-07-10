// handler_campaign.go — ساخت و مدیریت کمپین (فاز ۳).
package tgbot

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

// ── لیست کمپین‌ها ────────────────────────────────────────────────

func (h *Handler) renderCampaigns(ctx context.Context, status models.CampaignStatus) (string, *tele.ReplyMarkup) {
	list, err := h.store.ListCampaigns(ctx, status, 1, listPageSize)
	if err != nil {
		h.log.Error("list campaigns", portsF("err", err))
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, cm := range list {
		label := fmt.Sprintf("%s %s", campaignStatusIcon(cm.Status), cm.Name)
		rows = append(rows, kb.Row(cbBtn(kb, label, "cmp_view:"+cm.ID)))
	}
	// فیلتر وضعیت
	rows = append(rows, kb.Row(
		cbBtn(kb, "همه", "cmp_list"),
		cbBtn(kb, "در حال اجرا", "cmp_list:running"),
		cbBtn(kb, "پیش‌نویس", "cmp_list:draft"),
	))
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 منوی اصلی", "home")))
	kb.Inline(rows...)

	text := "📋 <b>کمپین‌ها</b>"
	if len(list) == 0 {
		text += "\n\nکمپینی در این دسته نیست. با «➕ کمپین جدید» یکی بسازید."
	}
	return text, kb
}

func (h *Handler) campaignsHome(c tele.Context) error {
	text, kb := h.renderCampaigns(context.Background(), "")
	return c.Send(text, tele.ModeHTML, kb)
}

func (h *Handler) campaignsList(c tele.Context, arg string) error {
	text, kb := h.renderCampaigns(context.Background(), models.CampaignStatus(arg))
	return c.Edit(text, tele.ModeHTML, kb)
}

// ── مشاهده‌ی کمپین ───────────────────────────────────────────────

// renderCampaignView متن و کیبورد نمای یک کمپین را می‌سازد.
func (h *Handler) renderCampaignView(ctx context.Context, id string) (string, *tele.ReplyMarkup, bool) {
	cm, err := h.store.FindCampaign(ctx, id)
	if err != nil || cm == nil {
		return "", nil, false
	}
	tags, _ := h.store.ListTags(ctx)
	ads, _ := h.store.ListAdsByCampaign(ctx, id)

	warn := ""
	if cm.IntervalMinutes < 1 {
		warn = "\n⚠️ فاصله بین پست‌ها صفر است؛ چیزی ارسال نمی‌شود. زمان‌بندی را ویرایش کنید."
	}

	text := fmt.Sprintf(
		"📋 <b>%s</b>\n\n"+
			"وضعیت: %s\n"+
			"بازه‌ی روزانه: %s\n"+
			"فاصله بین پست‌ها: %d دقیقه\n"+
			"عمر کل هر چرخه: %s\n"+
			"چرخش تبلیغ‌ها: %s\n"+
			"برچسب‌های هدف: %s\n"+
			"کانال‌های خاص: %d\n"+
			"تعداد تبلیغ‌ها: %d\n"+
			"نمایش‌ها: %d\n"+
			"🆔 <code>%s</code>%s",
		cm.Name, campaignStatusLabel(cm.Status),
		dailyWindowLabel(cm.StartHour, cm.StartMinute, cm.EndHour, cm.EndMinute),
		cm.IntervalMinutes, minutesLabel(cm.DeleteAfterMinutes), minutesLabel(cm.RotationMinutes),
		emptyDash(tagLabels(tags, cm.TargetTagIDs)),
		len(cm.TargetChannelIDs), len(ads), cm.TotalImpressions, cm.ID, warn,
	)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	switch cm.Status {
	case models.CampaignDraft:
		rows = append(rows, kb.Row(cbBtn(kb, "▶️ شروع", "cmp_start:"+id)))
	case models.CampaignRunning:
		rows = append(rows, kb.Row(cbBtn(kb, "⏸ توقف", "cmp_pause:"+id)))
	case models.CampaignPaused:
		rows = append(rows, kb.Row(cbBtn(kb, "▶️ ادامه", "cmp_resume:"+id)))
	}
	// ویرایش و هدف‌گذاری برای کمپین‌های فعال‌شدنی
	if cm.Status == models.CampaignDraft || cm.Status == models.CampaignRunning || cm.Status == models.CampaignPaused {
		rows = append(rows, kb.Row(
			cbBtn(kb, "✏️ نام", "cmp_ename:"+id),
			cbBtn(kb, "🕐 زمان‌بندی", "cmp_esched:"+id),
		))
		rows = append(rows, kb.Row(
			cbBtn(kb, "🎯 برچسب‌ها", "cmp_tags:"+id),
			cbBtn(kb, "📡 کانال‌ها", "cmp_chans:"+id),
		))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "📣 تبلیغ‌ها", "ad_list:"+id)))
	if cm.Status != models.CampaignRunning {
		rows = append(rows, kb.Row(cbBtn(kb, "🗑 حذف", "cmp_del:"+id)))
	} else {
		rows = append(rows, kb.Row(cbBtn(kb, "❌ لغو", "cmp_cancel:"+id)))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 بازگشت", "cmp_list")))
	kb.Inline(rows...)

	return text, kb, true
}

func (h *Handler) campaignView(c tele.Context, id string) error {
	text, kb, ok := h.renderCampaignView(context.Background(), id)
	if !ok {
		return c.Edit("کمپین پیدا نشد.")
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

// sendCampaignView نمای کمپین را به‌صورت پیام تازه می‌فرستد (بعد از مراحل متنی).
func (h *Handler) sendCampaignView(c tele.Context, id string) error {
	text, kb, ok := h.renderCampaignView(context.Background(), id)
	if !ok {
		return c.Send("کمپین پیدا نشد.", kbAdminMain())
	}
	return c.Send(text, tele.ModeHTML, kb)
}

// ── تغییر وضعیت ──────────────────────────────────────────────────

func (h *Handler) campaignSetStatus(c tele.Context, id, action string) error {
	ctx := context.Background()
	cm, err := h.store.FindCampaign(ctx, id)
	if err != nil || cm == nil {
		return c.Respond(&tele.CallbackResponse{Text: "کمپین پیدا نشد"})
	}

	switch action {
	case "start":
		ads, _ := h.store.ListAdsByCampaign(ctx, id)
		if len(ads) == 0 {
			return c.Respond(&tele.CallbackResponse{Text: "ابتدا حداقل یک تبلیغ اضافه کنید", ShowAlert: true})
		}
		if len(cm.TargetTagIDs) == 0 && len(cm.TargetChannelIDs) == 0 {
			return c.Respond(&tele.CallbackResponse{Text: "ابتدا حداقل یک برچسب هدف انتخاب کنید", ShowAlert: true})
		}
		_ = h.store.UpdateCampaignStatus(ctx, id, models.CampaignRunning)
		h.audit(ctx, c, models.AuditCampaignCreate, "campaign", id, "شروع کمپین")
	case "pause":
		_ = h.store.UpdateCampaignStatus(ctx, id, models.CampaignPaused)
		h.audit(ctx, c, models.AuditCampaignPause, "campaign", id, "توقف کمپین")
	case "resume":
		_ = h.store.UpdateCampaignStatus(ctx, id, models.CampaignRunning)
		h.audit(ctx, c, models.AuditCampaignResume, "campaign", id, "ادامه کمپین")
	case "cancel":
		_ = h.store.UpdateCampaignStatus(ctx, id, models.CampaignCancelled)
		_ = h.store.CancelCampaignJobs(ctx, id)
		_ = h.store.CancelCampaignReservations(ctx, id)
		h.audit(ctx, c, models.AuditCampaignEnd, "campaign", id, "لغو کمپین")
	}
	return h.campaignView(c, id)
}

func (h *Handler) campaignDelete(c tele.Context, id string) error {
	ctx := context.Background()
	_ = h.store.CancelCampaignJobs(ctx, id)
	_ = h.store.CancelCampaignReservations(ctx, id)
	if err := h.store.DeleteCampaign(ctx, id); err != nil {
		h.log.Error("delete campaign", portsF("err", err))
	}
	return h.campaignsList(c, "")
}

// ── هدف‌گذاری برچسب ──────────────────────────────────────────────

func (h *Handler) campaignTagsView(c tele.Context, id string) error {
	ctx := context.Background()
	cm, err := h.store.FindCampaign(ctx, id)
	if err != nil || cm == nil {
		return c.Edit("کمپین پیدا نشد.")
	}
	h.setCtx(ctx, c.Sender().ID, "cmp", id)
	tags, _ := h.store.ListTags(ctx)
	selected := map[string]bool{}
	for _, t := range cm.TargetTagIDs {
		selected[t] = true
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, t := range tags {
		mark := "▫️"
		if selected[t.ID] {
			mark = "✅"
		}
		rows = append(rows, kb.Row(cbBtn(kb, mark+" "+t.Name, "cmp_tagtoggle:"+t.ID)))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 بازگشت", "cmp_view:"+id)))
	kb.Inline(rows...)

	text := "🎯 <b>انتخاب برچسب‌های هدف</b>\nکانال‌های دارای این برچسب‌ها هدف کمپین می‌شوند."
	if len(tags) == 0 {
		text = "هیچ برچسبی نیست. ابتدا از بخش کانال‌ها برچسب بسازید."
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) campaignTagToggle(c tele.Context, tid string) error {
	ctx := context.Background()
	cid := h.getCtx(ctx, c.Sender().ID, "cmp")
	cm, err := h.store.FindCampaign(ctx, cid)
	if err != nil || cm == nil {
		return c.Respond(&tele.CallbackResponse{Text: "کمپین پیدا نشد"})
	}
	found := false
	var next []string
	for _, t := range cm.TargetTagIDs {
		if t == tid {
			found = true
			continue
		}
		next = append(next, t)
	}
	if !found {
		next = append(next, tid)
	}
	_ = h.store.UpdateCampaign(ctx, cid, bson.D{{Key: "target_tag_ids", Value: next}})
	return h.campaignTagsView(c, cid)
}

// ── هدف‌گذاری کانال خاص ──────────────────────────────────────────

func (h *Handler) campaignChannelsView(c tele.Context, id string) error {
	ctx := context.Background()
	cm, err := h.store.FindCampaign(ctx, id)
	if err != nil || cm == nil {
		return c.Edit("کمپین پیدا نشد.")
	}
	h.setCtx(ctx, c.Sender().ID, "cmp", id)
	channels, _ := h.store.ListChannels(ctx, models.ChannelActive, 1, listPageSize)
	selected := map[string]bool{}
	for _, ch := range cm.TargetChannelIDs {
		selected[ch] = true
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, ch := range channels {
		mark := "▫️"
		if selected[ch.ID] {
			mark = "✅"
		}
		rows = append(rows, kb.Row(cbBtn(kb, mark+" "+ch.Title, "cmp_chantoggle:"+ch.ID)))
	}
	rows = append(rows, kb.Row(cbBtn(kb, "🔙 بازگشت", "cmp_view:"+id)))
	kb.Inline(rows...)

	text := "📡 <b>انتخاب کانال‌های خاص</b>\nاین کانال‌ها مستقل از برچسب‌ها هدف کمپین می‌شوند."
	if len(channels) == 0 {
		text = "هیچ کانال فعالی نیست. ابتدا یک کانال اضافه و فعال کنید."
	}
	return c.Edit(text, tele.ModeHTML, kb)
}

func (h *Handler) campaignChannelToggle(c tele.Context, chID string) error {
	ctx := context.Background()
	cid := h.getCtx(ctx, c.Sender().ID, "cmp")
	cm, err := h.store.FindCampaign(ctx, cid)
	if err != nil || cm == nil {
		return c.Respond(&tele.CallbackResponse{Text: "کمپین پیدا نشد"})
	}
	found := false
	var next []string
	for _, ch := range cm.TargetChannelIDs {
		if ch == chID {
			found = true
			continue
		}
		next = append(next, ch)
	}
	if !found {
		next = append(next, chID)
	}
	_ = h.store.UpdateCampaign(ctx, cid, bson.D{{Key: "target_channel_ids", Value: next}})
	return h.campaignChannelsView(c, cid)
}

// ── ویرایش کمپین ─────────────────────────────────────────────────

func (h *Handler) campaignEditNameStart(c tele.Context, id string) error {
	ctx := context.Background()
	h.setState(ctx, c.Sender().ID, userState{Step: stepCampaignEditName, Data: map[string]string{"cid": id}})
	return c.Edit("نام جدید کمپین را بفرستید:")
}

func (h *Handler) handleCampaignEditName(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	name := trimText(text)
	if name == "" {
		return c.Send("نام نمی‌تواند خالی باشد:", kbCancelOnly())
	}
	id := st.Data["cid"]
	h.clearState(ctx, uid)
	if err := h.store.UpdateCampaign(ctx, id, bson.D{{Key: "name", Value: name}}); err != nil {
		h.log.Error("edit campaign name", portsF("err", err))
		return c.Send("❌ خطا در ویرایش.", kbAdminMain())
	}
	_ = c.Send("✅ نام به‌روزرسانی شد.")
	return h.sendCampaignView(c, id)
}

func (h *Handler) campaignEditSchedStart(c tele.Context, id string) error {
	ctx := context.Background()
	h.setState(ctx, c.Sender().ID, userState{Step: stepCampaignEditSchedule, Data: map[string]string{"cid": id}})
	return c.Edit(scheduleWizardPrompt, tele.ModeHTML)
}

func (h *Handler) handleCampaignEditSchedule(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	sh, sm, eh, em, interval, del, rot, ok := models.ParseSchedule(text)
	if !ok {
		return c.Send(scheduleWizardError, tele.ModeHTML, kbCancelOnly())
	}
	id := st.Data["cid"]
	h.clearState(ctx, uid)
	err := h.store.UpdateCampaign(ctx, id, bson.D{
		{Key: "start_hour", Value: sh},
		{Key: "start_minute", Value: sm},
		{Key: "end_hour", Value: eh},
		{Key: "end_minute", Value: em},
		{Key: "interval_minutes", Value: interval},
		{Key: "delete_after_minutes", Value: del},
		{Key: "rotation_minutes", Value: rot},
	})
	if err != nil {
		h.log.Error("edit campaign schedule", portsF("err", err))
		return c.Send("❌ خطا در ویرایش.", kbAdminMain())
	}
	_ = c.Send("✅ زمان‌بندی به‌روزرسانی شد.")
	return h.sendCampaignView(c, id)
}

// ── wizard ساخت (مراحل متنی) ─────────────────────────────────────

func (h *Handler) handleCampaignName(ctx context.Context, c tele.Context, _ userState, text string) error {
	name := trimText(text)
	if name == "" {
		return c.Send("نام کمپین نمی‌تواند خالی باشد. دوباره بفرستید:", kbCancelOnly())
	}
	h.setStepData(ctx, c.Sender().ID, stepCampaignSchedule, "name", name)
	return c.Send("✅ نام ثبت شد.\n\n"+scheduleWizardPrompt, tele.ModeHTML, kbCancelOnly())
}

func (h *Handler) handleCampaignSchedule(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID
	sh, sm, eh, em, interval, del, rot, ok := models.ParseSchedule(text)
	if !ok {
		return c.Send(scheduleWizardError, tele.ModeHTML, kbCancelOnly())
	}
	cm := &models.Campaign{
		Name:               st.Data["name"],
		StartHour:          sh,
		StartMinute:        sm,
		EndHour:            eh,
		EndMinute:          em,
		IntervalMinutes:    interval,
		DeleteAfterMinutes: del,
		RotationMinutes:    rot,
	}
	if err := h.store.CreateCampaign(ctx, cm); err != nil {
		h.log.Error("create campaign", portsF("err", err))
		h.clearState(ctx, uid)
		return c.Send("❌ ساخت کمپین با خطا مواجه شد.", kbAdminMain())
	}
	h.clearState(ctx, uid)
	h.audit(ctx, c, models.AuditCampaignCreate, "campaign", cm.ID, "ساخت کمپین "+cm.Name)

	return c.Send(
		fmt.Sprintf(
			"🎉 کمپین «%s» (پیش‌نویس) ساخته شد.\n\nمراحل بعدی:\n۱) برچسب یا کانال هدف را انتخاب کنید\n۲) حداقل یک تبلیغ (اصلی + ریپلی) اضافه کنید\n۳) دکمه‌ی «شروع» را بزنید\n\nاز «📋 کمپین‌ها» واردش شوید.",
			cm.Name,
		),
		kbAdminMain(),
	)
}

// ── helpers ──────────────────────────────────────────────────────

func campaignStatusIcon(s models.CampaignStatus) string {
	switch s {
	case models.CampaignRunning:
		return "🟢"
	case models.CampaignPaused:
		return "⏸"
	case models.CampaignDraft:
		return "📝"
	case models.CampaignCompleted:
		return "✅"
	case models.CampaignCancelled:
		return "❌"
	default:
		return "•"
	}
}

func campaignStatusLabel(s models.CampaignStatus) string {
	switch s {
	case models.CampaignRunning:
		return "🟢 در حال اجرا"
	case models.CampaignPaused:
		return "⏸ متوقف"
	case models.CampaignDraft:
		return "📝 پیش‌نویس"
	case models.CampaignCompleted:
		return "✅ پایان‌یافته"
	case models.CampaignCancelled:
		return "❌ لغوشده"
	default:
		return string(s)
	}
}

// minutesLabel نمایش خوانای دقیقه (۰ = غیرفعال).
func minutesLabel(n int) string {
	if n <= 0 {
		return "غیرفعال"
	}
	return fmt.Sprintf("%d دقیقه", n)
}

// dailyWindowLabel بازه‌ی روزانه را خوانا نمایش می‌دهد؛ اگر شروع و پایان
// برابر باشند یعنی کل شبانه‌روز پوشش داده می‌شود.
func dailyWindowLabel(sh, sm, eh, em int) string {
	if sh == eh && sm == em {
		return "🌐 کل شبانه‌روز"
	}
	label := fmt.Sprintf("%02d:%02d ← %02d:%02d", sh, sm, eh, em)
	startMin, endMin := sh*60+sm, eh*60+em
	if endMin <= startMin {
		label += " (عبور از نیمه‌شب)"
	}
	return label
}

// scheduleWizardPrompt متن راهنمای وارد کردن زمان‌بندی (۴ خط)، هم برای
// ساخت کمپین جدید و هم ویرایش، استفاده می‌شود.
const scheduleWizardPrompt = "⏰ <b>زمان‌بندی کمپین</b>\n" +
	"۴ خط جداگانه، هرکدام در یک پیام جدا، بفرستید:\n\n" +
	"<b>خط ۱</b> — بازه‌ی روزانه‌ی «شروع←پایان» (مثل <code>23:08-03:00</code>؛ اگر ساعت پایان کوچک‌تر از شروع باشد یعنی از نیمه‌شب عبور می‌کند)\n" +
	"<b>خط ۲</b> — فاصله بین پست‌ها به دقیقه\n" +
	"<b>خط ۳</b> — عمر کل هر چرخه (پست اصلی + ریپلی‌ها) به دقیقه، پیش از پاک‌شدن (۰ = هرگز پاک نشود)\n" +
	"<b>خط ۴</b> — فاصله‌ی چرخش بین تبلیغ‌ها به دقیقه (۰ = بدون چرخش)\n\n" +
	"نمونه:\n<code>23:08-03:00\n10\n60\n120</code>"

const scheduleWizardError = "❌ فرمت نادرست است. دقیقاً ۴ خط لازم است:\n" +
	"بازه‌ی روزانه (مثل <code>23:08-03:00</code>)\nفاصله‌ی پست‌ها (دقیقه)\nعمر کل چرخه (دقیقه)\nچرخش (دقیقه)"

// trimText فضای اضافی متن را حذف می‌کند (کمکی مشترک هندلرها).
func trimText(s string) string { return strings.TrimSpace(s) }
