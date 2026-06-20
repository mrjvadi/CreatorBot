// Package natspayclient کلاینت مشترک همه‌ی سرویس‌ها برای ارتباط با botpay است.
// به‌جای HTTP از NATS request/reply استفاده می‌کند و موجودی را در Redis کش می‌کند.
//
// الگوی خواندن موجودی:
//  1. اول از Redis می‌خواند (سریع)
//  2. اگر نبود → NATS request به botpay → نتیجه را در Redis می‌گذارد
//
// موجودی فقط زمانی در Redis تغییر می‌کند که botpay یک پرداخت/واریز را تأیید کند.
package natspayclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
)

// ErrInsufficientBalance خطای موجودی ناکافی.
var ErrInsufficientBalance = fmt.Errorf("insufficient balance")

// IsInsufficientBalance بررسی نوع خطا.
func IsInsufficientBalance(err error) bool {
	return err == ErrInsufficientBalance
}

// Config تنظیمات کلاینت.
type Config struct {
	ServiceID  string // مثلا "botmanager"
	ServiceKey string // کلید احراز هویت این سرویس
	Timeout    time.Duration
	CacheTTL   time.Duration // مدت کش موجودی در Redis (پیش‌فرض 30s)
}

// Client کلاینت NATS برای pay.
type Client struct {
	nc    *natsclient.Client
	cache ports.Cache // اختیاری — اگر nil باشد، همیشه از NATS می‌خواند
	cfg   Config
}

// BalanceResp سازگار با payclient قدیمی.
type BalanceResp struct {
	TelegramID int64   `json:"telegram_id"`
	TONBalance float64 `json:"ton_balance"`
	Credit     float64 `json:"credit"`
	Total      float64 `json:"total"`
	Frozen     float64 `json:"frozen"`
	TONAddress string  `json:"ton_address"`
}

func New(nc *natsclient.Client, cache ports.Cache, cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3 * time.Second
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 30 * time.Second
	}
	return &Client{nc: nc, cache: cache, cfg: cfg}
}

func (c *Client) base(tgID int64) protocol.PayRequest {
	return protocol.PayRequest{
		ServiceID:  c.cfg.ServiceID,
		ServiceKey: c.cfg.ServiceKey,
		TelegramID: tgID,
	}
}

func cacheKey(tgID int64) string { return fmt.Sprintf("wallet:%d", tgID) }

// Balance موجودی کاربر را برمی‌گرداند. اول Redis، بعد NATS.
func (c *Client) Balance(ctx context.Context, telegramID int64) (*BalanceResp, error) {
	// ① تلاش از Redis
	if c.cache != nil {
		if raw, err := c.cache.Get(ctx, cacheKey(telegramID)); err == nil && raw != "" {
			var b BalanceResp
			if json.Unmarshal([]byte(raw), &b) == nil {
				return &b, nil
			}
		}
	}

	// ② از botpay با NATS request
	var resp protocol.BalanceResponse
	err := c.nc.Request(ctx, protocol.SubjPayBalance, protocol.BalanceRequest{
		PayRequest: c.base(telegramID),
	}, &resp, c.cfg.Timeout)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("pay: %s", resp.Error)
	}

	b := &BalanceResp{
		TelegramID: resp.TelegramID,
		TONBalance: resp.TONBalance,
		Credit:     resp.Credit,
		Total:      resp.Total,
		Frozen:     resp.Frozen,
		TONAddress: resp.TONAddress,
	}

	// ③ نوشتن در Redis برای دفعات بعد
	if c.cache != nil {
		if data, e := json.Marshal(b); e == nil {
			_ = c.cache.Set(ctx, cacheKey(telegramID), string(data), c.cfg.CacheTTL)
		}
	}
	return b, nil
}

// Authorize بررسی می‌کند این سرویس مجاز به دسترسی حساب کاربر است.
// قبل از اولین دسترسی یا اولین پرداخت صدا زده می‌شود.
func (c *Client) Authorize(ctx context.Context, telegramID int64) (bool, error) {
	var resp protocol.AuthorizeResponse
	err := c.nc.Request(ctx, protocol.SubjPayAuthorize, protocol.AuthorizeRequest{
		PayRequest: c.base(telegramID),
	}, &resp, c.cfg.Timeout)
	if err != nil {
		return false, err
	}
	if resp.Error != "" {
		return false, fmt.Errorf("pay: %s", resp.Error)
	}
	return resp.Authorized, nil
}

// Deduct کسر از حساب (پرداخت). idempotencyKey از کسر دوباره جلوگیری می‌کند.
func (c *Client) Deduct(ctx context.Context, telegramID int64, amountTON float64, reason, idempotencyKey string) (float64, error) {
	return c.DeductWithMeta(ctx, telegramID, amountTON, reason, idempotencyKey, "", "")
}

// DeductWithMeta نسخه‌ی کامل کسر با ref و metadata شفاف.
func (c *Client) DeductWithMeta(ctx context.Context, telegramID int64, amountTON float64, reason, idempotencyKey, ref, metadata string) (float64, error) {
	var resp protocol.DeductResponse
	err := c.nc.Request(ctx, protocol.SubjPayDeduct, protocol.DeductRequest{
		PayRequest:     c.base(telegramID),
		AmountTON:      amountTON,
		Reason:         reason,
		Ref:            ref,
		Metadata:       metadata,
		IdempotencyKey: idempotencyKey,
	}, &resp, c.cfg.Timeout)
	if err != nil {
		return 0, err
	}
	if resp.Error != "" {
		if resp.Error == "insufficient balance" {
			return 0, ErrInsufficientBalance
		}
		return 0, fmt.Errorf("pay: %s", resp.Error)
	}

	// موجودی تغییر کرد → کش را باطل کن (botpay منبع حقیقت است)
	if c.cache != nil {
		_ = c.cache.Del(ctx, cacheKey(telegramID))
	}
	return resp.NewBalance, nil
}

// ── سازگاری با payclient قدیمی ─────────────────────────────────

// SubscribePayCompleted به رویداد اتمام پرداخت برای این سرویس گوش می‌دهد.
// هر سرویس (مثلا یک instance اپلودر) با این می‌فهمد پرداخت کاربرش تمام شد.
func (c *Client) SubscribePayCompleted(handler func(protocol.PayCompletedEvent)) error {
	subject := protocol.PayCompletedSubject(c.cfg.ServiceID)
	return c.nc.Subscribe(subject, func(data []byte) {
		var ev protocol.PayCompletedEvent
		if json.Unmarshal(data, &ev) == nil {
			handler(ev)
		}
	})
}

// SubscribeWalletUpdates به رویداد wallet.updated گوش می‌دهد و با هر تغییر
// موجودی، کش Redis آن کاربر را باطل می‌کند. سرویس‌ها باید این را در startup
// صدا بزنند تا موجودی نمایش‌داده‌شده همیشه تازه باشد.
func (c *Client) SubscribeWalletUpdates() error {
	if c.cache == nil {
		return nil // بدون کش، نیازی به invalidation نیست
	}
	return c.nc.Subscribe(protocol.SubjWalletUpdated, func(data []byte) {
		var ev protocol.WalletUpdatedEvent
		if json.Unmarshal(data, &ev) != nil {
			return
		}
		_ = c.cache.Del(context.Background(), cacheKey(ev.TelegramID))
	})
}

func (c *Client) DeductForService(ctx context.Context, telegramID int64, amountTON float64, planID string) (string, error) {
	reason := "plan:" + planID
	meta := fmt.Sprintf(`{"plan_id":%q}`, planID)
	_, err := c.DeductWithMeta(ctx, telegramID, amountTON, reason, reason, planID, meta)
	if err != nil {
		return "", err
	}
	return reason, nil
}

// Credit افزودن اعتبار به کیف پول کاربر (پرداخت پلتفرم به کاربر، نه کسر).
// برای موارد مثل پاداش owner ربات رایگان یا استرداد. metadata شفاف برای ردیابی.
func (c *Client) Credit(ctx context.Context, telegramID int64, amountTON float64, ref, metadata string) error {
	var resp protocol.DeductResponse
	err := c.nc.Request(ctx, protocol.SubjPayCredit, protocol.DeductRequest{
		PayRequest: c.base(telegramID),
		AmountTON:  amountTON,
		Reason:     ref,
		Ref:        ref,
		Metadata:   metadata,
	}, &resp, c.cfg.Timeout)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("pay: %s", resp.Error)
	}
	if c.cache != nil {
		_ = c.cache.Del(ctx, cacheKey(telegramID))
	}
	return nil
}

// RefundService استرداد در صورت شکست provisioning (کسر منفی = افزودن).
func (c *Client) RefundService(ctx context.Context, telegramID int64, amountTON float64, ref string) error {
	return c.Credit(ctx, telegramID, amountTON, "refund:"+ref, "")
}
