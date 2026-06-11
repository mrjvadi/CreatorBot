package tgbot

import (
	"context"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/search"
	"github.com/mrjvadi/creatorbot/archive-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Handler struct {
	sender ports.BotSender
	store  *store.Store
	db     ports.DB
	cache  ports.Cache
	log    ports.Logger
}

func NewHandler(sender ports.BotSender, st *store.Store, db ports.DB, cache ports.Cache, log ports.Logger) *Handler {
	return &Handler{sender: sender, store: st, db: db, cache: cache, log: log}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", h.handleStart)
	b.Handle("/add", h.handleAdd)
	b.Handle("/categories", h.handleCategories)
	b.Handle(tele.OnQuery, h.handleInlineQuery)
	b.Handle(tele.OnText, h.handleSearch)
	b.Handle(tele.OnDocument, h.handleAdminMedia)
	b.Handle(tele.OnVideo, h.handleAdminMedia)
	b.Handle(tele.OnAudio, h.handleAdminMedia)
}

func (h *Handler) handleStart(c tele.Context) error {
	return c.Send("جستجو کنید یا نام فایل را بنویسید.")
}

func (h *Handler) handleAdd(c tele.Context) error {
	return c.Send("فایل را ارسال کنید.")
}

func (h *Handler) handleCategories(c tele.Context) error {
	ctx := context.Background() // FIX 18: telebot has no c.Request()
	cats, err := h.store.ListCategories(ctx)
	if err != nil {
		return c.Send("خطا در دریافت دسته‌بندی‌ها.")
	}
	if len(cats) == 0 {
		return c.Send("هیچ دسته‌بندی‌ای وجود ندارد.")
	}
	msg := "📂 دسته‌بندی‌ها:\n"
	for _, cat := range cats {
		msg += "- " + cat.Name + "\n"
	}
	return c.Send(msg)
}

func (h *Handler) handleInlineQuery(c tele.Context) error {
	ctx := context.Background() // FIX 18
	files, err := search.Search(ctx, h.db.Conn(), c.Query().Text, 10)
	if err != nil {
		h.log.Error("inline search failed", ports.F("err", err))
		return c.Answer(nil)
	}
	var results []ports.InlineResult
	for _, f := range files {
		results = append(results, ports.InlineResult{
			ID:          f.ID.String(),
			Title:       f.Title,
			Description: f.Tags,
			FileID:      f.FileID,
			FileType:    f.FileType,
		})
	}
	return h.sender.AnswerInlineQuery(ctx, c.Query().ID, results)
}

func (h *Handler) handleSearch(c tele.Context) error {
	ctx := context.Background() // FIX 18
	files, err := search.Search(ctx, h.db.Conn(), c.Text(), 5)
	if err != nil || len(files) == 0 {
		return c.Send("نتیجه‌ای یافت نشد.")
	}
	// TODO: format and send files as album or list
	_ = files
	return c.Send("TODO: send search results")
}

func (h *Handler) handleAdminMedia(c tele.Context) error {
	// TODO: admin wizard — title → tags → description → category → confirm
	return c.Send("TODO: admin upload wizard")
}
