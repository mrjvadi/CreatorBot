package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// SendText sends a plain text message to destUsername (a user, bot,
// channel, or group) and returns the new message's ID. Used by watch_nats
// (internal/userbot) to bridge an arbitrary NATS message into Telegram.
func (c *Client) SendText(ctx context.Context, destUsername, text string) (int, error) {
	dest, err := c.resolvePeer(ctx, destUsername)
	if err != nil {
		return 0, fmt.Errorf("resolve destination: %w", err)
	}

	updates, err := c.api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     dest,
		Message:  text,
		RandomID: randID(),
	})
	if err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}
	return newMessageID(updates), nil
}
