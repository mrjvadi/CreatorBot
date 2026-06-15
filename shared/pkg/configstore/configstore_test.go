package configstore_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/configstore"
)

// ── Mock Cache ─────────────────────────────────────────────

type mockCache struct {
	mu   sync.RWMutex
	data map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key], nil
}

func (m *mockCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *mockCache) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// ── Mock Publisher ─────────────────────────────────────────

type mockPublisher struct {
	events []string
}

func (m *mockPublisher) PublishCore(subject string, _ any) error {
	m.events = append(m.events, subject)
	return nil
}

// ── Tests ─────────────────────────────────────────────────

func TestConfigStore_SetGet(t *testing.T) {
	cache := newMockCache()
	pub := &mockPublisher{}
	store := configstore.New(cache, pub)
	ctx := context.Background()

	entry, err := store.Set(ctx, "max_bots", 10, "admin", "initial")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if entry.Version != 1 {
		t.Errorf("first version should be 1, got %d", entry.Version)
	}

	got, err := store.Get(ctx, "max_bots")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if got.Key != "max_bots" {
		t.Errorf("key = %q, want max_bots", got.Key)
	}
}

func TestConfigStore_Versioning(t *testing.T) {
	cache := newMockCache()
	pub := &mockPublisher{}
	store := configstore.New(cache, pub)
	ctx := context.Background()

	// v1
	store.Set(ctx, "rate_limit", 100, "admin", "initial")
	// v2
	store.Set(ctx, "rate_limit", 200, "admin", "increased")
	// v3
	e3, _ := store.Set(ctx, "rate_limit", 300, "admin", "increased again")

	if e3.Version != 3 {
		t.Errorf("expected version 3, got %d", e3.Version)
	}

	// بررسی history
	v1, err := store.GetVersion(ctx, "rate_limit", 1)
	if err != nil {
		t.Fatalf("GetVersion(1) error = %v", err)
	}
	if v1.Version != 1 {
		t.Errorf("expected v1, got v%d", v1.Version)
	}
}

func TestConfigStore_Rollback(t *testing.T) {
	cache := newMockCache()
	pub := &mockPublisher{}
	store := configstore.New(cache, pub)
	ctx := context.Background()

	store.Set(ctx, "feature_x", false, "admin", "disabled")
	store.Set(ctx, "feature_x", true, "admin", "enabled")

	// rollback
	rolled, err := store.Rollback(ctx, "feature_x", "admin")
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if rolled.Version != 3 {
		t.Errorf("rollback should create v3, got v%d", rolled.Version)
	}
}

func TestConfigStore_PublishOnSet(t *testing.T) {
	cache := newMockCache()
	pub := &mockPublisher{}
	store := configstore.New(cache, pub)
	ctx := context.Background()

	store.Set(ctx, "key1", "val1", "admin", "")
	store.Set(ctx, "key2", "val2", "admin", "")

	if len(pub.events) != 2 {
		t.Errorf("expected 2 events, got %d", len(pub.events))
	}
	for _, e := range pub.events {
		if e != "config.updated" {
			t.Errorf("expected config.updated, got %s", e)
		}
	}
}

func TestConfigStore_GetString(t *testing.T) {
	cache := newMockCache()
	store := configstore.New(cache, nil)
	ctx := context.Background()

	// key موجود نیست → default
	val := store.GetString(ctx, "missing", "default_val")
	if val != "default_val" {
		t.Errorf("expected default_val, got %s", val)
	}

	// set و get
	store.SetString(ctx, "greeting", "سلام", "admin", "")
	val = store.GetString(ctx, "greeting", "")
	if val != "سلام" {
		t.Errorf("expected سلام, got %s", val)
	}
}

func TestConfigStore_RollbackAtV1(t *testing.T) {
	cache := newMockCache()
	store := configstore.New(cache, nil)
	ctx := context.Background()

	store.Set(ctx, "only_one", "value", "admin", "")

	_, err := store.Rollback(ctx, "only_one", "admin")
	if err == nil {
		t.Error("rollback at v1 should return error")
	}
}

func TestConfigStore_Delete(t *testing.T) {
	cache := newMockCache()
	pub := &mockPublisher{}
	store := configstore.New(cache, pub)
	ctx := context.Background()

	store.Set(ctx, "temp_key", "temp", "admin", "")
	store.Delete(ctx, "temp_key", "admin")

	got, _ := store.Get(ctx, "temp_key")
	if got != nil {
		t.Error("deleted key should return nil")
	}

	// باید config.deleted event داده شده باشه
	if len(pub.events) < 2 || pub.events[len(pub.events)-1] != "config.deleted" {
		t.Error("expected config.deleted event after delete")
	}
}
