// Package tgbot ربات تلگرام botpay — رابط کاربری مدیریت کیف پول TON.
//
// همه‌ی متن‌ها و دکمه‌ها از طریق بسته‌ی i18n ترجمه می‌شوند و هیچ رشته‌ی
// نمایشی به‌صورت hard-code در این بسته وجود ندارد. وضعیت گفتگوهای
// چندمرحله‌ای در stateStore (thread-safe) نگه‌داری می‌شود.
package tgbot

import (
	"context"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botpay/internal/i18n"
	"github.com/mrjvadi/creatorbot/botpay/internal/store"
	"github.com/mrjvadi/creatorbot/botpay/internal/wallet"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Handler تمام منطق رابط ربات را در خود دارد.
type Handler struct {
	wallet      *wallet.Service
	st          *store.Store
	ownerID     int64
	defaultLang i18n.Lang
	log         ports.Logger
	bot         *tele.Bot // برای ارسال push notification
	states      *stateStore
}

// New یک Handler جدید می‌سازد. defaultLang کد زبان پیش‌فرض است (مثلاً "fa").
func New(w *wallet.Service, st *store.Store, ownerID int64, defaultLang string, log ports.Logger) *Handler {
	i18n.SetDefault(defaultLang)
	return &Handler{
		wallet:      w,
		st:          st,
		ownerID:     ownerID,
		defaultLang: i18n.Normalize(defaultLang),
		log:         log,
		states:      newStateStore(),
	}
}

// SetBot ربات را تنظیم و notifier را فعال می‌کند.
func (h *Handler) SetBot(b *tele.Bot) {
	h.bot = b
	h.wallet.SetNotifier(h)
}

// SendHTML پیاده‌سازی wallet.Notifier — ارسال پیام HTML به یک کاربر.
func (h *Handler) SendHTML(ctx context.Context, telegramID int64, html string) error {
	if h.bot == nil {
		return nil
	}
	_, err := h.bot.Send(&tele.Chat{ID: telegramID}, html, tele.ModeHTML)
	return err
}

// Register هندلرهای ربات را ثبت می‌کند.
func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", h.onStart)
	b.Handle("/wallet", h.onWallet)
	b.Handle("/deposit", h.onDeposit)
	b.Handle("/withdraw", h.onWithdraw)
	b.Handle("/history", h.onHistory)
	b.Handle("/help", h.onHelp)
	b.Handle("/language", h.onLanguage)

	// ادمین
	b.Handle("/admin", h.onAdmin)
	b.Handle("/addcredit", h.onAddCredit)
	b.Handle("/withdraws", h.onAdminWithdraws)
	b.Handle("/approve", h.onApproveWithdraw)
	b.Handle("/reject", h.onRejectWithdraw)

	b.Handle(tele.OnText, h.onText)
	b.Handle(tele.OnCallback, h.onCallback)
}

// isAdmin بررسی می‌کند فرستنده، مالک ربات است.
func (h *Handler) isAdmin(c tele.Context) bool {
	return c.Sender() != nil && c.Sender().ID == h.ownerID
}

// langOf زبان فعال کاربر را تعیین می‌کند: ابتدا زبان ذخیره‌شده در DB، سپس
// language_code تلگرام به‌عنوان حدس اولیه، و در نهایت زبان پیش‌فرض.
func (h *Handler) langOf(ctx context.Context, c tele.Context) i18n.Lang {
	if c.Sender() == nil {
		return h.defaultLang
	}
	if w, err := h.st.GetWallet(ctx, c.Sender().ID); err == nil && w != nil && w.Lang != "" {
		return i18n.Normalize(w.Lang)
	}
	if lc := c.Sender().LanguageCode; lc != "" {
		if l, ok := i18n.Parse(lc); ok {
			return l
		}
	}
	return h.defaultLang
}
