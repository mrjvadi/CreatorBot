package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════
// Mock Telegram Bot
// ══════════════════════════════════════════════════════════════

type SentMessage struct {
	ChatID   int64
	Text     string
	Buttons  [][]string // [[btn1, btn2], [btn3]]
	ParseMode string
	SentAt   time.Time
}

type MockBot struct {
	mu       sync.Mutex
	messages []SentMessage
}

func NewMockBot() *MockBot { return &MockBot{} }

func (b *MockBot) Send(chatID int64, text string, buttons ...[][]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	msg := SentMessage{ChatID: chatID, Text: text, SentAt: time.Now()}
	if len(buttons) > 0 {
		msg.Buttons = buttons[0]
	}
	b.messages = append(b.messages, msg)
}

func (b *MockBot) Last() *SentMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.messages) == 0 {
		return nil
	}
	m := b.messages[len(b.messages)-1]
	return &m
}

func (b *MockBot) All() []SentMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]SentMessage, len(b.messages))
	copy(out, b.messages)
	return out
}

func (b *MockBot) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}

func (b *MockBot) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = nil
}

func (b *MockBot) HasButton(label string) bool {
	last := b.Last()
	if last == nil {
		return false
	}
	for _, row := range last.Buttons {
		for _, btn := range row {
			if btn == label {
				return true
			}
		}
	}
	return false
}

func (b *MockBot) TextContains(sub string) bool {
	last := b.Last()
	return last != nil && strings.Contains(last.Text, sub)
}

// ══════════════════════════════════════════════════════════════
// Mock Database (in-memory)
// ══════════════════════════════════════════════════════════════

type MockDB struct {
	mu     sync.RWMutex
	tables map[string]map[string]string // table→id→json
}

func NewMockDB() *MockDB {
	return &MockDB{tables: make(map[string]map[string]string)}
}

func (db *MockDB) Insert(table, id string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.tables[table] == nil {
		db.tables[table] = make(map[string]string)
	}
	db.tables[table][id] = string(b)
	return nil
}

func (db *MockDB) Find(table, id string, dest interface{}) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.tables[table] == nil {
		return false
	}
	raw, ok := db.tables[table][id]
	if !ok {
		return false
	}
	json.Unmarshal([]byte(raw), dest)
	return true
}

func (db *MockDB) Update(table, id string, v interface{}) error {
	return db.Insert(table, id, v)
}

func (db *MockDB) Delete(table, id string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.tables[table] != nil {
		delete(db.tables[table], id)
	}
}

func (db *MockDB) List(table string) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var out []string
	for _, v := range db.tables[table] {
		out = append(out, v)
	}
	return out
}

func (db *MockDB) Count(table string) int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.tables[table])
}

func (db *MockDB) FindWhere(table string, match func(string) bool) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var out []string
	for _, v := range db.tables[table] {
		if match(v) {
			out = append(out, v)
		}
	}
	return out
}

// ══════════════════════════════════════════════════════════════
// Mock Cache (Redis)
// ══════════════════════════════════════════════════════════════

type MockCache struct {
	mu   sync.RWMutex
	data map[string]cacheItem
}

type cacheItem struct {
	val string
	exp time.Time
}

func NewMockCache() *MockCache {
	c := &MockCache{data: make(map[string]cacheItem)}
	go func() {
		for range time.Tick(time.Second) {
			c.mu.Lock()
			for k, v := range c.data {
				if time.Now().After(v.exp) {
					delete(c.data, k)
				}
			}
			c.mu.Unlock()
		}
	}()
	return c
}

func (c *MockCache) Set(_ context.Context, key, val string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp := time.Now().Add(ttl)
	if ttl == 0 {
		exp = time.Now().Add(24 * time.Hour)
	}
	c.data[key] = cacheItem{val: val, exp: exp}
}

func (c *MockCache) Get(_ context.Context, key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.data[key]
	if !ok || time.Now().After(item.exp) {
		return "", false
	}
	return item.val, true
}

func (c *MockCache) Del(_ context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *MockCache) SetJSON(_ context.Context, key string, v interface{}, ttl time.Duration) {
	b, _ := json.Marshal(v)
	c.Set(context.Background(), key, string(b), ttl)
}

func (c *MockCache) GetJSON(_ context.Context, key string, dest interface{}) bool {
	raw, ok := c.Get(context.Background(), key)
	if !ok {
		return false
	}
	return json.Unmarshal([]byte(raw), dest) == nil
}

// ══════════════════════════════════════════════════════════════
// Mock NATS
// ══════════════════════════════════════════════════════════════

type NATSMsg struct {
	Subject string
	Data    map[string]interface{}
	SentAt  time.Time
}

type MockNATS struct {
	mu          sync.Mutex
	published   []NATSMsg
	subscribers map[string][]func(map[string]interface{})
}

func NewMockNATS() *MockNATS {
	return &MockNATS{
		subscribers: make(map[string][]func(map[string]interface{})),
	}
}

func (n *MockNATS) Publish(subject string, data map[string]interface{}) {
	n.mu.Lock()
	n.published = append(n.published, NATSMsg{Subject: subject, Data: data, SentAt: time.Now()})
	subs := append([]func(map[string]interface{}){}, n.subscribers[subject]...)
	n.mu.Unlock()
	for _, fn := range subs {
		go fn(data)
	}
}

func (n *MockNATS) Subscribe(subject string, fn func(map[string]interface{})) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subscribers[subject] = append(n.subscribers[subject], fn)
}

func (n *MockNATS) Events(subject string) []NATSMsg {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []NATSMsg
	for _, m := range n.published {
		if m.Subject == subject || strings.HasPrefix(m.Subject, strings.TrimSuffix(subject, "*")) {
			out = append(out, m)
		}
	}
	return out
}

func (n *MockNATS) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.published = nil
}

func (n *MockNATS) TotalPublished() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.published)
}

// ══════════════════════════════════════════════════════════════
// State Machine (wizard state)
// ══════════════════════════════════════════════════════════════

type WizardState struct {
	Step string
	Data map[string]string
}

type MockStateStore struct {
	cache *MockCache
}

func NewMockStateStore(c *MockCache) *MockStateStore { return &MockStateStore{cache: c} }

func (s *MockStateStore) Set(ctx context.Context, uid int64, step string, data map[string]string) {
	st := WizardState{Step: step, Data: data}
	s.cache.SetJSON(ctx, fmt.Sprintf("state:%d", uid), st, 15*time.Minute)
}

func (s *MockStateStore) Get(ctx context.Context, uid int64) WizardState {
	var st WizardState
	s.cache.GetJSON(ctx, fmt.Sprintf("state:%d", uid), &st)
	return st
}

func (s *MockStateStore) Clear(ctx context.Context, uid int64) {
	s.cache.Del(ctx, fmt.Sprintf("state:%d", uid))
}

// ══════════════════════════════════════════════════════════════
// Helpers
// ══════════════════════════════════════════════════════════════

func ptr[T any](v T) *T { return &v }

func jsonContains(raw, key, val string) bool {
	var m map[string]interface{}
	if json.Unmarshal([]byte(raw), &m) != nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	return fmt.Sprintf("%v", v) == val
}
