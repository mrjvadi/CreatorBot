package userbot

import (
	"context"
	"encoding/json"
)

type NatsWatchInfo struct {
	WatchID     string `json:"watch_id"`
	Subject     string `json:"subject"`
	DestChannel string `json:"dest_channel"`
}

type ListNatsWatchesResult struct {
	Watches []NatsWatchInfo `json:"watches"`
}

func (u *Userbot) handleListNatsWatches(ctx context.Context, _ string, _ json.RawMessage) (any, error) {
	rows, err := u.store.ListActiveNatsWatches(ctx, u.phone)
	if err != nil {
		return nil, err
	}

	out := make([]NatsWatchInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, NatsWatchInfo{WatchID: r.ID.String(), Subject: r.Subject, DestChannel: r.DestChannel})
	}
	return ListNatsWatchesResult{Watches: out}, nil
}
