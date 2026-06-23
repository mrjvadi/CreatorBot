package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
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
	case "👥 مالکان":
		return h.adminOwners(ctx, c)
	case "🔒 همه قفل‌ها":
		return h.adminLocks(ctx, c)
	case "📣 broadcast":
		h.setStep(ctx, c.Sender().ID, stepAdminBroadcast)
		return c.Send("پیام broadcast را ارسال کنید:", kbCancel())
	}
	return nil
}

func (h *Handler) adminStats(ctx context.Context, c tele.Context) error {
	owners, _ := h.store.ListOwners(ctx)
	locks, _ := h.store.ListAllLocks(ctx)
	bots, _ := h.store.FindActiveBots(ctx)

	active := 0
	for _, l := range locks {
		if l.Status == "active" {
			active++
		}
	}

	return c.Send(
		fmt.Sprintf(
			"<b>📊 آمار</b>\n\n"+
				"👤 مالکان: %d\n"+
				"🔒 قفل‌های فعال: %d / %d\n"+
				"🤖 Check Bot ها: %d",
			len(owners), active, len(locks), len(bots),
		),
		tele.ModeHTML, kbAdminMain(),
	)
}

func (h *Handler) adminOwners(ctx context.Context, c tele.Context) error {
	owners, _ := h.store.ListOwners(ctx)
	if len(owners) == 0 {
		return c.Send("هیچ مالکی وجود ندارد.", kbAdminMain())
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>👥 مالکان (%d)</b>\n\n", len(owners)))
	for _, o := range owners {
		blocked := ""
		if o.IsBlocked {
			blocked = " 🚫"
		}
		sb.WriteString(fmt.Sprintf("• <code>%d</code> @%s%s — %.0f تومان\n",
			o.TelegramID, o.Username, blocked, o.Balance))
	}
	return c.Send(sb.String(), tele.ModeHTML, kbAdminMain())
}

func (h *Handler) adminLocks(ctx context.Context, c tele.Context) error {
	locks, _ := h.store.ListAllLocks(ctx)
	if len(locks) == 0 {
		return c.Send("هیچ قفلی وجود ندارد.", kbAdminMain())
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🔒 قفل‌ها (%d)</b>\n\n", len(locks)))
	for _, l := range locks {
		sb.WriteString(fmtLock(l) + "\n\n")
	}
	return c.Send(sb.String(), tele.ModeHTML, kbAdminMain())
}

func (h *Handler) approvePayment(ctx context.Context, c tele.Context, payIDStr string) error {
	payID, _ := uuid.Parse(payIDStr)
	if err := h.store.ApprovePayment(ctx, payID); err != nil {
		return c.Edit("❌ خطا.")
	}
	return c.Edit(fmt.Sprintf("✅ پرداخت <code>%s</code> تأیید شد.", payIDStr), tele.ModeHTML)
}

func (h *Handler) doBroadcast(ctx context.Context, c tele.Context, text string) error {
	h.clearState(ctx, c.Sender().ID)
	owners, _ := h.store.ListOwners(ctx)
	sent, failed := 0, 0
	for _, o := range owners {
		if o.IsBlocked {
			continue
		}
		if _, err := h.bot.Send(&tele.User{ID: o.TelegramID}, text, tele.ModeHTML); err != nil {
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
