// Package ton مدیریت پرداخت با TON blockchain.
// از TON Pay API استفاده می‌شود — ساده‌ترین روش برای دریافت TON.
//
// Flow:
//  1. CreateInvoice → لینک پرداخت
//  2. کاربر لینک را باز می‌کند و در wallet خود پرداخت می‌کند
//  3. CheckPayment → بررسی دریافت تراکنش
//  4. تأیید و فعال‌سازی سرویس
package ton

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Config تنظیمات TON payment.
type Config struct {
	// WalletAddress آدرس کیف پول دریافت‌کننده (ادمین)
	WalletAddress string
	// APIKey کلید API سرویس TON (اختیاری برای چک خودکار)
	APIKey string
	// Network: mainnet یا testnet
	Network string
}

// Invoice یک فاکتور پرداخت.
type Invoice struct {
	ID        string  // شناسه یکتا (در comment تراکنش)
	Amount    float64 // مقدار TON
	PayURL    string  // لینک برای کاربر
	ExpiresAt time.Time
}

// Client مدیریت پرداخت TON.
type Client struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Client {
	return &Client{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateInvoice یک فاکتور پرداخت می‌سازد.
// کاربر باید amount TON به wallet بفرستد با comment = invoice.ID
func (c *Client) CreateInvoice(amount float64, description string) (*Invoice, error) {
	id := genInvoiceID()

	// لینک ton://transfer برای wallet app
	// فرمت استاندارد TON deep link
	payURL := fmt.Sprintf(
		"ton://transfer/%s?amount=%d&text=%s",
		c.cfg.WalletAddress,
		int64(amount*1e9), // nano TON
		id,
	)

	// همچنین لینک web wallet
	webURL := fmt.Sprintf(
		"https://app.tonkeeper.com/transfer/%s?amount=%d&text=%s",
		c.cfg.WalletAddress,
		int64(amount*1e9),
		id,
	)
	_ = webURL

	return &Invoice{
		ID:        id,
		Amount:    amount,
		PayURL:    payURL,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}, nil
}

// CheckPayment بررسی می‌کند آیا تراکنش با invoiceID دریافت شده.
// از TON API عمومی استفاده می‌کند.
func (c *Client) CheckPayment(ctx context.Context, invoiceID string, expectedAmount float64) (bool, string, error) {
	// استفاده از toncenter.com API
	baseURL := "https://toncenter.com/api/v2"
	if c.cfg.Network == "testnet" {
		baseURL = "https://testnet.toncenter.com/api/v2"
	}

	url := fmt.Sprintf("%s/getTransactions?address=%s&limit=20",
		baseURL, c.cfg.WalletAddress)
	if c.cfg.APIKey != "" {
		url += "&api_key=" + c.cfg.APIKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, "", err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("ton api: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result []struct {
			InMsg struct {
				Value   string `json:"value"`
				Message string `json:"message"`
				Source  string `json:"source"`
			} `json:"in_msg"`
			TransactionID struct {
				Hash string `json:"hash"`
			} `json:"transaction_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "", err
	}

	// بررسی تراکنش‌ها
	minNano := int64((expectedAmount - 0.01) * 1e9) // ۰.۰۱ TON tolerance
	for _, tx := range result.Result {
		// بررسی comment
		if tx.InMsg.Message != invoiceID {
			continue
		}
		// بررسی مقدار
		var nano int64
		fmt.Sscanf(tx.InMsg.Value, "%d", &nano)
		if nano >= minNano {
			return true, tx.TransactionID.Hash, nil
		}
	}

	return false, "", nil
}

func genInvoiceID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return "PAY" + hex.EncodeToString(b)
}
