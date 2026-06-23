package tgbot

import (
	"context"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
)

// onText پیام‌های متنی.
func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := c.Text()

	// لغو — همیشه اول
	if h.IsCancel(ctx, uid, text) {
		return h.onCancel(c)
	}

	// ── ادمین ────────────────────────────────────────────
	// دکمه‌های منوی اصلی همیشه اولویت دارند و state را پاک می‌کنند
	if h.IsInAdminMode(c) {
		switch text {
		case h.Btn(ctx, uid, i18n.KeyMenuUsers):
			h.ClearState(ctx, uid)
			return h.AdminUsersList(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuBots):
			h.ClearState(ctx, uid)
			return h.AdminBotsList(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuPlans):
			h.ClearState(ctx, uid)
			return h.AdminPlansList(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuTemplates):
			h.ClearState(ctx, uid)
			return h.AdminTemplatesList(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuServers):
			h.ClearState(ctx, uid)
			return h.AdminServersList(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuStats):
			h.ClearState(ctx, uid)
			return h.AdminStats(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuBroadcast):
			h.ClearState(ctx, uid)
			return h.AdminBroadcastMenu(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuSystem):
			h.ClearState(ctx, uid)
			return h.AdminSystemMenu(ctx, c)
		case h.Btn(ctx, uid, i18n.KeyMenuExitAdmin):
			h.ClearState(ctx, uid)
			h.SetAdminMode(ctx, uid, false)
			return c.Send(
				h.T(ctx, uid, i18n.KeyWelcomeUser, c.Sender().FirstName),
				h.KbUser(ctx, uid),
			)
		}

		// state فعال (ادمین)
		st := h.GetState(ctx, uid)
		if st.Step != stepIdle {
			return h.handleStep(ctx, c, st, text)
		}
		return nil
	}

	// ── کاربر ────────────────────────────────────────────
	// دکمه‌های منوی اصلی همیشه اولویت دارند و state را پاک می‌کنند
	switch text {
	case h.Btn(ctx, uid, i18n.KeyMenuMyBots):
		h.ClearState(ctx, uid)
		return h.UserBotsList(ctx, c)
	case h.Btn(ctx, uid, i18n.KeyMenuCreateBot):
		h.ClearState(ctx, uid)
		return h.WizardSelectType(ctx, c)
	case h.Btn(ctx, uid, i18n.KeyMenuWallet):
		h.ClearState(ctx, uid)
		return h.UserWallet(ctx, c)
	case h.Btn(ctx, uid, i18n.KeyMenuPlans):
		h.ClearState(ctx, uid)
		return h.UserPlans(ctx, c)
	case h.Btn(ctx, uid, i18n.KeyMenuSettings):
		h.ClearState(ctx, uid)
		return h.UserSettings(ctx, c)
	case h.Btn(ctx, uid, i18n.KeyMenuSupport):
		h.ClearState(ctx, uid)
		return h.UserSupport(c)
	case h.Btn(ctx, uid, i18n.KeyMenuLanguage):
		h.ClearState(ctx, uid)
		return h.UserLanguageMenu(ctx, c)
	case h.Btn(ctx, uid, i18n.KeyMenuAccount):
		h.ClearState(ctx, uid)
		return h.UserAccount(ctx, c)
	}

	// state فعال (کاربر)
	st := h.GetState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	return nil
}

// handleStep مدیریت state machine.
func (h *Handler) handleStep(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID

	if h.IsCancel(ctx, uid, text) {
		h.ClearState(ctx, uid)
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyCancelled))
	}

	switch st.Step {

	// ══ سرور ══════════════════════════════════════════════
	case stepServerName:
		h.SetStep(ctx, uid, stepServerIP, "name", text)
		return c.Send(h.T(ctx, uid, i18n.KeyServerAskIP),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	case stepServerIP:
		return h.AdminServerAdd(ctx, c, st.Data["name"], text)

	// ══ تمپلیت ════════════════════════════════════════════
	case stepTmplType:
		// نوعِ سرویس متن آزاد است (انواع پویا). نرمال‌سازی: trim + lower.
		bt := strings.ToLower(strings.TrimSpace(text))
		if bt == "" {
			return c.Send(h.T(ctx, uid, i18n.KeyTemplateAskType),
				tele.ModeHTML, h.KbBackCancel(ctx, uid))
		}
		h.SetStep(ctx, uid, stepTmplImage, "type", bt)
		return c.Send(h.T(ctx, uid, i18n.KeyTemplateAskImage),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	case stepTmplImage:
		h.SetStep(ctx, uid, stepTmplTag, "image", text)
		return c.Send(h.T(ctx, uid, i18n.KeyTemplateAskTag),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	case stepTmplTag:
		h.SetStep(ctx, uid, stepTmplName, "tag", text)
		return c.Send(h.T(ctx, uid, i18n.KeyTemplateAskName),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	case stepTmplName:
		return h.AdminTemplateAdd(ctx, c, st.Data["type"], st.Data["image"], st.Data["tag"], text)

	// ══ پلن ═══════════════════════════════════════════════
	case stepPlanTmpl:
		return h.AdminPlanStepName(ctx, c, text)
	case stepPlanName:
		h.SetStep(ctx, uid, stepPlanDays, "name", text)
		return c.Send(h.T(ctx, uid, i18n.KeyPlanAskDays),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	case stepPlanDays:
		days, err := strconv.Atoi(text)
		if err != nil || days < 0 {
			return c.Send(h.T(ctx, uid, i18n.KeyPlanInvalidNumber), h.KbBackCancel(ctx, uid))
		}
		h.SetStep(ctx, uid, stepPlanPrice, "days", text)
		return c.Send(h.T(ctx, uid, i18n.KeyPlanAskPrice),
			tele.ModeHTML, h.KbBackCancel(ctx, uid))
	case stepPlanPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil || price < 0 {
			return c.Send(h.T(ctx, uid, i18n.KeyPlanInvalidNumber), h.KbBackCancel(ctx, uid))
		}
		days, _ := strconv.Atoi(st.Data["days"])
		return h.AdminPlanAdd(ctx, c, st.Data["tmpl_id"], st.Data["name"], days, price)
	case stepPlanLimits:
		return h.AdminPlanSetLimits(ctx, c, st.Data["plan_id"], text)

	// ══ کاربر ═════════════════════════════════════════════
	case stepUserAction:
		return h.AdminUserHandleAction(ctx, c, st.Data["user_id"], text)

	// ══ افزودن اعتبار ═════════════════════════════════════
	case stepAdminCreditAmount:
		return h.AdminCreditExecute(ctx, c, st.Data["target_tid"], text)

	// ══ واریز کیف پول ════════════════════════════════════
	case stepWalletTopupAmount:
		return h.WalletTopupProcess(ctx, c, text)

	// ══ ارسال همگانی ══════════════════════════════════════
	case stepBroadcastText:
		return h.BroadcastExecute(ctx, c, text)

	// ══ دپلوی تستی ادمین ══════════════════════════════════
	case stepAdminTestToken:
		return h.AdminTestDeploy(ctx, c, st.Data["tmpl_id"], text)

	// ══ wizard ════════════════════════════════════════════
	case stepWizardToken:
		return h.WizardFinish(ctx, c, "", text)

	// ══ زبان ══════════════════════════════════════════════
	case stepLangSelect:
		h.ClearState(ctx, uid)
		switch text {
		case "🇮🇷 فارسی":
			h.Tr.SetLang(ctx, uid, i18n.FA)
		case "🇬🇧 English":
			h.Tr.SetLang(ctx, uid, i18n.EN)
		}
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyLangChanged))
	}

	return nil
}
