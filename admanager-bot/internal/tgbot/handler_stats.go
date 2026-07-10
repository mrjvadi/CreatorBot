// handler_stats.go — داشبورد آماری ادمین (فاز ۸).
package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
)

func (h *Handler) renderStats(ctx context.Context) (string, *tele.ReplyMarkup) {
	channels, _ := h.store.ListChannels(ctx, "", 1, statsScanLimit)
	activeCh := 0
	for _, ch := range channels {
		if ch.Status == models.ChannelActive {
			activeCh++
		}
	}

	campaigns, _ := h.store.ListCampaigns(ctx, "", 1, statsScanLimit)
	running, impressions := 0, 0
	for _, cm := range campaigns {
		if cm.Status == models.CampaignRunning {
			running++
		}
		impressions += cm.TotalImpressions
	}

	text := fmt.Sprintf(
		"📈 <b>آمار کلی</b>\n\n"+
			"📡 کانال‌ها: %d (فعال: %d)\n"+
			"📋 کمپین‌ها: %d (در حال اجرا: %d)\n"+
			"👁 مجموع نمایش‌ها: %d\n\n"+
			"برای زمان‌بندیِ دقیقِ ارسال‌های پیش‌رو از «🗓 زمان‌بندی» استفاده کنید.",
		len(channels), activeCh, len(campaigns), running, impressions,
	)

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(cbBtn(kb, "🔄 به‌روزرسانی", "stats")),
		kb.Row(cbBtn(kb, "🔙 منوی اصلی", "home")),
	)
	return text, kb
}

func (h *Handler) statsHome(c tele.Context) error {
	text, kb := h.renderStats(context.Background())
	return c.Send(text, tele.ModeHTML, kb)
}

func (h *Handler) adminStats(c tele.Context) error {
	text, kb := h.renderStats(context.Background())
	return c.Edit(text, tele.ModeHTML, kb)
}
