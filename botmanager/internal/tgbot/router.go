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
	data := c.Callback().Data
	defer c.Respond()

	if len(data) > 0 && data[0] == '\f' {
		data = data[1:]
	}

	parts := splitN(data, ":", 2)
	switch parts[0] {
	case "buy_plan":
		if len(parts) == 2 {
			return h.executePlanPurchase(ctx, c, parts[1])
		}
	case "select_plan":
		if len(parts) == 2 {
			return h.userSelectPlan(ctx, c, parts[1])
		}
	case "start_free":
		freePlan, _ := h.store.GetFreePlan(ctx)
		if freePlan != nil {
			u, _ := h.getOrCreateUser(ctx, c)
			if u != nil {
				return h.activateFreePlanInline(ctx, c, u, freePlan)
			}
		}
		return c.Edit(h.t(ctx, c.Sender().ID, i18n.KeyNoFreePlan))
	case "my_bots":
		c.Respond()
		return h.userBotsList(ctx, c)
	case "how_to_build":
		c.Respond()
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(h.t(ctx, c.Sender().ID, i18n.KeyHowToBuildDone), "cancel")))
		return c.Edit(h.t(ctx, c.Sender().ID, i18n.KeyHowToBuild), tele.ModeHTML, kb)
	case "check_plan":
		// check_plan:<plan_id>:<invoice_code>
		if len(data) > 10 {
			sub := strings.SplitN(data[len("check_plan:"):], ":", 2)
			if len(sub) == 2 {
				return h.checkPlanAfterDeposit(ctx, c, sub[0], sub[1])
			}
		}
	case "free_plan":
		if len(parts) == 2 {
			u, _ := h.getOrCreateUser(ctx, c)
			plan, _ := h.store.FindPlan(ctx, parts[1])
			if u != nil && plan != nil {
				return h.activateFreePlanInline(ctx, c, u, plan)
			}
		}
	case "show_plans":
		return h.userPlans(ctx, c)
	case "cancel":
		return h.sendMain(c, h.t(ctx, c.Sender().ID, i18n.KeyCancelled))
	}
	return nil
}

func splitN(s, sep string, n int) []string {
	result := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := 0
		for j := range s {
			if s[j:j+len(sep)] == sep {
				idx = j
				break
			}
		}
		if idx == 0 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := c.Text()

	// ── state فعال ────────────────────────────────────────
	st := h.getState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	// ── wizard pending ────────────────────────────────────
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

	// ── cancel/back همیشه ─────────────────────────────────
	if h.isCancel(ctx, uid, text) {
		return h.onCancel(c)
	}

	// ── انتخاب زبان ──────────────────────────────────────
	switch text {
	case "🇮🇷 فارسی":
		h.tr.SetLang(ctx, uid, i18n.FA)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyLangChanged))
	case "🇬🇧 English":
		h.tr.SetLang(ctx, uid, i18n.EN)
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyLangChanged))
	}

	// ── routing ادمین ─────────────────────────────────────
	if h.isAdmin(c) {
		switch text {
		case h.btn(ctx, uid, i18n.KeyMenuBots):
			return h.adminBotsList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuLinks):
			return h.adminLinksList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuServers):
			return h.adminServersList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuTemplates):
			return h.adminTemplatesList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuPlans):
			return h.adminPlansList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuUsers):
			return h.adminUsersList(ctx, c)
		case h.btn(ctx, uid, i18n.KeyMenuStats):
			return h.adminStats(ctx, c)
		}
	}

	// ── routing کاربر ─────────────────────────────────────
	switch text {
	case h.btn(ctx, uid, i18n.KeyMenuMyBots):
		return h.userBotsList(ctx, c)
	case h.btn(ctx, uid, i18n.KeyMenuHelp):
		return h.onHelp(c)
	case h.btn(ctx, uid, i18n.KeyMenuSupport):
		return h.userSupport(c)
	case "💎 خرید پلن", "💎 " + h.btn(ctx, uid, i18n.KeyMenuPlans):
		return h.userPlans(ctx, c)
	}

	return nil
}

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
			return c.Send(h.t(ctx, uid, i18n.KeyTemplateAskType),
				h.kbBotType(ctx, uid))
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
		// 0 = ابدی
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

	// ══ کاربر ═════════════════════════════════════════════
	case stepUserAction:
		return h.adminUserHandleAction(ctx, c, st.Data["user_id"], text)

	// ══ wizard ════════════════════════════════════════════
	case stepWizardToken:
		inviteToken := st.Data["invite_token"]
		h.clearState(ctx, uid)
		return h.wizardFinish(ctx, c, inviteToken, text)

	// ══ انتخاب زبان ═══════════════════════════════════════
	case stepLangSelect:
		h.clearState(ctx, uid)
		switch text {
		case "🇮🇷 فارسی":
			h.tr.SetLang(ctx, uid, i18n.FA)
			return h.sendMain(c, h.t(ctx, uid, i18n.KeyLangChanged))
		case "🇬🇧 English":
			h.tr.SetLang(ctx, uid, i18n.EN)
			return h.sendMain(c, h.t(ctx, uid, i18n.KeyLangChanged))
		}
	}

	return nil
}
