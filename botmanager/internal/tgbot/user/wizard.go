// Package tgbot - wizard.go
// Self-Service Bot Provisioning Wizard.
package user

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/google/uuid"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/core"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// wizard data keys
const (
	wkServiceType = "service_type"
	wkTag         = "service_tag"
	wkPlanID      = "plan_id"
	wkBotToken    = "bot_token"
)

// testTag تگِ مخصوص تست؛ از کاربران عادی مخفی است و فقط ادمین می‌تواند
// آن را نصب/تست کند.
const testTag = "test"

// wizardSteps تعداد کل مراحل ویزارد (سرویس → تگ → پلن → توکن).
const wizardSteps = 4

// wizStep عنوان یک مرحله را با نشانگر پیشرفت می‌سازد.
func (h *User) WizStep(ctx context.Context, uid int64, step int, body string) string {
	return h.T(ctx, uid, i18n.KeyWizardStep, step, wizardSteps) + "\n\n" + body
}

// ── Step 1: انتخاب سرویس (پویا از DB) ────────────────────
// انواع سرویس از روی templateهای فعال (distinct type) ساخته می‌شوند؛
// هیچ نوعی در کد hardcode نیست. با EditOrSend هم از callback و هم از دکمه‌ی
// منوی reply کار می‌کند.

func (h *User) WizardSelectType(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	types, err := h.Store.ListServiceTypes(ctx)
	if err != nil || len(types) == 0 {
		return c.EditOrSend(h.T(ctx, uid, i18n.KeyWizardNoTemplate))
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, t := range types {
		rows = append(rows, kb.Row(kb.Data(t, "svc_type:"+t)))
	}
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel")))
	kb.Inline(rows...)

	title := h.WizStep(ctx, uid, 1, h.T(ctx, uid, i18n.KeyServiceSelectType))
	return c.EditOrSend(title, tele.ModeHTML, kb)
}

// ── Step 2: انتخاب تگ (نسخه‌ی) سرویس ─────────────────────
// همه‌ی تگ‌های یک سرویس در همین یک پنل‌اند؛ کاربر نسخه‌ی دلخواه را نصب می‌کند.

func (h *User) WizardSelectTag(ctx context.Context, c tele.Context, serviceType string) error {
	uid := c.Sender().ID

	ok, _ := h.CheckBuildCapacityForType(ctx, c, serviceType)
	if !ok {
		return nil // پیام در checkBuildCapacityForType ارسال شده
	}

	tmpls, err := h.Store.ListTemplatesByType(ctx, serviceType)
	if err != nil || len(tmpls) == 0 {
		return c.Edit(h.T(ctx, uid, i18n.KeyWizardNoTemplate))
	}

	st := h.GetState(ctx, uid)
	if st.Data == nil {
		st.Data = make(map[string]string)
	}
	st.Step = "wizard_tag"
	st.Data[wkServiceType] = serviceType
	h.SetState(ctx, uid, st)

	// تگ‌های قابل‌نصب برای این کاربر (test فقط برای ادمین)
	isAdmin := h.IsAdmin(c)
	var avail []models.BotTemplate
	for _, t := range tmpls {
		if t.ImageTag == testTag && !isAdmin {
			continue
		}
		avail = append(avail, t)
	}
	if len(avail) == 0 {
		return c.Edit(h.T(ctx, uid, i18n.KeyWizardNoTemplate))
	}

	// اگر فقط یک تگ هست، مرحله را خودکار رد کن (کاهش اصطکاک)
	if len(avail) == 1 {
		return h.WizardSelectPlan(ctx, c, avail[0].ImageTag)
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, t := range avail {
		label := t.ImageTag
		if t.Name != "" {
			label = t.Name + " (" + t.ImageTag + ")"
		}
		if i == 0 { // جدیدترین (created_at desc)
			label += " · " + h.T(ctx, uid, i18n.KeyBadgeNewest)
		}
		rows = append(rows, kb.Row(kb.Data(label, "svc_tag:"+t.ImageTag)))
	}
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "svc_create")))
	kb.Inline(rows...)

	title := h.WizStep(ctx, uid, 2, h.T(ctx, uid, i18n.KeyServiceSelectTag, serviceType))
	return c.Edit(title, tele.ModeHTML, kb)
}

// ── Step 3: انتخاب پلن ───────────────────────────────────

func (h *User) WizardSelectPlan(ctx context.Context, c tele.Context, tag string) error {
	uid := c.Sender().ID
	st := h.GetState(ctx, uid)
	if st.Data == nil {
		st.Data = make(map[string]string)
	}
	serviceType := st.Data[wkServiceType]
	if serviceType == "" {
		return c.Edit(h.T(ctx, uid, i18n.KeyWizardRestart))
	}

	plans, err := h.Store.ListPlansByType(ctx, serviceType)
	if err != nil || len(plans) == 0 {
		return c.Edit(h.T(ctx, uid, i18n.KeyWizardNoPlan))
	}

	st.Step = "wizard_plan"
	st.Data[wkTag] = tag
	h.SetState(ctx, uid, st)

	// badge «محبوب» روی پلن میانی (anchor pricing) وقتی ۳ پلن یا بیشتر باشد
	popularIdx := -1
	if len(plans) >= 3 {
		popularIdx = len(plans) / 2
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, p := range plans {
		label := fmt.Sprintf("%s — %.1f TON", p.Name, p.Price)
		if p.IsFree {
			label = "🆓 " + p.Name
		}
		if i == popularIdx {
			label += " · " + h.T(ctx, uid, i18n.KeyBadgePopular)
		}
		rows = append(rows, kb.Row(kb.Data(label, "wizard_plan:"+p.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBack), "svc_type:"+serviceType)))
	kb.Inline(rows...)

	title := h.WizStep(ctx, uid, 3, h.T(ctx, uid, i18n.KeyServiceSelectPlan, serviceType))
	return c.Edit(title, tele.ModeHTML, kb)
}

// ── Step 3: ورود توکن ربات ──────────────────────────────

func (h *User) WizardEnterToken(ctx context.Context, c tele.Context, planID string) error {
	uid := c.Sender().ID
	plan, err := h.Store.FindPlan(ctx, planID)
	if err != nil || plan == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	st := h.GetState(ctx, uid)
	if st.Data == nil {
		st.Data = make(map[string]string)
	}
	// باید ثابتِ state.StepWizardToken باشد تا handleStep ورودیِ متنیِ توکن را
	// به wizardFinish بفرستد. مقدار خام "wizard_token" با هیچ case‌ای
	// مطابقت نمی‌کرد و باعث می‌شد ربات بعد از ارسال توکن هیچ پاسخی ندهد.
	st.Step = state.StepWizardToken
	st.Data[wkPlanID] = planID
	h.SetState(ctx, uid, st)

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel")))

	title := h.WizStep(ctx, uid, 4, h.T(ctx, uid, i18n.KeyServiceEnterToken, plan.Name, plan.Price))
	return c.Edit(title, tele.ModeHTML, kb)
}

// wizardFinish دریافت توکن در onText و رفتن به تأیید.
func (h *User) WizardFinish(ctx context.Context, c tele.Context, _ string, token string) error {
	uid := c.Sender().ID
	st := h.GetState(ctx, uid)
	data := st.Data
	if data == nil {
		data = map[string]string{}
	}

	planID := data[wkPlanID]
	serviceType := data[wkServiceType]

	if planID == "" || serviceType == "" {
		return c.Send(h.T(ctx, uid, i18n.KeyWizardRestart))
	}

	// اعتبارسنجی توکن
	botID, err := extractBotID(token)
	if err != nil {
		return c.Send(h.T(ctx, uid, i18n.KeyServiceInvalidToken), tele.ModeHTML)
	}

	// بررسی تکراری نبودن
	if existing, _ := h.Store.FindInstanceByBotID(ctx, botID); existing != nil {
		return c.Send(h.T(ctx, uid, i18n.KeyServiceDuplicate))
	}

	plan, _ := h.Store.FindPlan(ctx, planID)
	if plan == nil {
		return c.Send(h.T(ctx, uid, i18n.KeyError))
	}

	// ذخیره توکن
	st.Step = "wizard_confirm"
	st.Data[wkBotToken] = token
	h.SetState(ctx, uid, st)

	msg := h.T(ctx, uid, i18n.KeyServiceConfirm, serviceType, st.Data[wkTag], plan.Name, plan.Price)

	kb := &tele.ReplyMarkup{}
	if plan.IsFree || plan.Price == 0 {
		kb.Inline(
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCreateFree), "wizard_create:free"),
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel")),
		)
	} else {
		kb.Inline(
			kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnPayCreate), "wizard_pay"),
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel")),
		)
	}

	return c.Send(msg, tele.ModeHTML, kb)
}

// ── Step 5: پرداخت ──────────────────────────────────────

func (h *User) WizardPay(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	defer func() { _ = c.Respond() }()
	st := h.GetState(ctx, uid)
	data := st.Data
	if data == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyWizardIncomplete))
	}

	planID := data[wkPlanID]
	token := data[wkBotToken]
	serviceType := data[wkServiceType]

	plan, _ := h.Store.FindPlan(ctx, planID)
	u, _ := h.GetOrCreateUser(ctx, c)
	if plan == nil || u == nil || h.Pay == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	invoiceCode, err := h.Pay.DeductForService(ctx, u.TelegramID, plan.Price, planID)
	if err != nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyWizardLowBalance), tele.ModeHTML)
	}

	return h.Provision(ctx, c, u, plan, token, serviceType, data[wkTag], invoiceCode)
}

func (h *User) WizardCreateFree(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	defer func() { _ = c.Respond() }()
	st := h.GetState(ctx, uid)
	data := st.Data
	if data == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyWizardIncomplete))
	}

	plan, _ := h.Store.FindPlan(ctx, data[wkPlanID])
	u, _ := h.GetOrCreateUser(ctx, c)
	if plan == nil || u == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	return h.Provision(ctx, c, u, plan, data[wkBotToken], data[wkServiceType], data[wkTag], "")
}

// ── Core Provisioning ────────────────────────────────────

func (h *User) Provision(
	ctx context.Context, c tele.Context,
	u *models.User, plan *models.Plan,
	token, serviceType, tag, invoiceCode string,
) error {
	uid := c.Sender().ID
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = c.Edit(h.T(ctx, uid, i18n.KeyServiceCreating))

	server, err := h.Store.SelectLeastLoadedServer(ctx)
	if err != nil || server == nil {
		h.RefundOnFailure(ctx, u, plan, invoiceCode)
		return c.Send(h.T(ctx, uid, i18n.KeyWizardNoServer))
	}

	// تمپلیتِ دقیقِ سرویس+تگ انتخابی کاربر
	tmpl, err := h.Store.FindTemplateByTypeAndTag(ctx, serviceType, tag)
	if err != nil || tmpl == nil {
		h.RefundOnFailure(ctx, u, plan, invoiceCode)
		return c.Send(h.T(ctx, uid, i18n.KeyWizardNoTemplate))
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
		BotID:         botID,
		DBSchema:      fmt.Sprintf("inst_%d", botID),
		Status:        "pending",
		PlanID:        planID,
		LockMode:      lockMode,
	}

	if err := h.Store.CreateInstance(ctx, instance); err != nil {
		h.RefundOnFailure(ctx, u, plan, invoiceCode)
		return c.Send(h.T(ctx, uid, i18n.KeyWizardCreateError))
	}

	// اگر این instance قفل رایگان پلتفرم دارد، به ads-bot اطلاع بده تا آن را
	// به‌عنوان یک FreeBotSlot ثبت کند (بعداً قابل اجاره به خریداران است).
	if h.NC != nil && lockMode == models.LockModeFree {
		_ = h.NC.PublishCore(protocol.SubjFreeBotCreated, protocol.FreeBotCreatedEvent{
			InstanceID: instance.ID.String(),
			BotID:      botID,
		})
	}

	planIDStr := ""
	if plan != nil {
		planIDStr = plan.ID.String()
	}

	if h.NC != nil {
		_ = h.NC.PublishCore(protocol.ServiceCreationRequested, protocol.ServiceProvisionPayload{
			InstanceID:  instance.ID.String(),
			OwnerID:     u.ID.String(),
			ServiceType: serviceType,
			PlanID:      planIDStr,
			InvoiceCode: invoiceCode,
		})
	}

	jwtToken, _ := auth.GenerateAccessToken(
		u.ID.String(), "user",
		auth.JWTConfig{AccessSecret: h.EncryptKey},
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
			"PLAN_ID":        planIDStr,
			"JWT_TOKEN":      jwtToken,
		},
	}

	if h.NC == nil {
		h.RefundOnFailure(ctx, u, plan, invoiceCode)
		_ = h.Store.UpdateInstanceStatus(ctx, instance.ID.String(), "failed")
		return c.Send(h.T(ctx, uid, i18n.KeyWizardDeployError))
	}

	if err := h.NC.Publish(ctx, protocol.DeploySubject(server.ID.String()), cmd); err != nil {
		h.Log.Error("deploy failed", ports.F("err", err))
		h.RefundOnFailure(ctx, u, plan, invoiceCode)
		_ = h.Store.UpdateInstanceStatus(ctx, instance.ID.String(), "failed")
		return c.Send(h.T(ctx, uid, i18n.KeyWizardDeployError))
	}

	h.ClearState(ctx, uid)
	sub, _ := h.Store.GetActiveSubscription(ctx, u.ID)
	return c.Send(
		h.T(ctx, uid, i18n.KeyServiceCreated, serviceType, plan.Name),
		tele.ModeHTML,
		h.KbUserFull(ctx, uid, sub),
	)
}

func (h *User) RefundOnFailure(ctx context.Context, u *models.User, plan *models.Plan, invoiceCode string) {
	if invoiceCode == "" || plan.Price == 0 {
		return
	}
	if err := h.Pay.RefundService(ctx, u.TelegramID, plan.Price, invoiceCode); err != nil {
		h.Log.Error("refund failed", ports.F("err", err))
	}
}

// ── Helpers ──────────────────────────────────────────────

// extractBotID wrapper نازک به core.ExtractBotID (حفظ call siteها و تست).
func extractBotID(token string) (int64, error) { return core.ExtractBotID(token) }
