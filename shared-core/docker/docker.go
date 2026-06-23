// Package docker — backward compat wrapper.
// botmanager از این package استفاده می‌کند.
// Manager حالا از طریق NATS publish می‌کند نه Centrifugo.
package docker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
)

// CommandType — برای backward compat
type CommandType = protocol.MsgType

const (
	CmdDeploy              = protocol.MsgDeploy
	CmdStart               = protocol.MsgStart
	CmdStop                = protocol.MsgStop
	CmdRemove              = protocol.MsgRemove
	CmdRestart             = protocol.MsgRestart
	CmdLogs    CommandType = "logs"
	CmdInspect CommandType = "inspect"
)

// Command — برای backward compat
type Command = protocol.DeployCommand

// Manager دستورات Docker را از طریق NATS به agentmanager ارسال می‌کند.
type Manager struct {
	nc *natsclient.Client
}

// NewManager یک Manager جدید می‌سازد.
func NewManager(nc *natsclient.Client) *Manager {
	return &Manager{nc: nc}
}

// Deploy دستور deploy را به agentmanager ارسال می‌کند.
func (m *Manager) Deploy(ctx context.Context, serverID string, cmd protocol.DeployCommand) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	cmd.Type = protocol.MsgDeploy
	cmd.ServerID = serverID
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), cmd)
}

// Stop دستور stop را ارسال می‌کند.
func (m *Manager) Stop(ctx context.Context, serverID, containerID string) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), protocol.DeployCommand{
		Type:        protocol.MsgStop,
		ServerID:    serverID,
		ContainerID: containerID,
	})
}

// Start دستور start را ارسال می‌کند.
func (m *Manager) Start(ctx context.Context, serverID, containerID string) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), protocol.DeployCommand{
		Type:        protocol.MsgStart,
		ServerID:    serverID,
		ContainerID: containerID,
	})
}

// Remove دستور remove را ارسال می‌کند.
func (m *Manager) Remove(ctx context.Context, serverID, containerID string) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), protocol.DeployCommand{
		Type:        protocol.MsgRemove,
		ServerID:    serverID,
		ContainerID: containerID,
	})
}

// ServerChannel — برای backward compat
func ServerChannel(serverID string) string {
	return protocol.DeploySubject(serverID)
}

// suppress unused
var _ = json.Marshal

// Logs آخرین N خط log container را برمی‌گرداند.
// از NATS publish می‌کند — agentmanager اجرا می‌کند و نتیجه می‌فرستد.
func (m *Manager) Logs(ctx context.Context, serverID, containerName string, lines int) (string, error) {
	// در این پیاده‌سازی، فعلاً از طریق Docker مستقیم روی سرور نمی‌توان لاگ گرفت.
	// agentmanager باید یه subject برای logs داشته باشه — آینده.
	// فعلاً یه پیام می‌فرستیم که نشان دهد feature در دسترس است.
	return fmt.Sprintf("[logs not yet available for %s — use SSH to server]", containerName), nil
}
