package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/td/tg"
)

// ChannelPostEvent is a plain-data view of a channel post, handed to
// WatchChannelPosts callbacks. It deliberately exposes no gotd/td types, so
// callers (like internal/rules) don't need to depend on gotd/td.
type ChannelPostEvent struct {
	MessageID int
	Text      string
	// Sender is best-effort and often empty: a channel post only has an
	// identifiable human sender if the channel has "signed messages"
	// enabled, and even then this is just the numeric user ID (resolving a
	// username would cost an extra API call per post).
	Sender string
}

type rawChannelWatch struct {
	ID              string
	SourceChannelID int64
	Handler         func(ctx context.Context, ev ChannelPostEvent)
}

// rawChannelWatches is a second, independent registry from channelWatches
// (watch.go): that one always forwards to a fixed destination; this one
// hands the raw event to an arbitrary callback, which is what
// internal/rules needs to build its own trigger/condition/action pipeline
// on top. Kept separate so this addition can't affect the existing
// watch_channel behavior.
type rawChannelWatches struct {
	mu   sync.RWMutex
	byID map[string]rawChannelWatch
}

func newRawChannelWatches() *rawChannelWatches {
	return &rawChannelWatches{byID: make(map[string]rawChannelWatch)}
}

func (w *rawChannelWatches) add(watch rawChannelWatch) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.byID[watch.ID] = watch
}

func (w *rawChannelWatches) remove(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.byID, id)
}

func (w *rawChannelWatches) matching(channelID int64) []rawChannelWatch {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var out []rawChannelWatch
	for _, watch := range w.byID {
		if watch.SourceChannelID == channelID {
			out = append(out, watch)
		}
	}
	return out
}

// WatchChannelPosts calls handler for every future post in sourceUsername,
// with no fixed action attached — this is the generic primitive
// internal/rules builds "channel_post" triggers on. For the simpler
// "always forward to one destination" case, use AddWatch (watch.go)
// instead.
func (c *Client) WatchChannelPosts(ctx context.Context, id, sourceUsername string, handler func(ctx context.Context, ev ChannelPostEvent)) error {
	channel, err := c.resolveChannel(ctx, sourceUsername)
	if err != nil {
		return fmt.Errorf("resolve source channel: %w", err)
	}
	c.rawWatches.add(rawChannelWatch{ID: id, SourceChannelID: channel.ChannelID, Handler: handler})
	return nil
}

// UnwatchChannelPosts stops a WatchChannelPosts callback from firing.
func (c *Client) UnwatchChannelPosts(id string) {
	c.rawWatches.remove(id)
}

// senderFromMessage extracts a best-effort sender identifier (see
// ChannelPostEvent.Sender).
func senderFromMessage(m *tg.Message) string {
	if m.FromID == nil {
		return ""
	}
	if u, ok := m.FromID.(*tg.PeerUser); ok {
		return fmt.Sprintf("%d", u.UserID)
	}
	return ""
}
