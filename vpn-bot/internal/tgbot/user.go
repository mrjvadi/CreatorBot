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
// خرید اشتراک
// ════════════════════════════════════════════════════════════

func (h *Handler) onBuy(c tele.Context) error {
	ctx := context.Background()

	plans, err := h.store.ListPlans(ctx)
	if err != nil || len(plans) == 0 {
		return c.Send("در حال حاضر پلنی موجود نیست.")
	}

	var sb strings.Builder
	sb.WriteString("<b>🛒 انتخاب پلن</b>\n\n")
	for _, p := range plans {
		sb.WriteString(fmtPlan(p))
		sb.WriteString("\n\n")
	}
	sb.WriteString("پلن مورد نظر را انتخاب کنید:")

	return c.Send(sb.String(), tele.ModeHTML, kbPlans(plans))
}

// onPlanSelected کاربر روی یه پلن کلیک کرده.
func (h *Handler) onPlanSelected(ctx context.Context, c tele.Context, planIDStr string) error {
	uid := c.Sender().ID

	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return c.Edit("❌ پلن نامعتبر.")
	}

	plan, err := h.store.FindPlan(ctx, planID)
	if err != nil || plan == nil {
		return c.Edit("❌ پلن یافت نشد.")
	}

	u, _ := h.getOrCreate(ctx, c)

	h.setStep(ctx, uid, stepBuyPlan,
		"plan_id", planIDStr,
		"plan_price", fmt.Sprintf("%.0f", plan.Price),
	)

	// اگه موجودی کافیه → مستقیم بخر
	if u.Balance >= plan.Price {
		return c.Edit(
			fmt.Sprintf(
				"%s\n\n💳 موجودی شما: <b>%.0f تومان</b>\n\n✅ موجودی کافی است. خرید شود؟",
				fmtPlan(*plan), u.Balance,
			),
			tele.ModeHTML,
			confirmBuyKB(planIDStr),
		)
	}

	// موجودی کافی نیست → انتخاب روش پرداخت
	needed := plan.Price - u.Balance
	return c.Edit(
		fmt.Sprintf(
			"%s\n\n💳 موجودی: <b>%.0f</b> | کمبود: <b>%.0f تومان</b>\n\nروش پرداخت:",
			fmtPlan(*plan), u.Balance, needed,
		),
		tele.ModeHTML, kbPaymentGateway(),
	)
}

func confirmBuyKB(planID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("✅ بله، خرید شود", "confirm_buy:"+planID)),
		kb.Row(kb.Data("❌ انصراف", "cancel")),
	)
	return kb
}

// onGatewaySelected روش پرداخت انتخاب شد.
func (h *Handler) onGatewaySelected(ctx context.Context, c tele.Context, gw string) error {
	uid := c.Sender().ID
	st := h.getState(ctx, uid)
	h.setStep(ctx, uid, stepBuyPayment, "gw", gw,
		"plan_id", st.Data["plan_id"],
		"plan_price", st.Data["plan_price"],
	)

	switch gw {
	case "card":
		// دریافت شماره کارت
		// card info از DB
		h.setStep(ctx, uid, stepBuyReceipt, "plan_id", st.Data["plan_id"])
		return c.Edit(
			fmt.Sprintf(
				"💳 <b>پرداخت کارت به کارت</b>\n\n"+
					"مبلغ: <b>%.0f تومان</b>\n\n"+
					"عکس رسید پرداخت را ارسال کنید.",
				parseFloat(st.Data["plan_price"]),
			),
			tele.ModeHTML,
		)
	case "zarinpal", "nowpayments":
		amount := parseFloat(st.Data["plan_price"])
		resp, err := h.gateway.CreatePayment(ctx, ports.PaymentRequest{
			Amount: amount, Description: "خرید VPN", UserID: uid,
		})
		if err != nil {
			return c.Edit("❌ خطا در ایجاد لینک پرداخت.")
		}
		payURL, refID := resp.PaymentURL, resp.RefID
		h.setStep(ctx, uid, stepBuyPayment, "gw", gw,
			"plan_id", st.Data["plan_id"],
			"ref_id", refID,
		)
		kb := &tele.ReplyMarkup{}
		kb.Inline(
			kb.Row(kb.URL("💳 پرداخت آنلاین", payURL)),
			kb.Row(kb.Data("✅ پرداخت کردم", "verify_payment:"+refID)),
		)
		return c.Edit("برای پرداخت روی دکمه زیر کلیک کنید:", kb)
	}
	return nil
}

// handlePaymentInput متن وارد شده در مرحله پرداخت.
func (h *Handler) handlePaymentInput(ctx context.Context, c tele.Context, st wizardState, text string) error {
	// برای zarinpal: کد رهگیری وارد می‌شود
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	planID, _ := uuid.Parse(st.Data["plan_id"])
	plan, err := h.store.FindPlan(ctx, planID)
	if err != nil || plan == nil {
		return h.sendMain(c, "❌ پلن یافت نشد.")
	}

	// ثبت payment در حالت manual
	payment := &models.Payment{
		UserID:  getUserID(ctx, h, c),
		Amount:  plan.Price,
		Gateway: st.Data["gw"],
		RefCode: text,
		Status:  "pending",
		PlanID:  &planID,
	}
	if err := h.store.CreatePayment(ctx, payment); err != nil {
		return h.sendMain(c, "❌ خطا در ثبت پرداخت.")
	}

	// اطلاع به ادمین
	h.notifyAdmin(ctx, fmt.Sprintf(
		"💳 <b>پرداخت جدید</b>\n\nکاربر: <code>%d</code>\nپلن: %s\nمبلغ: %.0f\nکد: %s\nID: <code>%s</code>",
		c.Sender().ID, plan.Name, plan.Price, text, payment.ID,
	))

	return c.Send(
		"✅ کد پرداخت ثبت شد.\n\nبعد از تأیید توسط ادمین، اشتراک شما فعال می‌شود.",
		kbMain(),
	)
}

// handleReceiptPhoto عکس رسید کارت به کارت.
func (h *Handler) handleReceiptPhoto(ctx context.Context, c tele.Context, st wizardState, fileID string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	planID, _ := uuid.Parse(st.Data["plan_id"])
	plan, _ := h.store.FindPlan(ctx, planID)
	planName := ""
	planPrice := 0.0
	if plan != nil {
		planName = plan.Name
		planPrice = plan.Price
	}

	payment := &models.Payment{
		UserID:  getUserID(ctx, h, c),
		Amount:  planPrice,
		Gateway: "card",
		Receipt: fileID,
		Status:  "pending",
		PlanID:  &planID,
	}
	h.store.CreatePayment(ctx, payment)

	// ارسال عکس به ادمین
	h.notifyAdmin(ctx, fmt.Sprintf(
		"📸 <b>رسید کارت به کارت</b>\n\nکاربر: <code>%d</code>\nپلن: %s\nمبلغ: %.0f\nID: <code>%s</code>",
		uid, planName, planPrice, payment.ID,
	))
	h.sendPhotoToAdmin(ctx, fileID, "رسید پرداخت")

	return c.Send("✅ رسید دریافت شد.\n\nبعد از تأیید ادمین، اشتراک فعال می‌شود.", kbMain())
}

// ════════════════════════════════════════════════════════════
// اشتراک من
// ════════════════════════════════════════════════════════════

func (h *Handler) onMyVPN(c tele.Context) error {
	ctx := context.Background()
	u, _ := h.getOrCreate(ctx, c)

	subs, err := h.store.FindSubscriptionsByUserID(ctx, u.ID)
	if err != nil {
		return c.Send("❌ خطا.")
	}
	if len(subs) == 0 {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("🛒 خرید اشتراک", "buy")))
		return c.Send("هیچ اشتراکی ندارید.", kb)
	}

	for _, sub := range subs {
		subText := fmtSub(sub)
		err := c.Send(subText, tele.ModeHTML, kbSubscription(sub.ID.String()))
		if err != nil {
			h.log.Error("onMyVPN send", ports.F("err", err))
		}
	}
	return nil
}

// sendSubscriptionLink لینک اشتراک را می‌فرستد.
func (h *Handler) sendSubscriptionLink(ctx context.Context, c tele.Context, subIDStr string) error {
	subID, _ := uuid.Parse(subIDStr)
	sub, err := h.store.FindSubscriptionByID(ctx, subID)
	if err != nil || sub == nil {
		return c.Edit("❌ اشتراک یافت نشد.")
	}

	vpnUser, err := h.panel.GetUser(ctx, sub.Username)
	if err != nil || vpnUser == nil {
		return c.Edit("❌ خطا در دریافت اشتراک.")
	}

	if len(vpnUser.Links) == 0 {
		return c.Edit("❌ لینکی موجود نیست.")
	}

	var sb strings.Builder
	sb.WriteString("🔗 <b>لینک‌های اشتراک</b>\n\n")
	for i, link := range vpnUser.Links {
		sb.WriteString(fmt.Sprintf("<b>%d.</b> <code>%s</code>\n\n", i+1, link))
	}
	sb.WriteString("لینک را کپی کرده و در اپ VPN وارد کنید.")

	return c.Edit(sb.String(), tele.ModeHTML)
}

// sendSubscriptionQR QR Code اشتراک را می‌فرستد.
func (h *Handler) sendSubscriptionQR(ctx context.Context, c tele.Context, subIDStr string) error {
	subID, _ := uuid.Parse(subIDStr)
	sub, err := h.store.FindSubscriptionByID(ctx, subID)
	if err != nil || sub == nil {
		return c.Edit("❌ اشتراک یافت نشد.")
	}

	vpnUser, err := h.panel.GetUser(ctx, sub.Username)
	if err != nil || vpnUser == nil || len(vpnUser.Links) == 0 {
		return c.Edit("❌ خطا در دریافت اشتراک.")
	}

	// ارسال QR Code به‌صورت URL
	qr := qrURL(vpnUser.Links[0])
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.URL("📱 باز کردن QR", qr)))
	return c.Edit("برای دریافت QR Code روی دکمه زیر کلیک کنید:", kb)
}

// onRenewSelected تمدید اشتراک.
func (h *Handler) onRenewSelected(ctx context.Context, c tele.Context, subIDStr string) error {
	uid := c.Sender().ID

	plans, _ := h.store.ListPlans(ctx)
	if len(plans) == 0 {
		return c.Edit("هیچ پلنی موجود نیست.")
	}

	h.setStep(ctx, uid, stepRenewSub, "sub_id", subIDStr)
	return c.Edit("پلن تمدید را انتخاب کنید:", kbPlans(plans))
}

// ════════════════════════════════════════════════════════════
// کیف پول
// ════════════════════════════════════════════════════════════

func (h *Handler) onWallet(c tele.Context) error {
	ctx := context.Background()
	u, _ := h.getOrCreate(ctx, c)

	return c.Send(
		fmt.Sprintf(
			"<b>💳 کیف پول</b>\n\nموجودی: <b>%.0f تومان</b>",
			u.Balance,
		),
		tele.ModeHTML, kbMain(),
	)
}

// ════════════════════════════════════════════════════════════
// helpers
// ════════════════════════════════════════════════════════════

func (h *Handler) sendMain(c tele.Context, msg string) error {
	return c.Send(msg, kbMain())
}

func (h *Handler) notifyAdmin(ctx context.Context, msg string) {
	if h.ownerID == 0 {
		return
	}
	h.bot.Send(&tele.User{ID: h.ownerID}, msg, tele.ModeHTML)
}

func (h *Handler) sendPhotoToAdmin(ctx context.Context, fileID, caption string) {
	if h.ownerID == 0 {
		return
	}
	h.bot.Send(&tele.User{ID: h.ownerID}, &tele.Photo{File: tele.File{FileID: fileID}, Caption: caption})
}

func getUserID(ctx context.Context, h *Handler, c tele.Context) uuid.UUID {
	u, _ := h.getOrCreate(ctx, c)
	if u != nil {
		return u.ID
	}
	return uuid.Nil
}

// confirmBuyWithBalance خرید مستقیم از موجودی کیف پول.
func (h *Handler) confirmBuyWithBalance(ctx context.Context, c tele.Context, planIDStr string) error {
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return c.Edit("❌ پلن نامعتبر.")
	}
	plan, err := h.store.FindPlan(ctx, planID)
	if err != nil || plan == nil {
		return c.Edit("❌ پلن یافت نشد.")
	}
	u, _ := h.getOrCreate(ctx, c)
	if u == nil {
		return c.Edit("❌ خطا.")
	}
	if u.Balance < plan.Price {
		return c.Edit("❌ موجودی کافی نیست.")
	}
	if err := h.store.UpdateBalance(ctx, u.ID, -plan.Price); err != nil {
		return c.Edit("❌ خطا در کسر موجودی.")
	}
	h.clearState(ctx, c.Sender().ID)
	return h.activateSubscription(ctx, c, u.ID, plan)
}

// verifyOnlinePayment پرداخت آنلاین را با gateway تأیید می‌کند.
func (h *Handler) verifyOnlinePayment(ctx context.Context, c tele.Context, refID string) error {
	uid := c.Sender().ID
	st := h.getState(ctx, uid)
	planID, err := uuid.Parse(st.Data["plan_id"])
	if err != nil {
		return c.Edit("❌ اطلاعات پرداخت یافت نشد.")
	}
	plan, _ := h.store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Edit("❌ پلن یافت نشد.")
	}
	resp, err := h.gateway.VerifyPayment(ctx, refID)
	if err != nil || resp == nil || !resp.Success {
		return c.Edit("❌ تأیید پرداخت ناموفق بود. اگر مبلغ کسر شده با پشتیبانی تماس بگیرید.")
	}
	h.clearState(ctx, uid)
	u, _ := h.getOrCreate(ctx, c)
	if u == nil {
		return c.Edit("❌ خطا.")
	}
	return h.activateSubscription(ctx, c, u.ID, plan)
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// activateSubscription اشتراک را روی پنل VPN فعال می‌کند.
func (h *Handler) activateSubscription(ctx context.Context, c tele.Context, userID uuid.UUID, plan *models.Plan) error {
	// ساخت کاربر روی پنل
	username := genVPNUsername(c.Sender().ID)
	expiresAt := time.Now().AddDate(0, 0, plan.DurationDay)
	dataLimitBytes := int64(plan.DataGB * 1e9)

	vpnUser, err := h.panel.CreateUser(ctx, ports.CreateVPNUserRequest{
		Username:  username,
		DataLimit: dataLimitBytes,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return err
	}

	// پیدا کردن بهترین پنل
	// پنل از env تنظیم شده — یک پنل داریم
	panelID := uuid.Nil

	// ثبت subscription در DB
	sub := &models.Subscription{
		UserID:    userID,
		PanelID:   panelID,
		PlanID:    plan.ID,
		Username:  vpnUser.Username,
		Status:    models.SubActive,
		ExpiresAt: expiresAt,
		DataLimit: float64(dataLimitBytes),
	}
	if err := h.store.CreateSubscription(ctx, sub); err != nil {
		return err
	}

	// اطلاع به کاربر
	var sb strings.Builder
	sb.WriteString("🎉 <b>اشتراک فعال شد!</b>\n\n")
	sb.WriteString(fmtSub(*sub))
	sb.WriteString("\n\n🔗 لینک‌های اتصال:")
	for i, link := range vpnUser.Links {
		sb.WriteString(fmt.Sprintf("\n<b>%d.</b> <code>%s</code>", i+1, link))
	}

	return c.Send(sb.String(), tele.ModeHTML, kbSubscription(sub.ID.String()))
}
