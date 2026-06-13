package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/search"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Handler struct {
	sender  ports.BotSender
	store   *store.Store
	db      ports.DB
	cache   ports.Cache
	log     ports.Logger
	ownerID int64
}

func NewHandler(sender ports.BotSender, st *store.Store, db ports.DB, cache ports.Cache, log ports.Logger, ownerID int64) *Handler {
	return &Handler{sender: sender, store: st, db: db, cache: cache, log: log, ownerID: ownerID}
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
}

func (h *Handler) isAdmin(c tele.Context) bool { return c.Sender().ID == h.ownerID }

func (h *Handler) onStart(c tele.Context) error {
	ctx := context.Background()
	h.store.UpsertUserByID(ctx, c.Sender().ID, c.Sender().Username, c.Sender().FirstName)
	return c.Send(
		"<b>📚 آرشیو</b>\n\n🔍 برای جستجو متن بنویسید\n📂 /categories\n🔎 @"+c.Bot().Me.Username+" <query>",
		tele.ModeHTML, kbMain(h.isAdmin(c)),
	)
}

func (h *Handler) onHelp(c tele.Context) error {
	msg := "<b>❓ راهنما</b>\n\nجستجو: متن بنویسید\nInline: @bot <query>\nدسته‌بندی: /categories"
	if h.isAdmin(c) {
		msg += "\n\n<b>ادمین:</b>\n/add — آپلود\n/newcat — دسته جدید"
	}
	return c.Send(msg, tele.ModeHTML, kbMain(h.isAdmin(c)))
}

func (h *Handler) onCancel(c tele.Context) error {
	h.clearState(context.Background(), c.Sender().ID)
	return c.Send("لغو شد.", kbMain(h.isAdmin(c)))
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

var _ = search.Normalize
