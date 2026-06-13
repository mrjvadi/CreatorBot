// Package webhook یک Poller برای telebot می‌سازد که از NATS می‌خواند.
// به‌جای long-polling از تلگرام، gateway webhook ها را به NATS می‌فرستد
// و این adapter آن‌ها را به bot feed می‌کند.
//
// استفاده:
//
//	poller := webhook.NewNATSPoller(nc, botID, log)
//	b, _ := tele.NewBot(tele.Settings{
//	    Token:  token,
//	    Poller: poller,
//	})
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	tele "gopkg.in/telebot.v4"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// NATSPoller یک telebot.Poller که از NATS می‌خواند.
type NATSPoller struct {
	nc      *natsclient.Client
	botID   int64
	subject string
	log     ports.Logger

	updates chan tele.Update
	done    chan struct{}
	once    sync.Once
}

// NewNATSPoller یک Poller جدید می‌سازد.
func NewNATSPoller(nc *natsclient.Client, botID int64, log ports.Logger) *NATSPoller {
	return &NATSPoller{
		nc:      nc,
		botID:   botID,
		subject: fmt.Sprintf("webhook.%d", botID),
		log:     log,
		updates: make(chan tele.Update, 100),
		done:    make(chan struct{}),
	}
}

// Poll پیاده‌سازی telebot.Poller — update ها را از NATS می‌خواند.
func (p *NATSPoller) Poll(b *tele.Bot, updates chan tele.Update, stop chan struct{}) {
	p.log.Info("NATS poller started",
		ports.F("subject", p.subject),
		ports.F("bot_id", p.botID))

	// subscribe به NATS subject
	p.nc.Subscribe(p.subject, func(data []byte) {
		// تلگرام یک Update object می‌فرستد
		var update tele.Update
		if err := json.Unmarshal(data, &update); err != nil {
			p.log.Error("nats poller: unmarshal update",
				ports.F("err", err))
			return
		}

		select {
		case updates <- update:
		case <-stop:
			return
		default:
			// channel پر است — drop
			p.log.Info("nats poller: update dropped (channel full)")
		}
	})

	// منتظر stop signal
	<-stop
	p.log.Info("NATS poller stopped", ports.F("bot_id", p.botID))
}

// SetWebhook webhook URL را روی تلگرام تنظیم می‌کند.
// باید یک بار در startup صدا زده شود.
func SetWebhook(ctx context.Context, b *tele.Bot, gatewayURL, token string) error {
	webhook := &tele.Webhook{
		Listen:   gatewayURL + "/webhook/" + token,
		MaxConns: 40,
	}
	return b.SetWebhook(webhook)
}

// RemoveWebhook webhook را حذف می‌کند.
func RemoveWebhook(b *tele.Bot) error {
	return b.RemoveWebhook(false)
}
