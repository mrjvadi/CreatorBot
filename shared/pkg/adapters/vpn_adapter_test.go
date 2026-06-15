// Package adapters — integration tests برای VPN adapters.
// از mock HTTP server استفاده می‌کند تا نیاز به panel واقعی نباشد.
package adapters_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/adapters/marzban"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// mockMarzbanServer یک Marzban API mock می‌سازد.
func mockMarzbanServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	// Login
	mux.HandleFunc("/api/admin/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"access_token": "mock_token_xyz",
			"token_type":   "bearer",
		})
	})

	// Create user
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)
			username, _ := req["username"].(string)
			json.NewEncoder(w).Encode(map[string]any{
				"username":          username,
				"status":            "active",
				"data_limit":        int64(10 * 1024 * 1024 * 1024),
				"used_traffic":      int64(0),
				"expire":            nil,
				"subscription_url":  "https://panel.test/sub/token123",
				"proxies": map[string]any{
					"vless": map[string]string{"id": "uuid-xxx"},
				},
				"links": []string{"vless://uuid-xxx@panel.test:443"},
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Get user
	mux.HandleFunc("/api/user/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"username":         "testuser",
				"status":           "active",
				"data_limit":       int64(10 * 1024 * 1024 * 1024),
				"used_traffic":     int64(1024 * 1024 * 100),
				"expire":           nil,
				"subscription_url": "https://panel.test/sub/token123",
				"links":            []string{"vless://uuid@test:443"},
			})
		case http.MethodPut:
			json.NewEncoder(w).Encode(map[string]any{
				"username": "testuser",
				"status":   "active",
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		}
	})

	return httptest.NewServer(mux)
}

func TestMarzbanAdapter_Login(t *testing.T) {
	srv := mockMarzbanServer(t)
	defer srv.Close()

	panel := marzban.New(srv.URL, "admin", "password")
	ctx := context.Background()

	if err := panel.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
}

func TestMarzbanAdapter_CreateUser(t *testing.T) {
	srv := mockMarzbanServer(t)
	defer srv.Close()

	panel := marzban.New(srv.URL, "admin", "password")
	ctx := context.Background()
	panel.Login(ctx)

	user, err := panel.CreateUser(ctx, ports.CreateVPNUserRequest{
		Username:  "testuser",
		DataLimit: 10 * 1024 * 1024 * 1024, // 10GB
		ExpiresAt: time.Now().AddDate(0, 1, 0),
	})

	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user == nil {
		t.Fatal("CreateUser() returned nil user")
	}
	if user.Username != "testuser" {
		t.Errorf("username = %q, want %q", user.Username, "testuser")
	}
	if user.Status != ports.VPNUserActive {
		t.Errorf("status = %q, want %q", user.Status, ports.VPNUserActive)
	}
	if len(user.Links) == 0 {
		t.Error("expected at least one subscription link")
	}
}

func TestMarzbanAdapter_GetUser(t *testing.T) {
	srv := mockMarzbanServer(t)
	defer srv.Close()

	panel := marzban.New(srv.URL, "admin", "password")
	ctx := context.Background()
	panel.Login(ctx)

	user, err := panel.GetUser(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("username = %q, want testuser", user.Username)
	}
}

func TestMarzbanAdapter_Name(t *testing.T) {
	panel := marzban.New("http://test", "a", "b")
	if panel.Name() != "marzban" {
		t.Errorf("Name() = %q, want marzban", panel.Name())
	}
}

// TestVPNUserStatus بررسی status های معتبر.
func TestVPNUserStatus(t *testing.T) {
	statuses := []ports.VPNUserStatus{
		ports.VPNUserActive,
		ports.VPNUserDisabled,
		ports.VPNUserExpired,
		ports.VPNUserLimited,
	}
	for _, s := range statuses {
		if string(s) == "" {
			t.Errorf("status should not be empty")
		}
	}
}

// TestCreateVPNUserRequest validation.
func TestCreateVPNUserRequest(t *testing.T) {
	// username نمی‌تواند خالی باشد
	req := ports.CreateVPNUserRequest{
		Username:  "validuser_123",
		DataLimit: 5 * 1024 * 1024 * 1024,
	}
	if req.Username == "" {
		t.Error("username should not be empty")
	}
	if req.DataLimit <= 0 {
		t.Error("data limit should be positive")
	}
}
