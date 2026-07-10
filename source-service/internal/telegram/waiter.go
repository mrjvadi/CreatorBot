package telegram

import (
	"context"
	"sync"

	"github.com/gotd/td/tg"
)

// messageWaiter lets a goroutine block until a new message from a specific
// user (e.g. a bot we just messaged) arrives, without a polling loop. It
// supports one pending wait per sender at a time — if you need concurrent
// fetch_edit_send tasks against the *same* bot, extend `waiting` to hold a
// slice of channels per user ID instead of a single one.
type messageWaiter struct {
	mu      sync.Mutex
	waiting map[int64]chan *tg.Message
}

func newMessageWaiter() *messageWaiter {
	return &messageWaiter{waiting: make(map[int64]chan *tg.Message)}
}

// await blocks until a message from fromUserID arrives or ctx is done.
func (w *messageWaiter) await(ctx context.Context, fromUserID int64) (*tg.Message, error) {
	ch := make(chan *tg.Message, 1)
	w.mu.Lock()
	w.waiting[fromUserID] = ch
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.waiting, fromUserID)
		w.mu.Unlock()
	}()

	select {
	case msg := <-ch:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// handle is called from the update dispatcher for every new message; it
// delivers to whoever is waiting on that sender, if anyone.
func (w *messageWaiter) handle(msg tg.MessageClass) {
	m, ok := msg.(*tg.Message)
	if !ok {
		return
	}
	uid, ok := peerUserID(m.PeerID)
	if !ok {
		return
	}
	w.mu.Lock()
	ch, waiting := w.waiting[uid]
	w.mu.Unlock()
	if waiting {
		select {
		case ch <- m:
		default:
		}
	}
}
