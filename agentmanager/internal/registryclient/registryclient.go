// Package registryclient کلاینت HTTP سرویس image-registry برای agentmanager
// است. جایگزین whitelist محلیِ قدیمی (env var ALLOWED_IMAGES با prefix
// matching) — حالا هر agentmanager قبل از deploy از یک منبع حقیقت مرکزی
// می‌پرسد «آیا این image:tag مجاز است؟».
//
// چرا HTTP و نه NATS (بر خلاف بقیه‌ی پلتفرم): image-registry بر اساس IP
// واقعیِ فراخوان تصمیم می‌گیرد — چیزی که فقط یک اتصال TCP مستقیم می‌تواند
// به‌طور قابل‌اعتماد فراهم کند، نه یک پیام NATS. رجوع به
// image-registry/README.md برای توضیح کامل.
package registryclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

// New یک کلاینت می‌سازد. baseURL خالی یعنی همه‌ی چک‌ها fail-closed رد
// می‌شوند (رجوع به IsAllowed) — دقیقاً همان رفتار قدیمیِ «whitelist خالی».
func New(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}
}

type checkResponse struct {
	OK   bool `json:"ok"`
	Data struct {
		Allowed bool `json:"allowed"`
	} `json:"data"`
	Message string `json:"message"`
}

// IsAllowed می‌پرسد آیا name:tag مجاز است. Fail-closed در همه‌ی حالت‌های
// خطا (baseURL خالی، شبکه قطع، پاسخ غیرمنتظره) — یعنی هر مشکلی در ارتباط
// با image-registry به معنای رد deploy است، نه اجازه‌ی پیش‌فرض. این عمداً
// همان فلسفه‌ی امنیتیِ «whitelist خالی/در‌دسترس‌نبودن = رد همه‌چیز» است که
// در بقیه‌ی پلتفرم هم استفاده شده.
func (c *Client) IsAllowed(ctx context.Context, name, tag string) (bool, error) {
	if c.baseURL == "" {
		return false, fmt.Errorf("image-registry: IMAGE_REGISTRY_URL not configured")
	}

	u := fmt.Sprintf("%s/v1/check?name=%s&tag=%s", c.baseURL, url.QueryEscape(name), url.QueryEscape(tag))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("image-registry unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return false, fmt.Errorf("image-registry: this server's IP is not allow-listed")
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("image-registry: unexpected status %d", resp.StatusCode)
	}

	var out checkResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, fmt.Errorf("image-registry: decode response: %w", err)
	}
	if !out.OK {
		return false, fmt.Errorf("image-registry: %s", out.Message)
	}
	return out.Data.Allowed, nil
}
