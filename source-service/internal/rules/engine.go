package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/natsutil"
	"github.com/mrjvadi/creatorbot/source-service/internal/task"
	"github.com/mrjvadi/creatorbot/source-service/internal/telegram"
)

// telegramClient is the subset of *telegram.Client this engine needs.
type telegramClient interface {
	WatchChannelPosts(ctx context.Context, id, sourceUsername string, handler func(ctx context.Context, ev telegram.ChannelPostEvent)) error
	UnwatchChannelPosts(id string)
	ForwardMessage(ctx context.Context, sourceUsername, destUsername string, messageID int) (int, error)
	SendText(ctx context.Context, destUsername, text string) (int, error)
}

// Engine runs live rules: it subscribes to each rule's trigger and, when it
// fires and the rule's conditions pass, executes the rule's action.
type Engine struct {
	tg    telegramClient
	nc    *natsclient.Client
	tasks *task.Registry
	log   ports.Logger

	mu      sync.Mutex
	cleanup map[string]func()
}

func New(tg telegramClient, nc *natsclient.Client, tasks *task.Registry, log ports.Logger) *Engine {
	return &Engine{tg: tg, nc: nc, tasks: tasks, log: log, cleanup: make(map[string]func())}
}

// StartRule subscribes to r's trigger. Call this once per rule — at
// creation time, and again for every active rule on startup (see
// internal/userbot.RestoreRules).
func (e *Engine) StartRule(ctx context.Context, r StoredRule) error {
	switch r.TriggerType {
	case "channel_post":
		return e.startChannelPostRule(ctx, r)
	case "nats_message":
		return e.startNatsMessageRule(ctx, r)
	default:
		return fmt.Errorf("unknown trigger type %q", r.TriggerType)
	}
}

func (e *Engine) startChannelPostRule(ctx context.Context, r StoredRule) error {
	var cfg ChannelPostTrigger
	if err := json.Unmarshal(r.TriggerRaw, &cfg); err != nil {
		return fmt.Errorf("trigger config: %w", err)
	}
	if cfg.Channel == "" {
		return fmt.Errorf("trigger config: channel is required")
	}

	err := e.tg.WatchChannelPosts(ctx, r.ID, cfg.Channel, func(ctx context.Context, pe telegram.ChannelPostEvent) {
		e.handleEvent(ctx, r, Event{
			Text:          pe.Text,
			Sender:        pe.Sender,
			SourceChannel: cfg.Channel,
			MessageID:     pe.MessageID,
		})
	})
	if err != nil {
		return err
	}
	e.setCleanup(r.ID, func() { e.tg.UnwatchChannelPosts(r.ID) })
	return nil
}

func (e *Engine) startNatsMessageRule(ctx context.Context, r StoredRule) error {
	var cfg NatsMessageTrigger
	if err := json.Unmarshal(r.TriggerRaw, &cfg); err != nil {
		return fmt.Errorf("trigger config: %w", err)
	}
	if cfg.Subject == "" {
		return fmt.Errorf("trigger config: subject is required")
	}

	// natsutil.SubscribeRaw (not the shared client's wrapped Subscribe,
	// which discards its *nats.Subscription) so StopRule can actually
	// cancel this later.
	sub, err := natsutil.SubscribeRaw(e.nc, cfg.Subject, func(data []byte) {
		e.handleEvent(ctx, r, Event{Text: string(data), Subject: cfg.Subject})
	})
	if err != nil {
		return err
	}
	e.setCleanup(r.ID, func() { _ = sub.Unsubscribe() })
	return nil
}

// StopRule cancels a running rule's trigger subscription.
func (e *Engine) StopRule(id string) {
	e.mu.Lock()
	cleanup, ok := e.cleanup[id]
	delete(e.cleanup, id)
	e.mu.Unlock()
	if ok {
		cleanup()
	}
}

func (e *Engine) setCleanup(id string, fn func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanup[id] = fn
}

func (e *Engine) handleEvent(ctx context.Context, r StoredRule, ev Event) {
	if !evaluate(r.Conditions, ev) {
		return
	}
	if err := e.executeAction(ctx, r.Action, ev); err != nil {
		e.log.Error("rule action failed",
			ports.F("rule_id", r.ID), ports.F("trigger_type", r.TriggerType), ports.F("err", err))
	}
}
