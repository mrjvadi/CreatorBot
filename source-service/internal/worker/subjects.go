package worker

import "fmt"

// Registration and heartbeat-to-botmanager subjects live in
// shared-core/protocol (SubjSourceWorkerRegister, SubjSourceWorkerHeartbeat,
// SubjSourceWorkerUpdate) since botmanager needs the same contract — see
// identity.go/heartbeat.go. The subjects below stay local to source-service:
// only core services call worker.<id>.tasks / worker.pool.tasks directly,
// and that's expected to be restricted at the NATS account/permission
// level, not via an app-level shared secret on every call.
const (
	// PoolTasksSubject is shared by every worker via a queue group: publish
	// a task here and whichever worker is free picks it up.
	PoolTasksSubject = "worker.pool.tasks"
	// PoolQueueGroup is the NATS queue group name backing PoolTasksSubject.
	PoolQueueGroup = "workers"
)

// TasksSubject addresses one specific worker by ID — use this when the task
// must run on this exact instance (e.g. it's the one logged into the
// relevant Telegram account).
func TasksSubject(id string) string { return fmt.Sprintf("worker.%s.tasks", id) }

// HeartbeatSubject is where this worker publishes its own liveness pings.
func HeartbeatSubject(id string) string { return fmt.Sprintf("worker.%s.heartbeat", id) }

// AuthCodeSubject is where this worker requests the Telegram login code
// during initial authentication (see internal/telegram.NATSCodeSource).
func AuthCodeSubject(id string) string { return fmt.Sprintf("worker.%s.auth.code", id) }
