package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/tg"
)

// resolveUsername resolves a "@username" (or bare "username") to the chats
// and users Telegram knows about it.
func (c *Client) resolveUsername(ctx context.Context, username string) (*tg.ContactsResolvedPeer, error) {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	return c.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
}

// resolveChannel resolves a channel/supergroup username to an *tg.InputChannel.
func (c *Client) resolveChannel(ctx context.Context, username string) (*tg.InputChannel, error) {
	resolved, err := c.resolveUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", username, err)
	}
	for _, chat := range resolved.Chats {
		if ch, ok := chat.(*tg.Channel); ok {
			return &tg.InputChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}, nil
		}
	}
	return nil, fmt.Errorf("%q is not a channel or supergroup", username)
}

// resolvePeer resolves a "@username" to something usable as a message
// destination/source (user, bot, channel, or group).
func (c *Client) resolvePeer(ctx context.Context, username string) (tg.InputPeerClass, error) {
	resolved, err := c.resolveUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("resolve %q: %w", username, err)
	}
	for _, user := range resolved.Users {
		if u, ok := user.(*tg.User); ok {
			return &tg.InputPeerUser{UserID: u.ID, AccessHash: u.AccessHash}, nil
		}
	}
	for _, chat := range resolved.Chats {
		switch ch := chat.(type) {
		case *tg.Channel:
			return &tg.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}, nil
		case *tg.Chat:
			return &tg.InputPeerChat{ChatID: ch.ID}, nil
		}
	}
	return nil, fmt.Errorf("%q did not resolve to a user, chat, or channel", username)
}

// peerUserID extracts the numeric user ID a peer refers to, if it's a user.
func peerUserID(p tg.PeerClass) (int64, bool) {
	if u, ok := p.(*tg.PeerUser); ok {
		return u.UserID, true
	}
	return 0, false
}
