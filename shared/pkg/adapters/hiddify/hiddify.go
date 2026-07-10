// Package hiddify implements ports.VPNPanel for Hiddify panel.
// API docs: https://github.com/hiddify/hiddify-manager
package hiddify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Panel implements ports.VPNPanel for Hiddify Next.
type Panel struct {
	baseURL    string
	apiKey     string // Hiddify از API Key استفاده می‌کند نه user/pass
	adminPath  string // مسیر ادمین (مثل /b2fa2eac-9e9b/)
	httpClient *http.Client
}

var _ ports.VPNPanel = (*Panel)(nil)

// New یک Hiddify adapter جدید می‌سازد.
// adminPath: مسیر پنل ادمین (از URL پنل)
// apiKey: کلید API از تنظیمات پنل
func New(baseURL, adminPath, apiKey string) *Panel {
	return &Panel{
		baseURL:    strings.TrimRight(baseURL, "/"),
		adminPath:  strings.TrimRight(adminPath, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *Panel) Name() string { return "hiddify" }

// Login — Hiddify از API Key استفاده می‌کند، login لازم نیست.
func (p *Panel) Login(ctx context.Context) error { return nil }

func (p *Panel) CreateUser(ctx context.Context, req ports.CreateVPNUserRequest) (*ports.VPNUser, error) {
	var expireDay int
	if !req.ExpiresAt.IsZero() {
		expireDay = int(time.Until(req.ExpiresAt).Hours() / 24)
	}

	payload := map[string]any{
		"name":           req.Username,
		"comment":        "created by creatorbot",
		"package_days":   expireDay,
		"usage_limit_GB": float64(req.DataLimit) / 1e9,
		"enable":         true,
		"telegram_id":    nil,
		"added_by_uuid":  nil,
		"last_online":    nil,
	}

	body, _ := json.Marshal(payload)
	resp, err := p.do(ctx, http.MethodPost, "/api/v2/admin/user/", body)
	if err != nil {
		return nil, fmt.Errorf("hiddify create user: %w", err)
	}

	var result hiddifyUser
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.toVPNUser(), nil
}

func (p *Panel) GetUser(ctx context.Context, username string) (*ports.VPNUser, error) {
	resp, err := p.do(ctx, http.MethodGet, "/api/v2/admin/user/"+username+"/", nil)
	if err != nil {
		return nil, err
	}
	var result hiddifyUser
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return result.toVPNUser(), nil
}

func (p *Panel) UpdateUser(ctx context.Context, username string, req ports.UpdateVPNUserRequest) (*ports.VPNUser, error) {
	payload := map[string]any{}
	if req.DataLimit != nil {
		payload["usage_limit_GB"] = float64(*req.DataLimit) / 1e9
	}
	if req.ExpiresAt != nil {
		payload["package_days"] = int(time.Until(*req.ExpiresAt).Hours() / 24)
	}
	body, _ := json.Marshal(payload)
	resp, err := p.do(ctx, http.MethodPatch, "/api/v2/admin/user/"+username+"/", body)
	if err != nil {
		return nil, err
	}
	var result hiddifyUser
	json.Unmarshal(resp, &result)
	return result.toVPNUser(), nil
}

func (p *Panel) EnableUser(ctx context.Context, username string) error {
	body, _ := json.Marshal(map[string]any{"enable": true})
	_, err := p.do(ctx, http.MethodPatch, "/api/v2/admin/user/"+username+"/", body)
	return err
}

func (p *Panel) DisableUser(ctx context.Context, username string) error {
	body, _ := json.Marshal(map[string]any{"enable": false})
	_, err := p.do(ctx, http.MethodPatch, "/api/v2/admin/user/"+username+"/", body)
	return err
}

func (p *Panel) DeleteUser(ctx context.Context, username string) error {
	_, err := p.do(ctx, http.MethodDelete, "/api/v2/admin/user/"+username+"/", nil)
	return err
}

func (p *Panel) ActiveCount(ctx context.Context) (int, error) {
	resp, err := p.do(ctx, http.MethodGet, "/api/v2/admin/user/?enabled=true", nil)
	if err != nil {
		return 0, err
	}
	var users []hiddifyUser
	if err := json.Unmarshal(resp, &users); err != nil {
		return 0, err
	}
	return len(users), nil
}

// ── HTTP helper ──────────────────────────────────────────────

func (p *Panel) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := p.baseURL + p.adminPath + path
	var reader *strings.Reader
	if body != nil {
		reader = strings.NewReader(string(body))
	} else {
		reader = strings.NewReader("")
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Hiddify-API-Key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hiddify %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("hiddify %s %s: HTTP %d", method, path, resp.StatusCode)
	}

	var buf []byte
	buf = make([]byte, 0, 1024)
	tmp := make([]byte, 512)
	for {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return buf, nil
}

// ── Hiddify response types ────────────────────────────────────

type hiddifyUser struct {
	UUID           string   `json:"uuid"`
	Name           string   `json:"name"`
	Enable         bool     `json:"enable"`
	UsageLimitGB   float64  `json:"usage_limit_GB"`
	CurrentUsageGB float64  `json:"current_usage_GB"`
	PackageDays    int      `json:"package_days"`
	StartDate      *string  `json:"start_date"`
	Links          []string `json:"configs"`
}

func (u *hiddifyUser) toVPNUser() *ports.VPNUser {
	status := ports.VPNUserDisabled
	if u.Enable {
		status = ports.VPNUserActive
	}

	var exp time.Time
	if u.StartDate != nil && u.PackageDays > 0 {
		start, _ := time.Parse("2006-01-02", *u.StartDate)
		exp = start.AddDate(0, 0, u.PackageDays)
	}

	return &ports.VPNUser{
		Username:  u.Name,
		Status:    status,
		DataLimit: int64(u.UsageLimitGB * 1e9),
		UsedData:  int64(u.CurrentUsageGB * 1e9),
		ExpiresAt: exp,
		Links:     u.Links,
	}
}
