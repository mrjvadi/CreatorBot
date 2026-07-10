package userbot

import (
	"context"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// RestoreNatsWatches reloads this account's persisted, active NATS-watch
// rules into live subscriptions after a restart. Call this once the
// telegram client is ready (SendText inside the subscription callback needs
// it), same as RestoreWatches.
func (u *Userbot) RestoreNatsWatches(ctx context.Context) {
	rows, err := u.store.ListActiveNatsWatches(ctx, u.phone)
	if err != nil {
		u.log.Error("restore nats watches: list", ports.F("err", err))
		return
	}

	for _, w := range rows {
		id := w.ID.String()
		if err := u.startNatsWatch(ctx, id, w.Subject, w.DestChannel); err != nil {
			u.log.Error("restore nats watch", ports.F("watch_id", id), ports.F("err", err))
			continue
		}
		u.log.Info("nats watch restored", ports.F("watch_id", id), ports.F("subject", w.Subject), ports.F("dest", w.DestChannel))
	}
}
