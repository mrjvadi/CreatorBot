// Package docker — backward compat wrapper.
// botmanager از این package استفاده می‌کند.
// Manager حالا از طریق NATS publish می‌کند نه Centrifugo.
package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
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
	// serviceID/hmacSecret برای امضای envelope دستورها استفاده می‌شوند تا
	// agentmanager اصالت آن‌ها را تأیید کند. اگر hmacSecret خالی باشد (Manager
	// ساخته‌شده با NewManager) دستور بدون امضا می‌رود و یک agentmanager به‌روز
	// آن را رد می‌کند.
	serviceID  string
	hmacSecret string
}

// NewManager یک Manager بدون امضا می‌سازد (backward compat). دستورهای آن توسط
// agentmanagerِ به‌روز رد می‌شوند؛ فقط جایی استفاده شود که SERVICE_HMAC_SECRET
// در دسترس نیست.
func NewManager(nc *natsclient.Client) *Manager {
	return &Manager{nc: nc}
}

// NewSignedManager یک Manager می‌سازد که هر دستور را با HMAC امضا می‌کند.
// serviceID باید هویت مجازِ فرستنده باشد (مثلاً "botmanager" یا "apimanager")
// و hmacSecret همان SERVICE_HMAC_SECRET مشترک پلتفرم.
func NewSignedManager(nc *natsclient.Client, serviceID, hmacSecret string) *Manager {
	return &Manager{nc: nc, serviceID: serviceID, hmacSecret: hmacSecret}
}

// sign فیلدهای envelope auth را روی دستور پیش از publish پر می‌کند. اگر
// hmacSecret خالی باشد، دست‌نخورده می‌ماند (Manager بدون امضا).
func (m *Manager) sign(cmd *protocol.DeployCommand) {
	if m.hmacSecret == "" {
		return
	}
	cmd.ServiceID = m.serviceID
	cmd.IssuedAt = time.Now().Unix()
	cmd.Nonce = genNonce()
	cmd.ServiceKey = auth.ComputeServiceKey(m.hmacSecret, m.serviceID)
}

// genNonce یک nonce تصادفی ۱۶ بایتی (hex) می‌سازد.
func genNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Send یک DeployCommandِ از پیش‌ساخته را امضا و به سرور هدف ارسال می‌کند —
// برای publisher هایی که خودشان دستور را می‌سازند (مثل wizard خرید/تست
// سرویس) به‌جای دور زدن Manager با یک nc.Publish خام. Type دست‌نخورده می‌ماند
// (deploy یا start/stop/...). این تنها راهِ درست publish کردن deploy است تا
// همه‌ی دستورها امضا شوند.
func (m *Manager) Send(ctx context.Context, serverID string, cmd protocol.DeployCommand) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	cmd.ServerID = serverID
	m.sign(&cmd)
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), cmd)
}

// Deploy دستور deploy را به agentmanager ارسال می‌کند.
func (m *Manager) Deploy(ctx context.Context, serverID string, cmd protocol.DeployCommand) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	cmd.Type = protocol.MsgDeploy
	cmd.ServerID = serverID
	m.sign(&cmd)
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), cmd)
}

// Stop دستور stop را ارسال می‌کند.
func (m *Manager) Stop(ctx context.Context, serverID, containerID string) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	cmd := protocol.DeployCommand{
		Type:        protocol.MsgStop,
		ServerID:    serverID,
		ContainerID: containerID,
	}
	m.sign(&cmd)
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), cmd)
}

// Start دستور start را ارسال می‌کند.
func (m *Manager) Start(ctx context.Context, serverID, containerID string) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	cmd := protocol.DeployCommand{
		Type:        protocol.MsgStart,
		ServerID:    serverID,
		ContainerID: containerID,
	}
	m.sign(&cmd)
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), cmd)
}

// Restart دستور restart را ارسال می‌کند.
func (m *Manager) Restart(ctx context.Context, serverID, containerID string) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	cmd := protocol.DeployCommand{
		Type:        protocol.MsgRestart,
		ServerID:    serverID,
		ContainerID: containerID,
	}
	m.sign(&cmd)
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), cmd)
}

// Remove دستور remove را ارسال می‌کند.
func (m *Manager) Remove(ctx context.Context, serverID, containerID string) error {
	if m.nc == nil {
		return fmt.Errorf("NATS not configured")
	}
	cmd := protocol.DeployCommand{
		Type:        protocol.MsgRemove,
		ServerID:    serverID,
		ContainerID: containerID,
	}
	m.sign(&cmd)
	return m.nc.Publish(ctx, protocol.DeploySubject(serverID), cmd)
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
