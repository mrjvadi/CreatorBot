// Package tgbot ربات آپلودر فایل.
// قابلیت‌ها:
//   - ادمین: آپلود فایل، ساخت کد، مدیریت آلبوم، تنظیمات، آمار
//   - کاربر: دریافت فایل با کد، بررسی عضویت اجباری
//
// فایل‌ها:
//   bot.go      ← Handler، Register، /start
//   admin.go    ← همه قابلیت‌های ادمین
//   user.go     ← قابلیت‌های کاربر
//   state.go    ← state machine در Redis
//   keyboards.go← keyboard ها
//   helpers.go  ← توابع کمکی
package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/docstore"
	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Handler handler اصلی uploader-bot.
type Handler struct {
	eng       *engine.Engine
	sender    ports.BotSender
	ownerID   int64
	channelID int64 // اگه صفر باشه force_join غیرفعاله

	// stores
	codes *docstore.CodeStore
	files *docstore.FileStore
	users *docstore.BotUserStore
}

// Deps وابستگی‌های Handler.
type Deps struct {
	Engine    *engine.Engine
	Sender    ports.BotSender
	OwnerID   int64
	ChannelID int64
}

func New(
	eng *engine.Engine,
	sender ports.BotSender,
	ownerID int64,
	channelID int64,
) *Handler {
	return &Handler{
		eng:       eng,
		sender:    sender,
		ownerID:   ownerID,
		channelID: channelID,
		codes:     docstore.NewCodeStore(eng.Mongo, eng.InstanceID),
		files:     docstore.NewFileStore(eng.Mongo, eng.InstanceID),
		users:     docstore.NewBotUserStore(eng.Mongo, eng.InstanceID),
	}
}

// NewHandler با Deps struct می‌سازد.
func NewHandler(d Deps) *Handler {
	return New(d.Engine, d.Sender, d.OwnerID, d.ChannelID)
}

// Register همه handler ها را روی bot وصل می‌کند.
func Register(b *tele.Bot, h *Handler) {
	// ── کاربر ─────────────────────────────────────────────
	b.Handle("/start",  h.onStart)
	b.Handle("/help",   h.onHelp)
	b.Handle(tele.OnText, h.onText)

	// ── ادمین ─────────────────────────────────────────────
	b.Handle("/panel",    h.adminPanel)
	b.Handle("/newcode",  h.adminNewCode)
	b.Handle("/codes",    h.adminCodeList)
	b.Handle("/delcode",  h.adminDelCode)
	b.Handle("/stats",    h.adminStats)
	b.Handle("/settings", h.adminSettings)
	b.Handle("/block",    h.adminBlock)
	b.Handle("/unblock",  h.adminUnblock)
	b.Handle("/users",    h.adminUsers)
	b.Handle("/broadcast",h.adminBroadcast)

	// ── آپلود فایل ────────────────────────────────────────
	b.Handle(tele.OnDocument,  h.onMedia)
	b.Handle(tele.OnVideo,     h.onMedia)
	b.Handle(tele.OnAudio,     h.onMedia)
	b.Handle(tele.OnPhoto,     h.onMedia)
	b.Handle(tele.OnVoice,     h.onMedia)
	b.Handle(tele.OnAnimation, h.onMedia)
	b.Handle(tele.OnVideoNote, h.onMedia)

	// ── callback ──────────────────────────────────────────
	b.Handle(tele.OnCallback, h.onCallback)
}

// ── /start ────────────────────────────────────────────────

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()

	// ثبت/آپدیت کاربر
	h.users.Upsert(ctx, &documents.BotUser{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
	})
	h.eng.Stats.IncrementDaily(ctx, "new_users", 1)

	// بررسی بلاک
	user, _ := h.users.FindByTelegramID(ctx, c.Sender().ID)
	if user != nil && user.IsBlocked {
		return c.Send(h.setting(ctx, "blocked_text", "⛔️ دسترسی شما محدود شده است."))
	}

	// بررسی عضویت اجباری
	if err := h.checkMembership(ctx, c); err != nil {
		return err
	}

	if h.isAdmin(c) {
		return c.Send(
			fmt.Sprintf("سلام %s 👑\nبه ربات آپلودر خوش آمدید.", c.Sender().FirstName),
			kbAdmin(),
		)
	}

	welcome := h.setting(ctx, "welcome_text",
		fmt.Sprintf("سلام %s 👋\nکد دریافت فایل را ارسال کنید.", c.Sender().FirstName))
	return c.Send(fmt.Sprintf(welcome, c.Sender().FirstName), kbUser())
}

func (h *Handler) onHelp(c tele.Context) error {
	if h.isAdmin(c) {
		return c.Send(adminHelp, tele.ModeHTML, kbAdmin())
	}
	return c.Send(userHelp, tele.ModeHTML, kbUser())
}

const adminHelp = `<b>📖 دستورات ادمین</b>

<b>فایل و کد:</b>
فایل بفرست → ذخیره و دریافت ID
/newcode — ساخت کد جدید
/codes — لیست کدها
/delcode &lt;code&gt; — حذف کد

<b>کاربران:</b>
/users — لیست کاربران
/block &lt;id&gt; — بلاک
/unblock &lt;id&gt; — آنبلاک
/broadcast — پیام به همه

<b>مدیریت:</b>
/stats — آمار
/settings — تنظیمات
/panel — پنل مدیریت`

const userHelp = `<b>❓ راهنما</b>

کد دریافتی را اینجا ارسال کنید تا فایل دریافت کنید.`

func (h *Handler) isAdmin(c tele.Context) bool {
	return c.Sender().ID == h.ownerID
}

func (h *Handler) setting(ctx context.Context, key, defaultVal string) string {
	// اول از config MongoDB (hot-reload) بخوان
	if cfg := h.eng.Config.Get(); cfg != nil {
		if val := cfg.Get(key, ""); val != "" {
			return val
		}
	}
	// fallback به SettingStore (docstore)
	val, _ := h.eng.Settings.Get(ctx, key)
	if val == "" {
		return defaultVal
	}
	return val
}

func (h *Handler) log() ports.Logger {
	return h.eng.Log
}
