// Package licenseclient کلاینت مشترک برای صحبت با license-service است —
// درست مثل natspayclient برای botpay، از NATS request/reply استفاده می‌کند.
package licenseclient

import (
	"context"
	"fmt"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Config تنظیمات کلاینت.
type Config struct {
	ServiceID  string // فقط برای Issue/Revoke لازم است: "agentmanager" یا "botmanager"
	ServiceKey string
	Timeout    time.Duration
}

type Client struct {
	nc  *natsclient.Client
	cfg Config
}

func New(nc *natsclient.Client, cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Client{nc: nc, cfg: cfg}
}

// Issue یک لایسنس تازه برای یک instance صادر می‌کند. فقط agentmanager/botmanager
// (که ServiceKey معتبر دارند) اجازه‌ی این کار را دارند.
func (c *Client) Issue(ctx context.Context, botID int64, instanceID, ownerID, serverID, planID string) (string, error) {
	var resp protocol.LicenseIssueResponse
	err := c.nc.Request(ctx, protocol.SubjLicenseIssue, protocol.LicenseIssueRequest{
		ServiceID:  c.cfg.ServiceID,
		ServiceKey: c.cfg.ServiceKey,
		BotID:      botID,
		InstanceID: instanceID,
		OwnerID:    ownerID,
		ServerID:   serverID,
		PlanID:     planID,
	}, &resp, c.cfg.Timeout)
	if err != nil {
		return "", fmt.Errorf("license issue: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("license issue: %s", resp.Error)
	}
	return resp.Token, nil
}

// Verify از خودِ bot instance صدا زده می‌شود — نیازی به ServiceKey ندارد
// (احراز آن با تطبیق خودِ token در سمت سرور انجام می‌شود).
func (c *Client) Verify(ctx context.Context, botID int64, token, serverID string) (valid bool, status string, cloneWarning bool, err error) {
	var resp protocol.LicenseVerifyResponse
	reqErr := c.nc.Request(ctx, protocol.SubjLicenseVerify, protocol.LicenseVerifyRequest{
		BotID:    botID,
		Token:    token,
		ServerID: serverID,
	}, &resp, c.cfg.Timeout)
	if reqErr != nil {
		return false, "", false, fmt.Errorf("license verify: %w", reqErr)
	}
	if resp.Error != "" {
		return false, resp.Status, false, fmt.Errorf("license verify: %s", resp.Error)
	}
	return resp.Valid, resp.Status, resp.CloneWarning, nil
}

// RequireValid در startup یک‌بار صدا زده می‌شود و fail-closed عمل می‌کند:
// اگر token خالی باشد، NATS وصل نباشد، یا license-service invalid برگرداند →
// خطا برمی‌گرداند و فراخوان باید bot را متوقف کند.
func RequireValid(ctx context.Context, nc *natsclient.Client, botID int64, token, serverID string) error {
	if token == "" {
		return fmt.Errorf("LICENSE_TOKEN is not set — bot cannot start without a valid license")
	}
	if nc == nil {
		return fmt.Errorf("NATS not connected — cannot verify license with license-service")
	}
	lc := New(nc, Config{Timeout: 15 * time.Second})
	valid, status, _, err := lc.Verify(ctx, botID, token, serverID)
	if err != nil {
		return fmt.Errorf("license verify failed: %w", err)
	}
	if !valid {
		return fmt.Errorf("license is not valid (status=%s)", status)
	}
	return nil
}

// RunLicenseLoop از داخل هر bot instance صدا زده می‌شود (uploader/vpn/archive/
// member) تا دوره‌ای به license-service بگوید «من هنوز روی همین سرورم».
// عمداً fail-open است — قطعی license-service یا شبکه هرگز یک ربات مشتری
// واقعی را متوقف نمی‌کند؛ فقط clone-warning/invalid را لاگ می‌کند تا تیم
// پلتفرم رسیدگی کند. token خالی یعنی این instance با نسخه‌ی قبل از
// license-service ساخته شده — فقط یک هشدار یک‌بار چاپ می‌شود، حلقه اجرا
// نمی‌شود.
func RunLicenseLoop(ctx context.Context, nc *natsclient.Client, botID int64, token, serverID string, log ports.Logger) {
	if token == "" {
		log.Warn("LICENSE_TOKEN not set — skipping license check-in", ports.F("bot_id", botID))
		return
	}
	lc := New(nc, Config{})
	check := func() {
		vctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		valid, status, cloneWarning, err := lc.Verify(vctx, botID, token, serverID)
		if err != nil {
			log.Warn("license check-in failed (fail-open, continuing)",
				ports.F("bot_id", botID), ports.F("err", err))
			return
		}
		if cloneWarning {
			log.Error("LICENSE CLONE WARNING — this instance_id checked in from an unexpected server",
				ports.F("bot_id", botID), ports.F("server_id", serverID))
		}
		if !valid {
			log.Error("license is not valid", ports.F("bot_id", botID), ports.F("status", status))
		}
	}
	check()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

// Revoke لایسنس یک instance را باطل می‌کند (فقط سرویس‌های مرکزی).
func (c *Client) Revoke(ctx context.Context, botID int64, reason string) error {
	var resp protocol.LicenseRevokeResponse
	err := c.nc.Request(ctx, protocol.SubjLicenseRevoke, protocol.LicenseRevokeRequest{
		ServiceID:  c.cfg.ServiceID,
		ServiceKey: c.cfg.ServiceKey,
		BotID:      botID,
		Reason:     reason,
	}, &resp, c.cfg.Timeout)
	if err != nil {
		return fmt.Errorf("license revoke: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("license revoke: %s", resp.Error)
	}
	return nil
}
