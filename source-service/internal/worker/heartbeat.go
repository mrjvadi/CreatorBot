package worker

import (
	"context"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// StartHeartbeat publishes a liveness ping — on both this worker's own
// subject and botmanager's shared one (shared-core/protocol
// SubjSourceWorkerHeartbeat) — every interval, until ctx is done.
// PublishCore (not Publish) is used deliberately: this is a fire-and-forget
// signal, not something that needs JetStream persistence/replay.
func (i *Identity) StartHeartbeat(ctx context.Context, nc *natsclient.Client, log ports.Logger, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		start := time.Now()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				hb := protocol.SourceWorkerHeartbeat{
					ServiceID: i.ServiceID, ServiceKey: i.ServiceKey, WorkerID: i.ID,
					Status: "ok", UptimeSeconds: int(time.Since(start).Seconds()), Timestamp: time.Now().Unix(),
				}
				if err := nc.PublishCore(HeartbeatSubject(i.ID), hb); err != nil {
					log.Error("heartbeat publish", ports.F("err", err))
				}
				if err := nc.PublishCore(protocol.SubjSourceWorkerHeartbeat, hb); err != nil {
					log.Error("heartbeat publish (botmanager)", ports.F("err", err))
				}
			}
		}
	}()
}
