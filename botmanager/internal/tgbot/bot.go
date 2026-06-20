// Package tgbot handler اصلی botmanager.
//
//	bot.go          ← Handler struct، Register، /start، /cancel، /lang
//	router.go       ← onText routing + state machine
//	admin_server.go ← سرورها
//	admin_tmpl.go   ← تمپلیت‌ها
//	admin_plan.go   ← پلن‌ها
//	admin_link.go   ← لینک‌های دعوت
//	admin_bot.go    ← مدیریت ربات‌ها
//	admin_user.go   ← مدیریت کاربران
//	admin_stats.go  ← آمار
//	user_bot.go     ← ربات‌های کاربر
//	wizard.go       ← ساخت ربات از InviteLink
//	keyboards.go    ← همه keyboard ها
//	state.go        ← state machine در Redis
//	helpers.go      ← فرمت‌دهی و توابع کمکی
//	i18n/           ← سیستم چندزبانگی
package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared-core/ton"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	natsadapter "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Handler نگهدارنده همه dependency های ربات.
type Handler struct {
	store       *store.Store
	cache       ports.Cache
	docker      *sharedocker.Manager
	log         ports.Logger
	ownerID     int64
	botUsername string
	encryptKey  string
	ton         *ton.Client
	pay         *natspayclient.Client
	tr          *i18n.Translator
	nc          *natsadapter.Client
}

func NewHandler(
	bot *tele.Bot,
	st *store.Store,
	cache ports.Cache,
	docker *sharedocker.Manager,
	log ports.Logger,
	ownerID int64,
	encryptKey string,
	tonClient *ton.Client,
	payClient *natspayclient.Client,
	nc *natsadapter.Client,
) *Handler {
	return &Handler{
		store:       st,
		cache:       cache,
		docker:      docker,
		log:         log,
		ownerID:     ownerID,
		botUsername: bot.Me.Username,
		encryptKey:  encryptKey,
		ton:         tonClient,
		pay:         payClient,
		tr:          i18n.New(cache),
		nc:          nc,
	}
}

// Register همه handler ها را روی bot وصل می‌کند.
// safeHandler هر handler را در panic recovery wrap می‌کند.
func safeHandler(name string, fn tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) (retErr error) {
		defer func() {
			if r := recover(); r != nil {
				retErr = fmt.Errorf("panic in %s: %v", name, r)
			}
		}()
		return fn(c)
	}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start",  h.onStart)
	b.Handle("/cancel", h.onCancel)
	b.Handle("/help",   h.onHelp)
	b.Handle("/lang",   h.onLang)
	b.Handle(tele.OnText, h.onText)
	b.Handle(tele.OnCallback, h.onCallback)
}

// ── /start ────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	// اگه payload داشت → InviteLink wizard
	if token := c.Message().Payload; token != "" {
		return h.wizardStart(ctx, c, token)
	}

	// اولین بار → detect زبان از تلگرام
	if h.tr.GetLang(ctx, uid) == i18n.Default {
		detectedLang := i18n.DetectFromTelegram(c.Sender().LanguageCode)
		h.tr.SetLang(ctx, uid, detectedLang)
	}

	u, _ := h.getOrCreateUser(ctx, c)

	if h.isAdmin(c) {
		name := c.Sender().FirstName
		if u != nil && u.Role == models.RoleOwner {
			name += " 👑"
		} else {
			name += " 🛡"
		}
		return c.Send(
			h.t(ctx, uid, i18n.KeyWelcomeAdmin, name),
			h.kbAdmin(ctx, uid),
		)
	}

	return c.Send(
		h.t(ctx, uid, i18n.KeyWelcomeUser, c.Sender().FirstName),
		h.kbUser(ctx, uid),
	)
}

// ── /cancel ───────────────────────────────────────────────

func (h *Handler) onCancel(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	h.clearState(ctx, uid)
	h.clearWizardPending(ctx, uid)
	return h.sendMain(c, h.t(ctx, uid, i18n.KeyCancelled))
}

// ── /help ─────────────────────────────────────────────────

func (h *Handler) onHelp(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	if h.isAdmin(c) {
		return c.Send(adminHelpText(ctx, h, uid), tele.ModeHTML, h.kbAdmin(ctx, uid))
	}
	return c.Send(h.t(ctx, uid, i18n.KeyHelpText), tele.ModeHTML, h.kbUser(ctx, uid))
}

// ── /lang ─────────────────────────────────────────────────

func (h *Handler) onLang(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	h.setStep(ctx, uid, stepLangSelect)
	return c.Send(h.t(ctx, uid, i18n.KeySelectLang), kbLanguage())
}

func adminHelpText(ctx context.Context, h *Handler, uid int64) string {
	return h.t(ctx, uid, i18n.KeyHelpText)
}

// ── i18n shortcuts ────────────────────────────────────────

// t ترجمه متن با args.
func (h *Handler) t(ctx context.Context, uid int64, key i18n.Key, args ...any) string {
	return h.tr.T(ctx, uid, key, args...)
}

// btn دکمه ترجمه‌شده.
func (h *Handler) btn(ctx context.Context, uid int64, key i18n.Key) string {
	return h.tr.Btn(ctx, uid, key)
}

// ── helpers ───────────────────────────────────────────────

func (h *Handler) sendMain(c tele.Context, text string) error {
	ctx := context.Background()
	uid := c.Sender().ID
	if h.isAdmin(c) {
		return c.Send(text, h.kbAdmin(ctx, uid))
	}
	return c.Send(text, h.kbUser(ctx, uid))
}

// botTypeFromText دکمه نوع ربات را به string تبدیل می‌کند.
func (h *Handler) botTypeFromText(ctx context.Context, uid int64, text string) string {
	types := map[i18n.Key]string{
		i18n.KeyBotTypeUploader: "uploader",
		i18n.KeyBotTypeVPN:      "vpn",
		i18n.KeyBotTypeArchive:  "archive",
		i18n.KeyBotTypeMember:   "member",
	}
	for k, v := range types {
		if text == h.btn(ctx, uid, k) {
			return v
		}
	}
	return ""
}

// linkLimitFromText دکمه محدودیت را به عدد تبدیل می‌کند.
func (h *Handler) linkLimitFromText(ctx context.Context, uid int64, text string) int {
	limits := map[i18n.Key]int{
		i18n.KeyBtnLimit1:  1,
		i18n.KeyBtnLimit3:  3,
		i18n.KeyBtnLimit5:  5,
		i18n.KeyBtnLimit10: 10,
		i18n.KeyBtnLimitNo: 0,
	}
	for k, v := range limits {
		if text == h.btn(ctx, uid, k) {
			return v
		}
	}
	return -1
}

// isCancel بررسی می‌کند متن دکمه cancel یا back است.
func (h *Handler) isCancel(ctx context.Context, uid int64, text string) bool {
	return text == h.btn(ctx, uid, i18n.KeyBtnCancel) ||
		text == h.btn(ctx, uid, i18n.KeyBtnBack) ||
		text == "/cancel"
}

// suppress unused
var _ = strings.Title
