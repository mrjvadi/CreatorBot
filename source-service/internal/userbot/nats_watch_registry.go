package userbot

import "sync"

// natsSubRegistry tracks live NATS subscriptions backing watch_nats rules,
// keyed by watch ID, so remove_nats_watch can actually unsubscribe (rather
// than just marking the database row inactive).
type natsSubRegistry struct {
	mu   sync.Mutex
	subs map[string]func() error
}

func newNatsSubRegistry() *natsSubRegistry {
	return &natsSubRegistry{subs: make(map[string]func() error)}
}

func (r *natsSubRegistry) add(id string, unsubscribe func() error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs[id] = unsubscribe
}

func (r *natsSubRegistry) remove(id string) {
	r.mu.Lock()
	unsub, ok := r.subs[id]
	delete(r.subs, id)
	r.mu.Unlock()
	if ok {
		_ = unsub()
	}
}
