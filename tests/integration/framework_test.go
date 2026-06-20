// Package integration — تست یکپارچه همه ربات‌های اصلی.
// این فایل mock های مشترک و framework تست را تعریف می‌کند.
package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ── Mock Telegram Bot ─────────────────────────────────────────

// MockBot شبیه‌سازی تلگرام bot برای تست.
type MockBot struct {
	mu       sync.Mutex
	messages []SentMessage
	updates  chan TelegramUpdate
}

type SentMessage struct {
	ChatID  int64
	Text    string
	ReplyMarkup interface{}
	ParseMode   string
	SentAt  time.Time
}

type TelegramUpdate struct {
	UpdateID int64
	Message  *TelegramMessage
	Callback *TelegramCallback
}

type TelegramMessage struct {
	ID     int64
	ChatID int64
	UserID int64
	Text   string
}

type TelegramCallback struct {
	ID      string
	UserID  int64
	ChatID  int64
	Data    string
	MsgID   int64
}

func NewMockBot() *MockBot {
	return &MockBot{
		updates: make(chan TelegramUpdate, 100),
	}
}

func (b *MockBot) Send(chatID int64, text string, opts ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, SentMessage{
		ChatID: chatID,
		Text:   text,
		SentAt: time.Now(),
	})
}

func (b *MockBot) LastMessage() *SentMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.messages) == 0 {
		return nil
	}
	return &b.messages[len(b.messages)-1]
}

func (b *MockBot) MessageCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}

func (b *MockBot) ClearMessages() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = nil
}

func (b *MockBot) SimulateMessage(userID, chatID int64, text string) {
	b.updates <- TelegramUpdate{
		UpdateID: time.Now().UnixNano(),
		Message:  &TelegramMessage{ID: time.Now().UnixNano(), ChatID: chatID, UserID: userID, Text: text},
	}
}

func (b *MockBot) SimulateCallback(userID, chatID, msgID int64, data string) {
	b.updates <- TelegramUpdate{
		UpdateID: time.Now().UnixNano(),
		Callback: &TelegramCallback{ID: fmt.Sprintf("cb_%d", time.Now().UnixNano()), UserID: userID, ChatID: chatID, Data: data, MsgID: msgID},
	}
}

// ── Mock Database ─────────────────────────────────────────────

// MockDB شبیه‌سازی دیتابیس in-memory.
type MockDB struct {
	mu      sync.RWMutex
	tables  map[string]map[string][]byte // table → id → json
}

func NewMockDB() *MockDB {
	return &MockDB{
		tables: make(map[string]map[string][]byte),
	}
}

func (db *MockDB) Set(table, id string, data interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.tables[table] == nil {
		db.tables[table] = make(map[string][]byte)
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	db.tables[table][id] = b
	return nil
}

func (db *MockDB) Get(table, id string, dest interface{}) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.tables[table] == nil {
		return false
	}
	b, ok := db.tables[table][id]
	if !ok {
		return false
	}
	json.Unmarshal(b, dest)
	return true
}

func (db *MockDB) Delete(table, id string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.tables[table] != nil {
		delete(db.tables[table], id)
	}
}

func (db *MockDB) Count(table string) int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.tables[table])
}

func (db *MockDB) List(table string) [][]byte {
	db.mu.RLock()
	defer db.mu.RUnlock()
	result := make([][]byte, 0, len(db.tables[table]))
	for _, v := range db.tables[table] {
		result = append(result, v)
	}
	return result
}

// ── Mock Cache (Redis) ─────────────────────────────────────────

type MockCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

func NewMockCache() *MockCache {
	c := &MockCache{data: make(map[string]cacheEntry)}
	go c.cleanup()
	return c
}

func (c *MockCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp := time.Now().Add(ttl)
	if ttl == 0 {
		exp = time.Now().Add(24 * time.Hour)
	}
	c.data[key] = cacheEntry{value: value, expiresAt: exp}
	return nil
}

func (c *MockCache) Get(_ context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.data[key]
	if !ok || time.Now().After(e.expiresAt) {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return e.value, nil
}

func (c *MockCache) Del(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

func (c *MockCache) cleanup() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		c.mu.Lock()
		for k, e := range c.data {
			if time.Now().After(e.expiresAt) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}

// ── Mock NATS ────────────────────────────────────────────────

type MockNATS struct {
	mu           sync.Mutex
	published    []NATSMessage
	subscribers  map[string][]func([]byte)
}

type NATSMessage struct {
	Subject string
	Data    interface{}
	SentAt  time.Time
}

func NewMockNATS() *MockNATS {
	return &MockNATS{
		subscribers: make(map[string][]func([]byte)),
	}
}

func (n *MockNATS) Publish(subject string, data interface{}) {
	n.mu.Lock()
	n.published = append(n.published, NATSMessage{
		Subject: subject,
		Data:    data,
		SentAt:  time.Now(),
	})
	subs := append([]func([]byte){}, n.subscribers[subject]...)
	n.mu.Unlock()

	// notify subscribers
	b, _ := json.Marshal(data)
	for _, fn := range subs {
		go fn(b)
	}
}

func (n *MockNATS) Subscribe(subject string, fn func([]byte)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subscribers[subject] = append(n.subscribers[subject], fn)
}

func (n *MockNATS) Published(subject string) []NATSMessage {
	n.mu.Lock()
	defer n.mu.Unlock()
	var result []NATSMessage
	for _, m := range n.published {
		if m.Subject == subject {
			result = append(result, m)
		}
	}
	return result
}

func (n *MockNATS) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.published = nil
}

// ── Test Context ──────────────────────────────────────────────

type TestContext struct {
	DB    *MockDB
	Cache *MockCache
	NATS  *MockNATS
	Bot   *MockBot
	Ctx   context.Context
}

func NewTestContext() *TestContext {
	return &TestContext{
		DB:    NewMockDB(),
		Cache: NewMockCache(),
		NATS:  NewMockNATS(),
		Bot:   NewMockBot(),
		Ctx:   context.Background(),
	}
}

// ── Assertions ────────────────────────────────────────────────

type T interface {
	Helper()
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Logf(format string, args ...interface{})
}

func AssertMessageContains(t T, bot *MockBot, text string) {
	t.Helper()
	last := bot.LastMessage()
	if last == nil {
		t.Errorf("expected message containing %q, but no messages sent", text)
		return
	}
	if last.Text == "" || (text != "" && last.Text != text &&
		len(last.Text) > 0 && last.Text[:min(len(text), len(last.Text))] != text) {
		// simple contains check
		found := false
		for i := 0; i <= len(last.Text)-len(text); i++ {
			if last.Text[i:i+len(text)] == text {
				found = true
				break
			}
		}
		if !found && text != "" {
			t.Errorf("last message %q does not contain %q", last.Text, text)
		}
	}
}

func AssertNATSPublished(t T, nats *MockNATS, subject string) {
	t.Helper()
	msgs := nats.Published(subject)
	if len(msgs) == 0 {
		t.Errorf("expected NATS message on subject %q, none found", subject)
	}
}

func AssertDBHas(t T, db *MockDB, table, id string) {
	t.Helper()
	var dummy interface{}
	if !db.Get(table, id, &dummy) {
		t.Errorf("expected record in table %q with id %q", table, id)
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
