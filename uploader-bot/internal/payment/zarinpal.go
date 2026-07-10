package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Zarinpal درگاه زرین‌پال (API v4).
type Zarinpal struct {
	Merchant string
}

const (
	zpRequestURL = "https://api.zarinpal.com/pg/v4/payment/request.json"
	zpVerifyURL  = "https://api.zarinpal.com/pg/v4/payment/verify.json"
	zpStartURL   = "https://www.zarinpal.com/pg/StartPay/"
)

func (z *Zarinpal) Request(amount int64, desc, callback string) (string, string, error) {
	body := map[string]any{
		"merchant_id":  z.Merchant,
		"amount":       amount,
		"callback_url": callback,
		"description":  desc,
	}
	var resp struct {
		Data struct {
			Authority string `json:"authority"`
			Code      int    `json:"code"`
		} `json:"data"`
	}
	if err := zpPost(zpRequestURL, body, &resp); err != nil {
		return "", "", err
	}
	if resp.Data.Authority == "" {
		return "", "", fmt.Errorf("zarinpal: درخواست ناموفق (code=%d)", resp.Data.Code)
	}
	return resp.Data.Authority, zpStartURL + resp.Data.Authority, nil
}

func (z *Zarinpal) Verify(ref string, amount int64) (bool, string, error) {
	body := map[string]any{
		"merchant_id": z.Merchant,
		"amount":      amount,
		"authority":   ref,
	}
	var resp struct {
		Data struct {
			Code  int    `json:"code"`
			RefID int64  `json:"ref_id"`
			Msg   string `json:"message"`
		} `json:"data"`
	}
	if err := zpPost(zpVerifyURL, body, &resp); err != nil {
		return false, "", err
	}
	// 100 = موفق، 101 = قبلاً تایید شده
	if resp.Data.Code == 100 || resp.Data.Code == 101 {
		return true, fmt.Sprintf("%d", resp.Data.RefID), nil
	}
	return false, "", nil
}

func zpPost(url string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("zarinpal: encode request: %w", err)
	}
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("zarinpal: read response: %w", err)
	}
	return json.Unmarshal(data, out)
}
