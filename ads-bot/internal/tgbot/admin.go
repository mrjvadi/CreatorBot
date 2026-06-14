package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

func (h *Handler) onAdmin(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	return c.Send("👑 پنل ادمین:", kbAdminMain())
}

func (h *Handler) handleAdminText(ctx context.Context, c tele.Context, text string) error {
	switch text {
	case "📊 آمار":
		return h.adminStats(ctx, c)
	case "⏳ در انتظار تأیید":
		return h.adminPending(ctx, c)
	case "📢 کانال‌ها":
		return h.adminChannels(ctx, c)
	case "📣 broadcast":
		h.setStep(ctx, c.Sender().ID, stepAdminBroadcast)
		return c.Send("پیام broadcast را ارسال کنید:", kbCancel())
	}
	return nil
}

// ── آمار ────────────────────────────────────────────────

func (h *Handler) adminStats(ctx context.Context, c tele.Context) error {
	stats, _ := h.store.GetStats(ctx)
	pubs, _ := h.store.ListPublishers(ctx)
	return c.Send(
		fmt.Sprintf(
			"<b>📊 آمار Ads Bot</b>\n\n"+
				"👤 ناشران: %d\n"+
				"📣 کل کمپین‌ها: %d\n"+
				"🟢 فعال: %d\n"+
				"📢 کانال‌های تأییدشده: %d\n"+
				"💸 کل خرج: %.4f TON\n"+
				"👥 کل عضو جذب: %d",
			len(pubs),
			stats.TotalCampaigns,
			stats.ActiveCampaigns,
			stats.TotalChannels,
			stats.TotalSpent,
			stats.TotalJoins,
		),
		tele.ModeHTML, kbAdminMain(),
	)
}

// ── کمپین‌های در انتظار ───────────────────────────────────

func (h *Handler) adminPending(ctx context.Context, c tele.Context) error {
	camps, _ := h.store.FindPendingCampaigns(ctx)
	if len(camps) == 0 {
		return c.Send("هیچ کمپین منتظری وجود ندارد.", kbAdminMain())
	}
	for _, camp := range camps {
		cp := camp
		text := fmt.Sprintf(
			"📣 <b>%s</b>\n"+
				"💰 %.2f TON | CPJ: %.3f\n"+
				"🆔 <code>%s</code>",
			cp.Name, cp.Budget, cp.CPJ, cp.ID,
		)
		if err := c.Send(text, tele.ModeHTML, kbReview(cp.ID.String())); err != nil {
			h.log.Error("adminPending send", ports.F("err", err))
		}
		// اگه media داره نمایش بده
		if cp.MediaFileID != "" {
			sendCampaignMedia(c, &cp)
		}
	}
	return nil
}

func sendCampaignMedia(c tele.Context, camp *store.Campaign) {
	file := tele.File{FileID: camp.MediaFileID}
	switch camp.MediaType {
	case "photo":
		c.Send(&tele.Photo{File: file, Caption: "پیش‌نمایش"})
	case "video":
		c.Send(&tele.Video{File: file, Caption: "پیش‌نمایش"})
	}
}

// ── تأیید / رد کمپین ─────────────────────────────────────

func (h *Handler) approveCampaign(ctx context.Context, c tele.Context, campIDStr string) error {
	campID, _ := uuid.Parse(campIDStr)
	if err := h.store.ApproveCampaign(ctx, campID, c.Sender().ID); err != nil {
		return c.Edit("❌ خطا.")
	}

	// فوری توزیع
	camp, _ := h.store.FindCampaign(ctx, campID)
	if camp != nil {
		go h.engine.DistributeCampaign(context.Background(), camp)

		// اطلاع به صاحب کمپین
		h.notifyPublisher(ctx, camp, "✅ <b>کمپین تأیید شد!</b>\n\n📌 "+camp.Name+"\n🟢 اکنون فعال است.")
	}

	return c.Edit(fmt.Sprintf("✅ کمپین <code>%s</code> تأیید شد.", campIDStr), tele.ModeHTML)
}

func (h *Handler) startReject(ctx context.Context, c tele.Context, campIDStr string) error {
	h.setStep(ctx, c.Sender().ID, stepRejectNote, "camp_id", campIDStr)
	return c.Edit("دلیل رد کمپین را وارد کنید:")
}

func (h *Handler) doReject(ctx context.Context, c tele.Context, st wizardState, note string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	campID, _ := uuid.Parse(st.Data["camp_id"])
	if err := h.store.RejectCampaign(ctx, campID, uid, note); err != nil {
		return c.Send("❌ خطا.")
	}

	camp, _ := h.store.FindCampaign(ctx, campID)
	if camp != nil {
		// بازگشت بودجه
		pub, _ := h.store.FindPublisher(ctx, c.Sender().ID)
		if pub != nil {
			h.store.UpdateBalance(ctx, pub.ID, camp.Budget)
		}
		h.notifyPublisher(ctx, camp,
			"❌ <b>کمپین رد شد</b>\n\n📌 "+camp.Name+"\n📝 دلیل: "+note+"\n\n💰 بودجه برگشت داده شد.")
	}

	return c.Send(fmt.Sprintf("🚫 کمپین <code>%s</code> رد شد.", st.Data["camp_id"]), tele.ModeHTML)
}

// ── کانال‌ها ──────────────────────────────────────────────

func (h *Handler) adminChannels(ctx context.Context, c tele.Context) error {
	channels, _ := h.store.ListUnverifiedChannels(ctx)
	if len(channels) == 0 {
		return c.Send("هیچ کانال منتظر تأییدی وجود ندارد.", kbAdminMain())
	}
	for _, ch := range channels {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(
			kb.Data("✅ تأیید", "verify_ch:"+ch.ID.String()),
			kb.Data("❌ رد", "reject_ch:"+ch.ID.String()),
		))
		c.Send(fmt.Sprintf(
			"📢 <b>%s</b>\n👥 %d اعضا\n💰 CPJ: %.3f TON\n🆔 <code>%s</code>",
			ch.ChannelName, ch.MemberCount, ch.EffectiveCPJ, ch.ID,
		), tele.ModeHTML, kb)
	}
	return nil
}

func (h *Handler) verifyChannel(ctx context.Context, c tele.Context, chIDStr string) error {
	chID, _ := uuid.Parse(chIDStr)
	if err := h.store.VerifyChannel(ctx, chID); err != nil {
		return c.Edit("❌ خطا.")
	}
	return c.Edit(fmt.Sprintf("✅ کانال <code>%s</code> تأیید شد.", chIDStr), tele.ModeHTML)
}

// ── Broadcast ────────────────────────────────────────────

func (h *Handler) doBroadcast(ctx context.Context, c tele.Context, text string) error {
	h.clearState(ctx, c.Sender().ID)
	pubs, _ := h.store.ListPublishers(ctx)
	sent, failed := 0, 0
	for _, p := range pubs {
		if p.IsBlocked {
			continue
		}
		chat := &tele.Chat{ID: p.TelegramID}
		if _, err := h.bot.Send(chat, text, tele.ModeHTML); err != nil {
			failed++
		} else {
			sent++
		}
	}
	return c.Send(
		fmt.Sprintf("📣 Broadcast ارسال شد.\n✅ %d\n❌ %d", sent, failed),
		kbAdminMain(),
	)
}

func (h *Handler) notifyPublisher(ctx context.Context, camp *store.Campaign, msg string) {
	// پیدا کردن publisher از campaign
	pub, _ := h.store.FindPublisherByID(ctx, camp.PublisherID)
	if pub == nil {
		return
	}
	chat := &tele.Chat{ID: pub.TelegramID}
	h.bot.Send(chat, msg, tele.ModeHTML)
}

// suppress
var _ = ports.F

// ── تنظیمات CPJ ─────────────────────────────────────────

func (h *Handler) adminConfig(ctx context.Context, c tele.Context) error {
	cfg, _ := h.store.GetConfig(ctx)
	if cfg == nil {
		return c.Send("❌ خطا در بارگذاری تنظیمات.")
	}

	cats, _ := h.store.ListCategories(ctx)
	var catLines strings.Builder
	for _, cat := range cats {
		catLines.WriteString(fmt.Sprintf(
			"• %s (%s) — ضریب: <b>×%.1f</b>\n",
			cat.Label, cat.Name, cat.CPJMultiplier,
		))
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("✏️ ویرایش CPJ پایه", "cfg_cpj")),
		kb.Row(kb.Data("✏️ ویرایش کمیسیون", "cfg_commission")),
		kb.Row(kb.Data("📂 ویرایش ضریب دسته‌ها", "cfg_categories")),
	)

	return c.Send(
		fmt.Sprintf(
			"<b>⚙️ تنظیمات سیستم تبلیغات</b>\n\n"+
				"💰 CPJ پایه: <b>%.4f TON</b>\n"+
				"🏢 کمیسیون پلتفرم: <b>%.0f%%</b>\n"+
				"📊 حداقل امتیاز کانال: <b>%d</b>\n"+
				"🤖 حداکثر fake مجاز: <b>%.0f%%</b>\n\n"+
				"<b>ضرایب دسته‌بندی:</b>\n%s",
			cfg.BaseCPJ,
			cfg.PlatformCommission,
			cfg.MinChannelScore,
			cfg.MaxFakePercent,
			catLines.String(),
		),
		tele.ModeHTML, kb,
	)
}

func (h *Handler) analyzeChannel(ctx context.Context, c tele.Context, chIDStr string) error {
	defer c.Respond()
	return c.Edit("🔄 آنالیز کانال در دست توسعه است.")
}

func (h *Handler) rejectChannel(ctx context.Context, c tele.Context, chIDStr string) error {
	defer c.Respond()
	chID, _ := uuid.Parse(chIDStr)
	h.store.DeactivateChannel(ctx, chID)
	return c.Edit(fmt.Sprintf("❌ کانال <code>%s</code> رد شد.", chIDStr), tele.ModeHTML)
}
