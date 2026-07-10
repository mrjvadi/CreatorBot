package userbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type RemoveNatsWatchPayload struct {
	WatchID string `json:"watch_id"`
}

func (u *Userbot) handleRemoveNatsWatch(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p RemoveNatsWatchPayload
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

	u.natsSubs.remove(p.WatchID)
	if err := u.store.DeactivateNatsWatch(ctx, id); err != nil {
		return nil, err
	}
	return map[string]bool{"removed": true}, nil
}
