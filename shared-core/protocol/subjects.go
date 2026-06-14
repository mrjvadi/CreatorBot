// Package protocol همه NATS subjects و message types را تعریف می‌کند.
//
// معماری:
//   هر bot یک engine کامل دارد — مستقیم به DB وصل است.
//   NATS فقط برای: deploy، heartbeat، رویدادهای cross-service
//
// Subjects:
//   deploy.{server_id}          ← apimanager → agentmanager (deploy bot)
//   agent.{server_id}.heartbeat ← agentmanager → apimanager
//   agent.{server_id}.result    ← agentmanager → apimanager (نتیجه دستور)
//   event.payment.{bot_id}      ← bot → apimanager (پرداخت تأیید شد)
//   event.instance.{bot_id}     ← apimanager → bot (تغییر وضعیت instance)
package protocol

import "fmt"

// ── Streams ────────────────────────────────────────────────

const (
	StreamDeploy = "DEPLOY"  // دستورات deploy/stop/remove
	StreamAgent  = "AGENT"   // heartbeat + نتایج
	StreamEvents = "EVENTS"  // رویدادهای cross-service
)

// ── Subjects ───────────────────────────────────────────────

// DeploySubject دستور deploy/stop/remove به یک agentmanager.
func DeploySubject(serverID string) string {
	return fmt.Sprintf("deploy.%s", serverID)
}

// HeartbeatSubject heartbeat از agentmanager به apimanager.
func HeartbeatSubject(serverID string) string {
	return fmt.Sprintf("agent.%s.heartbeat", serverID)
}

// ResultSubject نتیجه اجرای یک دستور Docker.
func ResultSubject(serverID string) string {
	return fmt.Sprintf("agent.%s.result", serverID)
}

// PaymentEventSubject پرداخت تأیید شده از bot به apimanager.
func PaymentEventSubject(botID int64) string {
	return fmt.Sprintf("event.payment.%d", botID)
}

// InstanceEventSubject تغییر وضعیت instance (از apimanager به bot).
func InstanceEventSubject(botID int64) string {
	return fmt.Sprintf("event.instance.%d", botID)
}

// ── Message Types ──────────────────────────────────────────

type MsgType string

const (
	// deploy
	MsgDeploy  MsgType = "deploy"
	MsgStop    MsgType = "stop"
	MsgRemove  MsgType = "remove"
	MsgStart   MsgType = "start"
	MsgRestart MsgType = "restart"

	// agent
	MsgHeartbeat MsgType = "heartbeat"
	MsgResult    MsgType = "result"

	// events
	MsgPaymentConfirmed MsgType = "payment_confirmed"
	MsgInstanceUpdated  MsgType = "instance_updated"
	MsgInstanceExpired  MsgType = "instance_expired"
)

// ── Deploy Command ─────────────────────────────────────────

// DeployCommand دستوری که apimanager به agentmanager می‌فرستد.
type DeployCommand struct {
	Type          MsgType           `json:"type"`
	ServerID      string            `json:"server_id"`
	ContainerName string            `json:"container_name"`
	ImageName     string            `json:"image_name"`
	ImageTag      string            `json:"image_tag"`
	EnvVars       map[string]string `json:"env_vars"`
	NetworkName   string            `json:"network_name,omitempty"`
	// ContainerID فقط برای stop/remove/restart لازم است
	ContainerID string `json:"container_id,omitempty"`
}

// ── Heartbeat ──────────────────────────────────────────────

// HeartbeatMsg وضعیت سرور که agentmanager هر N ثانیه ارسال می‌کند.
type HeartbeatMsg struct {
	Type       MsgType           `json:"type"`
	ServerID   string            `json:"server_id"`
	Timestamp  int64             `json:"ts"`
	Containers []ContainerStatus `json:"containers"`
}

// ContainerStatus وضعیت یک container.
type ContainerStatus struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`  // running | exited | paused
	Status string `json:"status"` // "Up 2 hours"
}

// ResultMsg نتیجه اجرای یک دستور Docker.
type ResultMsg struct {
	Type          MsgType `json:"type"`
	ServerID      string  `json:"server_id"`
	ContainerName string  `json:"container_name"`
	CommandType   string  `json:"command_type"`
	Success       bool    `json:"success"`
	Output        string  `json:"output,omitempty"`
	Error         string  `json:"error,omitempty"`
	Timestamp     int64   `json:"ts"`
}

// ── Events ─────────────────────────────────────────────────

// PaymentConfirmedEvent پرداخت تأیید شده.
// bot این رویداد را publish می‌کند تا apimanager رکورد را update کند.
type PaymentConfirmedEvent struct {
	Type       MsgType `json:"type"`
	BotID      int64   `json:"bot_id"`
	UserID     int64   `json:"user_id"`
	PlanID     string  `json:"plan_id"`
	Amount     float64 `json:"amount"`
	Gateway    string  `json:"gateway"`
	RefCode    string  `json:"ref_code"`
	Timestamp  int64   `json:"ts"`
}

// InstanceUpdatedEvent تغییر وضعیت instance از apimanager به bot.
// مثلاً انقضا، تمدید، تغییر تنظیمات.
type InstanceUpdatedEvent struct {
	Type       MsgType           `json:"type"`
	BotID      int64             `json:"bot_id"`
	ChangeType string            `json:"change_type"` // expired | renewed | settings_changed
	Data       map[string]any    `json:"data,omitempty"`
	Timestamp  int64             `json:"ts"`
}

// ── Service Provisioning Events ─────────────────────────────

const (
	// ServiceCreationRequested کاربر درخواست ایجاد سرویس داد.
	ServiceCreationRequested = "service.creation.requested"
	// ServiceCreationStarted agentmanager شروع به deploy کرد.
	ServiceCreationStarted = "service.creation.started"
	// ServiceCreationCompleted deploy موفق.
	ServiceCreationCompleted = "service.creation.completed"
	// ServiceCreationFailed deploy ناموفق → refund باید صورت گیرد.
	ServiceCreationFailed = "service.creation.failed"
	// ServiceStatusChanged تغییر وضعیت سرویس.
	ServiceStatusChanged = "service.status.changed"
)

// ServiceProvisionPayload payload رویداد provisioning.
type ServiceProvisionPayload struct {
	InstanceID  string `json:"instance_id"`
	OwnerID     string `json:"owner_id"`
	ServiceType string `json:"service_type"`
	PlanID      string `json:"plan_id"`
	BotToken    string `json:"bot_token,omitempty"`
	ServerID    string `json:"server_id,omitempty"`
	InvoiceCode string `json:"invoice_code,omitempty"`
	AmountNano  int64  `json:"amount_nano,omitempty"`
	Error       string `json:"error,omitempty"`
}
