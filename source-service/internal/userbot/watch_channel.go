package userbot

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/source-service/internal/models"
)

// WatchChannelPayload is "if source_channel posts, forward it to
// dest_channel", checked in real time as new posts arrive (not on a poll).
type WatchChannelPayload struct {
	SourceChannel string `json:"source_channel"` // "@channel_username"
	DestChannel   string `json:"dest_channel"`
}

type WatchChannelResult struct {
	WatchID string `json:"watch_id"`
}

func (u *Userbot) handleWatchChannel(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p WatchChannelPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.SourceChannel == "" || p.DestChannel == "" {
		return nil, errors.New("source_channel and dest_channel are required")
	}

	id := uuid.New()
	if err := u.tg.AddWatch(ctx, id.String(), p.SourceChannel, p.DestChannel); err != nil {
		return nil, err
	}

	row := &models.ChannelWatch{
		Base:          models.Base{ID: id},
		Phone:         u.phone,
		SourceChannel: p.SourceChannel,
		DestChannel:   p.DestChannel,
		Active:        true,
	}
	if err := u.store.CreateChannelWatch(ctx, row); err != nil {
		u.tg.RemoveWatch(id.String()) // don't leave a live rule with no record of it
		return nil, err
	}

	return WatchChannelResult{WatchID: id.String()}, nil
}
