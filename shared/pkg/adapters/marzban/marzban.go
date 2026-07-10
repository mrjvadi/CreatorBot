// Package marzban implements ports.VPNPanel for the Marzban panel.
// To add a new panel (e.g. Hiddify): implement ports.VPNPanel in a new package and wire in main.go.
package marzban

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Panel implements ports.VPNPanel for Marzban.
type Panel struct {
	baseURL    string
	username   string
	password   string
	token      string
	httpClient *http.Client
}

var _ ports.VPNPanel = (*Panel)(nil)

func New(baseURL, username, password string) *Panel {
	return &Panel{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Panel) Name() string { return "marzban" }

func (p *Panel) Login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", p.username)
	form.Set("password", p.password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/api/admin/token",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marzban login: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	p.token = result.AccessToken
	return nil
}

func (p *Panel) CreateUser(ctx context.Context, req ports.CreateVPNUserRequest) (*ports.VPNUser, error) {
	body, _ := json.Marshal(map[string]any{
		"username":                  req.Username,
		"proxies":                   map[string]any{"vless": map[string]any{}, "vmess": map[string]any{}},
		"data_limit":                req.DataLimit,
		"expire":                    unixOrZero(req.ExpiresAt),
		"data_limit_reset_strategy": "no_reset",
	})
	return p.doUserRequest(ctx, http.MethodPost, "/api/user", body)
}

func (p *Panel) GetUser(ctx context.Context, username string) (*ports.VPNUser, error) {
	return p.doUserRequest(ctx, http.MethodGet, "/api/user/"+username, nil)
}

func (p *Panel) UpdateUser(ctx context.Context, username string, req ports.UpdateVPNUserRequest) (*ports.VPNUser, error) {
	payload := map[string]any{}
	if req.DataLimit != nil {
		payload["data_limit"] = *req.DataLimit
	}
	if req.ExpiresAt != nil {
		payload["expire"] = unixOrZero(*req.ExpiresAt)
	}
	body, _ := json.Marshal(payload)
	return p.doUserRequest(ctx, http.MethodPut, "/api/user/"+username, body)
}

func (p *Panel) EnableUser(ctx context.Context, username string) error {
	body, _ := json.Marshal(map[string]any{"status": "active"})
	_, err := p.doUserRequest(ctx, http.MethodPut, "/api/user/"+username, body)
	return err
}

func (p *Panel) DisableUser(ctx context.Context, username string) error {
	body, _ := json.Marshal(map[string]any{"status": "disabled"})
	_, err := p.doUserRequest(ctx, http.MethodPut, "/api/user/"+username, body)
	return err
}

func (p *Panel) DeleteUser(ctx context.Context, username string) error {
	req, err := p.newAuthRequest(ctx, http.MethodDelete, "/api/user/"+username, nil)
	if err != nil {
		return err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (p *Panel) ActiveCount(ctx context.Context) (int, error) {
	req, err := p.newAuthRequest(ctx, http.MethodGet, "/api/system", nil)
	if err != nil {
		return 0, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		UsersActive int `json:"users_active"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.UsersActive, nil
}

// ---- helpers ----

type marzbanUser struct {
	Username    string   `json:"username"`
	Status      string   `json:"status"`
	DataLimit   int64    `json:"data_limit"`
	UsedTraffic int64    `json:"used_traffic"`
	Expire      *int64   `json:"expire"`
	Links       []string `json:"links"`
}

func (p *Panel) doUserRequest(ctx context.Context, method, path string, body []byte) (*ports.VPNUser, error) {
	req, err := p.newAuthRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// Token expired — re-login and retry once
		if err := p.Login(ctx); err != nil {
			return nil, fmt.Errorf("marzban: re-login failed: %w", err)
		}
		req, _ = p.newAuthRequest(ctx, method, path, body)
		resp, err = p.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	var mu marzbanUser
	if err := json.NewDecoder(resp.Body).Decode(&mu); err != nil {
		return nil, err
	}
	return toVPNUser(mu), nil
}

func (p *Panel) newAuthRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func toVPNUser(m marzbanUser) *ports.VPNUser {
	u := &ports.VPNUser{
		Username:  m.Username,
		Status:    ports.VPNUserStatus(m.Status),
		DataLimit: m.DataLimit,
		UsedData:  m.UsedTraffic,
		Links:     m.Links,
	}
	if m.Expire != nil && *m.Expire > 0 {
		t := time.Unix(*m.Expire, 0)
		u.ExpiresAt = t
	}
	return u
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
