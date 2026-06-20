package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

// ── helper ─────────────────────────────────────────────────

func (h *Handler) b(ctx context.Context, uid int64, k i18n.Key) tele.Btn {
	kb := &tele.ReplyMarkup{}
	return kb.Text(h.btn(ctx, uid, k))
}

// ── منوی اصلی کاربر (Reply Keyboard) ──────────────────────
// ساده: فقط آنچه برای خرید و مدیریت ربات لازم است

func (h *Handler) kbUser(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(h.b(ctx, uid, i18n.KeyMenuCreateBot), h.b(ctx, uid, i18n.KeyMenuMyBots)),
		kb.Row(h.b(ctx, uid, i18n.KeyMenuAccount), h.b(ctx, uid, i18n.KeyMenuPlans)),
		kb.Row(h.b(ctx, uid, i18n.KeyMenuTutorials), h.b(ctx, uid, i18n.KeyMenuSupport)),
		kb.Row(h.b(ctx, uid, i18n.KeyMenuLanguage)),
	)
	return kb
}

func (h *Handler) kbUserFull(ctx context.Context, uid int64, sub *models.Subscription) *tele.ReplyMarkup {
	return h.kbUser(ctx, uid)
}

// ── منوی اصلی ادمین (Reply Keyboard) ───────────────────────

func (h *Handler) kbAdmin(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(h.b(ctx, uid, i18n.KeyMenuUsers), h.b(ctx, uid, i18n.KeyMenuBots)),
		kb.Row(h.b(ctx, uid, i18n.KeyMenuServers), h.b(ctx, uid, i18n.KeyMenuTemplates)),
		kb.Row(h.b(ctx, uid, i18n.KeyMenuPlans), h.b(ctx, uid, i18n.KeyMenuStats)),
		kb.Row(h.b(ctx, uid, i18n.KeyMenuBroadcast), h.b(ctx, uid, i18n.KeyMenuSystem)),
		kb.Row(h.b(ctx, uid, i18n.KeyMenuExitAdmin)),
	)
	return kb
}

// ── پلن‌ها (Inline) ──────────────────────────────────────────

func kbPlanList(plans []models.Plan) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("💎 %s — %.1f TON", p.Name, p.Price)
		if p.IsFree {
			label = "🆓 " + p.Name + " (رایگان)"
		}
		rows = append(rows, kb.Row(kb.Data(label, "select_plan:"+p.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "cancel")))
	kb.Inline(rows...)
	return kb
}

func kbPlanDetail(planID string, isFree bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	if isFree {
		kb.Inline(
			kb.Row(kb.Data("✅ فعال‌سازی رایگان", "start_free:"+planID)),
			kb.Row(kb.Data("🔙 بازگشت", "show_plans")),
		)
	} else {
		kb.Inline(
			kb.Row(kb.Data("💳 خرید با کیف پول TON", "buy_plan:"+planID)),
			kb.Row(kb.Data("🔙 بازگشت", "show_plans")),
		)
	}
	return kb
}

// ── سرویس‌های کاربر (Inline) ─────────────────────────────────

func kbServiceRunning(instanceID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("📊 آمار",      "svc_stats:"+instanceID),
			kb.Data("⚙️ تنظیمات", "svc_settings:"+instanceID),
		),
		kb.Row(
			kb.Data("🔄 ری‌استارت", "bot_restart:"+instanceID),
			kb.Data("⏸ توقف",      "bot_stop:"+instanceID),
		),
		kb.Row(kb.Data("🗑 حذف سرویس", "bot_delete:"+instanceID)),
	)
	return kb
}

func kbServiceStopped(instanceID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("▶️ شروع", "bot_start:"+instanceID),
			kb.Data("🗑 حذف",  "bot_delete:"+instanceID),
		),
	)
	return kb
}

func kbServicePending(instanceID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🔄 بررسی وضعیت", "svc_status:"+instanceID)),
	)
	return kb
}

func kbServiceFailed(instanceID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🔄 تلاش مجدد", "bot_restart:"+instanceID),
			kb.Data("🗑 حذف",       "bot_delete:"+instanceID),
		),
	)
	return kb
}

// ── ایجاد سرویس (Inline) ─────────────────────────────────────

func kbServiceCreate() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🌐 VPN",        "svc_type:vpn"),
			kb.Data("📤 آپلودر",     "svc_type:uploader"),
		),
		kb.Row(
			kb.Data("🔒 ممبرشیپ",   "svc_type:member"),
			kb.Data("📦 آرشیو",     "svc_type:archive"),
		),
		kb.Row(kb.Data("❌ لغو", "cancel")),
	)
	return kb
}

// ── ادمین — کاربران ───────────────────────────────────────────

func (h *Handler) kbUserActions(ctx context.Context, uid int64, targetID int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🚫 مسدود",    fmt.Sprintf("block_user:%d", targetID)),
			kb.Data("🛡 ادمین",    fmt.Sprintf("make_admin:%d", targetID)),
		),
		kb.Row(kb.Data("🔙 بازگشت", "admin_users")),
	)
	return kb
}

// ── مشترک ─────────────────────────────────────────────────────

func (h *Handler) kbBotType(ctx context.Context, uid int64) *tele.ReplyMarkup {
	return kbServiceCreate()
}

func (h *Handler) kbLinkLimit(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("1️⃣ یک بار",  "limit:1"),
			kb.Data("3️⃣ سه بار",  "limit:3"),
		),
		kb.Row(
			kb.Data("5️⃣ پنج بار", "limit:5"),
			kb.Data("🔟 ده بار",  "limit:10"),
		),
		kb.Row(kb.Data("♾️ نامحدود", "limit:0")),
	)
	return kb
}

func (h *Handler) kbBack(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("🔙 بازگشت", "cancel")))
	return kb
}

func (h *Handler) kbBackCancel(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🔙 بازگشت", "cancel"),
			kb.Data("❌ لغو",    "cancel"),
		),
	)
	return kb
}

func (h *Handler) kbCancel(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("❌ لغو", "cancel")))
	return kb
}

func (h *Handler) kbWizardConfirm(ctx context.Context, uid int64) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("✅ بله، ربات دارم", "wizard_confirm"),
			kb.Data("❌ لغو",           "cancel"),
		),
	)
	return kb
}

func kbLanguage() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🇮🇷 فارسی", "lang:fa"),
			kb.Data("🇬🇧 English", "lang:en"),
		),
	)
	return kb
}
