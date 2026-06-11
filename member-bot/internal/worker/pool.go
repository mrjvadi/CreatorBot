package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

type Pool struct {
	mu      sync.Mutex
	workers []*BotWorker
	wg      sync.WaitGroup
}

func NewPool(workers []*BotWorker) *Pool {
	return &Pool{workers: workers}
}

func (p *Pool) Start(ctx context.Context) {
	for _, w := range p.workers {
		p.wg.Add(1)
		go func(w *BotWorker) {
			defer p.wg.Done()
			w.Run(ctx)
		}(w)
	}
	p.wg.Wait()
}

func (p *Pool) Add(ctx context.Context, w *BotWorker) {
	p.mu.Lock()
	p.workers = append(p.workers, w)
	p.mu.Unlock()
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		w.Run(ctx)
	}()
}

// MemberChecker is the interface for calling Telegram getChatMember.
// HTTPChecker is the default — no tele.Bot needed, just a token.
type MemberChecker interface {
	IsMember(ctx context.Context, channelID, userID int64) (bool, error)
}

// BotWorker processes membership check jobs from the Redis stream.
type BotWorker struct {
	BotID   string
	cache   ports.Cache
	checker MemberChecker
	limiter *rateLimiter
	log     ports.Logger
}

func NewBotWorker(botID string, ratePerSec int, cache ports.Cache, checker MemberChecker, log ports.Logger) *BotWorker {
	if ratePerSec <= 0 {
		ratePerSec = 20
	}
	return &BotWorker{
		BotID:   botID,
		cache:   cache,
		checker: checker,
		limiter: newRateLimiter(ratePerSec),
		log:     log,
	}
}

func (w *BotWorker) Run(ctx context.Context) {
	w.log.Info("worker started", ports.F("bot", w.BotID[:8]))
	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker stopped", ports.F("bot", w.BotID[:8]))
			return
		default:
		}

		msgs, err := w.cache.XReadGroup(ctx, ConsumerGroup, w.BotID, StreamKey, 1, 2*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(300 * time.Millisecond)
			continue
		}
		for _, msg := range msgs {
			w.handle(ctx, msg)
		}
	}
}

func (w *BotWorker) handle(ctx context.Context, msg ports.StreamMessage) {
	payload, ok := msg.Values["payload"].(string)
	if !ok {
		w.ack(ctx, msg.ID)
		return
	}

	var job CheckJob
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		log.Printf("[worker:%s] bad payload: %v", w.BotID[:8], err)
		w.ack(ctx, msg.ID)
		return
	}

	// 1. Am I a member of this channel?
	isMember, _ := w.cache.SIsMember(ctx, BotChannelKey(w.BotID), job.ChannelID)
	if !isMember {
		return // leave in PEL for another bot
	}

	// 2. Rate limit check
	if !w.limiter.Allow() {
		w.log.Info("rate-limited", ports.F("bot", w.BotID[:8]), ports.F("job", job.JobID))
		return
	}

	// 3. Race to claim
	claimed, err := TryClaim(ctx, w.cache, job.JobID, w.BotID)
	if err != nil || !claimed {
		w.ack(ctx, msg.ID)
		return
	}

	// 4. Call Telegram
	member, err := w.checker.IsMember(ctx, job.ChannelID, job.UserID)

	result := CheckResult{
		JobID:    job.JobID,
		IsMember: member,
		BotID:    w.BotID,
		ReplyKey: job.ReplyKey,
	}
	if err != nil {
		result.Err = err.Error()
	}

	// 5. Write result & ACK
	if wErr := WriteResult(ctx, w.cache, result); wErr != nil {
		w.log.Error("WriteResult failed", ports.F("err", wErr))
	}
	w.ack(ctx, msg.ID)
	w.log.Info("job done", ports.F("bot", w.BotID[:8]), ports.F("member", member))
}

func (w *BotWorker) ack(ctx context.Context, msgID string) {
	if err := w.cache.XAck(ctx, StreamKey, ConsumerGroup, msgID); err != nil {
		w.log.Error("XAck failed", ports.F("err", err))
	}
}
