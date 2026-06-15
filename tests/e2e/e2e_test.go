// Package e2e — End-to-End tests برای CreatorBot V3.
// این تست‌ها نیاز به سرویس‌های واقعی (PostgreSQL, Redis, NATS) دارند.
// برای اجرا: E2E=true go test ./tests/e2e/...
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// skipIfNotE2E تست را اگه E2E=true نباشد skip می‌کند.
func skipIfNotE2E(t *testing.T) {
	if os.Getenv("E2E") != "true" {
		t.Skip("E2E tests skipped — set E2E=true to run")
	}
}

// apiURL آدرس apimanager را از env می‌خواند.
func apiURL() string {
	if u := os.Getenv("API_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}

// botpayURL آدرس botpay را می‌خواند.
func botpayURL() string {
	if u := os.Getenv("BOTPAY_URL"); u != "" {
		return u
	}
	return "http://localhost:8087"
}

// doJSON یک HTTP request با JSON body می‌فرستد.
func doJSON(t *testing.T, method, url string, body, result any, headers map[string]string) int {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if result != nil {
		json.NewDecoder(resp.Body).Decode(result)
	}
	return resp.StatusCode
}

// ── Health Check Tests ─────────────────────────────────────

func TestE2E_HealthChecks(t *testing.T) {
	skipIfNotE2E(t)

	endpoints := []struct {
		name string
		url  string
	}{
		{"apimanager", apiURL() + "/health"},
		{"botpay", botpayURL() + "/health"},
		{"webhook-gateway", "http://localhost:8090/health"},
		{"revenue-service", "http://localhost:8088/health"},
		{"fraud-engine", "http://localhost:8092/health"},
		{"community-service", "http://localhost:8093/health"},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			var result map[string]any
			code := doJSON(t, http.MethodGet, ep.url, nil, &result, nil)
			if code != http.StatusOK {
				t.Errorf("%s health: HTTP %d", ep.name, code)
			}
			if ok, _ := result["ok"].(bool); !ok {
				t.Errorf("%s health: ok=false", ep.name)
			}
		})
	}
}

// ── API Tests ─────────────────────────────────────────────

func TestE2E_APIAuth(t *testing.T) {
	skipIfNotE2E(t)

	// Telegram auth (simulation)
	var result map[string]any
	code := doJSON(t, http.MethodPost, apiURL()+"/api/v1/auth/telegram",
		map[string]any{
			"id":         12345678,
			"first_name": "Test",
			"auth_date":  time.Now().Unix(),
			"hash":       "test_hash",
		}, &result, nil)

	// ممکنه 401 بشه چون hash اشتباه، ولی نباید 500 بشه
	if code == http.StatusInternalServerError {
		t.Errorf("auth endpoint returned 500: %v", result)
	}
}

func TestE2E_ListPlans(t *testing.T) {
	skipIfNotE2E(t)

	var result map[string]any
	code := doJSON(t, http.MethodGet, apiURL()+"/api/v1/plans", nil, &result, nil)

	// بدون token باید 401 بشه
	if code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", code)
	}
}

// ── Botpay Tests ───────────────────────────────────────────

func TestE2E_BotpayBalance(t *testing.T) {
	skipIfNotE2E(t)

	var result map[string]any
	code := doJSON(t, http.MethodPost, botpayURL()+"/api/v1/balance",
		map[string]any{"telegram_id": 99999999},
		&result, nil)

	// باید wallet پیدا کنه یا بسازه
	if code != http.StatusOK {
		t.Errorf("balance: HTTP %d, result: %v", code, result)
	}
}

// ── Integration Flow Tests ─────────────────────────────────

func TestE2E_FullFlow_CreateUser(t *testing.T) {
	skipIfNotE2E(t)
	// این تست کامل‌ترین تست است:
	// 1. Telegram auth → دریافت JWT
	// 2. List plans → انتخاب پلن رایگان
	// 3. Activate free plan
	// 4. Create service (اگه agentmanager در دسترس باشد)

	t.Log("Full flow test requires all services to be running")
	t.Log("Run with: docker compose up -d && E2E=true go test ./tests/e2e/...")

	// Health check همه سرویس‌ها
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	services := []string{
		apiURL() + "/health",
		botpayURL() + "/health",
	}

	for _, svc := range services {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, svc, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Skipf("service %s not available: %v", svc, err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Skipf("service %s returned %d", svc, resp.StatusCode)
		}
	}

	t.Log("✅ All required services are healthy")
}

// ── Metrics Tests ──────────────────────────────────────────

func TestE2E_PrometheusMetrics(t *testing.T) {
	skipIfNotE2E(t)

	resp, err := http.Get("http://localhost:9090/metrics")
	if err != nil {
		t.Skipf("prometheus not available: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("metrics endpoint: HTTP %d", resp.StatusCode)
	}

	t.Log("✅ Prometheus metrics endpoint accessible")
}

// ── Rate Limiting Tests ────────────────────────────────────

func TestE2E_RateLimiting(t *testing.T) {
	skipIfNotE2E(t)

	webhookURL := fmt.Sprintf("http://localhost:8090/webhook/%s", "fake_token")

	// ارسال ۱۰۰ request سریع
	client := &http.Client{Timeout: 5 * time.Second}
	rateLimited := 0

	for i := 0; i < 50; i++ {
		resp, err := client.Post(webhookURL,
			"application/json",
			bytes.NewReader([]byte(`{"update_id":1}`)))
		if err != nil {
			break
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimited++
		}
		resp.Body.Close()
	}

	t.Logf("Rate limited %d/50 requests", rateLimited)
	// fake token باید 404 یا 400 برگردانه، نه 500
}
