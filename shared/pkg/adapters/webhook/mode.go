// mode.go — انتخاب خودکار polling یا webhook بر اساس محیط.
//
// محیط تست/توسعه:  BOT_MODE=polling  → مستقیم از تلگرام long-poll می‌کند (بدون gateway)
// محیط production:  BOT_MODE=webhook  → از NATS می‌خواند (webhook-gateway آن را feed می‌کند)
//
// اگر BOT_MODE خالی باشد، پیش‌فرض polling است (امن‌ترین حالت برای dev).
package webhook

import (
	"context"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Mode حالت دریافت update.
type Mode string

const (
	ModePolling Mode = "polling"
	ModeWebhook Mode = "webhook"
)

// BotIDFromToken شناسه‌ی عددی ربات را از توکن تلگرام استخراج می‌کند.
// توکن فرمت "<bot_id>:<hash>" دارد. در صورت خطا 0 برمی‌گرداند.
func BotIDFromToken(token string) int64 {
	idx := strings.IndexByte(token, ':')
	if idx <= 0 {
		return 0
	}
	id, err := strconv.ParseInt(token[:idx], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// ParseMode رشته‌ی env را به Mode تبدیل می‌کند. پیش‌فرض polling.
func ParseMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "webhook", "production", "prod":
		return ModeWebhook
	default:
		return ModePolling
	}
}

// PollerConfig تنظیمات لازم برای ساخت Poller.
type PollerConfig struct {
	Mode       Mode
	BotID      int64
	Token      string
	GatewayURL string            // فقط در حالت webhook لازم است
	NATS       *natsclient.Client // فقط در حالت webhook لازم است
	Log        ports.Logger
}

// BuildPoller بر اساس Mode، Poller مناسب را برمی‌گرداند.
//   - polling → tele.LongPoller (مستقیم از تلگرام)
//   - webhook → NATSPoller (از NATS؛ gateway آن را feed می‌کند)
func BuildPoller(cfg PollerConfig) tele.Poller {
	if cfg.Mode == ModeWebhook {
		cfg.Log.Info("bot mode: webhook (NATS poller)",
			ports.F("bot_id", cfg.BotID))
		return NewNATSPoller(cfg.NATS, cfg.BotID, cfg.Log)
	}
	cfg.Log.Info("bot mode: polling (long poll)")
	return &tele.LongPoller{Timeout: 10 * time.Second}
}

// Setup کارهای startup مخصوص هر mode را انجام می‌دهد.
// در حالت webhook: روی تلگرام SetWebhook می‌زند.
// در حالت polling: webhook قبلی را حذف می‌کند (تا تداخل نشود).
func Setup(ctx context.Context, b *tele.Bot, cfg PollerConfig) error {
	if cfg.Mode == ModeWebhook {
		return SetWebhook(ctx, b, cfg.GatewayURL, cfg.Token)
	}
	// در polling مطمئن می‌شویم webhook قبلی پاک شده (وگرنه تلگرام long-poll نمی‌دهد)
	_ = RemoveWebhook(b)
	return nil
}
