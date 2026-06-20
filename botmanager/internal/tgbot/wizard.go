// Package tgbot - wizard.go
// Self-Service Bot Provisioning Wizard.
package tgbot

import (
	"context"
	"time"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// wizard data keys
const (
	wkServiceType = "service_type"
	wkPlanID      = "plan_id"
	wkBotToken    = "bot_token"
)

// ── Step 1: انتخاب نوع سرویس ─────────────────────────────

func (h *Handler) wizardSelectType(ctx context.Context, c tele.Context) error {
	return c.Edit(
		"<b>🤖 ایجاد سرویس جدید</b>\n\nنوع سرویس مورد نظر را انتخاب کنید:",
		tele.ModeHTML, kbServiceCreate(),
	)
}

// ── Step 2: انتخاب پلن ──────────────────────────────────

func (h *Handler) wizardSelectPlan(ctx context.Context, c tele.Context, serviceType string) error {
	uid := c.Sender().ID
	u, _ := h.getOrCreateUser(ctx, c)
	if u == nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
	}

	ok, _ := h.checkBuildCapacityForType(ctx, c, serviceType)
	if !ok {
		// پیام قبلاً در checkBuildCapacityForType ارسال شده
		return nil
	}

	plans, err := h.store.ListPlansByType(ctx, serviceType)
	if err != nil || len(plans) == 0 {
		return c.Edit("❌ پلنی برای این سرویس یافت نشد.")
	}

	// ذخیره service type در wizard state
	st := h.getState(ctx, uid)
	if st.Data == nil {
		st.Data = make(map[string]string)
	}
	st.Step = "wizard_plan"
	st.Data[wkServiceType] = serviceType
	h.setState(ctx, uid, st)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		label := fmt.Sprintf("%s — %.1f TON", p.Name, p.Price)
		if p.IsFree {
			label = "🆓 " + p.Name + " (رایگان)"
		}
		rows = append(rows, kb.Row(kb.Data(label, "wizard_plan:"+p.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "back_services")))
	kb.Inline(rows...)

	return c.Edit(
		fmt.Sprintf("<b>💎 انتخاب پلن</b>\n\nسرویس: %s\nپلن خود را انتخاب کنید:",
			serviceTypeLabel(serviceType)),
		tele.ModeHTML, kb,
	)
}

// ── Step 3: ورود توکن ربات ──────────────────────────────

func (h *Handler) wizardEnterToken(ctx context.Context, c tele.Context, planID string) error {
	uid := c.Sender().ID
	plan, err := h.store.FindPlan(ctx, planID)
	if err != nil || plan == nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
	}

	st := h.getState(ctx, uid)
	if st.Data == nil {
		st.Data = make(map[string]string)
	}
	st.Step = "wizard_token"
	st.Data[wkPlanID] = planID
	h.setState(ctx, uid, st)

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("❌ لغو", "cancel")))

	return c.Edit(
		fmt.Sprintf(
			"<b>🔑 توکن ربات</b>\n\n"+
				"پلن: <b>%s</b> — %.2f TON\n\n"+
				"توکن ربات تلگرام خود را از @BotFather دریافت و ارسال کنید:\n\n"+
				"<code>1234567890:ABCDefgh...</code>",
			plan.Name, plan.Price,
		),
		tele.ModeHTML, kb,
	)
}

// wizardFinish دریافت توکن در onText و رفتن به تأیید.
func (h *Handler) wizardFinish(ctx context.Context, c tele.Context, _ string, token string) error {
	uid := c.Sender().ID
	st := h.getState(ctx, uid)
	data := st.Data
	if data == nil {
		data = map[string]string{}
	}

	planID := data[wkPlanID]
	serviceType := data[wkServiceType]

	if planID == "" || serviceType == "" {
		return c.Send("❌ لطفاً از ابتدا شروع کنید.")
	}

	// اعتبارسنجی توکن
	botID, err := extractBotID(token)
	if err != nil {
		return c.Send("❌ توکن نامعتبر است.\nمثال: <code>1234567890:ABC...</code>",
			tele.ModeHTML)
	}

	// بررسی تکراری نبودن
	if existing, _ := h.store.FindInstanceByBotID(ctx, botID); existing != nil {
		return c.Send("❌ این ربات قبلاً ثبت شده است.")
	}

	plan, _ := h.store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Send(h.t(ctx, uid, i18n.KeyError))
	}

	// ذخیره توکن
	st.Step = "wizard_confirm"
	st.Data[wkBotToken] = token
	h.setState(ctx, uid, st)

	msg := fmt.Sprintf(
		"<b>✅ تأیید ایجاد سرویس</b>\n\n"+
			"🤖 نوع: <b>%s</b>\n"+
			"💎 پلن: <b>%s</b>\n"+
			"💰 قیمت: <b>%.2f TON</b>\n\n"+
			"آیا مطمئن هستید؟",
		serviceTypeLabel(serviceType), plan.Name, plan.Price,
	)

	kb := &tele.ReplyMarkup{}
	if plan.IsFree || plan.Price == 0 {
		kb.Inline(
			kb.Row(kb.Data("✅ ایجاد رایگان", "wizard_create:free"),
				kb.Data("❌ لغو", "cancel")),
		)
	} else {
		kb.Inline(
			kb.Row(kb.Data("✅ پرداخت و ایجاد", "wizard_pay"),
				kb.Data("❌ لغو", "cancel")),
		)
	}

	return c.Send(msg, tele.ModeHTML, kb)
}

// ── Step 5: پرداخت ──────────────────────────────────────

func (h *Handler) wizardPay(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	defer c.Respond()
	st := h.getState(ctx, uid)
	data := st.Data
	if data == nil {
		return c.Edit("❌ اطلاعات ناقص است. از ابتدا شروع کنید.")
	}

	planID := data[wkPlanID]
	token := data[wkBotToken]
	serviceType := data[wkServiceType]

	plan, _ := h.store.FindPlan(ctx, planID)
	u, _ := h.getOrCreateUser(ctx, c)
	if plan == nil || u == nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
	}

	invoiceCode, err := h.pay.DeductForService(ctx, u.TelegramID, plan.Price, planID)
	if err != nil {
		return c.Edit("❌ موجودی کافی نیست. لطفاً کیف پول خود را شارژ کنید.")
	}

	return h.provision(ctx, c, u, plan, token, serviceType, invoiceCode)
}

func (h *Handler) wizardCreateFree(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	defer c.Respond()
	st := h.getState(ctx, uid)
	data := st.Data
	if data == nil {
		return c.Edit("❌ اطلاعات ناقص است.")
	}

	plan, _ := h.store.FindPlan(ctx, data[wkPlanID])
	u, _ := h.getOrCreateUser(ctx, c)
	if plan == nil || u == nil {
		return c.Edit(h.t(ctx, uid, i18n.KeyError))
	}

	return h.provision(ctx, c, u, plan, data[wkBotToken], data[wkServiceType], "")
}

// ── Core Provisioning ────────────────────────────────────

func (h *Handler) provision(
	ctx context.Context, c tele.Context,
	u *models.User, plan *models.Plan,
	token, serviceType, invoiceCode string,
) error {
	uid := c.Sender().ID
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = c.Edit("⏳ در حال راه‌اندازی سرویس...")

	server, err := h.store.SelectLeastLoadedServer(ctx)
	if err != nil || server == nil {
		h.refundOnFailure(ctx, u, plan, invoiceCode)
		return c.Send("❌ هیچ سروری در دسترس نیست.")
	}

	tmpl, err := h.store.FindTemplateByType(ctx, serviceType)
	if err != nil || tmpl == nil {
		h.refundOnFailure(ctx, u, plan, invoiceCode)
		return c.Send("❌ قالب سرویس یافت نشد.")
	}

	botID, _ := extractBotID(token)
	containerName := fmt.Sprintf("%s_%d", serviceType, botID)

	// LockMode اولیه: پلن رایگان → free (تبلیغ خودمان)، در غیر این صورت none
	// (در فاز اجاره، ads-bot می‌تواند بعداً LockMode را به rented تغییر دهد)
	lockMode := models.LockModeNone
	var planID *uuid.UUID
	if plan != nil {
		planID = &plan.ID
		if plan.IsFree {
			lockMode = models.LockModeFree
		}
	}

	instance := &models.BotInstance{
		OwnerID:       u.ID,
		TemplateID:    tmpl.ID,
		ServerID:      server.ID,
		BotToken:      token,
		ContainerName: containerName,
		Status:        "pending",
		PlanID:        planID,
		LockMode:      lockMode,
	}

	if err := h.store.CreateInstance(ctx, instance); err != nil {
		h.refundOnFailure(ctx, u, plan, invoiceCode)
		return c.Send("❌ خطا در ایجاد سرویس.")
	}

	// اگر این instance قفل رایگان پلتفرم دارد، به ads-bot اطلاع بده تا آن را
	// به‌عنوان یک FreeBotSlot ثبت کند (بعداً قابل اجاره به خریداران است).
	if h.nc != nil && lockMode == models.LockModeFree {
		h.nc.PublishCore(protocol.SubjFreeBotCreated, protocol.FreeBotCreatedEvent{
			InstanceID: instance.ID.String(),
			BotID:      botID,
		})
	}

	if h.nc != nil {
		h.nc.PublishCore(protocol.ServiceCreationRequested, protocol.ServiceProvisionPayload{
			InstanceID:  instance.ID.String(),
			OwnerID:     u.ID.String(),
			ServiceType: serviceType,
			PlanID:      plan.ID.String(),
			InvoiceCode: invoiceCode,
		})
	}

	jwtToken, _ := auth.GenerateAccessToken(
		u.ID.String(), "user",
		auth.JWTConfig{AccessSecret: h.encryptKey},
	)

	cmd := protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ContainerName: containerName,
		ImageName:     tmpl.ImageName,
		ImageTag:      tmpl.ImageTag,
		EnvVars: map[string]string{
			"BOT_TOKEN":      token,
			"INSTANCE_ID":    "bot_" + fmt.Sprint(botID),
			"OWNER_TELEGRAM": fmt.Sprint(u.TelegramID),
			"PLAN_ID":        plan.ID.String(),
			"JWT_TOKEN":      jwtToken,
		},
	}

	if err := h.nc.Publish(ctx, protocol.DeploySubject(server.ID.String()), cmd); err != nil {
		h.log.Error("deploy failed", ports.F("err", err))
		h.refundOnFailure(ctx, u, plan, invoiceCode)
		h.store.UpdateInstanceStatus(ctx, instance.ID.String(), "failed")
		return c.Send("❌ خطا در ارسال دستور deploy.")
	}

	h.clearState(ctx, uid)
	sub, _ := h.store.GetActiveSubscription(ctx, u.ID)
	return c.Send(
		fmt.Sprintf(
			"<b>🎉 سرویس در حال راه‌اندازی است!</b>\n\n"+
				"🤖 نوع: %s\n💎 پلن: %s\n📦 وضعیت: <b>در حال راه‌اندازی</b>\n\n"+
				"ظرف ۲-۳ دقیقه آماده می‌شود.",
			serviceTypeLabel(serviceType), plan.Name,
		),
		tele.ModeHTML,
		h.kbUserFull(ctx, uid, sub),
	)
}

func (h *Handler) refundOnFailure(ctx context.Context, u *models.User, plan *models.Plan, invoiceCode string) {
	if invoiceCode == "" || plan.Price == 0 {
		return
	}
	if err := h.pay.RefundService(ctx, u.TelegramID, plan.Price, invoiceCode); err != nil {
		h.log.Error("refund failed", ports.F("err", err))
	}
}

// ── Backward compat: invite link flow ────────────────────

func (h *Handler) wizardStart(ctx context.Context, c tele.Context, token string) error {
	uid := c.Sender().ID
	link, err := h.store.FindInviteLinkByToken(ctx, token)
	if err != nil {
		return c.Send(h.t(ctx, uid, i18n.KeyError))
	}
	if link == nil {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardInvalidLink))
	}
	if link.IsExpired() {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardExpiredLink))
	}
	if link.IsExhausted() {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardUsedLink))
	}
	return h.wizardSelectType(ctx, c)
}

// ── Helpers ──────────────────────────────────────────────

func serviceTypeLabel(t string) string {
	m := map[string]string{
		"vpn":      "🌐 VPN",
		"uploader": "📤 آپلودر",
		"member":   "🔒 قفل ممبرشیپ",
		"archive":  "📦 آرشیو",
	}
	if l, ok := m[t]; ok {
		return l
	}
	return t
}

func extractBotID(token string) (int64, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid token")
	}
	var id int64
	if _, err := fmt.Sscan(parts[0], &id); err != nil {
		return 0, err
	}
	return id, nil
}
