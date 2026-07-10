// Package admin - admin_broadcast.go
// دو حالتِ باقی‌مانده‌ی broadcast که قبلاً stub بودند: فوروارد همگانی (هر نوع
// پیام) و ارسال فیلترشده (بر اساسِ پلن). ارسالِ متنیِ ساده در admin_menu.go است.
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

// ── فوروارد همگانی ───────────────────────────────────────────

// BroadcastForwardStart منتظرِ پیامی می‌ماند که ادمین می‌فرستد یا فوروارد
// می‌کند — هر نوعی (متن/عکس/ویدیو/فایل...). گرفتنِ خودِ پیام در
// BroadcastForwardCapture انجام می‌شود که هم از onText (پیامِ متنی) و هم از
// یک هندلرِ tele.OnMedia در bot.go صدا زده می‌شود.
func (h *Admin) BroadcastForwardStart(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()
	h.SetStep(ctx, uid, state.StepBroadcastForwardWait)
	return c.Send(h.T(ctx, uid, i18n.KeyBroadcastForwardAsk), tele.ModeHTML, h.KbCancel(ctx, uid))
}

// BroadcastForwardCapture ارجاعِ سبکِ پیام (chat_id + message_id) را نگه
// می‌دارد — نه محتوای کامل را — چون tele.StoredMessage همین دو مقدار را
// برای Forward بعدی لازم دارد و در Redis (به‌صورت رشته) به‌راحتی جا می‌شود.
func (h *Admin) BroadcastForwardCapture(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	msg := c.Message()
	if msg == nil {
		return nil
	}

	h.SetStep(ctx, uid, state.StepBroadcastForwardConfirm,
		"fwd_chat_id", strconv.FormatInt(msg.Chat.ID, 10),
		"fwd_msg_id", strconv.Itoa(msg.ID),
	)

	users, _ := h.Store.ListUsers(ctx)
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.T(ctx, uid, i18n.KeyBroadcastConfirm), "bc_fwd_confirm")),
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyCancel), "cancel")),
	)
	return c.Send(
		fmt.Sprintf(h.T(ctx, uid, i18n.KeyBroadcastForwardPreview), len(users)),
		tele.ModeHTML, kb,
	)
}

// BroadcastForwardConfirm فوروارد را واقعاً برای همه‌ی کاربران اجرا می‌کند.
func (h *Admin) BroadcastForwardConfirm(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()
	st := h.GetState(ctx, uid)
	chatIDStr, msgIDStr := st.Data["fwd_chat_id"], st.Data["fwd_msg_id"]
	h.ClearState(ctx, uid)

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || chatID == 0 || msgIDStr == "" {
		return c.Edit(h.T(ctx, uid, i18n.KeyError))
	}

	users, err := h.Store.ListUsers(ctx)
	if err != nil || len(users) == 0 {
		return c.Edit(h.T(ctx, uid, i18n.KeyBroadcastEmptyAudience))
	}

	stored := tele.StoredMessage{MessageID: msgIDStr, ChatID: chatID}
	go h.RunForwardBroadcast(uid, stored, users)
	return c.Edit(h.T(ctx, uid, i18n.KeyBroadcastStarted), tele.ModeHTML)
}

// RunForwardBroadcast همان محدودیتِ نرخِ RunBroadcast (broadcastRate) را
// رعایت می‌کند تا به rate limitِ تلگرام نخوریم.
func (h *Admin) RunForwardBroadcast(adminUID int64, msg tele.StoredMessage, users []models.User) {
	ticker := time.NewTicker(time.Second / broadcastRate)
	defer ticker.Stop()

	sent, failed := 0, 0
	for _, u := range users {
		if u.TelegramID == adminUID {
			continue
		}
		<-ticker.C
		if _, err := h.Bot.Forward(tele.ChatID(u.TelegramID), msg); err != nil {
			failed++
		} else {
			sent++
		}
	}

	h.Log.Info("forward broadcast finished", h.F("sent", sent), h.F("failed", failed))
	_, _ = h.Bot.Send(
		tele.ChatID(adminUID),
		h.T(context.Background(), adminUID, i18n.KeyBroadcastDone, sent, failed),
		tele.ModeHTML,
	)
}

// ── ارسال فیلترشده ───────────────────────────────────────────

// BroadcastFilteredMenu مخاطبِ هدف را می‌پرسد: همه / بدون پلن فعال / کاربرانِ
// یک پلنِ خاص. بعد از انتخاب، همان مسیرِ متنیِ معمولی (BroadcastExecute) ادامه
// پیدا می‌کند — فقط فیلتر در state ذخیره می‌شود.
func (h *Admin) BroadcastFilteredMenu(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	_ = c.Respond()

	plans, _ := h.Store.ListActivePlans(ctx)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	rows = append(rows, kb.Row(kb.Data(h.T(ctx, uid, i18n.KeyBcFilterAll), "bc_filter:all")))
	rows = append(rows, kb.Row(kb.Data(h.T(ctx, uid, i18n.KeyBcFilterNoPlan), "bc_filter:no_plan")))
	for _, p := range plans {
		label := fmt.Sprintf(h.T(ctx, uid, i18n.KeyBcFilterPlan), p.Name)
		rows = append(rows, kb.Row(kb.Data(label, "bc_filter:plan:"+p.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnCancel), "cancel")))
	kb.Inline(rows...)
	return c.Send(h.T(ctx, uid, i18n.KeyBcFilterTitle), tele.ModeHTML, kb)
}

// BroadcastFilterSelect فیلترِ انتخابی را ذخیره و متنِ پیام را می‌پرسد.
func (h *Admin) BroadcastFilterSelect(ctx context.Context, c tele.Context, filter string) error {
	uid := c.Sender().ID
	_ = c.Respond()
	h.SetStep(ctx, uid, state.StepBroadcastText, "bc_filter", filter)
	return c.Send(h.T(ctx, uid, i18n.KeyBroadcastAskText), tele.ModeHTML, h.KbCancel(ctx, uid))
}

// resolveBroadcastAudience رشته‌ی فیلترِ ذخیره‌شده در state را به لیستِ
// کاربرانِ واقعی تبدیل می‌کند. filter خالی = همه (سازگار با مسیرِ قدیمیِ
// ارسالِ متنیِ بدون فیلتر که هرگز bc_filter را ست نمی‌کند).
func (h *Admin) resolveBroadcastAudience(ctx context.Context, filter string) ([]models.User, error) {
	switch {
	case filter == "" || filter == "all":
		return h.Store.ListUsers(ctx)
	case filter == "no_plan":
		return h.Store.ListUsersWithoutActivePlan(ctx)
	case strings.HasPrefix(filter, "plan:"):
		return h.Store.ListUsersByActivePlan(ctx, strings.TrimPrefix(filter, "plan:"))
	default:
		return h.Store.ListUsers(ctx)
	}
}
