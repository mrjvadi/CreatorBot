package userbot

import (
	"context"
	"encoding/json"

	"github.com/mrjvadi/creatorbot/source-service/internal/telegram"
)

type ListWatchesResult struct {
	Watches []telegram.ChannelWatch `json:"watches"`
}

func (u *Userbot) handleListWatches(_ context.Context, _ string, _ json.RawMessage) (any, error) {
	return ListWatchesResult{Watches: u.tg.ListWatches()}, nil
}
