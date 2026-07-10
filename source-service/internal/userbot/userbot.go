// Package userbot registers this worker's Telegram-backed capabilities into
// a task.Registry. To add a new command: create a new file in this package
// with a handler method (task.Handler signature) and its payload/result
// types, then add one line in Register(). Nothing in internal/bus or
// main.go needs to change.
//
// Each existing capability lives in its own file:
//   - extract_members.go
//   - fetch_edit_send.go
//   - forward_message.go
//   - watch_channel.go / list_watches.go / remove_watch.go (real-time,
//     fixed "if a Telegram channel posts, forward to dest" rules)
//   - watch_nats.go / list_nats_watches.go / remove_nats_watch.go
//     (real-time, fixed "if a NATS subject gets a message, send it to
//     dest" rules)
//   - run_bot_command.go (run a bot command, register the resulting file,
//     report it to botmanager)
//   - create_rule.go / list_rules.go / delete_rule.go (generic
//     trigger+condition+action rules — see internal/rules — for
//     combinations the fixed watch_* tasks don't cover)
//
// restore_watches.go / restore_nats_watches.go / restore_rules.go reload
// persisted rules of all three kinds after a restart.
package userbot

import (
	"context"

	"github.com/google/uuid"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/botmanager"
	"github.com/mrjvadi/creatorbot/source-service/internal/models"
	"github.com/mrjvadi/creatorbot/source-service/internal/rules"
	"github.com/mrjvadi/creatorbot/source-service/internal/task"
	"github.com/mrjvadi/creatorbot/source-service/internal/telegram"
)

// telegramClient is the subset of *telegram.Client the handlers in this
// package (and the rules.Engine it builds) need. Declaring it here (rather
// than depending on the concrete type) keeps each handler file testable in
// isolation.
type telegramClient interface {
	ExtractMembers(ctx context.Context, channelUsername string) ([]telegram.Member, error)
	FetchEditSend(ctx context.Context, botUsername, command, newCaption, destUsername string) error
	FetchFromBot(ctx context.Context, botUsername, command string) (data []byte, fileName, mimeType string, replyMessageID int, err error)
	ForwardMessage(ctx context.Context, sourceUsername, destUsername string, messageID int) (int, error)
	SendText(ctx context.Context, destUsername, text string) (int, error)
	AddWatch(ctx context.Context, id, sourceUsername, destUsername string) error
	RemoveWatch(id string)
	ListWatches() []telegram.ChannelWatch
	WatchChannelPosts(ctx context.Context, id, sourceUsername string, handler func(ctx context.Context, ev telegram.ChannelPostEvent)) error
	UnwatchChannelPosts(id string)
}

// dataStore is the subset of *store.Store the handlers in this package
// need.
type dataStore interface {
	CreateArchiveFile(ctx context.Context, f *models.ArchiveFile) error
	CreateChannelWatch(ctx context.Context, w *models.ChannelWatch) error
	ListActiveChannelWatches(ctx context.Context, phone string) ([]models.ChannelWatch, error)
	DeactivateChannelWatch(ctx context.Context, id uuid.UUID) error
	CreateNatsWatch(ctx context.Context, w *models.NatsWatch) error
	ListActiveNatsWatches(ctx context.Context, phone string) ([]models.NatsWatch, error)
	DeactivateNatsWatch(ctx context.Context, id uuid.UUID) error
	CreateRule(ctx context.Context, r *models.Rule) error
	ListActiveRules(ctx context.Context, phone string) ([]models.Rule, error)
	DeactivateRule(ctx context.Context, id uuid.UUID) error
}

type Userbot struct {
	tg    telegramClient
	store dataStore
	nc    *natsclient.Client
	bmCfg botmanager.Config // ServiceID/ServiceKey used when reporting to botmanager
	log   ports.Logger
	phone string // which account's data (watches, rules, ...) belongs to this worker

	natsSubs *natsSubRegistry
	rules    *rules.Engine
}

// New builds a Userbot. tasks is the same registry passed to Register — the
// rules engine needs it too, so any task type can be used as a rule action
// (see internal/rules, action type "run_task").
func New(tg telegramClient, st dataStore, nc *natsclient.Client, bmCfg botmanager.Config, tasks *task.Registry, log ports.Logger, phone string) *Userbot {
	return &Userbot{
		tg:       tg,
		store:    st,
		nc:       nc,
		bmCfg:    bmCfg,
		log:      log,
		phone:    phone,
		natsSubs: newNatsSubRegistry(),
		rules:    rules.New(tg, nc, tasks, log),
	}
}

// Register wires every Telegram-backed task type into reg. This is the one
// place that lists what this worker knows how to do.
func (u *Userbot) Register(reg *task.Registry) {
	reg.Register("extract_members", u.handleExtractMembers)
	reg.Register("fetch_edit_send", u.handleFetchEditSend)
	reg.Register("forward_message", u.handleForwardMessage)
	reg.Register("watch_channel", u.handleWatchChannel)
	reg.Register("list_watches", u.handleListWatches)
	reg.Register("remove_watch", u.handleRemoveWatch)
	reg.Register("watch_nats", u.handleWatchNats)
	reg.Register("list_nats_watches", u.handleListNatsWatches)
	reg.Register("remove_nats_watch", u.handleRemoveNatsWatch)
	reg.Register("run_bot_command", u.handleRunBotCommand)
	reg.Register("create_rule", u.handleCreateRule)
	reg.Register("list_rules", u.handleListRules)
	reg.Register("delete_rule", u.handleDeleteRule)

	// To add a new capability: write handle_your_task.go with a
	// func(ctx, id string, json.RawMessage) (any, error) method, then:
	//   reg.Register("your_new_task", u.handleYourNewTask)
}
