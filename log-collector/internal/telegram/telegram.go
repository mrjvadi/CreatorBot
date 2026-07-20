// Package telegram هشدار لاگ‌ها را به یک سوپرگروه فوروم تلگرام می‌فرستد —
// هر سرویس یک topic اختصاصی می‌گیرد (اولین لاگ آن سرویس topic را می‌سازد).
// عمداً با HTTP خام به Bot API پیاده شده (نه یک کتابخانه‌ی ربات کامل) چون
// فقط به دو متد نیاز داریم: createForumTopic و sendMessage.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Notifier یک کلاینت سبک برای Bot API.
type Notifier struct {
	baseURL string // مثلاً https://api.telegram.org یا یک local bot API
	token   string
	chatID  int64 // شناسه‌ی سوپرگروه فوروم (باید Forum فعال باشد)
	http    *http.Client
}

func New(baseURL, token string, chatID int64) *Notifier {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	return &Notifier{
		baseURL: baseURL,
		token:   token,
		chatID:  chatID,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Enabled یعنی توکن/چت تنظیم شده — اگر نه، فراخوانی‌های Notify باید نادیده گرفته شوند.
func (n *Notifier) Enabled() bool { return n.token != "" && n.chatID != 0 }

func (n *Notifier) call(ctx context.Context, method string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/bot%s/%s", n.baseURL, n.token, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// CreateTopic یک forum topic تازه با نام سرویس می‌سازد و message_thread_id برمی‌گرداند.
func (n *Notifier) CreateTopic(ctx context.Context, name string) (int, error) {
	var resp struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageThreadID int `json:"message_thread_id"`
		} `json:"result"`
		Description string `json:"description"`
	}
	err := n.call(ctx, "createForumTopic", map[string]any{
		"chat_id": n.chatID,
		"name":    name,
	}, &resp)
	if err != nil {
		return 0, err
	}
	if !resp.OK {
		return 0, fmt.Errorf("createForumTopic failed: %s", resp.Description)
	}
	return resp.Result.MessageThreadID, nil
}

// SendToTopic یک پیام متنی به یک topic مشخص می‌فرستد.
func (n *Notifier) SendToTopic(ctx context.Context, threadID int, text string) error {
	_, err := n.sendToTopic(ctx, threadID, text)
	return err
}

// SendToTopicGetID دقیقاً مثل SendToTopic ولی message_id پیامِ ارسال‌شده را
// هم برمی‌گرداند — برای پیام‌هایی که بعداً قرار است با EditMessage جای‌گزین
// شوند (مثل داشبوردِ وضعیتِ سرویس‌ها) لازم است.
func (n *Notifier) SendToTopicGetID(ctx context.Context, threadID int, text string) (int, error) {
	return n.sendToTopic(ctx, threadID, text)
}

func (n *Notifier) sendToTopic(ctx context.Context, threadID int, text string) (int, error) {
	var resp struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
		Description string `json:"description"`
	}
	body := map[string]any{
		"chat_id":    n.chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if threadID != 0 {
		body["message_thread_id"] = threadID
	}
	if err := n.call(ctx, "sendMessage", body, &resp); err != nil {
		return 0, err
	}
	if !resp.OK {
		return 0, fmt.Errorf("sendMessage failed: %s", resp.Description)
	}
	return resp.Result.MessageID, nil
}

// EditMessage متنِ یک پیامِ از قبل ارسال‌شده را جای‌گزین می‌کند (برای
// داشبوردهایی که به‌جای فرستادنِ پیامِ جدید، همان یک پیام را آپدیت می‌کنند).
// اگر پیام دیگر وجود نداشته باشد (مثلاً کاربر آن را پاک کرده)، خطا برمی‌گرداند
// تا caller بتواند با فرستادنِ پیامِ تازه جبران کند.
func (n *Notifier) EditMessage(ctx context.Context, messageID int, text string) error {
	var resp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	body := map[string]any{
		"chat_id":    n.chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if err := n.call(ctx, "editMessageText", body, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("editMessageText failed: %s", resp.Description)
	}
	return nil
}
