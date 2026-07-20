package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/joinevents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Handler struct {
	bot         *tele.Bot
	store       *store.Store
	cache       ports.Cache
	log         ports.Logger
	ownerID     int64
	botID       int64
	botUsername string
	nats        *natsclient.Client

	// rentalStatus/joinPublisher فقط وقتی این instance رایگان به یک کمپینِ
	// اجاره‌ی قفلِ فعال در ads-bot وصل است معنی دارند — nil یعنی راه‌اندازی
	// نشده (مثلاً NATS در دسترس نبوده، رجوع main.go).
	rentalStatus  *memberclient.RentalStatus
	joinPublisher *joinevents.Publisher
}

func NewHandler(
	bot *tele.Bot, st *store.Store, cache ports.Cache, log ports.Logger,
	ownerID int64, botUsername string, botID int64, nc *natsclient.Client,
	rentalStatus *memberclient.RentalStatus, joinPublisher *joinevents.Publisher,
) *Handler {
	return &Handler{
		bot: bot, store: st, cache: cache, log: log,
		ownerID: ownerID, botUsername: botUsername, botID: botID, nats: nc,
		rentalStatus: rentalStatus, joinPublisher: joinPublisher,
	}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start",      h.onStart)
	b.Handle("/help",       h.onHelp)
	b.Handle("/add",        h.onAdd)
	b.Handle("/categories", h.onCategories)
	b.Handle("/newcat",     h.onNewCategory)
	b.Handle("/cancel",     h.onCancel)
	b.Handle(tele.OnQuery,    h.onInlineQuery)
	b.Handle(tele.OnText,     h.onText)
	b.Handle(tele.OnDocument, h.onMedia)
	b.Handle(tele.OnVideo,    h.onMedia)
	b.Handle(tele.OnAudio,    h.onMedia)
	b.Handle(tele.OnPhoto,    h.onMedia)
	b.Handle(tele.OnCallback, h.onCallback)
	b.Handle(tele.OnMyChatMember, h.onMyChatMember)
	if h.joinPublisher != nil {
		b.Handle(tele.OnChatMember, h.joinPublisher.HandleChatMember)
		b.Handle(tele.OnUserJoined, h.joinPublisher.HandleUserJoined)
		b.Handle(tele.OnUserLeft, h.joinPublisher.HandleUserLeft)
	}
}

func (h *Handler) isAdmin(c tele.Context) bool { return c.Sender().ID == h.ownerID }

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	h.store.UpsertUserByID(ctx, c.Sender().ID, c.Sender().Username, c.Sender().FirstName)
	return c.Send(
		"<b>📚 آرشیو</b>\n\n🔍 برای جستجو متن بنویسید\n📂 /categories\n🔎 @"+h.botUsername+" &lt;query&gt;",
		tele.ModeHTML, kbMain(h.isAdmin(c)),
	)
}

func (h *Handler) onHelp(c tele.Context) error {
	msg := "<b>❓ راهنما</b>\n\nجستجو: متن بنویسید\nInline: @bot &lt;query&gt;\nدسته‌بندی: /categories"
	if h.isAdmin(c) {
		msg += "\n\n<b>ادمین:</b>\n/add — آپلود\n/newcat — دسته جدید"
	}
	return c.Send(msg, tele.ModeHTML, kbMain(h.isAdmin(c)))
}

func (h *Handler) onCancel(c tele.Context) error {
	h.clearState(context.Background(), c.Sender().ID)
	return c.Send("لغو شد.", kbMain(h.isAdmin(c)))
}

// onFileCommand ارسال فایل از طریق inline query.
// وقتی کاربر نتیجه inline رو انتخاب می‌کند این command ارسال می‌شود.
func (h *Handler) onFileCommand(c tele.Context) error {
	ctx := context.Background()
	// متن: /file_<uuid>
	text := strings.TrimSpace(c.Text())
	if !strings.HasPrefix(text, "/file_") {
		return nil
	}
	fileID := strings.TrimPrefix(text, "/file_")
	f, err := h.store.FindFileByID(ctx, fileID)
	if err != nil || f == nil {
		return c.Send("❌ فایل یافت نشد.")
	}
	sendArchiveFile(c, *f, h.isAdmin(c))
	return nil
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
	case btnSearch:
		return c.Send("متن جستجو را وارد کنید:", kbCancel())
	case btnCategories:
		return h.onCategories(c)
	case btnHelp:
		return h.onHelp(c)
	case "➕ فایل جدید":
		return h.onAdd(c)
	case "📂 دسته جدید":
		return h.onNewCategory(c)
	case btnCancel, btnBack:
		h.clearState(ctx, uid)
		return c.Send("لغو شد.", kbMain(h.isAdmin(c)))
	}

	return h.doSearch(ctx, c, text)
}

func (h *Handler) onMedia(c tele.Context) error {
	ctx := context.Background()
	if !h.isAdmin(c) {
		return nil
	}
	st := h.getState(ctx, c.Sender().ID)
	if st.Step == stepUploadConfirm {
		return h.confirmUpload(ctx, c, st)
	}
	return h.startUploadWizard(ctx, c)
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
	case "cat":
		if len(parts) == 2 {
			return h.showCategory(ctx, c, parts[1])
		}
	case "del":
		if len(parts) == 2 && h.isAdmin(c) {
			return h.deleteFile(ctx, c, parts[1])
		}
	case "back":
		return h.onCategories(c)
	}
	return nil
}

func (h *Handler) handleStep(ctx context.Context, c tele.Context, st wizardState, text string) error {
	if text == btnCancel {
		h.clearState(ctx, c.Sender().ID)
		return c.Send("لغو شد.", kbMain(h.isAdmin(c)))
	}
	switch st.Step {
	case stepUploadTitle:    return h.handleTitle(ctx, c, st, text)
	case stepUploadTags:     return h.handleTags(ctx, c, st, text)
	case stepUploadDesc:     return h.handleDesc(ctx, c, st, text)
	case stepUploadCategory: return h.handleCategory(ctx, c, st, text)
	case stepNewCategory:    return h.createCategory(ctx, c, text)
	case stepUploadConfirm:
		if text == btnConfirm {
			return h.confirmUpload(ctx, c, st)
		}
		h.clearState(ctx, c.Sender().ID)
		return c.Send("لغو شد.", kbMain(h.isAdmin(c)))
	}
	return nil
}

func (h *Handler) onCategories(c tele.Context) error {
	ctx := context.Background()
	cats, err := h.store.ListCategories(ctx)
	if err != nil {
		return c.Send("❌ خطا.")
	}
	if len(cats) == 0 {
		return c.Send("دسته‌بندی‌ای وجود ندارد.", kbMain(h.isAdmin(c)))
	}
	return c.Send("📂 دسته‌بندی‌ها:", kbCategories(cats))
}

func (h *Handler) showCategory(ctx context.Context, c tele.Context, catIDStr string) error {
	files, err := h.store.FindFilesByCategory(ctx, catIDStr)
	if err != nil || len(files) == 0 {
		return c.Edit("این دسته خالی است.")
	}
	for _, f := range files {
		sendArchiveFile(c, f, h.isAdmin(c))
	}
	return nil
}

func (h *Handler) onNewCategory(c tele.Context) error {
	if !h.isAdmin(c) { return nil }
	h.setStep(context.Background(), c.Sender().ID, stepNewCategory)
	return c.Send("نام دسته‌بندی جدید:", kbCancel())
}

func (h *Handler) createCategory(ctx context.Context, c tele.Context, name string) error {
	h.clearState(ctx, c.Sender().ID)
	if len(name) < 2 {
		return c.Send("نام خیلی کوتاه است.")
	}
	cat, err := h.store.FindOrCreateCategory(ctx, name)
	if err != nil {
		return c.Send("❌ خطا.")
	}
	return c.Send("✅ دسته‌بندی <b>"+cat.Name+"</b> ساخته شد.", tele.ModeHTML, kbMain(true))
}

func (h *Handler) deleteFile(ctx context.Context, c tele.Context, fileIDStr string) error {
	if err := h.store.DeleteFile(ctx, fileIDStr); err != nil {
		return c.Edit("❌ خطا.")
	}
	return c.Edit("🗑 فایل حذف شد.")
}

