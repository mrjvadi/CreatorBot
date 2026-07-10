// Package botmanager provides the generic, correlation-ID-based way workers
// report results back to botmanager — as opposed to a synchronous NATS
// reply. Some tasks (e.g. run_bot_command) take multiple steps and the
// "real" answer isn't ready by the time the synchronous task.Result goes
// back; instead, the worker reports it later, tagged with the same
// correlation ID the original instruction carried, so botmanager can match
// it up regardless of timing.
//
// Subject and payload shapes live in shared-core/protocol
// (SubjSourceWorkerUpdate / SourceWorkerUpdateRequest) — the same package
// botmanager imports — so this can't drift out of sync the way an
// assumption embedded only in source-service could.
package botmanager

import (
	"context"
	"fmt"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
)

// Config carries the ServiceID/ServiceKey Report authenticates with —
// same credential source-service uses to register (internal/worker.Config).
type Config struct {
	ServiceID  string
	ServiceKey string
}

// Report tells botmanager about a result tagged with id — the same id the
// originating instruction carried (task.Envelope.ID) — so botmanager can
// match this update to whatever asked for it, no matter how many steps or
// how long it took to produce.
func Report(ctx context.Context, nc *natsclient.Client, cfg Config, id string, tags map[string]any) error {
	req := protocol.SourceWorkerUpdateRequest{
		ServiceID:  cfg.ServiceID,
		ServiceKey: cfg.ServiceKey,
		ID:         id,
		Tags:       tags,
	}

	var reply protocol.SourceWorkerUpdateResponse
	if err := nc.Request(ctx, protocol.SubjSourceWorkerUpdate, req, &reply, 15*time.Second); err != nil {
		return fmt.Errorf("botmanager report request: %w", err)
	}
	if !reply.Success {
		return fmt.Errorf("botmanager rejected report: %s", reply.Error)
	}
	return nil
}
