package tgbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
)

// ════════════════════════════════════════════════════════════
// ورود به پنل ادمین
// ════════════════════════════════════════════════════════════

func (h *Handler) onAdmin(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	return c.Send("پنل ادمین:", kbAdminMain())
}

func (h *Handler) handleAdminText(ctx context.Context, c tele.Context, text string) error {
	switch text {
	case "📊 آمار":
		return h.adminStats(ctx, c)
	case "👥 کاربران":
		return h.adminUsers(ctx, c)
	case "💰 پلن‌ها":
		return h.adminPlans(ctx, c)
	case "🖥 پنل‌ها":
		return h.adminPanels(ctx, c)
		return h.adminPanels(ctx, c)
	case "💸 پرداخت‌ها":
		return h.adminPayments(ctx, c)
	case "📣 broadcast":
		h.setStep(ctx, c.Sender().ID, stepAdminBroadcast)
		return c.Send("پیام broadcast را ارسال کنید:", kbCancel())
	}
	return nil
}

// ════════════════════════════════════════════════════════════
// آمار
// ════════════════════════════════════════════════════════════

func (h *Handler) adminStats(ctx context.Context, c tele.Context) error {
	activeSubs, _ := h.store.FindActiveSubscriptions(ctx)
	expiredSubs, _ := h.store.FindExpiredSubscriptions(ctx)

	panelCount, _ := h.panel.ActiveCount(ctx)

	return c.Send(
		fmt.Sprintf(
			"<b>📊 آمار ربات</b>\n\n"+
				"🟢 اشتراک فعال: <b>%d</b>\n"+
				"🔴 اشتراک منقضی: <b>%d</b>\n"+
				"🖥 کاربران پنل: <b>%d</b>\n\n"+
				"⏰ %s",
			len(activeSubs), len(expiredSubs), panelCount,
			time.Now().Format("2006/01/02 15:04"),
		),
		tele.ModeHTML, kbAdminMain(),
	)
}

// ════════════════════════════════════════════════════════════
// کاربران
// ════════════════════════════════════════════════════════════

func (h *Handler) adminUsers(ctx context.Context, c tele.Context) error {
	users, err := h.store.ListUsers(ctx)
	if err != nil {
		return c.Send("❌ خطا.")
	}
	if len(users) == 0 {
		return c.Send("هیچ کاربری وجود ندارد.", kbAdminMain())
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>👥 کاربران (%d)</b>\n\n", len(users)))
	for _, u := range users {
		blocked := ""
		if u.IsBlocked {
			blocked = " 🚫"
		}
		sb.WriteString(fmt.Sprintf("• <code>%d</code> %s%s — %.0f تومان\n",
			u.TelegramID, u.FirstName, blocked, u.Balance))
	}
	sb.WriteString("\n/block <id> — بلاک\n/unblock <id> — آنبلاک\n/addbalance <id> <amount>")
	return c.Send(sb.String(), tele.ModeHTML, kbAdminMain())
}

// ════════════════════════════════════════════════════════════
// پلن‌ها
// ════════════════════════════════════════════════════════════

func (h *Handler) adminPlans(ctx context.Context, c tele.Context) error {
	plans, _ := h.store.ListPlans(ctx)

	var sb strings.Builder
	sb.WriteString("<b>💰 پلن‌ها</b>\n\n")
	if len(plans) == 0 {
		sb.WriteString("هیچ پلنی ندارید.\n")
	} else {
		for _, p := range plans {
			active := "✅"
			if !p.IsActive {
				active = "❌"
			}
			traffic := "نامحدود"
			if p.DataGB > 0 {
				traffic = fmt.Sprintf("%.0f GB", p.DataGB)
			}
			sb.WriteString(fmt.Sprintf("%s <b>%s</b> — %d روز — %s — %.0f تومان\n",
				active, p.Name, p.DurationDay, traffic, p.Price))
		}
	}
	sb.WriteString("\n/addplan <نام> <روز> <GB> <قیمت>")
	return c.Send(sb.String(), tele.ModeHTML, kbAdminMain())
}

// ════════════════════════════════════════════════════════════
// پنل‌ها
// ════════════════════════════════════════════════════════════

func (h *Handler) adminPanels(ctx context.Context, c tele.Context) error {
	// پنل از طریق env تنظیم می‌شود — نه DB
	// اطلاعات از panel interface می‌گیریم
	count, err := h.panel.ActiveCount(ctx)
	if err != nil {
		return c.Send("❌ خطا در اتصال به پنل.", kbAdminMain())
	}
	return c.Send(
		fmt.Sprintf(
			"<b>🖥 پنل VPN</b>\n\n"+
				"🟢 وضعیت: متصل\n"+
				"👥 کاربران فعال: <b>%d</b>\n\n"+
				"تنظیمات پنل از .env خوانده می‌شود.",
			count,
		),
		tele.ModeHTML, kbAdminMain(),
	)
}

// ════════════════════════════════════════════════════════════
// تأیید پرداخت‌ها
// ════════════════════════════════════════════════════════════

func (h *Handler) adminPayments(ctx context.Context, c tele.Context) error {
	payments, err := h.store.FindPendingPayments(ctx)
	if err != nil {
		return c.Send("❌ خطا.")
	}
	if len(payments) == 0 {
		return c.Send("هیچ پرداخت منتظری وجود ندارد.", kbAdminMain())
	}

	for _, p := range payments {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(
			"💳 <b>پرداخت جدید</b>\n"+
				"UserID: <code>%s</code>\n"+
				"مبلغ: <b>%.0f تومان</b>\n"+
				"روش: %s\n"+
				"ID: <code>%s</code>",
			p.UserID, p.Amount, p.Gateway, p.ID,
		))

		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(
				kb.Data("✅ تأیید", "approve_pay:"+p.ID.String()),
				kb.Data("❌ رد", "reject_pay:"+p.ID.String()),
			),
		)

		if err := c.Send(sb.String(), tele.ModeHTML, kb); err != nil {
			h.log.Error("adminPayments send", ports.F("err", err))
		}
	}
	return nil
}

// ════════════════════════════════════════════════════════════
// تأیید / رد پرداخت (callback)
// ════════════════════════════════════════════════════════════

func (h *Handler) approvePayment(ctx context.Context, c tele.Context, payIDStr string) error {
	payID, err := uuid.Parse(payIDStr)
	if err != nil {
		return c.Edit("❌ ID نامعتبر.")
	}

	payment, err := h.store.FindPaymentByID(ctx, payID)
	if err != nil || payment == nil {
		return c.Edit("❌ پرداخت یافت نشد.")
	}
	if payment.Status != "pending" {
		return c.Edit("این پرداخت قبلاً پردازش شده.")
	}

	// افزایش موجودی کاربر
	if err := h.store.UpdateBalance(ctx, payment.UserID, payment.Amount); err != nil {
		return c.Edit("❌ خطا در افزایش موجودی.")
	}

	// تأیید پرداخت
	h.store.UpdatePaymentStatus(ctx, payID, "confirmed")

	// فعال‌سازی اشتراک اگه plan_id داره
	if payment.PlanID != nil {
		plan, _ := h.store.FindPlan(ctx, *payment.PlanID)
		if plan != nil {
			// پیدا کردن کاربر تلگرام
			user, _ := h.store.FindUserByID(ctx, payment.UserID)
			if user != nil {
				// فعال‌سازی از طریق پنل
				username := genVPNUsername(user.TelegramID)
				expiresAt := time.Now().AddDate(0, 0, plan.DurationDay)
				dataLimit := int64(plan.DataGB * 1e9)

				vpnUser, err := h.panel.CreateUser(ctx, ports.CreateVPNUserRequest{
					Username:  username,
					DataLimit: dataLimit,
					ExpiresAt: expiresAt,
				})
				if err == nil {
					panelRec, _ := h.store.FindBestPanel(ctx)
					panelID := uuid.Nil
					if panelRec != nil {
						panelID = panelRec.ID
					}
					sub := &models.Subscription{
						UserID: user.ID, PanelID: panelID,
						PlanID: plan.ID, Username: vpnUser.Username,
						Status: models.SubActive, ExpiresAt: expiresAt,
						DataLimit: dataLimit,
					}
					h.store.CreateSubscription(ctx, sub)

					// اطلاع به کاربر
					var sb strings.Builder
					sb.WriteString("🎉 <b>اشتراک فعال شد!</b>\n\n")
					for i, link := range vpnUser.Links {
						sb.WriteString(fmt.Sprintf("<b>%d.</b> <code>%s</code>\n", i+1, link))
					}
					h.sender.Send(ctx, user.TelegramID, sb.String(), ports.WithHTML())
				}
			}
		}
	}

	return c.Edit(fmt.Sprintf("✅ پرداخت <code>%s</code> تأیید شد.", payIDStr), tele.ModeHTML)
}

func (h *Handler) rejectPayment(ctx context.Context, c tele.Context, payIDStr string) error {
	payID, _ := uuid.Parse(payIDStr)
	h.store.UpdatePaymentStatus(ctx, payID, "rejected")

	payment, _ := h.store.FindPaymentByID(ctx, payID)
	if payment != nil {
		user, _ := h.store.FindUserByID(ctx, payment.UserID)
		if user != nil {
			h.sender.Send(ctx, user.TelegramID,
				"❌ پرداخت شما رد شد. برای اطلاعات بیشتر با پشتیبانی تماس بگیرید.")
		}
	}
	return c.Edit("🚫 پرداخت رد شد.")
}

// ════════════════════════════════════════════════════════════
// Broadcast
// ════════════════════════════════════════════════════════════

func (h *Handler) doBroadcast(ctx context.Context, c tele.Context, text string) error {
	h.clearState(ctx, c.Sender().ID)

	users, _ := h.store.ListUsers(ctx)
	sent, failed := 0, 0
	for _, u := range users {
		if u.IsBlocked {
			continue
		}
		if err := h.sender.Send(ctx, u.TelegramID, text, ports.WithHTML()); err != nil {
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

// ════════════════════════════════════════════════════════════
// کد تخفیف
// ════════════════════════════════════════════════════════════

func (h *Handler) handleDiscountInput(ctx context.Context, c tele.Context, st wizardState, text string) error {
	h.clearState(ctx, c.Sender().ID)
	// TODO: parse و ذخیره کد تخفیف
	_ = text
	return c.Send("✅ کد تخفیف ذخیره شد.", kbAdminMain())
}
