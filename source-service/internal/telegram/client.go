// Package telegram wraps gotd/td (MTProto) for the operations this worker
// exposes as tasks: extracting channel members, fetching a file from a bot
// and re-sending it, forwarding messages, and watching a channel in
// real time (watch.go) so new posts are auto-forwarded as they arrive.
//
// ⚠️ Real MTProto/UserBot code. Not compile- or runtime-verified in this
// environment: this session has no persisted Go module cache and no real
// Telegram session to authenticate with, so `go build`/`go vet` against the
// actual gotd/td v0.159.0 API could not be run here. The shapes below match
// gotd/td's documented usage patterns as of mid-2025; run `go build ./...`
// locally and treat any compile errors here as the next fix, not a sign the
// whole approach is wrong.
package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Config is one worker's Telegram identity. AppID/AppHash/Phone normally
// come from botmanager (see internal/worker.Identity.Telegram).
// SessionStorage is where the MTProto session is persisted — the real
// worker (cmd/service) always uses DBSessionStorage (Postgres, encrypted),
// since a Docker volume can disappear and force a re-login; the standalone
// cmd/login tool can use either DBSessionStorage or a plain
// session.FileStorage for quick offline testing.
type Config struct {
	AppID          int
	AppHash        string
	Phone          string
	SessionStorage session.Storage
}

// CodeSource supplies the login code Telegram sends when this account isn't
// authorized yet. In a headless worker there's no terminal to type it into,
// so this is implemented over NATS — see NATSCodeSource. cmd/login uses
// StdinCodeSource instead.
type CodeSource interface {
	Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error)
}

// Client is a running MTProto connection for one Telegram account.
type Client struct {
	cfg    Config
	client *telegram.Client
	api    *tg.Client
	code   CodeSource
	log    ports.Logger

	waiter     *messageWaiter
	watches    *channelWatches
	rawWatches *rawChannelWatches

	ready     chan struct{}
	readyOnce sync.Once
}

func New(cfg Config, code CodeSource, log ports.Logger) *Client {
	c := &Client{
		cfg:        cfg,
		code:       code,
		log:        log,
		waiter:     newMessageWaiter(),
		watches:    newChannelWatches(),
		rawWatches: newRawChannelWatches(),
		ready:      make(chan struct{}),
	}

	dispatcher := tg.NewUpdateDispatcher()
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		c.waiter.handle(u.Message)
		return nil
	})
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		c.handleChannelPost(ctx, u.Message)
		return nil
	})

	c.client = telegram.NewClient(cfg.AppID, cfg.AppHash, telegram.Options{
		SessionStorage: cfg.SessionStorage,
		UpdateHandler:  dispatcher,
	})
	c.api = c.client.API()
	return c
}

// Start connects, authorizes (requesting a login code if this account isn't
// authorized yet — see CodeSource), and then blocks — keeping the MTProto
// connection alive — until ctx is cancelled. Run this once in its own
// goroutine; other methods on Client (ExtractMembers, FetchEditSend,
// ForwardMessage, AddWatch, ...) use the same live connection and can be
// called concurrently once Ready() is closed.
func (c *Client) Start(ctx context.Context) error {
	return c.client.Run(ctx, func(ctx context.Context) error {
		if err := c.authorize(ctx); err != nil {
			return err
		}
		c.log.Info("telegram client authorized", ports.F("phone", c.cfg.Phone))
		c.readyOnce.Do(func() { close(c.ready) })
		<-ctx.Done()
		return ctx.Err()
	})
}

// Ready returns a channel that's closed once Start has connected and
// authorized successfully. Wait on this (e.g. before restoring persisted
// watches) rather than calling API methods immediately after launching
// Start in a goroutine.
func (c *Client) Ready() <-chan struct{} {
	return c.ready
}

// Login connects, authorizes if necessary, and returns as soon as the
// session is saved — it does not block waiting for ctx.Done(). Use this for
// one-off setup/recovery (see cmd/login) rather than running a worker,
// which should use Start instead.
func (c *Client) Login(ctx context.Context) error {
	return c.client.Run(ctx, func(ctx context.Context) error {
		if err := c.authorize(ctx); err != nil {
			return err
		}
		c.log.Info("telegram login complete, session saved", ports.F("phone", c.cfg.Phone))
		return nil
	})
}

func (c *Client) authorize(ctx context.Context) error {
	status, err := c.client.Auth().Status(ctx)
	if err != nil {
		return fmt.Errorf("auth status: %w", err)
	}
	if status.Authorized {
		return nil
	}
	flow := auth.NewFlow(auth.CodeOnly(c.cfg.Phone, c.code), auth.SendCodeOptions{})
	if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
		return fmt.Errorf("auth flow: %w", err)
	}
	return nil
}
