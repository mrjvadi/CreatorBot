// Package zarinpal implements ports.PaymentGateway using the Zarinpal API.
// To swap to a different gateway: implement ports.PaymentGateway and wire in main.go.
package zarinpal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const (
	requestURL  = "https://api.zarinpal.com/pg/v4/payment/request.json"
	verifyURL   = "https://api.zarinpal.com/pg/v4/payment/verify.json"
	paymentPage = "https://www.zarinpal.com/pg/StartPay/"
)

// Gateway implements ports.PaymentGateway for Zarinpal.
type Gateway struct {
	merchantID string
	httpClient *http.Client
}

var _ ports.PaymentGateway = (*Gateway)(nil)

func New(merchantID string) *Gateway {
	return &Gateway{
		merchantID: merchantID,
		httpClient: &http.Client{},
	}
}

func (g *Gateway) Name() string { return "zarinpal" }

func (g *Gateway) CreatePayment(ctx context.Context, req ports.PaymentRequest) (*ports.PaymentResponse, error) {
	body, _ := json.Marshal(map[string]any{
		"merchant_id":  g.merchantID,
		"amount":       int(req.Amount),
		"description":  req.Description,
		"callback_url": req.CallbackURL,
		"metadata":     map[string]any{"order_id": req.OrderID},
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Code      int    `json:"code"`
			Authority string `json:"authority"`
			Message   string `json:"message"`
		} `json:"data"`
		Errors any `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Data.Code != 100 {
		return nil, fmt.Errorf("zarinpal: %s (code %d)", result.Data.Message, result.Data.Code)
	}

	return &ports.PaymentResponse{
		PaymentURL: paymentPage + result.Data.Authority,
		RefID:      result.Data.Authority,
	}, nil
}

func (g *Gateway) VerifyPayment(ctx context.Context, refID string) (*ports.VerifyResponse, error) {
	// refID here is "authority" from callback
	body, _ := json.Marshal(map[string]any{
		"merchant_id": g.merchantID,
		"authority":   refID,
		// amount must be stored and passed here — caller responsibility
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Code    int     `json:"code"`
			RefID   int64   `json:"ref_id"`
			Message string  `json:"message"`
			CardPAN string  `json:"card_pan"`
			Fee     float64 `json:"fee"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	success := result.Data.Code == 100 || result.Data.Code == 101
	return &ports.VerifyResponse{
		RefID:   fmt.Sprintf("%d", result.Data.RefID),
		Success: success,
	}, nil
}
