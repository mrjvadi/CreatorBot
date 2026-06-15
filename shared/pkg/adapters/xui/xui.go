// Package xui implements ports.VPNPanel for 3x-ui and x-ui panels.
// API: https://github.com/MHSanaei/3x-ui
package xui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Panel implements ports.VPNPanel for 3x-ui.
type Panel struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	inboundID  int // ID اینباند پیش‌فرض
}

var _ ports.VPNPanel = (*Panel)(nil)

func New(baseURL, username, password string, inboundID int) *Panel {
	jar, _ := cookiejar.New(nil)
	return &Panel{
		baseURL:   strings.TrimRight(baseURL, "/"),
		username:  username,
		password:  password,
		inboundID: inboundID,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Jar:     jar, // 3x-ui از cookie session استفاده می‌کند
		},
	}
}

func (p *Panel) Name() string { return "xui" }

func (p *Panel) Login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", p.username)
	form.Set("password", p.password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/login",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("xui login: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("xui login failed: %s", result.Msg)
	}
	return nil
}

func (p *Panel) CreateUser(ctx context.Context, req ports.CreateVPNUserRequest) (*ports.VPNUser, error) {
	expiry := int64(0)
	if !req.ExpiresAt.IsZero() {
		expiry = req.ExpiresAt.UnixMilli()
	}

	// 3x-ui از client config استفاده می‌کند
	client := map[string]any{
		"id":         generateUUID(), // client UUID
		"email":      req.Username,
		"enable":     true,
		"expiryTime": expiry,
		"totalGB":    req.DataLimit / (1024 * 1024 * 1024),
		"flow":       "xtls-rprx-vision",
	}

	clients, _ := json.Marshal([]any{client})
	payload := map[string]any{
		"id":      p.inboundID,
		"settings": fmt.Sprintf(`{"clients":%s}`, string(clients)),
	}

	body, _ := json.Marshal(payload)
	resp, err := p.do(ctx, http.MethodPost, "/xui/inbound/addClient", body)
	if err != nil {
		return nil, fmt.Errorf("xui create client: %w", err)
	}

	var result xuiResponse
	json.Unmarshal(resp, &result)
	if !result.Success {
		return nil, fmt.Errorf("xui: %s", result.Msg)
	}

	// برگرداندن اطلاعات user
	return p.GetUser(ctx, req.Username)
}

func (p *Panel) GetUser(ctx context.Context, username string) (*ports.VPNUser, error) {
	resp, err := p.do(ctx, http.MethodGet,
		fmt.Sprintf("/xui/inbound/%d/clientTraffics/%s", p.inboundID, username), nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Success bool      `json:"success"`
		Obj     xuiClient `json:"obj"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.Obj.toVPNUser(), nil
}

func (p *Panel) UpdateUser(ctx context.Context, username string, req ports.UpdateVPNUserRequest) (*ports.VPNUser, error) {
	user, err := p.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}

	if req.DataLimit != nil {
		user.DataLimit = *req.DataLimit
	}
	if req.ExpiresAt != nil {
		user.ExpiresAt = *req.ExpiresAt
	}

	// آپدیت از طریق delete + create
	if err := p.DeleteUser(ctx, username); err != nil {
		return nil, err
	}
	return p.CreateUser(ctx, ports.CreateVPNUserRequest{
		Username:  username,
		DataLimit: user.DataLimit,
		ExpiresAt: user.ExpiresAt,
	})
}

func (p *Panel) EnableUser(ctx context.Context, username string) error {
	return p.setUserEnabled(ctx, username, true)
}

func (p *Panel) DisableUser(ctx context.Context, username string) error {
	return p.setUserEnabled(ctx, username, false)
}

func (p *Panel) setUserEnabled(ctx context.Context, username string, enable bool) error {
	payload := map[string]any{
		"id":      p.inboundID,
		"email":   username,
		"enable":  enable,
	}
	body, _ := json.Marshal(payload)
	_, err := p.do(ctx, http.MethodPost, "/xui/inbound/updateClientByEmail", body)
	return err
}

func (p *Panel) DeleteUser(ctx context.Context, username string) error {
	resp, err := p.do(ctx, http.MethodPost,
		fmt.Sprintf("/xui/inbound/%d/delClientByEmail/%s", p.inboundID, username), nil)
	if err != nil {
		return err
	}
	var result xuiResponse
	json.Unmarshal(resp, &result)
	if !result.Success {
		return fmt.Errorf("xui delete: %s", result.Msg)
	}
	return nil
}

func (p *Panel) ActiveCount(ctx context.Context) (int, error) {
	resp, err := p.do(ctx, http.MethodGet,
		fmt.Sprintf("/xui/inbound/getClientTraffics/%d", p.inboundID), nil)
	if err != nil {
		return 0, err
	}
	var result struct {
		Success bool        `json:"success"`
		Obj     []xuiClient `json:"obj"`
	}
	json.Unmarshal(resp, &result)
	active := 0
	for _, c := range result.Obj {
		if c.Enable {
			active++
		}
	}
	return active, nil
}

// ── HTTP helper ──────────────────────────────────────────────

func (p *Panel) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xui %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		// re-login
		if err := p.Login(ctx); err != nil {
			return nil, fmt.Errorf("xui re-login failed: %w", err)
		}
		// retry
		return p.do(ctx, method, path, body)
	}

	return io.ReadAll(resp.Body)
}

// ── types ────────────────────────────────────────────────────

type xuiResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

type xuiClient struct {
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`
	Total      int64  `json:"total"` // bytes
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
}

func (c *xuiClient) toVPNUser() *ports.VPNUser {
	status := ports.VPNUserDisabled
	if c.Enable {
		status = ports.VPNUserActive
	}
	var exp time.Time
	if c.ExpiryTime > 0 {
		exp = time.UnixMilli(c.ExpiryTime)
	}
	return &ports.VPNUser{
		Username:  c.Email,
		Status:    status,
		DataLimit: c.Total,
		UsedData:  c.Up + c.Down,
		ExpiresAt: exp,
	}
}

func generateUUID() string {
	// ساده‌ترین UUID v4 بدون dependency
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		time.Now().UnixNano()&0xFFFFFFFF,
		time.Now().UnixNano()&0xFFFF,
		(time.Now().UnixNano()&0x0FFF)|0x4000,
		(time.Now().UnixNano()&0x3FFF)|0x8000,
		time.Now().UnixNano()&0xFFFFFFFFFFFF,
	)
}
