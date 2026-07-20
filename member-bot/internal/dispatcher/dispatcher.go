// Package dispatcher loads CheckBots from DB, syncs channel memberships to Redis,
// and manages the worker pool lifecycle.
package dispatcher

import (
	"context"
	"time"

	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
	"github.com/mrjvadi/creatorbot/member-bot/internal/store"
	"github.com/mrjvadi/creatorbot/member-bot/internal/worker"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Dispatcher struct {
	st         *store.Store
	cache      ports.Cache
	log        ports.Logger
	encryptKey string
	pool       *worker.Pool
	balancer   *Balancer
}

func New(st *store.Store, cache ports.Cache, log ports.Logger, encryptKey string) *Dispatcher {
	return &Dispatcher{
		st: st, cache: cache,
		log: log, encryptKey: encryptKey,
		balancer: NewBalancer(st, cache, log),
	}
}

func (d *Dispatcher) Start(ctx context.Context) error {
	if err := worker.EnsureGroup(ctx, d.cache); err != nil {
		return err
	}

	bots, err := d.loadBots(ctx)
	if err != nil {
		return err
	}
	d.log.Info("dispatcher: loaded bots", ports.F("count", len(bots)))

	if err := d.syncMemberships(ctx, bots); err != nil {
		d.log.Error("dispatcher: initial membership sync failed", ports.F("err", err))
	}

	workers, err := d.buildWorkers(bots)
	if err != nil {
		return err
	}

	// channel assignment با load balance
	if err := d.balancer.Assign(ctx); err != nil {
		d.log.Error("initial channel assignment failed", ports.F("err", err))
	}

	go d.syncLoop(ctx)

	reclaimer := worker.NewReclaimer(d.cache, d.log)
	go reclaimer.Run(ctx)

	d.pool = worker.NewPool(workers)
	d.pool.Start(ctx)
	return nil
}

func (d *Dispatcher) AddBot(ctx context.Context, bot models.CheckBot) error {
	w, err := d.buildWorker(bot)
	if err != nil {
		return err
	}
	if err := d.syncBotMembership(ctx, bot); err != nil {
		d.log.Error("AddBot: sync failed", ports.F("err", err))
	}
	d.pool.Add(ctx, w)
	d.log.Info("dispatcher: added bot", ports.F("bot", bot.ID))
	return nil
}

func (d *Dispatcher) loadBots(ctx context.Context) ([]models.CheckBot, error) {
	return d.st.FindActiveBots(ctx)
}

func (d *Dispatcher) buildWorkers(bots []models.CheckBot) ([]*worker.BotWorker, error) {
	var workers []*worker.BotWorker
	for _, bot := range bots {
		w, err := d.buildWorker(bot)
		if err != nil {
			d.log.Error("buildWorker failed", ports.F("bot", bot.ID), ports.F("err", err))
			continue
		}
		workers = append(workers, w)
	}
	return workers, nil
}

func (d *Dispatcher) buildWorker(bot models.CheckBot) (*worker.BotWorker, error) {
	// FIX 3: use decrypted token directly with lightweight HTTP checker
	// no need to create a full tele.Bot (which requires a poller) for each check-bot
	token, err := auth.Decrypt(bot.Token, d.encryptKey)
	if err != nil {
		return nil, err
	}
	checker := worker.NewHTTPChecker(token)
	return worker.NewBotWorker(bot.ID.String(), bot.RateLimit, d.cache, checker, d.log), nil
}

func (d *Dispatcher) syncMemberships(ctx context.Context, bots []models.CheckBot) error {
	for _, bot := range bots {
		if err := d.syncBotMembership(ctx, bot); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) syncBotMembership(ctx context.Context, bot models.CheckBot) error {
	key := worker.BotChannelKey(bot.ID.String())
	d.cache.Del(ctx, key)
	if len(bot.Memberships) == 0 {
		return nil
	}
	members := make([]any, len(bot.Memberships))
	for i, m := range bot.Memberships {
		members[i] = m.ChannelID
	}
	if err := d.cache.SAdd(ctx, key, members...); err != nil {
		return err
	}
	return d.cache.Set(ctx, key+"_ttl", "1", 10*time.Minute)
}

func (d *Dispatcher) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bots, err := d.loadBots(ctx)
			if err != nil {
				d.log.Error("syncLoop: loadBots failed", ports.F("err", err))
				continue
			}
			if err := d.syncMemberships(ctx, bots); err != nil {
				d.log.Error("syncLoop: sync failed", ports.F("err", err))
			}
			d.log.Info("dispatcher: sync done", ports.F("bots", len(bots)))
		}
	}
}
