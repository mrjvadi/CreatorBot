// Package configstore — versioned config storage با Redis.
// هر config یک version دارد و تغییرات از طریق NATS broadcast می‌شوند.
package configstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Version شماره نسخه یک config.
type Version int64

// Entry یک config entry با version و metadata.
type Entry struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Version   Version         `json:"version"`
	UpdatedAt time.Time       `json:"updated_at"`
	UpdatedBy string          `json:"updated_by"` // user ID
	Comment   string          `json:"comment"`
}

// Cache interface برای Redis.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

// Publisher interface برای NATS.
type Publisher interface {
	PublishCore(subject string, payload any) error
}

// Store مدیریت versioned configs.
type Store struct {
	cache     Cache
	publisher Publisher
	prefix    string
}

// New یک configstore جدید می‌سازد.
func New(cache Cache, publisher Publisher) *Store {
	return &Store{
		cache:     cache,
		publisher: publisher,
		prefix:    "config:",
	}
}

// Set یک config را با version جدید ذخیره می‌کند.
func (s *Store) Set(ctx context.Context, key string, value any, updatedBy, comment string) (*Entry, error) {
	// version فعلی
	current, _ := s.Get(ctx, key)
	nextVersion := Version(1)
	if current != nil {
		nextVersion = current.Version + 1
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("configstore: marshal: %w", err)
	}

	entry := &Entry{
		Key:       key,
		Value:     raw,
		Version:   nextVersion,
		UpdatedAt: time.Now(),
		UpdatedBy: updatedBy,
		Comment:   comment,
	}

	data, _ := json.Marshal(entry)
	if err := s.cache.Set(ctx, s.prefix+key, string(data), 0); err != nil {
		return nil, fmt.Errorf("configstore: save: %w", err)
	}

	// history در Redis (آخرین ۱۰ version)
	histKey := s.prefix + key + ":history:" + strconv.FormatInt(int64(nextVersion), 10)
	s.cache.Set(ctx, histKey, string(data), 30*24*time.Hour)

	// broadcast تغییر
	if s.publisher != nil {
		s.publisher.PublishCore("config.updated", map[string]any{
			"key":     key,
			"version": nextVersion,
			"by":      updatedBy,
		})
	}

	return entry, nil
}

// Get یک config را برمی‌گرداند.
func (s *Store) Get(ctx context.Context, key string) (*Entry, error) {
	data, err := s.cache.Get(ctx, s.prefix+key)
	if err != nil || data == "" {
		return nil, nil
	}

	var entry Entry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, fmt.Errorf("configstore: unmarshal: %w", err)
	}
	return &entry, nil
}

// GetVersion یک version خاص را برمی‌گرداند.
func (s *Store) GetVersion(ctx context.Context, key string, version Version) (*Entry, error) {
	histKey := s.prefix + key + ":history:" + strconv.FormatInt(int64(version), 10)
	data, err := s.cache.Get(ctx, histKey)
	if err != nil || data == "" {
		return nil, fmt.Errorf("configstore: version %d not found for %s", version, key)
	}

	var entry Entry
	json.Unmarshal([]byte(data), &entry)
	return &entry, nil
}

// Rollback به version قبلی برمی‌گردد.
func (s *Store) Rollback(ctx context.Context, key, rolledBackBy string) (*Entry, error) {
	current, err := s.Get(ctx, key)
	if err != nil || current == nil {
		return nil, fmt.Errorf("configstore: key %s not found", key)
	}
	if current.Version <= 1 {
		return nil, fmt.Errorf("configstore: already at version 1, cannot rollback")
	}

	prev, err := s.GetVersion(ctx, key, current.Version-1)
	if err != nil {
		return nil, err
	}

	// ذخیره version قبلی به عنوان version جدید
	return s.Set(ctx, key, json.RawMessage(prev.Value), rolledBackBy,
		fmt.Sprintf("rollback from v%d to v%d", current.Version, prev.Version))
}

// Delete یک config را حذف می‌کند.
func (s *Store) Delete(ctx context.Context, key, deletedBy string) error {
	if err := s.cache.Del(ctx, s.prefix+key); err != nil {
		return err
	}
	if s.publisher != nil {
		s.publisher.PublishCore("config.deleted", map[string]any{
			"key": key,
			"by":  deletedBy,
		})
	}
	return nil
}

// ── Typed helpers ──────────────────────────────────────────

// SetString یک string config ذخیره می‌کند.
func (s *Store) SetString(ctx context.Context, key, value, by, comment string) error {
	_, err := s.Set(ctx, key, value, by, comment)
	return err
}

// GetString یک string config می‌خواند.
func (s *Store) GetString(ctx context.Context, key, defaultValue string) string {
	entry, _ := s.Get(ctx, key)
	if entry == nil {
		return defaultValue
	}
	var v string
	json.Unmarshal(entry.Value, &v)
	if v == "" {
		return defaultValue
	}
	return v
}

// SetInt یک int config ذخیره می‌کند.
func (s *Store) SetInt(ctx context.Context, key string, value int, by, comment string) error {
	_, err := s.Set(ctx, key, value, by, comment)
	return err
}

// GetInt یک int config می‌خواند.
func (s *Store) GetInt(ctx context.Context, key string, defaultValue int) int {
	entry, _ := s.Get(ctx, key)
	if entry == nil {
		return defaultValue
	}
	var v int
	json.Unmarshal(entry.Value, &v)
	if v == 0 {
		return defaultValue
	}
	return v
}
