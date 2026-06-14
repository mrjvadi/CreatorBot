// Package payclient کلاینت HTTP برای ارتباط با botpay.
// همه سرویس‌هایی که نیاز به عملیات مالی دارند از این کلاینت استفاده می‌کنند.
package payclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client کلاینت botpay API.
type Client struct {
	baseURL    string
	apiKey     string
	serviceID  string
	httpClient *http.Client
}

// Config تنظیمات کلاینت.
type Config struct {
	URL       string // http://botpay:8087
	APIKey    string // SERVICE_KEY_<SERVICE_ID>
	ServiceID string // مثلاً "botmanager"
}

func New(cfg Config) *Client {
	return &Client{
		baseURL:    cfg.URL,
		apiKey:     cfg.APIKey,
		serviceID:  cfg.ServiceID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Balance موجودی کاربر را برمی‌گرداند.
func (c *Client) Balance(ctx context.Context, telegramID int64) (*BalanceResp, error) {
	var resp BalanceResp
	if err := c.post(ctx, "/balance", map[string]any{"telegram_id": telegramID}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Deduct مبلغ را از کیف پول کاربر کسر می‌کند.
func (c *Client) Deduct(ctx context.Context, telegramID int64, amountTON float64, ref, desc string) (*DeductResp, error) {
	var resp DeductResp
	if err := c.post(ctx, "/deduct", map[string]any{
		"telegram_id": telegramID,
		"amount_ton":  amountTON,
		"ref":         ref,
		"description": desc,
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateInvoice فاکتور واریز می‌سازد.
// کاربر باید TON بفرستد با comment = invoice_code
func (c *Client) CreateInvoice(ctx context.Context, telegramID int64, amountTON float64, ref string) (*InvoiceResp, error) {
	var resp InvoiceResp
	if err := c.post(ctx, "/invoice/create", map[string]any{
		"telegram_id": telegramID,
		"amount_ton":  amountTON,
		"ref":         ref,
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AddCredit اعتبار به کاربر اضافه می‌کند (فقط با admin key).
func (c *Client) AddCredit(ctx context.Context, telegramID int64, amountTON float64, desc string) error {
	return c.post(ctx, "/credit/add", map[string]any{
		"telegram_id": telegramID,
		"amount_ton":  amountTON,
		"description": desc,
	}, nil)
}

// ── Response types ─────────────────────────────────────────

type BalanceResp struct {
	TelegramID int64   `json:"telegram_id"`
	TONBalance float64 `json:"ton_balance"`
	Credit     float64 `json:"credit"`
	Total      float64 `json:"total"`
	Frozen     float64 `json:"frozen"`
	TONAddress string  `json:"ton_address"`
}

type DeductResp struct {
	TxID      string  `json:"tx_id"`
	AmountTON float64 `json:"amount_ton"`
	Ref       string  `json:"ref"`
}

type InvoiceResp struct {
	InvoiceCode  string `json:"invoice_code"`
	PayURL       string `json:"pay_url"`
	AmountTON    float64 `json:"amount_ton"`
	ExpiresIn    string `json:"expires_in"`
	NATSSubject  string `json:"nats_subject"`
}

// ── internal ───────────────────────────────────────────────

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/pay"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Service-ID", c.serviceID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("botpay: %w", err)
	}
	defer resp.Body.Close()

	var envelope struct {
		OK      bool            `json:"ok"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("botpay decode: %w", err)
	}
	if !envelope.OK {
		if resp.StatusCode == http.StatusPaymentRequired {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("botpay error: %s", envelope.Message)
	}

	if result != nil && len(envelope.Data) > 0 {
		return json.Unmarshal(envelope.Data, result)
	}
	return nil
}

// DeductForService پرداخت برای ایجاد سرویس — ref = plan_id.
func (c *Client) DeductForService(ctx context.Context, telegramID int64, amountTON float64, planID string) (string, error) {
	resp, err := c.Deduct(ctx, telegramID, amountTON, "plan:"+planID, "خرید سرویس")
	if err != nil {
		return "", err
	}
	return resp.TxID, nil
}

// RefundService استرداد در صورت شکست provisioning.
func (c *Client) RefundService(ctx context.Context, telegramID int64, amountTON float64, invoiceCode string) error {
	return c.AddCredit(ctx, telegramID, amountTON, "استرداد: "+invoiceCode)
}

// ErrInsufficientBalance خطای موجودی ناکافی.
var ErrInsufficientBalance = fmt.Errorf("insufficient balance")

// IsInsufficientBalance بررسی نوع خطا.
func IsInsufficientBalance(err error) bool {
	return err == ErrInsufficientBalance
}
