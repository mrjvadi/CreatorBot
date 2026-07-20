// Package task defines the generic command envelope that turns
// source-service into a worker: every capability (extract members, watch a
// channel, run a bot command, ...) is one Handler registered under a task
// type string. Adding a new capability never touches the transport (NATS)
// code — it's a single Registry.Register call.
//
// Every instruction carries a correlation ID (Envelope.ID). Fast handlers
// just return their answer and it comes back as Result.ID = Envelope.ID —
// but slower/multi-step handlers (e.g. run_bot_command) can also report
// their real result later, out of band, to botmanager tagged with that same
// ID (see internal/botmanager.Report), so whoever's tracking the original
// instruction can match the eventual answer to it regardless of timing.
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Handler implements one task type. It receives the envelope's correlation
// ID (pass it along to internal/botmanager.Report if this handler reports
// results asynchronously) and the raw JSON payload, and returns any
// JSON-serializable result, or an error.
type Handler func(ctx context.Context, id string, payload json.RawMessage) (any, error)

// Envelope is what callers send, over NATS request-reply, to run a task.
//
//	{"id": "abc123", "type": "extract_members", "payload": {"channel": "@some_channel"}}
//
// ID is optional for fire-and-forget callers that only care about the
// synchronous Result; it's required if the task type reports results
// asynchronously to botmanager and you need to correlate that report.
type Envelope struct {
	ID         string          `json:"id,omitempty"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	ServiceID  string          `json:"service_id"`
	ServiceKey string          `json:"service_key"`
	TenantID   string          `json:"tenant_id"`
	IssuedAt   int64           `json:"issued_at"`
	Nonce      string          `json:"nonce"`
}

// Result is always what callers get back, on both success and failure —
// never a bare NATS/transport-level error. ID echoes Envelope.ID.
type Result struct {
	ID    string `json:"id,omitempty"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Type  string `json:"type,omitempty"`
	Data  any    `json:"data,omitempty"`
}

// Registry maps task type -> Handler. It is safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{handlers: make(map[string]Handler)}
}

// Register adds (or replaces) the handler for a task type. This is the only
// thing a new capability needs to do to become callable by workers:
//
//	reg.Register("extract_members", myHandlerFunc)
func (r *Registry) Register(taskType string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = h
}

// Types lists all registered task types, e.g. for a "capabilities" query.
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// Dispatch decodes an Envelope from data, runs the matching handler, and
// always returns a valid, marshaled Result — suitable for direct use as a
// NATS request-reply handler body.
func (r *Registry) Dispatch(ctx context.Context, data []byte) ([]byte, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return marshal(Result{OK: false, Error: "invalid task envelope: " + err.Error()})
	}

	r.mu.RLock()
	h, ok := r.handlers[env.Type]
	r.mu.RUnlock()
	if !ok {
		return marshal(Result{ID: env.ID, OK: false, Error: fmt.Sprintf("unknown task type %q", env.Type), Type: env.Type})
	}

	resultData, err := h(ctx, env.ID, env.Payload)
	if err != nil {
		return marshal(Result{ID: env.ID, OK: false, Error: err.Error(), Type: env.Type})
	}
	return marshal(Result{ID: env.ID, OK: true, Type: env.Type, Data: resultData})
}

func marshal(r Result) ([]byte, error) {
	return json.Marshal(r)
}
