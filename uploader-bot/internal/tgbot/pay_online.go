package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/payment"
)

// merchantFor مرچنت تنظیم‌شده‌ی یک درگاه را برمی‌گرداند.
func (h *Handler) merchantFor(ctx context.Context, gateway string) string {
	switch gateway {
	case "zarinpal":
		return h.Store.GetSetting(ctx, models.SettingZarinpalMerchant)
	case "zibal":
		return h.Store.GetSetting(ctx, models.SettingZibalMerchant)
	}
	return ""
}

func (h *Handler) payCallbackURL(ctx context.Context) string {
	if h.Bot != nil && h.Bot.Me != nil && h.Bot.Me.Username != "" {
		return "https://t.me/" + h.Bot.Me.Username
	}
	return "https://example.com/callback"
}

// startOnlinePayment تراکنش آنلاین می‌سازد و لینک پرداخت را می‌فرستد.
func (h *Handler) startOnlinePayment(ctx context.Context, c tele.Context, plan *models.SubPlan, gateway string) error {
	merchant := h.merchantFor(ctx, gateway)
	if merchant == "" {
		return c.Edit("❌ این درگاه هنوز پیکربندی نشده است (مرچنت خالی است).")
	}
	gw := payment.New(gateway, merchant)
	if gw == nil {
		return c.Edit("❌ درگاه نامعتبر.")
	}

	amount := payment.TomanToRial(plan.Price)
	ref, url, err := gw.Request(amount, "خرید اشتراک "+plan.Name, h.payCallbackURL(ctx))
	if err != nil {
		return c.Edit("❌ خطا در اتصال به درگاه:\n" + err.Error())
	}

	uid := c.Sender().ID
	user, err := h.Store.GetUser(ctx, uid)
	h.LogErr("startOnlinePayment: get user", err)
	userID := ""
	if user != nil {
		userID = user.ID
	}
	pay := &models.Payment{
		UserID:     userID,
		TelegramID: uid,
		PlanID:     plan.ID,
		Gateway:    models.PaymentGateway(gateway),
		Amount:     plan.Price,
		Status:     models.PaymentPending,
		Authority:  ref,
	}
	if err := h.Store.CreatePayment(ctx, pay); err != nil {
		h.LogErr("startOnlinePayment: create payment", err)
		return c.Edit("❌ ثبت پرداخت با خطا مواجه شد. دوباره امتحان کنید.")
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.URL("💳 پرداخت", url)),
		kb.Row(kb.Data("✅ پرداخت کردم", "pay_verify:"+pay.ID)),
	)
	return c.Edit(fmt.Sprintf("برای پرداخت «%s» (%.0f تومان) روی «💳 پرداخت» بزنید،\nبعد از پرداخت «✅ پرداخت کردم» را بزنید.", plan.Name, plan.Price), kb)
}

// payVerify پرداخت آنلاین را بررسی و در صورت موفقیت اشتراک را فعال می‌کند.
func (h *Handler) payVerify(ctx context.Context, c tele.Context, payID string) error {
	pay, err := h.Store.FindPayment(ctx, payID)
	h.LogErr("payVerify: find payment", err)
	if pay == nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ تراکنش یافت نشد"})
	}
	if pay.Status == models.PaymentConfirmed {
		return c.Respond(&tele.CallbackResponse{Text: "قبلاً تایید شده"})
	}
	merchant := h.merchantFor(ctx, string(pay.Gateway))
	gw := payment.New(string(pay.Gateway), merchant)
	if gw == nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ درگاه نامعتبر"})
	}
	ok, track, err := gw.Verify(pay.Authority, payment.TomanToRial(pay.Amount))
	if err != nil {
		h.LogErr("payVerify: gateway verify", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در بررسی پرداخت"})
	}
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: "❌ پرداخت تایید نشد"})
	}

	// فعال‌سازی اشتراک — قبل از ConfirmPayment چک می‌شود تا اگر پلن دیگر
	// وجود نداشت، پیام موفقیت گمراه‌کننده («اشتراک فعال شد» با نام خالی)
	// نمایش داده نشود.
	days := 0
	name := ""
	for _, p := range mustPlans(ctx, h) {
		if p.ID == pay.PlanID {
			days = p.Days
			name = p.Name
			break
		}
	}
	if days == 0 {
		return c.Edit("⚠️ پرداخت شما تایید شد ولی پلن مربوطه دیگر موجود نیست. لطفاً با پشتیبانی تماس بگیرید.\n🔖 کد رهگیری: " + track)
	}
	if err := h.Store.SetUserSub(ctx, pay.TelegramID, pay.PlanID, days); err != nil {
		h.LogErr("payVerify: set sub", err)
		return c.Edit("⚠️ پرداخت تایید شد ولی فعال‌سازی اشتراک با خطا مواجه شد. با پشتیبانی تماس بگیرید.\n🔖 کد رهگیری: " + track)
	}
	h.LogErr("payVerify: confirm payment", h.Store.ConfirmPayment(ctx, pay.ID))
	h.LogErr("payVerify: respond", c.Respond(&tele.CallbackResponse{Text: "✅ پرداخت موفق"}))
	return c.Edit(fmt.Sprintf("✅ پرداخت تایید شد و اشتراک «%s» فعال شد.\n🔖 کد رهگیری: %s", name, track))
}

func mustPlans(ctx context.Context, h *Handler) []models.SubPlan {
	plans, err := h.Store.ListSubPlans(ctx)
	h.LogErr("mustPlans", err)
	return plans
}
