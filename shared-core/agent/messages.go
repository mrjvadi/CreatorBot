// Package agent پیام‌های مشترکی را تعریف می‌کند که بین botmanager و agentmanager
// از طریق Centrifugo رد و بدل می‌شوند.
//
// هر دو طرف این package را import می‌کنند — نه internal یکدیگر را.
//
// کانال‌ها:
//   "server_<serverID>"  ← botmanager publish می‌کند، agentmanager subscribe می‌کند
//   "botmanager"         ← agentmanager publish می‌کند، botmanager subscribe می‌کند
package agent

import "time"

// ── پیام‌های دریافتی توسط botmanager (کانال "botmanager") ──────────────────

// MessageType نوع پیام ورودی به botmanager را مشخص می‌کند.
type MessageType string

const (
	TypeHeartbeat     MessageType = "heartbeat"
	TypeCommandResult MessageType = "command_result"
)

// Envelope پوشش مشترک همه پیام‌ها — برای dispatch بر اساس Type.
type Envelope struct {
	Type MessageType `json:"type"`
}

// HeartbeatPayload اطلاعات وضعیت سرور که agentmanager هر N ثانیه ارسال می‌کند.
type HeartbeatPayload struct {
	Type       MessageType       `json:"type"` // همیشه TypeHeartbeat
	ServerID   string            `json:"server_id"`
	TS         int64             `json:"ts"`
	Containers []ContainerStatus `json:"containers"`
}

// ContainerStatus وضعیت یک container Docker روی سرور.
type ContainerStatus struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`  // running | exited | paused | restarting
	Status string `json:"status"` // مثلاً "Up 2 hours"
}

// CommandResultPayload نتیجه اجرای یک دستور Docker که agentmanager ارسال می‌کند.
type CommandResultPayload struct {
	Type        MessageType `json:"type"` // همیشه TypeCommandResult
	CommandType string      `json:"command_type"`
	ServerID    string      `json:"server_id"`
	Container   string      `json:"container"`
	Success     bool        `json:"success"`
	Output      string      `json:"output,omitempty"`
	Error       string      `json:"error,omitempty"`
	Timestamp   int64       `json:"ts"`
}

// NewHeartbeat یک HeartbeatPayload جدید می‌سازد.
func NewHeartbeat(serverID string, containers []ContainerStatus) HeartbeatPayload {
	return HeartbeatPayload{
		Type:       TypeHeartbeat,
		ServerID:   serverID,
		TS:         time.Now().Unix(),
		Containers: containers,
	}
}

// NewCommandResult یک CommandResultPayload جدید می‌سازد.
func NewCommandResult(cmdType, serverID, container string, success bool, output, errMsg string) CommandResultPayload {
	return CommandResultPayload{
		Type:        TypeCommandResult,
		CommandType: cmdType,
		ServerID:    serverID,
		Container:   container,
		Success:     success,
		Output:      output,
		Error:       errMsg,
		Timestamp:   time.Now().Unix(),
	}
}
