// Package userbot wraps gotd/td for MTProto-based file forwarding.
// ⚠️ Violates Telegram ToS — use only for personal archiving.
package userbot

import (
	"context"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Config struct {
	AppID           int
	AppHash         string
	Phone           string
	SessionFile     string
	SourceChannel   int64
	DeliveryChannel int64
}

type Userbot struct {
	cfg Config
	log ports.Logger
}

func New(cfg Config, log ports.Logger) *Userbot {
	return &Userbot{cfg: cfg, log: log}
}

func (u *Userbot) Start(ctx context.Context) {
	// TODO: initialize gotd/td client, login, listen for new messages in SourceChannel
	u.log.Info("userbot started (TODO: implement gotd/td)")
	<-ctx.Done()
}

func (u *Userbot) ForwardToDelivery(ctx context.Context, sourceMessageID int) (int, error) {
	// TODO: forward message from SourceChannel to DeliveryChannel via MTProto
	return 0, nil
}
