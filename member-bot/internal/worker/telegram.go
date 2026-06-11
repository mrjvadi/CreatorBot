// Package worker - HTTPChecker implements MemberChecker via direct Telegram Bot API call.
// No tele.Bot instance needed — just a token and net/http.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// HTTPChecker calls Telegram getChatMember directly over HTTP.
type HTTPChecker struct {
	token  string
	client *http.Client
}

func NewHTTPChecker(token string) *HTTPChecker {
	return &HTTPChecker{
		token:  token,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (h *HTTPChecker) IsMember(ctx context.Context, channelID, userID int64) (bool, error) {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember", h.token)

	params := url.Values{}
	params.Set("chat_id", strconv.FormatInt(channelID, 10))
	params.Set("user_id", strconv.FormatInt(userID, 10))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return false, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Status string `json:"status"`
		} `json:"result"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if !result.OK {
		return false, fmt.Errorf("telegram: %s", result.Description)
	}

	switch result.Result.Status {
	case "member", "administrator", "creator":
		return true, nil
	default:
		return false, nil
	}
}
