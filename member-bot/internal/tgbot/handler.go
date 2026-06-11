package tgbot

import (
	tele "gopkg.in/telebot.v4"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/member-bot/internal/store"
)

type Handler struct {
	sender ports.BotSender
	store  *store.Store
	cache  ports.Cache
	log    ports.Logger
}

func NewHandler(sender ports.BotSender, st *store.Store, cache ports.Cache, log ports.Logger) *Handler {
	return &Handler{sender: sender, store: st, cache: cache, log: log}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", func(c tele.Context) error { return c.Send("به ربات مدیریت قفل ممبر خوش آمدید.") })
	b.Handle("/register", func(c tele.Context) error { return c.Send("TODO: register lock channel") })
	b.Handle("/mylocks", func(c tele.Context) error { return c.Send("TODO: list locks") })
}
