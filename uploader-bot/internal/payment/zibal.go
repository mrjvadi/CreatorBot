package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Zibal درگاه زیبال.
type Zibal struct {
	Merchant string
}

const (
	zbRequestURL = "https://gateway.zibal.ir/v1/request"
	zbVerifyURL  = "https://gateway.zibal.ir/v1/verify"
	zbStartURL   = "https://gateway.zibal.ir/start/"
)

func (z *Zibal) Request(amount int64, desc, callback string) (string, string, error) {
	body := map[string]any{
		"merchant":    z.Merchant,
		"amount":      amount,
		"callbackUrl": callback,
		"description": desc,
	}
	var resp struct {
		Result  int    `json:"result"`
		TrackID int64  `json:"trackId"`
		Message string `json:"message"`
	}
	if err := zbPost(zbRequestURL, body, &resp); err != nil {
		return "", "", err
	}
	if resp.Result != 100 || resp.TrackID == 0 {
		return "", "", fmt.Errorf("zibal: درخواست ناموفق (result=%d)", resp.Result)
	}
	track := fmt.Sprintf("%d", resp.TrackID)
	return track, zbStartURL + track, nil
}

func (z *Zibal) Verify(ref string, amount int64) (bool, string, error) {
	body := map[string]any{
		"merchant": z.Merchant,
		"trackId":  ref,
	}
	var resp struct {
		Result    int    `json:"result"`
		RefNumber any    `json:"refNumber"`
		Message   string `json:"message"`
	}
	if err := zbPost(zbVerifyURL, body, &resp); err != nil {
		return false, "", err
	}
	// 100 = موفق، 201 = قبلاً تایید شده
	if resp.Result == 100 || resp.Result == 201 {
		return true, fmt.Sprintf("%v", resp.RefNumber), nil
	}
	return false, "", nil
}

func zbPost(url string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("zibal: encode request: %w", err)
	}
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("zibal: read response: %w", err)
	}
	return json.Unmarshal(data, out)
}
