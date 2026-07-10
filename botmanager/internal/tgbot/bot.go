// Package tgbot لایه‌ی مسیریابی و wiring ربات.
// منطق مشترک در internal/tgbot/core، و (در گام‌های بعد) منطق admin/user در
// package‌های جدا قرار می‌گیرند.
package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/admin"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/core"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/user"
	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/licenseclient"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/natspayclient"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared-core/ton"
	natsadapter "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Handler هندلرِ اصلی (routing).
//   - Deps: وابستگی‌ها و helperهای مشترک (core)
//   - Admin/User: منطقِ هر بخش در package جدا؛ متدهایشان promote می‌شوند تا
//     router بتواند مستقیم h.AdminX / h.UserX را صدا بزند.
type Handler struct {
	*core.Deps
	*admin.Admin
	*user.User
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
	licenseClient *licenseclient.Client,
) *Handler {
	deps := core.New(bot, st, cache, docker, log, ownerID, encryptKey, tonClient, payClient, nc, licenseClient)
	return &Handler{
		Deps:  deps,
		Admin: admin.New(deps),
		User:  user.New(deps),
	}
}

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
	b.Handle("/start", safeHandler("start", h.onStart))
	b.Handle("/cancel", safeHandler("cancel", h.onCancel))
	b.Handle("/help", safeHandler("help", h.onHelp))
	b.Handle("/lang", safeHandler("lang", h.onLang))
	b.Handle("/admin", safeHandler("admin", h.onEnterAdmin))
	b.Handle(tele.OnText, safeHandler("text", h.onText))
	b.Handle(tele.OnCallback, safeHandler("callback", h.onCallback))
	// فقط برای «فوروارد همگانی»: وقتی ادمین در انتظارِ ارسال/فورواردِ پیامِ
	// غیرمتنی (عکس/ویدیو/فایل/...) است. پیامِ متنی از همان مسیرِ OnText/
	// handleStep می‌رود، چون telebot پیام‌های متنیِ فوروارد‌شده را هم OnText
	// می‌بیند.
	b.Handle(tele.OnMedia, safeHandler("broadcast_forward_media", h.onBroadcastForwardMedia))
}

// onBroadcastForwardMedia فقط وقتی کاری می‌کند که فرستنده ادمین باشد و در
// حالتِ انتظارِ «فوروارد همگانی» باشد؛ در غیر این صورت بی‌صدا نادیده می‌گیرد
// (نه اینکه پیام‌های عادیِ رسانه‌ای کاربران را اشتباهی پردازش کند).
func (h *Handler) onBroadcastForwardMedia(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	if !h.IsAdmin(c) {
		return nil
	}
	st := h.GetState(ctx, uid)
	if st.Step != stepBroadcastForwardWait {
		return nil
	}
	return h.BroadcastForwardCapture(ctx, c)
}

// ── /start ────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID

	// اولین بار → detect زبان از تلگرام
	if h.Tr.GetLang(ctx, uid) == i18n.Default {
		detectedLang := i18n.DetectFromTelegram(c.Sender().LanguageCode)
		h.Tr.SetLang(ctx, uid, detectedLang)
	}

	u, _ := h.GetOrCreateUser(ctx, c)

	if h.IsInAdminMode(c) {
		name := c.Sender().FirstName
		if u != nil && u.Role == models.RoleOwner {
			name += " 👑"
		} else {
			name += " 🛡"
		}
		return c.Send(
			h.T(ctx, uid, i18n.KeyWelcomeAdmin, name),
			h.KbAdmin(ctx, uid),
		)
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyWelcomeUser, c.Sender().FirstName),
		h.KbUser(ctx, uid),
	)
}

// ── /cancel ───────────────────────────────────────────────

func (h *Handler) onCancel(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	return h.SendMain(c, h.T(ctx, uid, i18n.KeyCancelled))
}

// ── /help ─────────────────────────────────────────────────

func (h *Handler) onHelp(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	if h.IsInAdminMode(c) {
		return c.Send(h.T(ctx, uid, i18n.KeyHelpText), tele.ModeHTML, h.KbAdmin(ctx, uid))
	}
	return c.Send(h.T(ctx, uid, i18n.KeyHelpText), tele.ModeHTML, h.KbUser(ctx, uid))
}

// ── /lang ─────────────────────────────────────────────────

func (h *Handler) onLang(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	h.SetStep(ctx, uid, stepLangSelect)
	return c.Send(h.T(ctx, uid, i18n.KeySelectLang), core.KbLanguage())
}

// ── /admin — ورود به پنل ادمین ──────────────────────────

func (h *Handler) onEnterAdmin(c tele.Context) error {
	ctx := context.Background()
	uid := c.Sender().ID
	if !h.IsAdmin(c) {
		return c.Send(h.T(ctx, uid, i18n.KeyNoAccess))
	}
	h.SetAdminMode(ctx, uid, true)
	return c.Send(
		h.T(ctx, uid, i18n.KeyWelcomeAdmin, c.Sender().FirstName+" 👑"),
		h.KbAdmin(ctx, uid),
	)
}
