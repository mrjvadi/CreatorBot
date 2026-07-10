// Package natsutil has one job: give callers a NATS subscription they can
// actually cancel. The shared client's own (*natsclient.Client).Subscribe
// discards the underlying *nats.Subscription, so anything that needs to
// unsubscribe later (watch_nats in internal/userbot, the nats_message
// trigger in internal/rules) goes through the raw connection (nc.NC())
// instead. This is the one place in source-service that imports nats-io/
// nats.go directly, so that dependency doesn't spread across packages.
package natsutil

import (
	"github.com/nats-io/nats.go"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
)

// Subscription is satisfied by *nats.Subscription.
type Subscription interface {
	Unsubscribe() error
}

// SubscribeRaw subscribes to subject on the underlying *nats.Conn and
// returns a real, cancelable Subscription.
func SubscribeRaw(nc *natsclient.Client, subject string, handler func(data []byte)) (Subscription, error) {
	return nc.NC().Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
}
