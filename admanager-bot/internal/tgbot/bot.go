// Package tgbot — هندلر اصلی admanager-bot.
//
// این ربات تک‌نقشه (ادمین‌محور) است: فقط مالک/ادمین کانال‌ها (OWNER_ID)
// به آن دسترسی دارد. الگوی کلی مشابه uploader-bot:
//   - Deps ساختار وابستگی‌ها
//   - NewHandler سازنده
//   - Handler تمام callback/text handlerها را دارد
package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/admanager-bot/internal/models"
	"github.com/mrjvadi/creatorbot/admanager-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// portsF میان‌بر ساخت Field برای لاگ ساختاریافته.
func portsF(key string, value any) ports.Field { return ports.F(key, value) }

// audit یک رویداد را در AuditLog ثبت می‌کند (بدون توقف در صورت خطا).
func (h *Handler) audit(ctx context.Context, c tele.Context, action models.AuditAction, targetType, targetID, desc string) {
	if err := h.store.LogAudit(ctx, &models.AuditLog{
		Action:        action,
		ActorID:       c.Sender().ID,
		ActorUsername: c.Sender().Username,
		TargetID:      targetID,
		TargetType:    targetType,
		Description:   desc,
	}); err != nil {
		h.log.Error("audit log", portsF("err", err))
	}
}

// Deps وابستگی‌های Handler.
type Deps struct {
	Engine  *engine.Engine
	Bot     *tele.Bot
	OwnerID int64 // TelegramID مالک/ادمین اصلی
}

// Handler تمام handlerهای ربات را نگه می‌دارد.
type Handler struct {
	bot        *tele.Bot
	store      *store.Store
	engine     *engine.Engine
	cache      ports.Cache
	instanceID string
	ownerID    int64
	log        ports.Logger
}

// NewHandler یک Handler جدید می‌سازد و routerها را ثبت می‌کند.
func NewHandler(d Deps) *Handler {
	h := &Handler{
		bot:        d.Bot,
		store:      store.New(d.Engine.Mongo, d.Engine.InstanceID, d.Engine.Cache),
		engine:     d.Engine,
		cache:      d.Engine.Cache,
		instanceID: d.Engine.InstanceID,
		ownerID:    d.OwnerID,
		log:        d.Engine.Log,
	}
	h.register()
	return h
}

// register همه‌ی handlerها را روی bot ثبت می‌کند.
func (h *Handler) register() {
	b := h.bot

	// دستورات
	b.Handle("/start", h.onStart)
	b.Handle("/help", h.onHelp)

	// متن آزاد
	b.Handle(tele.OnText, h.onText)

	// رسانه (برای محتوای تبلیغ)
	b.Handle(tele.OnPhoto, h.onMedia)
	b.Handle(tele.OnVideo, h.onMedia)
	b.Handle(tele.OnDocument, h.onMedia)

	// callback inline keyboard
	b.Handle(tele.OnCallback, h.onCallback)
}

// onMedia رسانه‌ی ارسالی را در صورت بودن در مرحله‌ی محتوای تبلیغ پردازش می‌کند.
func (h *Handler) onMedia(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	ctx := context.Background()
	st := h.getState(ctx, c.Sender().ID)
	switch st.Step {
	case stepAdMain:
		return h.handleAdMain(ctx, c, st)
	case stepAdReplies:
		return h.handleAdReplies(ctx, c, st)
	case stepAdEditMain:
		return h.handleAdEditMain(ctx, c, st)
	case stepChannelAdd:
		// پیام forward‌شده‌ی رسانه‌ای برای ثبت کانال
		return h.handleChannelAdd(ctx, c, "")
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────

// isAdmin بررسی می‌کند فرستنده ادمین/مالک است.
func (h *Handler) isAdmin(c tele.Context) bool {
	return c.Sender().ID == h.ownerID
}

// ── دستورها ───────────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	if !h.isAdmin(c) {
		return c.Send(denyText, tele.ModeHTML)
	}

	h.clearState(ctx, uid)
	h.log.Info("admin start", portsF("uid", uid))

	welcome := defWelcome
	if st, err := h.store.GetSettings(ctx); err == nil && st.WelcomeMessage != "" {
		welcome = st.WelcomeMessage
	}
	return c.Send(welcome, tele.ModeHTML, kbAdminMain())
}

func (h *Handler) onHelp(c tele.Context) error {
	if !h.isAdmin(c) {
		return c.Send(denyText, tele.ModeHTML)
	}
	help := defHelp
	if st, err := h.store.GetSettings(context.Background()); err == nil && st.HelpMessage != "" {
		help = st.HelpMessage
	}
	return c.Send(help, tele.ModeHTML, kbAdminMain())
}

// ── متن آزاد ──────────────────────────────────────────────────────

func (h *Handler) onText(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	if !h.isAdmin(c) {
		// در گفتگوی خصوصی پیام رد دسترسی؛ در گروه/کانال سکوت.
		if c.Chat() != nil && c.Chat().Type == tele.ChatPrivate {
			return c.Send(denyText, tele.ModeHTML)
		}
		return nil
	}

	// لغو در هر مرحله‌ای
	if text == btnCancel || text == btnBack {
		h.clearState(ctx, uid)
		return c.Send("لغو شد.", kbAdminMain())
	}

	// دکمه‌های منوی اصلی همیشه از هر wizard خارج می‌شوند.
	if isMainMenuButton(text) {
		h.clearState(ctx, uid)
		return h.adminOnText(ctx, c, text)
	}

	// اگر کاربر وسط یک wizard است
	if st := h.getState(ctx, uid); st.Step != stepIdle {
		return h.handleStep(ctx, c, st, text)
	}

	// routing منوی اصلی
	return h.adminOnText(ctx, c, text)
}

// adminOnText متن منوی اصلی ادمین را مسیر‌دهی می‌کند.
func (h *Handler) adminOnText(ctx context.Context, c tele.Context, text string) error {
	switch text {
	case btnNewCampaign:
		h.setStep(ctx, c.Sender().ID, stepCampaignName)
		return c.Send("➕ نام کمپین جدید را بفرستید:", kbCancelOnly())

	case btnChannels:
		return h.channelsHome(c)

	case btnCampaigns:
		return h.campaignsHome(c)

	case btnSchedule:
		return h.scheduleHome(c)

	case btnStats:
		return h.statsHome(c)

	case btnTemplates:
		return h.templatesHome(c)

	case btnSettings:
		return h.settingsHome(c)

	case btnHelp:
		return h.onHelp(c)

	default:
		return c.Send("یکی از گزینه‌های منو را انتخاب کنید 👇", kbAdminMain())
	}
}
