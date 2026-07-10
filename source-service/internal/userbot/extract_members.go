package userbot

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/mrjvadi/creatorbot/source-service/internal/telegram"
)

type ExtractMembersPayload struct {
	Channel string `json:"channel"` // "@channel_username"
}

type ExtractMembersResult struct {
	Members []telegram.Member `json:"members"`
	Count   int               `json:"count"`
}

func (u *Userbot) handleExtractMembers(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p ExtractMembersPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Channel == "" {
		return nil, errors.New("channel is required")
	}

	members, err := u.tg.ExtractMembers(ctx, p.Channel)
	if err != nil {
		return nil, err
	}
	return ExtractMembersResult{Members: members, Count: len(members)}, nil
}
