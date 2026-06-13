// Package nats implements ports.Notifier and a subscriber using NATS JetStream.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Config تنظیمات اتصال به NATS.
type Config struct {
	URL      string // nats://localhost:4222
	// اختیاری — برای auth
	Username string
	Password string
}

// Client یک wrapper روی NATS connection است.
type Client struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// New اتصال به NATS برقرار می‌کند.
func New(cfg Config) (*Client, error) {
	opts := []nats.Option{
		nats.Name("creatorbot"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
	}
	if cfg.Username != "" && cfg.Password != "" {
		opts = append(opts, nats.UserInfo(cfg.Username, cfg.Password))
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats: connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}

	return &Client{nc: nc, js: js}, nil
}

// Publish یک پیام JSON به یک subject publish می‌کند.
func (c *Client) Publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.js.Publish(ctx, subject, data)
	return err
}

// PublishCore پیام را بدون JetStream (fire-and-forget) publish می‌کند.
func (c *Client) PublishCore(subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.nc.Publish(subject, data)
}

// Subscribe یک handler برای یک subject ثبت می‌کند.
// برای heartbeat و رویدادهای ساده استفاده می‌شود.
func (c *Client) Subscribe(subject string, handler func([]byte)) (*nats.Subscription, error) {
	return c.nc.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
}

// QueueSubscribe چند instance از یک سرویس می‌توانند روی یک queue باشند.
// هر پیام فقط به یک instance می‌رسد (load balancing).
func (c *Client) QueueSubscribe(subject, queue string, handler func([]byte)) (*nats.Subscription, error) {
	return c.nc.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		handler(msg.Data)
	})
}

// EnsureStream یک JetStream stream می‌سازد اگه وجود نداشته باشد.
func (c *Client) EnsureStream(ctx context.Context, name string, subjects []string) error {
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      name,
		Subjects:  subjects,
		Retention: jetstream.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		Storage:   jetstream.FileStorage,
	})
	return err
}

// JetStream برگرداندن JetStream برای استفاده مستقیم.
func (c *Client) JetStream() jetstream.JetStream { return c.js }

// NC برگرداندن connection خام.
func (c *Client) NC() *nats.Conn { return c.nc }

// Close اتصال را می‌بندد.
func (c *Client) Close() { c.nc.Close() }

// ── Dead Letter Queue ──────────────────────────────────────

const DLQSubject = "errors.dlq"

// DLQMessage پیامی که به DLQ رفته.
type DLQMessage struct {
	OriginalSubject string `json:"original_subject"`
	Payload         []byte `json:"payload"`
	Error           string `json:"error"`
	Attempts        int    `json:"attempts"`
	Timestamp       int64  `json:"timestamp"`
}

// PublishToDLQ پیام fail شده را به DLQ ارسال می‌کند.
func (c *Client) PublishToDLQ(subject string, payload []byte, err error, attempts int) {
	msg := DLQMessage{
		OriginalSubject: subject,
		Payload:         payload,
		Error:           err.Error(),
		Attempts:        attempts,
		Timestamp:       time.Now().Unix(),
	}
	c.PublishCore(DLQSubject, msg)
}

// SubscribeWithRetry با retry و DLQ subscribe می‌کند.
// maxRetries: تعداد تلاش مجدد قبل از ارسال به DLQ
func (c *Client) SubscribeWithRetry(subject string, maxRetries int, handler func([]byte) error) error {
	return c.Subscribe(subject, func(data []byte) {
		var lastErr error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if lastErr = handler(data); lastErr == nil {
				return // موفق
			}
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			}
		}
		// همه تلاش‌ها ناموفق → DLQ
		c.PublishToDLQ(subject, data, lastErr, maxRetries)
	})
}
