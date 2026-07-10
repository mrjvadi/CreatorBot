package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/td/tg"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ChannelWatch is one live "if SourceUsername posts, forward it to
// DestUsername" rule. SourceChannelID is resolved once (in AddWatch) so
// matching an incoming post never needs an extra API call.
type ChannelWatch struct {
	ID              string `json:"id"`
	SourceChannelID int64  `json:"-"`
	SourceUsername  string `json:"source_username"`
	DestUsername    string `json:"dest_username"`
}

// channelWatches is the in-memory, live set of watch rules this Client
// checks against every incoming channel post. It starts empty on process
// start; internal/userbot.RestoreWatches repopulates it from Postgres via
// AddWatch once the client is authorized (see Client.Ready).
type channelWatches struct {
	mu   sync.RWMutex
	byID map[string]ChannelWatch
}

func newChannelWatches() *channelWatches {
	return &channelWatches{byID: make(map[string]ChannelWatch)}
}

func (w *channelWatches) add(watch ChannelWatch) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.byID[watch.ID] = watch
}

func (w *channelWatches) remove(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.byID, id)
}

func (w *channelWatches) list() []ChannelWatch {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]ChannelWatch, 0, len(w.byID))
	for _, watch := range w.byID {
		out = append(out, watch)
	}
	return out
}

func (w *channelWatches) matching(channelID int64) []ChannelWatch {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var out []ChannelWatch
	for _, watch := range w.byID {
		if watch.SourceChannelID == channelID {
			out = append(out, watch)
		}
	}
	return out
}

// AddWatch resolves sourceUsername/destUsername once and registers a live
// rule: any future post in sourceUsername is forwarded to destUsername.
// Persist the rule yourself (internal/store) if it should survive a
// restart — this only affects the in-memory/live behavior.
func (c *Client) AddWatch(ctx context.Context, id, sourceUsername, destUsername string) error {
	channel, err := c.resolveChannel(ctx, sourceUsername)
	if err != nil {
		return fmt.Errorf("resolve source channel: %w", err)
	}
	if _, err := c.resolvePeer(ctx, destUsername); err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}

	c.watches.add(ChannelWatch{
		ID:              id,
		SourceChannelID: channel.ChannelID,
		SourceUsername:  sourceUsername,
		DestUsername:    destUsername,
	})
	return nil
}

// RemoveWatch stops a live watch rule. It does not touch persisted state —
// callers are expected to also deactivate it in internal/store.
func (c *Client) RemoveWatch(id string) {
	c.watches.remove(id)
}

// ListWatches returns every currently-live watch rule.
func (c *Client) ListWatches() []ChannelWatch {
	return c.watches.list()
}

// handleChannelPost is the update-dispatcher callback for new channel
// posts (see New in client.go). It forwards the post to every fixed
// watch_channel rule whose source matches, and also hands the raw event to
// every generic WatchChannelPosts callback (used by internal/rules) whose
// source matches.
func (c *Client) handleChannelPost(ctx context.Context, msg tg.MessageClass) {
	m, ok := msg.(*tg.Message)
	if !ok {
		return
	}
	channelID, ok := peerChannelID(m.PeerID)
	if !ok {
		return
	}

	for _, watch := range c.watches.matching(channelID) {
		if err := c.forwardWatchedPost(ctx, watch, m.ID); err != nil {
			c.log.Error("watch forward failed",
				ports.F("watch_id", watch.ID),
				ports.F("source", watch.SourceUsername),
				ports.F("dest", watch.DestUsername),
				ports.F("err", err))
		}
	}

	if raw := c.rawWatches.matching(channelID); len(raw) > 0 {
		ev := ChannelPostEvent{MessageID: m.ID, Text: m.Message, Sender: senderFromMessage(m)}
		for _, watch := range raw {
			watch.Handler(ctx, ev)
		}
	}
}

func (c *Client) forwardWatchedPost(ctx context.Context, watch ChannelWatch, messageID int) error {
	source, err := c.resolvePeer(ctx, watch.SourceUsername)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}
	dest, err := c.resolvePeer(ctx, watch.DestUsername)
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}

	_, err = c.api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: source,
		ID:       []int{messageID},
		ToPeer:   dest,
		RandomID: []int64{randID()},
	})
	return err
}

// peerChannelID extracts the numeric channel ID a peer refers to, if it's a
// channel/supergroup.
func peerChannelID(p tg.PeerClass) (int64, bool) {
	if c, ok := p.(*tg.PeerChannel); ok {
		return c.ChannelID, true
	}
	return 0, false
}
