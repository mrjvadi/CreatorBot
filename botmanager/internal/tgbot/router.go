package tgbot

import (
	"context"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
)

// onCallback inline keyboard callbacks.
func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	data := c.Callback().Data
	defer c.Respond()

	if len(data) > 0 && data[0] == '\f' {
		data = data[1:]
	}

	parts := strings.SplitN(data, ":", 2)
	action := parts[0]
	arg := ""
	if len(parts) == 2 {
		arg = parts[1]
	}

	switch action {

	case "lang":
		return h.setLanguage(ctx, c, arg)
	case "back_main":
		_ = c.Respond()
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyDone))
	case "wallet_topup", "redeem_promo", "bc_text", "bc_forward",
		"bc_filtered", "sys_notif":
		return c.Respond(&tele.CallbackResponse{Text: h.t(ctx, uid, i18n.KeyComingSoon)})
	case "sys_lang":
		_ = c.Respond()
		return h.userLanguageMenu(ctx, c)

	// ── پلن‌ها ────────────────────────────────────────────
	case "show_plans":
		return h.userPlans(ctx, c)
	case "select_plan":
		return h.userSelectPlan(ctx, c, arg)
	case "buy_plan":
		return h.executePlanPurchase(ctx, c, arg)
	case "start_free":
		plan, _ := h.store.FindPlan(ctx, arg)
		u, _ := h.getOrCreateUser(ctx, c)
		if plan != nil && u != nil {
			return h.activateFreePlanInline(ctx, c, u, plan)
		}
		return c.Edit(h.t(ctx, uid, i18n.KeyNoFreePlan))
	case "check_plan":
		sub := strings.SplitN(arg, ":", 2)
		if len(sub) == 2 {
			return h.checkPlanAfterDeposit(ctx, c, sub[0], sub[1])
		}
	case "plan_current":
		return h.userPlans(ctx, c)

	// ── سرویس‌های من ─────────────────────────────────────
	case "my_bots":
		return h.userBotsList(ctx, c)
	case "svc_create":
		return h.wizardSelectType(ctx, c)
	case "svc_type":
		return h.wizardSelectPlan(ctx, c, arg)
	case "wizard_plan":
		return h.wizardEnterToken(ctx, c, arg)
	case "wizard_pay":
		return h.wizardPay(ctx, c)
	case "wizard_create":
		return h.wizardCreateFree(ctx, c)
	case "svc_status":
		return h.instanceStatus(ctx, c, arg)

	// ── عملیات روی سرویس ─────────────────────────────────
	case "bot_stop":
		return h.instanceAction(ctx, c, arg, "stop")
	case "bot_start":
		return h.instanceAction(ctx, c, arg, "start")
	case "bot_restart":
		return h.instanceAction(ctx, c, arg, "restart")
	case "bot_delete":
		return h.instanceAction(ctx, c, arg, "delete")
	case "svc_stats":
		return h.instanceStats(ctx, c, arg)

	// ── راهنما ────────────────────────────────────────────
	case "how_to_build":
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("✅ متوجه شدم", "cancel")))
		return c.Edit(h.t(ctx, uid, i18n.KeyHowToBuild), tele.ModeHTML, kb)

	// ── ادمین — کاربران ───────────────────────────────────
	case "block_user":
		return h.adminUserAction(ctx, c, arg, "block")
	case "unblock_user":
		return h.adminUserAction(ctx, c, arg, "unblock")
	case "make_admin":
		return h.adminUserAction(ctx, c, arg, "make_admin")
	case "make_user":
		return h.adminUserAction(ctx, c, arg, "make_user")
	case "admin_users":
		return h.adminUsersList(ctx, c)
	case "add_server":
		return h.adminServerStart(ctx, c)
	case "create_link":
		h.setStep(ctx, uid, stepLinkType)
		return c.Edit(h.t(ctx, uid, i18n.KeyLinkAskType), h.kbBotType(ctx, uid))
	case "create_tmpl":
		h.setStep(ctx, uid, stepTmplType)
		return c.Edit(h.t(ctx, uid, i18n.KeyTemplateAskType), h.kbBotType(ctx, uid))

	// ── لغو ───────────────────────────────────────────────
	case "cancel":
		h.clearState(ctx, uid)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyCancelled))
	}

	return nil
}

// onText پیام‌های متنی.
func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := c.Text()

	// state فعال
	st := h.getState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	// wizard pending (invite link)
	if token := h.getWizardPending(ctx, uid); token != "" {
		switch text {
		case h.btn(ctx, uid, i18n.KeyBtnYesBuild):
			h.clearWizardPending(ctx, uid)
			h.setStep(ctx, uid, stepWizardToken, "invite_token", token)
			return c.Send(h.t(ctx, uid, i18n.KeyWizardAskToken),
				tele.ModeHTML, h.kbBackCancel(ctx, uid))
		default:
			if h.isCancel(ctx, uid, text) {
				h.clearWizardPending(ctx, uid)
				return h.sendMain(c, h.t(ctx, uid, i18n.KeyCancelled))
			}
		}
	}

	// cancel
	if h.isCancel(ctx, uid, text) {
		return h.onCancel(c)
	}

	// ── ادمین ────────────────────────────────────────────
	if h.isAdmin(c) {
		switch text {
		case h.btn(ctx, uid, i18n.KeyMenuUsers):
			return h.adminUsersList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuBots):
			return h.adminBotsList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuPlans):
			return h.adminPlansList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuLinks):
			return h.adminLinksList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuServers):
			return h.adminServersList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuTemplates):
			return h.adminTemplatesList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuStats):
			return h.adminStats(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuBroadcast):
			return h.adminBroadcastMenu(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuSystem):
			return h.adminSystemMenu(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuExitAdmin):
			return h.sendMain(c, h.t(ctx, uid, i18n.KeyDone))
		}
		return nil
	}

	// ── کاربر ────────────────────────────────────────────
	switch text {
	case h.btn(ctx, uid, i18n.KeyMenuCreateBot):
		return h.userPlans(ctx, c)
	case h.btn(ctx, uid, i18n.KeyMenuMyBots):
		return h.userBotsList(ctx, c)
	case h.btn(ctx, uid, i18n.KeyMenuAccount):
		return h.userAccount(ctx, c)
	case h.btn(ctx, uid, i18n.KeyMenuPlans):
		return h.userPlans(ctx, c)
	case h.btn(ctx, uid, i18n.KeyMenuTutorials):
		return h.onHelp(c)
	case h.btn(ctx, uid, i18n.KeyMenuSupport):
		return h.userSupport(c)
	case h.btn(ctx, uid, i18n.KeyMenuLanguage):
		return h.userLanguageMenu(ctx, c)
	}

	return nil
}

// handleStep مدیریت state machine.
func (h *Handler) handleStep(ctx context.Context, c tele.Context, st userState, text string) error {
	uid := c.Sender().ID

	if h.isCancel(ctx, uid, text) {
		h.clearState(ctx, uid)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyCancelled))
	}

	switch st.Step {

	// ══ سرور ══════════════════════════════════════════════
	case stepServerName:
		h.setStep(ctx, uid, stepServerIP, "name", text)
		return c.Send(h.t(ctx, uid, i18n.KeyServerAskIP),
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	case stepServerIP:
		return h.adminServerAdd(ctx, c, st.Data["name"], text)

	// ══ تمپلیت ════════════════════════════════════════════
	case stepTmplType:
		bt := h.botTypeFromText(ctx, uid, text)
		if bt == "" {
			return c.Send(h.t(ctx, uid, i18n.KeyTemplateAskType), h.kbBotType(ctx, uid))
		}
		h.setStep(ctx, uid, stepTmplImage, "type", bt)
		return c.Send(h.t(ctx, uid, i18n.KeyTemplateAskImage),
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	case stepTmplImage:
		h.setStep(ctx, uid, stepTmplTag, "image", text)
		return c.Send(h.t(ctx, uid, i18n.KeyTemplateAskTag),
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	case stepTmplTag:
		h.setStep(ctx, uid, stepTmplName, "tag", text)
		return c.Send(h.t(ctx, uid, i18n.KeyTemplateAskName),
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	case stepTmplName:
		return h.adminTemplateAdd(ctx, c, st.Data["type"], st.Data["image"], st.Data["tag"], text)

	// ══ لینک ══════════════════════════════════════════════
	case stepLinkType:
		bt := h.botTypeFromText(ctx, uid, text)
		if bt == "" {
			return c.Send(h.t(ctx, uid, i18n.KeyLinkAskType), h.kbBotType(ctx, uid))
		}
		h.setStep(ctx, uid, stepLinkLimit, "type", bt)
		return c.Send(h.t(ctx, uid, i18n.KeyLinkAskLimit), h.kbLinkLimit(ctx, uid))
	case stepLinkLimit:
		limit := h.linkLimitFromText(ctx, uid, text)
		if limit < 0 {
			return c.Send(h.t(ctx, uid, i18n.KeyLinkAskLimit), h.kbLinkLimit(ctx, uid))
		}
		h.setStep(ctx, uid, stepLinkLabel, "limit", strconv.Itoa(limit))
		return c.Send(h.t(ctx, uid, i18n.KeyLinkAskLabel),
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	case stepLinkLabel:
		label := text
		if label == "0" {
			label = ""
		}
		limit, _ := strconv.Atoi(st.Data["limit"])
		return h.adminLinkCreate(ctx, c, st.Data["type"], limit, label)

	// ══ پلن ═══════════════════════════════════════════════
	case stepPlanTmpl:
		return h.adminPlanStepName(ctx, c, text)
	case stepPlanName:
		h.setStep(ctx, uid, stepPlanDays, "name", text)
		return c.Send(h.t(ctx, uid, i18n.KeyPlanAskDays),
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	case stepPlanDays:
		days, err := strconv.Atoi(text)
		if err != nil || days < 0 {
			return c.Send(h.t(ctx, uid, i18n.KeyPlanInvalidNumber), h.kbBackCancel(ctx, uid))
		}
		h.setStep(ctx, uid, stepPlanPrice, "days", text)
		return c.Send(h.t(ctx, uid, i18n.KeyPlanAskPrice),
			tele.ModeHTML, h.kbBackCancel(ctx, uid))
	case stepPlanPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil || price < 0 {
			return c.Send(h.t(ctx, uid, i18n.KeyPlanInvalidNumber), h.kbBackCancel(ctx, uid))
		}
		days, _ := strconv.Atoi(st.Data["days"])
		return h.adminPlanAdd(ctx, c, st.Data["tmpl_id"], st.Data["name"], days, price)
	case stepPlanLimits:
		return h.adminPlanSetLimits(ctx, c, st.Data["plan_id"], text)

	// ══ کاربر ═════════════════════════════════════════════
	case stepUserAction:
		return h.adminUserHandleAction(ctx, c, st.Data["user_id"], text)

	// ══ wizard ════════════════════════════════════════════
	case stepWizardToken:
		return h.wizardFinish(ctx, c, "", text)

	// ══ زبان ══════════════════════════════════════════════
	case stepLangSelect:
		h.clearState(ctx, uid)
		switch text {
		case "🇮🇷 فارسی":
			h.tr.SetLang(ctx, uid, i18n.FA)
		case "🇬🇧 English":
			h.tr.SetLang(ctx, uid, i18n.EN)
		}
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyLangChanged))
	}

	return nil
}
