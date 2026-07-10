// Package worker gives each source-service instance an identity so you can
// run many of them side by side. On startup, an instance activates a
// LICENSE_KEY with botmanager over NATS (identity.go), using the shared
// contract in github.com/mrjvadi/creatorbot/shared-core/protocol
// (SubjSourceWorkerRegister and friends — the same package botmanager
// itself imports, so this can't drift silently). It gets back a worker ID
// plus the Telegram credentials it should run as. That ID is then used to
// namespace this instance's own NATS subjects (subjects.go), so callers can
// either target one specific worker or drop a task into the shared pool for
// whichever worker is free. It also sends periodic liveness pings
// (heartbeat.go).
//
// source-service is a core-services-only internal tool, not a customer
// BotInstance — see README "دسترسی و اعتماد". SubjSourceWorkerRegister
// therefore requires a ServiceID/ServiceKey, the same trust pattern already
// used for SubjLicenseIssue/SubjPayCredit in shared-core/protocol.
package worker

import (
	"context"
	"fmt"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
)

// Config identifies this instance to botmanager.
type Config struct {
	// ServiceID/ServiceKey authenticate this call as coming from a trusted
	// core service (see shared-core/protocol source_worker.go doc comment).
	ServiceID  string
	ServiceKey string
	// LicenseKey selects which configured Telegram account/instance this
	// process should activate. Give every worker instance its own key to
	// run more than one — see README.
	LicenseKey string
}

// Identity is what this process is, once activated: a worker ID plus the
// Telegram account it should operate.
type Identity struct {
	ID       string
	Telegram protocol.SourceWorkerTelegramCreds
}

// Register activates cfg.LicenseKey with botmanager and returns this
// worker's assigned ID and Telegram credentials. Call this once at startup.
func Register(ctx context.Context, nc *natsclient.Client, cfg Config) (*Identity, error) {
	if cfg.LicenseKey == "" {
		return nil, fmt.Errorf("LICENSE_KEY is required")
	}

	req := protocol.SourceWorkerRegisterRequest{
		ServiceID:  cfg.ServiceID,
		ServiceKey: cfg.ServiceKey,
		LicenseKey: cfg.LicenseKey,
	}

	var reply protocol.SourceWorkerRegisterResponse
	if err := nc.Request(ctx, protocol.SubjSourceWorkerRegister, req, &reply, 15*time.Second); err != nil {
		return nil, fmt.Errorf("botmanager registration request: %w", err)
	}
	if !reply.Success {
		return nil, fmt.Errorf("botmanager rejected license: %s", reply.Error)
	}
	if reply.WorkerID == "" {
		return nil, fmt.Errorf("botmanager reply missing worker_id")
	}

	return &Identity{ID: reply.WorkerID, Telegram: reply.Telegram}, nil
}
