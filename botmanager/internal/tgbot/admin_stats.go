package tgbot

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminStats(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	instances, _ := h.store.ListAllInstances(ctx)
	users, _ := h.store.ListUsers(ctx)
	servers, _ := h.store.ListServers(ctx)
	templates, _ := h.store.ListTemplates(ctx)
	plans, _ := h.store.ListPlans(ctx)

	running, stopped, pending, errored := 0, 0, 0, 0
	for _, inst := range instances {
		switch inst.Status {
		case models.StatusRunning: running++
		case models.StatusStopped: stopped++
		case models.StatusPending: pending++
		case models.StatusError:   errored++
		}
	}
	onlineSrv, admins, blocked := 0, 0, 0
	for _, s := range servers { if s.IsOnline { onlineSrv++ } }
	for _, u := range users {
		if u.Role == models.RoleAdmin || u.Role == models.RoleOwner { admins++ }
		if u.IsBlocked { blocked++ }
	}

	text := fmt.Sprintf(
		"%s\n⏰ %s\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"<b>📦</b> %d  |  <b>💰</b> %d",
		h.t(ctx, uid, i18n.KeyStatsTitle),
		time.Now().Format("2006-01-02 | 15:04"),
		h.t(ctx, uid, i18n.KeyStatsBotsLine,
			len(instances), running, stopped, pending, errored),
		h.t(ctx, uid, i18n.KeyStatsServersLine,
			len(servers), onlineSrv, len(servers)-onlineSrv),
		h.t(ctx, uid, i18n.KeyStatsUsersLine,
			len(users), admins, blocked),
		len(templates), len(plans),
	)

	return c.Send(text, tele.ModeHTML, h.kbAdmin(ctx, uid))
}

// ── Admin Menu Section Handlers (spec) ───────────────────

func (h *Handler) adminCommunitiesList(ctx context.Context, c tele.Context) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🔍 جستجو", "admin_comm_search"),
			   kb.Data("📋 لیست",  "admin_comm_list")),
		kb.Row(kb.Data("📈 آمار",  "admin_comm_stats"),
			   kb.Data("🚨 مشکوک","admin_comm_suspicious")),
	)
	return c.Send("🏘 کامیونیتی‌ها",
		tele.ModeHTML, kb)
}

func (h *Handler) adminCampaignsList(ctx context.Context, c tele.Context) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🔍 جستجو",   "admin_camp_search"),
			   kb.Data("📋 فعال",    "admin_camp_active")),
		kb.Row(kb.Data("📋 تمام‌شده","admin_camp_done"),
			   kb.Data("📊 آمار",    "admin_camp_stats")),
	)
	return c.Send("📢 کمپین‌ها", tele.ModeHTML, kb)
}

func (h *Handler) adminFinance(ctx context.Context, c tele.Context) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("💰 درآمد",   "admin_fin_revenue"),
			   kb.Data("📤 برداشت‌ها","admin_fin_withdraw")),
		kb.Row(kb.Data("📊 گزارش‌ها","admin_fin_reports"),
			   kb.Data("💳 کیف‌پول‌ها","admin_fin_wallets")),
	)
	return c.Send("💰 مالی", tele.ModeHTML, kb)
}

func (h *Handler) adminFraud(ctx context.Context, c tele.Context) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🚩 رویدادها",    "admin_fraud_events"),
			   kb.Data("👤 امتیاز کاربران","admin_fraud_users")),
		kb.Row(kb.Data("🏘 امتیاز کامیونیتی","admin_fraud_comm"),
			   kb.Data("🔍 تحقیقات",     "admin_fraud_inv")),
	)
	return c.Send("🚨 تقلب", tele.ModeHTML, kb)
}

func (h *Handler) adminSystem(ctx context.Context, c tele.Context) error {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("📦 پلن‌ها",    "admin_sys_plans"),
			   kb.Data("🖥 سرورها",    "admin_sys_servers")),
		kb.Row(kb.Data("🔌 ربات‌های ممبر","admin_sys_member"),
			   kb.Data("📡 NATS",     "admin_sys_nats")),
		kb.Row(kb.Data("🗄 دیتابیس",  "admin_sys_db"),
			   kb.Data("📈 متریک‌ها", "admin_sys_metrics")),
	)
	return c.Send("⚙️ سیستم", tele.ModeHTML, kb)
}
