// Package rotation — Secret Rotation برای CreatorBot V3.
// از dual-key strategy استفاده می‌کند:
// وقتی secret جدید ایجاد می‌شود، secret قدیمی برای مدت grace period هنوز valid است.
package rotation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// KeyPair شامل key فعلی و key قبلی برای دوره grace است.
type KeyPair struct {
	Current   string
	Previous  string
	RotatedAt time.Time
}

// Manager مدیریت rotation سکرت‌ها.
type Manager struct {
	mu          sync.RWMutex
	keys        map[string]*KeyPair
	gracePeriod time.Duration
	store       KeyStore
}

// KeyStore interface برای ذخیره key ها.
type KeyStore interface {
	SaveKey(ctx context.Context, name, current, previous string, rotatedAt time.Time) error
	LoadKey(ctx context.Context, name string) (current, previous string, rotatedAt time.Time, err error)
}

// New یک Manager جدید می‌سازد.
// gracePeriod: مدت زمانی که key قدیمی هنوز valid است (پیش‌فرض ۲۴ ساعت).
func New(store KeyStore, gracePeriod time.Duration) *Manager {
	if gracePeriod == 0 {
		gracePeriod = 24 * time.Hour
	}
	return &Manager{
		keys:        make(map[string]*KeyPair),
		gracePeriod: gracePeriod,
		store:       store,
	}
}

// Rotate یک secret جدید تولید می‌کند و قدیمی را به previous منتقل می‌کند.
func (m *Manager) Rotate(ctx context.Context, name string) (newKey string, err error) {
	newKey, err = generateKey(32)
	if err != nil {
		return "", fmt.Errorf("rotation: generate key: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pair := &KeyPair{RotatedAt: time.Now()}

	if existing, ok := m.keys[name]; ok {
		pair.Previous = existing.Current
	}
	pair.Current = newKey
	m.keys[name] = pair

	if m.store != nil {
		if err := m.store.SaveKey(ctx, name, newKey, pair.Previous, pair.RotatedAt); err != nil {
			return "", fmt.Errorf("rotation: save: %w", err)
		}
	}

	return newKey, nil
}

// Validate بررسی می‌کند آیا یک key (فعلی یا قبلی) valid است.
func (m *Manager) Validate(name, key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pair, ok := m.keys[name]
	if !ok {
		return false
	}

	if key == pair.Current {
		return true
	}

	// grace period — key قدیمی هنوز valid است
	if key == pair.Previous && pair.Previous != "" {
		if time.Since(pair.RotatedAt) < m.gracePeriod {
			return true
		}
	}

	return false
}

// Current key فعلی را برمی‌گرداند.
func (m *Manager) Current(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pair, ok := m.keys[name]
	if !ok {
		return "", fmt.Errorf("rotation: key %q not found", name)
	}
	return pair.Current, nil
}

// Load key ها را از store بارگذاری می‌کند.
func (m *Manager) Load(ctx context.Context, name string) error {
	if m.store == nil {
		return nil
	}

	current, previous, rotatedAt, err := m.store.LoadKey(ctx, name)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys[name] = &KeyPair{
		Current:   current,
		Previous:  previous,
		RotatedAt: rotatedAt,
	}
	return nil
}

// RotateAll همه key های ثبت‌شده را rotate می‌کند.
func (m *Manager) RotateAll(ctx context.Context) (map[string]string, error) {
	m.mu.RLock()
	names := make([]string, 0, len(m.keys))
	for name := range m.keys {
		names = append(names, name)
	}
	m.mu.RUnlock()

	result := make(map[string]string)
	for _, name := range names {
		newKey, err := m.Rotate(ctx, name)
		if err != nil {
			return result, fmt.Errorf("rotate %s: %w", name, err)
		}
		result[name] = newKey
	}
	return result, nil
}

// ScheduleRotation rotation خودکار را در فواصل زمانی مشخص اجرا می‌کند.
func (m *Manager) ScheduleRotation(ctx context.Context, names []string, interval time.Duration,
	onRotate func(name, newKey string)) {

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, name := range names {
					newKey, err := m.Rotate(ctx, name)
					if err == nil && onRotate != nil {
						onRotate(name, newKey)
					}
					_ = err
				}
			}
		}
	}()
}

// ── Helpers ────────────────────────────────────────────────

func generateKey(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateKey یک random key تولید می‌کند (برای استفاده خارجی).
func GenerateKey(length int) (string, error) {
	return generateKey(length)
}
