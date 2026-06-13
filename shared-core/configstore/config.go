// Package configstore مدیریت config ربات‌ها در MongoDB.
//
// Flow:
//  1. startup: Load() → MongoDB → in-memory
//  2. runtime: Get() → from memory (zero-lock)
//  3. update:  Update() → MongoDB → NATS config.updated
//  4. fallback: RunFallbackPoller() → re-fetch هر ۵ دقیقه
package configstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const colName = "bot_configs"

// BotConfig سند config یک bot در MongoDB.
type BotConfig struct {
	BotID     string                       `bson:"bot_id"   json:"bot_id"`
	Type      string                       `bson:"type"     json:"type"`
	Settings  map[string]string            `bson:"settings" json:"settings"`
	I18n      map[string]map[string]string `bson:"i18n"     json:"i18n"`
	AdminIDs  []int64                      `bson:"admin_ids" json:"admin_ids"`
	UpdatedAt time.Time                    `bson:"updated_at" json:"updated_at"`
}

// Get یک setting را برمی‌گرداند.
func (c *BotConfig) Get(key, defaultVal string) string {
	if c == nil || c.Settings == nil {
		return defaultVal
	}
	if v, ok := c.Settings[key]; ok && v != "" {
		return v
	}
	return defaultVal
}

// T ترجمه یک key را برمی‌گرداند.
func (c *BotConfig) T(lang, key string) string {
	if c == nil || c.I18n == nil {
		return key
	}
	if texts, ok := c.I18n[lang]; ok {
		if v, ok := texts[key]; ok {
			return v
		}
	}
	if texts, ok := c.I18n["fa"]; ok {
		if v, ok := texts[key]; ok {
			return v
		}
	}
	return key
}

// IsAdmin بررسی می‌کند آیا user ادمین است.
func (c *BotConfig) IsAdmin(userID int64) bool {
	if c == nil {
		return false
	}
	for _, id := range c.AdminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// ── Store ──────────────────────────────────────────────────

// Store مدیریت bot config با in-memory cache.
// از ports.DocumentStore استفاده می‌کند — مستقل از MongoDB driver.
type Store struct {
	col   ports.Collection
	botID string

	mu     sync.RWMutex
	cached *BotConfig

	fallbackInterval time.Duration
}

func New(ds ports.DocumentStore, botID string) *Store {
	return &Store{
		col:              ds.Collection(colName),
		botID:            botID,
		fallbackInterval: 5 * time.Minute,
	}
}

// Load config را از MongoDB بارگذاری می‌کند.
func (s *Store) Load(ctx context.Context) (*BotConfig, error) {
	cfg, err := s.fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("configstore load: %w", err)
	}
	if cfg == nil {
		cfg = &BotConfig{
			BotID:     s.botID,
			Settings:  map[string]string{},
			I18n:      map[string]map[string]string{},
			AdminIDs:  []int64{},
			UpdatedAt: time.Now(),
		}
	}
	s.mu.Lock()
	s.cached = cfg
	s.mu.Unlock()
	return cfg, nil
}

// Get config از memory برمی‌گرداند.
func (s *Store) Get() *BotConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cached
}

// Reload config را از MongoDB دوباره می‌خواند.
func (s *Store) Reload(ctx context.Context) error {
	cfg, err := s.fetch(ctx)
	if err != nil || cfg == nil {
		return err
	}
	s.mu.Lock()
	s.cached = cfg
	s.mu.Unlock()
	return nil
}

// Update یک setting را در MongoDB آپدیت می‌کند.
func (s *Store) Update(ctx context.Context, key, value string) error {
	filter := bson.M{"bot_id": s.botID}
	update := bson.M{
		"$set": bson.M{
			"settings." + key: value,
			"updated_at":      time.Now(),
		},
	}
	return s.col.UpdateOne(ctx, filter, update)
}

// UpdateMany چند setting را یکجا آپدیت می‌کند.
func (s *Store) UpdateMany(ctx context.Context, settings map[string]string) error {
	setFields := bson.M{"updated_at": time.Now()}
	for k, v := range settings {
		setFields["settings."+k] = v
	}
	return s.col.UpdateOne(ctx,
		bson.M{"bot_id": s.botID},
		bson.M{"$set": setFields},
	)
}

// SetAdmins لیست admin ها را آپدیت می‌کند.
func (s *Store) SetAdmins(ctx context.Context, adminIDs []int64) error {
	return s.col.UpdateOne(ctx,
		bson.M{"bot_id": s.botID},
		bson.M{"$set": bson.M{
			"admin_ids":  adminIDs,
			"updated_at": time.Now(),
		}},
	)
}

// RunFallbackPoller هر fallbackInterval از Mongo re-fetch می‌کند.
func (s *Store) RunFallbackPoller(ctx context.Context) {
	ticker := time.NewTicker(s.fallbackInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Reload(ctx)
		}
	}
}

// Snapshot JSON snapshot از config فعلی.
func (s *Store) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.cached)
}

// ── internal ──────────────────────────────────────────────

func (s *Store) fetch(ctx context.Context) (*BotConfig, error) {
	var cfg BotConfig
	err := s.col.FindOne(ctx, bson.M{"bot_id": s.botID}, &cfg)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
