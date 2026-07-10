package userbot

import (
	"context"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// RestoreWatches reloads this account's persisted, active channel-watch
// rules into the live telegram.Client after a restart — otherwise a worker
// that comes back up would silently stop auto-forwarding until someone
// re-runs watch_channel. Call this once the client is authorized (see
// telegram.Client.Ready); a failed individual watch is logged and skipped
// rather than aborting the rest.
func (u *Userbot) RestoreWatches(ctx context.Context, log ports.Logger) {
	watches, err := u.store.ListActiveChannelWatches(ctx, u.phone)
	if err != nil {
		log.Error("restore watches: list", ports.F("err", err))
		return
	}

	for _, w := range watches {
		id := w.ID.String()
		if err := u.tg.AddWatch(ctx, id, w.SourceChannel, w.DestChannel); err != nil {
			log.Error("restore watch", ports.F("watch_id", id), ports.F("err", err))
			continue
		}
		log.Info("watch restored", ports.F("watch_id", id), ports.F("source", w.SourceChannel), ports.F("dest", w.DestChannel))
	}
}
