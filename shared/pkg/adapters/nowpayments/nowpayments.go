// Package nowpayments implements ports.PaymentGateway using the NOWPayments API.
package nowpayments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const baseURL = "https://api.nowpayments.io/v1"

// Gateway implements ports.PaymentGateway for NOWPayments (crypto).
type Gateway struct {
	apiKey     string
	httpClient *http.Client
}

var _ ports.PaymentGateway = (*Gateway)(nil)

func New(apiKey string) *Gateway {
	return &Gateway{apiKey: apiKey, httpClient: &http.Client{}}
}

func (g *Gateway) Name() string { return "nowpayments" }

func (g *Gateway) CreatePayment(ctx context.Context, req ports.PaymentRequest) (*ports.PaymentResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"price_amount":    req.Amount,
		"price_currency":  req.Currency,
		"pay_currency":    "TON",
		"order_id":        req.OrderID,
		"order_description": req.Description,
		"ipn_callback_url": req.CallbackURL,
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/payment", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", g.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		PaymentID      string `json:"payment_id"`
		PayAddress     string `json:"pay_address"`
		PayAmount      float64 `json:"pay_amount"`
		PayCurrency    string `json:"pay_currency"`
		PaymentStatus  string `json:"payment_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.PaymentID == "" {
		return nil, fmt.Errorf("nowpayments: empty payment_id")
	}

	return &ports.PaymentResponse{
		Address: result.PayAddress,
		RefID:   result.PaymentID,
	}, nil
}

func (g *Gateway) VerifyPayment(ctx context.Context, refID string) (*ports.VerifyResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/payment/"+refID, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		PaymentID     string  `json:"payment_id"`
		PaymentStatus string  `json:"payment_status"`
		ActuallyPaid  float64 `json:"actually_paid"`
		PayCurrency   string  `json:"pay_currency"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	success := result.PaymentStatus == "finished" || result.PaymentStatus == "confirmed"
	return &ports.VerifyResponse{
		RefID:    result.PaymentID,
		Amount:   result.ActuallyPaid,
		Currency: result.PayCurrency,
		Success:  success,
	}, nil
}
