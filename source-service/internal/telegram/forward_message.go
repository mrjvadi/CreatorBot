package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
)

// ForwardMessage forwards messageID from sourceUsername to destUsername and
// returns the new message ID in the destination chat, if determinable.
func (c *Client) ForwardMessage(ctx context.Context, sourceUsername, destUsername string, messageID int) (int, error) {
	sourcePeer, err := c.resolvePeer(ctx, sourceUsername)
	if err != nil {
		return 0, fmt.Errorf("resolve source: %w", err)
	}
	destPeer, err := c.resolvePeer(ctx, destUsername)
	if err != nil {
		return 0, fmt.Errorf("resolve destination: %w", err)
	}

	updates, err := c.api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: sourcePeer,
		ID:       []int{messageID},
		ToPeer:   destPeer,
		RandomID: []int64{randID()},
	})
	if err != nil {
		return 0, fmt.Errorf("forward message: %w", err)
	}

	return newMessageID(updates), nil
}
