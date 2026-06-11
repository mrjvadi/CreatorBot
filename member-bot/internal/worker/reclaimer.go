package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Reclaimer periodically re-enqueues stream messages that have been idle
// in the PEL longer than JobTimeout — meaning all eligible workers skipped them
// (rate-limited or not a member of the channel) or a worker crashed.
type Reclaimer struct {
	cache ports.Cache
	log   ports.Logger
}

func NewReclaimer(cache ports.Cache, log ports.Logger) *Reclaimer {
	return &Reclaimer{cache: cache, log: log}
}

// Run starts the reclaim loop in the background.
func (r *Reclaimer) Run(ctx context.Context) {
	ticker := time.NewTicker(JobTimeout)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.reclaim(ctx)
		}
	}
}

// reclaim reads idle PEL entries and re-injects them as fresh stream messages.
// We use XReadGroup with a synthetic "reclaimer" consumer to read pending entries
// and then re-push them so the full worker pool sees them again.
//
// Note: Full XAUTOCLAIM support requires the raw Redis client. Here we approximate
// by reading pending entries via XReadGroup with idle consumer "reclaimer".
// For production, expose XAUTOCLAIM through ports.Cache or use the raw client.
func (r *Reclaimer) reclaim(ctx context.Context) {
	// Read pending messages assigned to the reclaimer consumer
	msgs, err := r.cache.XReadGroup(ctx, ConsumerGroup, "reclaimer", StreamKey, 50, 0)
	if err != nil || len(msgs) == 0 {
		return
	}

	for _, msg := range msgs {
		payload, ok := msg.Values["payload"].(string)
		if !ok {
			r.cache.XAck(ctx, StreamKey, ConsumerGroup, msg.ID)
			continue
		}

		var job CheckJob
		if err := json.Unmarshal([]byte(payload), &job); err != nil {
			r.cache.XAck(ctx, StreamKey, ConsumerGroup, msg.ID)
			continue
		}

		// Check whether the job was already claimed
		alreadyClaimed, _ := r.cache.Exists(ctx, "memberbot:claim:"+job.JobID)
		if alreadyClaimed {
			r.cache.XAck(ctx, StreamKey, ConsumerGroup, msg.ID)
			continue
		}

		// Re-enqueue as a fresh message
		if err := Enqueue(ctx, r.cache, job); err != nil {
			r.log.Error("reclaimer: re-enqueue failed",
				ports.F("job", job.JobID), ports.F("err", err))
		} else {
			r.log.Info("reclaimer: re-enqueued abandoned job", ports.F("job", job.JobID))
			r.cache.XAck(ctx, StreamKey, ConsumerGroup, msg.ID)
		}
	}
}
