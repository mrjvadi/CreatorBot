package userbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type RemoveWatchPayload struct {
	WatchID string `json:"watch_id"`
}

func (u *Userbot) handleRemoveWatch(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p RemoveWatchPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.WatchID == "" {
		return nil, errors.New("watch_id is required")
	}

	id, err := uuid.Parse(p.WatchID)
	if err != nil {
		return nil, fmt.Errorf("invalid watch_id: %w", err)
	}

	u.tg.RemoveWatch(p.WatchID)
	if err := u.store.DeactivateChannelWatch(ctx, id); err != nil {
		return nil, err
	}
	return map[string]bool{"removed": true}, nil
}
