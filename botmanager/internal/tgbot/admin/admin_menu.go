package admin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

// ── افزودن اعتبار ────────────────────────────────────────────

func (h *Admin) AdminCreditStart(ctx context.Context, c tele.Context, targetTelegramID int64) error {
	uid := c.Sender().ID
	h.SetStep(ctx, uid, state.StepAdminCreditAmount,
		"target_tid", strconv.FormatInt(targetTelegramID, 10))
	return c.Send(h.T(ctx, uid, i18n.KeyAdminCreditAsk, targetTelegramID), tele.ModeHTML, h.KbCancel(ctx, uid))
}

func (h *Admin) AdminCreditExecute(ctx context.Context, c tele.Context, targetTidStr, amountStr string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	amountStr = strings.TrimSpace(amountStr)
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyAdminCreditInvalid), h.KbAdmin(ctx, uid))
	}

	targetTid, err := strconv.ParseInt(targetTidStr, 10, 64)
	if err != nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyError))
	}

	if h.Pay != nil {
		if credErr := h.Pay.Credit(ctx, targetTid, amount, "admin_credit", ""); credErr != nil {
			h.Log.Error("admin credit failed", h.F("err", credErr), h.F("target", targetTid))
			return c.Send(h.T(ctx, uid, i18n.KeyAdminCreditError), h.KbAdmin(ctx, uid))
		}
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyAdminCreditDone, amount, targetTid),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}

// ── ارسال همگانی ─────────────────────────────────────────────

func (h *Admin) AdminBroadcastMenu(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBcText), "bc_text")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBcForward), "bc_forward")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnBcFiltered), "bc_filtered")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBack), "back_main")),
	)
	return c.Send(h.T(ctx, uid, i18n.KeyBroadcastMenu), tele.ModeHTML, kb)
}

func (h *Admin) BroadcastStartText(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()
	h.SetStep(ctx, uid, state.StepBroadcastText)
	return c.Send(h.T(ctx, uid, i18n.KeyBroadcastAskText), tele.ModeHTML, h.KbCancel(ctx, uid))
}

func (h *Admin) BroadcastExecute(ctx context.Context, c tele.Context, text string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	users, err := h.Store.ListUsers(ctx)
	if err != nil || len(users) == 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyError), h.KbAdmin(ctx, uid))
	}

	h.SetStep(ctx, uid, state.StepBroadcastText, "msg", text)
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.T(ctx, uid, i18n.KeyBroadcastConfirm), "bc_confirm")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyCancel), "cancel")),
	)
	return c.Send(
		fmt.Sprintf(h.T(ctx, uid, i18n.KeyBroadcastPreview), text, len(users)),
		tele.ModeHTML, kb,
	)
}

func (h *Admin) BroadcastConfirm(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()
	st := h.GetState(ctx, uid)
	msg := st.Data["msg"]
	h.ClearState(ctx, uid)

	if msg == "" {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	users, _ := h.Store.ListUsers(ctx)
	go h.RunBroadcast(uid, msg, users)
	return c.Edit(h.T(ctx, uid, i18n.KeyBroadcastStarted), tele.ModeHTML)
}

const broadcastRate = 25

func (h *Admin) RunBroadcast(adminUID int64, msg string, users []models.User) {
	ticker := time.NewTicker(time.Second / broadcastRate)
	defer ticker.Stop()

	sent, failed := 0, 0
	for _, u := range users {
		if u.TelegramID == adminUID {
			continue
		}
		<-ticker.C
		if _, err := h.Bot.Send(tele.ChatID(u.TelegramID), msg, tele.ModeHTML); err != nil {
			failed++
		} else {
			sent++
		}
	}

	h.Log.Info("broadcast finished", h.F("sent", sent), h.F("failed", failed))
	_, _ = h.Bot.Send(
		tele.ChatID(adminUID),
		h.T(context.Background(), adminUID, i18n.KeyBroadcastDone, sent, failed),
		tele.ModeHTML,
	)
}

// ── سیستم ────────────────────────────────────────────────────

func (h *Admin) AdminSystemMenu(ctx context.Context, c tele.Context) error {
	return h.adminSysView(ctx, c, false)
}

func (h *Admin) AdminSysInfo(ctx context.Context, c tele.Context) error {
	_ = c.Respond()
	return h.adminSysView(ctx, c, true)
}

func (h *Admin) adminSysView(ctx context.Context, c tele.Context, edit bool) error {
	uid := c.Sender().ID
	plans, _ := h.Store.ListPlans(ctx)
	servers, _ := h.Store.ListServers(ctx)
	templates, _ := h.Store.ListTemplates(ctx)
	users, _ := h.Store.ListUsers(ctx)

	online := 0
	for _, s := range servers {
		if s.IsOnline {
			online++
		}
	}

	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyMenuPlans), "admin_sys_plans"), kb.Data(h.Btn(ctx, uid, i18n.KeyMenuServers), "admin_sys_servers")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyMenuTemplates), "admin_sys_templates")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBack), "back_main")),
	)
	msg := fmt.Sprintf(h.T(ctx, uid, i18n.KeyAdminSysInfo),
		len(plans), len(servers), online, len(templates), len(users))
	if edit {
		return c.Edit(msg, tele.ModeHTML, kb)
	}
	return c.Send(msg, tele.ModeHTML, kb)
}
