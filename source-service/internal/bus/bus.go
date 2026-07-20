// Package bus wires source-service's operations onto NATS: the archive
// file registry (see files.go) as request-reply, plus generic worker task
// dispatch (see internal/task and internal/worker) both targeted at one
// specific worker and load-balanced across a pool.
//
// This uses the real shared NATS client (github.com/mrjvadi/creatorbot/
// shared/pkg/adapters/nats.Client) — verified against the actual shared
// repo, not assumed. Its request-reply methods are Respond/QueueRespond
// (handler: func([]byte) (any, error), no ctx — Bus captures ctx via
// closure at Start time) and Request (auto JSON marshal/unmarshal via an
// `out` pointer). Fire-and-forget events use PublishCore (no JetStream
// stream required), not Publish.
//
// Handlers here never return a Go error from Respond/QueueRespond — the
// shared client would otherwise reply with its own generic {"error":"..."}
// shape instead of our FileEnvelope/task.Result. Failures are always
// encoded inside the returned envelope (see errEnvelope in files.go).
package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"

	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/source-service/internal/store"
	"github.com/mrjvadi/creatorbot/source-service/internal/task"
	"github.com/mrjvadi/creatorbot/source-service/internal/worker"
)

// Bus binds NATS subjects to store/cache operations (files.go) and to this
// worker's task registry.
type Bus struct {
	nc         *natsclient.Client
	cache      ports.Cache
	store      *store.Store
	log        ports.Logger
	tasks      *task.Registry
	workerID   string
	hmacSecret string
}

func New(nc *natsclient.Client, cache ports.Cache, st *store.Store, log ports.Logger, tasks *task.Registry, workerID, hmacSecret string) *Bus {
	return &Bus{nc: nc, cache: cache, store: st, log: log, tasks: tasks, workerID: workerID, hmacSecret: hmacSecret}
}

const requestAuthWindow = 5 * time.Minute

func (b *Bus) authorize(ctx context.Context, serviceID, serviceKey, tenantID string, issuedAt int64, nonce string) error {
	if b.hmacSecret == "" || !auth.ValidateServiceKey(b.hmacSecret, serviceID, serviceKey) {
		return fmt.Errorf("unauthorized")
	}
	if tenantID == "" || nonce == "" {
		return fmt.Errorf("tenant_id and nonce are required")
	}
	issued := time.Unix(issuedAt, 0)
	if issuedAt <= 0 || time.Since(issued) > requestAuthWindow || time.Until(issued) > 30*time.Second {
		return fmt.Errorf("request expired")
	}
	ok, err := b.cache.SetNX(ctx, "source:nonce:"+serviceID+":"+nonce, "1", requestAuthWindow)
	if err != nil {
		return fmt.Errorf("nonce store unavailable")
	}
	if !ok {
		return fmt.Errorf("replayed request")
	}
	return nil
}

// Start registers every subject handler — file registry, targeted worker
// tasks, and pool worker tasks — then blocks until ctx is done.
func (b *Bus) Start(ctx context.Context) error {
	if err := b.nc.Respond(SubjectFilesRegister, func(data []byte) (any, error) {
		return b.handleRegister(ctx, data)
	}); err != nil {
		return err
	}
	if err := b.nc.Respond(SubjectFilesGet, func(data []byte) (any, error) {
		return b.handleGet(ctx, data)
	}); err != nil {
		return err
	}
	if err := b.nc.Respond(SubjectFilesCache, func(data []byte) (any, error) {
		return b.handleCache(ctx, data)
	}); err != nil {
		return err
	}

	// Targeted: "worker #3, do this specific thing" (e.g. because only #3
	// is logged into the relevant Telegram account).
	if err := b.nc.Respond(worker.TasksSubject(b.workerID), b.dispatchTask(ctx)); err != nil {
		return err
	}
	// Pool: "any free worker, do this" — load-balanced via a queue group so
	// exactly one worker in the fleet handles each task.
	if err := b.nc.QueueRespond(worker.PoolTasksSubject, worker.PoolQueueGroup, b.dispatchTask(ctx)); err != nil {
		return err
	}

	b.log.Info("bus ready", ports.F("worker_id", b.workerID), ports.F("task_types", b.tasks.Types()))
	<-ctx.Done()
	return nil
}

// dispatchTask wraps task.Registry.Dispatch (which already returns
// marshaled JSON) as a Respond/QueueRespond handler. json.RawMessage passed
// as the `any` result makes the shared client's own json.Marshal call a
// no-op passthrough of those already-marshaled bytes.
func (b *Bus) dispatchTask(ctx context.Context) func(data []byte) (any, error) {
	return func(data []byte) (any, error) {
		var env task.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return task.Result{OK: false, Error: "invalid task envelope"}, nil
		}
		if err := b.authorize(ctx, env.ServiceID, env.ServiceKey, env.TenantID, env.IssuedAt, env.Nonce); err != nil {
			return task.Result{ID: env.ID, Type: env.Type, OK: false, Error: err.Error()}, nil
		}
		result, _ := b.tasks.Dispatch(ctx, data)
		return json.RawMessage(result), nil
	}
}
