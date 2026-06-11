package tgbot

import (
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/vpn-bot/internal/store"
)

type Handler struct {
	sender    ports.BotSender
	store     *store.Store
	panel     ports.VPNPanel
	gateway   ports.PaymentGateway
	cache     ports.Cache
	log       ports.Logger
	channelID int64
}

func NewHandler(sender ports.BotSender, st *store.Store, panel ports.VPNPanel, gateway ports.PaymentGateway, cache ports.Cache, log ports.Logger, channelID int64) *Handler {
	return &Handler{sender: sender, store: st, panel: panel, gateway: gateway, cache: cache, log: log, channelID: channelID}
}

func Register(b *tele.Bot, h *Handler) {
	b.Handle("/start", func(c tele.Context) error { return c.Send("به ربات VPN خوش آمدید.") })
	b.Handle("/buy", func(c tele.Context) error { return c.Send("TODO: buy") })
	b.Handle("/renew", func(c tele.Context) error { return c.Send("TODO: renew") })
	b.Handle("/status", func(c tele.Context) error { return c.Send("TODO: status") })
	b.Handle("/wallet", func(c tele.Context) error { return c.Send("TODO: wallet") })
}
