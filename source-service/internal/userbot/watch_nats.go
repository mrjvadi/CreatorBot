package userbot

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/models"
	"github.com/mrjvadi/creatorbot/source-service/internal/natsutil"
)

// WatchNatsPayload is "if a message arrives on subject, send it to
// dest_channel" — the NATS-triggered equivalent of watch_channel, for
// bridging events from the rest of the system into Telegram.
type WatchNatsPayload struct {
	Subject     string `json:"subject"`
	DestChannel string `json:"dest_channel"` // Telegram chat/bot/channel to send matching messages to
}

type WatchNatsResult struct {
	WatchID string `json:"watch_id"`
}

func (u *Userbot) handleWatchNats(ctx context.Context, _ string, raw json.RawMessage) (any, error) {
	var p WatchNatsPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	if p.Subject == "" || p.DestChannel == "" {
		return nil, errors.New("subject and dest_channel are required")
	}

	id := uuid.New()
	if err := u.startNatsWatch(ctx, id.String(), p.Subject, p.DestChannel); err != nil {
		return nil, err
	}

	row := &models.NatsWatch{
		Base:        models.Base{ID: id},
		Phone:       u.phone,
		Subject:     p.Subject,
		DestChannel: p.DestChannel,
		Active:      true,
	}
	if err := u.store.CreateNatsWatch(ctx, row); err != nil {
		u.natsSubs.remove(id.String())
		return nil, err
	}

	return WatchNatsResult{WatchID: id.String()}, nil
}

// startNatsWatch subscribes to subject and, for every message received,
// sends its raw content as a text message to destChannel. This is the
// "and many other things" seam: swap the body of the callback below to do
// something other than a plain text send (e.g. decode the payload and run
// a different action) for a different kind of NATS-to-Telegram bridge.
//
// Uses natsutil.SubscribeRaw (the raw *nats.Conn) rather than the shared
// client's wrapped Subscribe, because that one discards its
// *nats.Subscription and gives no way to unsubscribe — which remove_nats_watch
// needs.
func (u *Userbot) startNatsWatch(ctx context.Context, id, subject, destChannel string) error {
	sub, err := natsutil.SubscribeRaw(u.nc, subject, func(data []byte) {
		if _, err := u.tg.SendText(ctx, destChannel, string(data)); err != nil {
			u.log.Error("nats watch: send failed",
				ports.F("watch_id", id), ports.F("subject", subject), ports.F("err", err))
		}
	})
	if err != nil {
		return err
	}
	u.natsSubs.add(id, sub.Unsubscribe)
	return nil
}
