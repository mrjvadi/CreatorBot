// Package marzneshin implements ports.VPNPanel for MarzNeshin panel.
// API: https://github.com/marzneshin/marzneshin
// MarzNeshin از API مشابه Marzban استفاده می‌کند با تفاوت‌های جزئی.
package marzneshin

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

// Panel implements ports.VPNPanel for MarzNeshin.
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
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *Panel) Name() string { return "marzneshin" }

func (p *Panel) Login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", p.username)
	form.Set("password", p.password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/api/admins/token",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marzneshin login: %w", err)
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
	var expire *int64
	if !req.ExpiresAt.IsZero() {
		t := req.ExpiresAt.Unix()
		expire = &t
	}

	body, _ := json.Marshal(map[string]any{
		"username":   req.Username,
		"services":   []string{}, // از service های پنل استفاده می‌شه
		"data_limit": req.DataLimit,
		"expire":     expire,
		"status":     "active",
	})

	resp, err := p.do(ctx, http.MethodPost, "/api/users", body)
	if err != nil {
		return nil, fmt.Errorf("marzneshin create user: %w", err)
	}

	var user marzneshinUser
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, err
	}
	return user.toVPNUser(), nil
}

func (p *Panel) GetUser(ctx context.Context, username string) (*ports.VPNUser, error) {
	resp, err := p.do(ctx, http.MethodGet, "/api/users/"+username, nil)
	if err != nil {
		return nil, err
	}
	var user marzneshinUser
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, err
	}
	return user.toVPNUser(), nil
}

func (p *Panel) UpdateUser(ctx context.Context, username string, req ports.UpdateVPNUserRequest) (*ports.VPNUser, error) {
	payload := map[string]any{}
	if req.DataLimit != nil {
		payload["data_limit"] = *req.DataLimit
	}
	if req.ExpiresAt != nil {
		t := req.ExpiresAt.Unix()
		payload["expire"] = t
	}
	body, _ := json.Marshal(payload)
	resp, err := p.do(ctx, http.MethodPut, "/api/users/"+username, body)
	if err != nil {
		return nil, err
	}
	var user marzneshinUser
	json.Unmarshal(resp, &user)
	return user.toVPNUser(), nil
}

func (p *Panel) EnableUser(ctx context.Context, username string) error {
	body, _ := json.Marshal(map[string]any{"status": "active"})
	_, err := p.do(ctx, http.MethodPut, "/api/users/"+username, body)
	return err
}

func (p *Panel) DisableUser(ctx context.Context, username string) error {
	body, _ := json.Marshal(map[string]any{"status": "disabled"})
	_, err := p.do(ctx, http.MethodPut, "/api/users/"+username, body)
	return err
}

func (p *Panel) DeleteUser(ctx context.Context, username string) error {
	_, err := p.do(ctx, http.MethodDelete, "/api/users/"+username, nil)
	return err
}

func (p *Panel) ActiveCount(ctx context.Context) (int, error) {
	resp, err := p.do(ctx, http.MethodGet, "/api/users?status=active", nil)
	if err != nil {
		return 0, err
	}
	var result struct {
		Total int `json:"total"`
	}
	json.Unmarshal(resp, &result)
	return result.Total, nil
}

// ── HTTP helper ──────────────────────────────────────────────

func (p *Panel) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
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
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("marzneshin %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if err := p.Login(ctx); err != nil {
			return nil, fmt.Errorf("marzneshin re-login: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+p.token)
		return p.do(ctx, method, path, body)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("marzneshin %s %s: HTTP %d", method, path, resp.StatusCode)
	}

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

// ── Response types ───────────────────────────────────────────

type marzneshinUser struct {
	Username        string `json:"username"`
	Status          string `json:"status"`
	DataLimit       int64  `json:"data_limit"`
	UsedTraffic     int64  `json:"used_traffic"`
	Expire          *int64 `json:"expire"`
	SubscriptionURL string `json:"subscription_url"`
}

func (u *marzneshinUser) toVPNUser() *ports.VPNUser {
	status := ports.VPNUserDisabled
	switch u.Status {
	case "active":
		status = ports.VPNUserActive
	case "expired":
		status = ports.VPNUserExpired
	case "limited":
		status = ports.VPNUserLimited
	}

	var exp time.Time
	if u.Expire != nil && *u.Expire > 0 {
		exp = time.Unix(*u.Expire, 0)
	}

	links := []string{}
	if u.SubscriptionURL != "" {
		links = []string{u.SubscriptionURL}
	}

	return &ports.VPNUser{
		Username:  u.Username,
		Status:    status,
		DataLimit: u.DataLimit,
		UsedData:  u.UsedTraffic,
		ExpiresAt: exp,
		Links:     links,
	}
}
