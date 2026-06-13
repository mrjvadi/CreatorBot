// Package tgbot رابط تلگرامی member-bot را پیاده‌سازی می‌کند.
// owner ها از طریق این ربات قفل‌های ممبرشیپ خود را مدیریت می‌کنند.
//
// فایل‌ها:
//   handler.go  ← Handler، Register، /start، routing
//   owner.go    ← مدیریت قفل، check bot، موجودی
//   admin.go    ← پنل ادمین
//   state.go    ← state machine
//   keyboards.go← keyboard ها
package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/member-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Handler struct {
	sender     ports.BotSender
	store      *store.Store
	cache      ports.Cache
	log        ports.Logger
	ownerID    int64
	encryptKey string
}

func NewHandler(
	sender ports.BotSender,
	st *store.Store,
	cache ports.Cache,
	log ports.Logger,
	ownerID int64,
	encryptKey string,
) *Handler {
	return &Handler{
		sender: sender, store: st, cache: cache,
		log: log, ownerID: ownerID, encryptKey: encryptKey,
	}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start",    h.onStart)
	b.Handle("/help",     h.onHelp)
	b.Handle("/mylocks",  h.onMyLocks)
	b.Handle("/newlock",  h.onNewLock)
	b.Handle("/mybots",   h.onMyBots)
	b.Handle("/addbot",   h.onAddBot)
	b.Handle("/balance",  h.onBalance)
	b.Handle("/admin",    h.onAdmin)
	b.Handle("/cancel",   h.onCancel)

	b.Handle(tele.OnText,            h.onText)
	b.Handle(tele.OnChannelPost,     h.onChannelPost) // forward از کانال
	b.Handle(tele.OnCallback,        h.onCallback)
}

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()

	owner, err := h.store.FindOwnerByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("❌ خطا. دوباره امتحان کنید.")
	}
	if owner == nil {
		owner, err = h.createOwner(ctx, c)
		if err != nil {
			return c.Send("❌ خطا در ثبت‌نام.")
		}
	}
	if owner.IsBlocked {
		return c.Send("⛔️ دسترسی شما محدود شده است.")
	}

	if h.isAdmin(c) {
		return c.Send("👑 پنل ادمین:", kbAdminMain())
	}

	locks, _ := h.store.FindLocksByOwnerID(ctx, owner.ID)
	return c.Send(
		"🔒 <b>Member Lock Bot</b>\n\n"+
			"با این ربات می‌توانید قفل ممبرشیپ برای کانال‌های خود بسازید.\n\n"+
			"🔒 قفل‌های فعال: <b>"+countStr(locks)+"</b>",
		tele.ModeHTML, kbMain(),
	)
}

func (h *Handler) onHelp(c tele.Context) error {
	return c.Send(
		"<b>❓ راهنما</b>\n\n"+
			"<b>🔒 قفل جدید:</b>\n"+
			"یک پیام از کانال مورد نظر را forward کنید.\n"+
			"سپس مدت و قیمت را وارد کنید.\n\n"+
			"<b>🤖 Check Bot:</b>\n"+
			"برای بررسی ممبرشیپ، باید bot های خود را اضافه کنید.\n"+
			"bot ها باید عضو کانال‌های قفل شده باشند.\n\n"+
			"<b>💰 درآمد:</b>\n"+
			"به ازای هر بررسی ممبرشیپ که از قفل شما انجام می‌شود،\n"+
			"مبلغی به موجودی شما اضافه می‌شود.",
		tele.ModeHTML, kbMain(),
	)
}

func (h *Handler) onCancel(c tele.Context) error {
	h.clearState(context.Background(), c.Sender().ID)
	return c.Send("لغو شد.", kbMain())
}

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	st := h.getState(ctx, uid)
	if st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	switch text {
	case btnMyLocks:  return h.onMyLocks(c)
	case btnNewLock:  return h.onNewLock(c)
	case btnMyBots:   return h.onMyBots(c)
	case btnAddBot:   return h.onAddBot(c)
	case btnBalance:  return h.onBalance(c)
	case btnHelp:     return h.onHelp(c)
	case btnCancel, btnBack:
		h.clearState(ctx, uid)
		return c.Send("لغو شد.", kbMain())
	}

	if h.isAdmin(c) {
		return h.handleAdminText(ctx, c, text)
	}
	return nil
}

func (h *Handler) onChannelPost(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	st := h.getState(ctx, uid)
	if st.Step == stepLockChannel {
		return h.handleChannelForward(ctx, c, st)
	}
	return nil
}

func (h *Handler) onCallback(c tele.Context) error {
	ctx := context.Background()
	data := c.Callback().Data
	if len(data) > 0 && data[0] == '\f' {
		data = data[1:]
	}
	defer c.Respond()

	parts := strings.SplitN(data, ":", 2)
	switch parts[0] {
	case "lock_pause":
		if len(parts) == 2 {
			return h.pauseLock(ctx, c, parts[1])
		}
	case "lock_delete":
		if len(parts) == 2 {
			return h.deleteLock(ctx, c, parts[1])
		}
	case "approve_pay":
		if len(parts) == 2 {
			return h.approvePayment(ctx, c, parts[1])
		}
	case "cancel":
		h.clearState(ctx, c.Sender().ID)
		return c.Edit("لغو شد.")
	}
	return nil
}

func (h *Handler) handleStep(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	if text == btnCancel || text == btnBack {
		h.clearState(ctx, uid)
		return c.Send("لغو شد.", kbMain())
	}
	switch st.Step {
	case stepLockDuration:
		return h.handleLockDuration(ctx, c, st, text)
	case stepLockPrice:
		return h.handleLockPrice(ctx, c, st, text)
	case stepAddBot:
		return h.handleBotToken(ctx, c, st, text)
	case stepAdminBroadcast:
		return h.doBroadcast(ctx, c, text)
	}
	return nil
}

func (h *Handler) isAdmin(c tele.Context) bool {
	return c.Sender().ID == h.ownerID
}

func countStr(v any) string {
	switch val := v.(type) {
	case int:
		return strings.Repeat("", 0) + string(rune('0'+val))
	}
	return "0"
}
