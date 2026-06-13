// Package registry نگه‌داری map از bot_token → NATS subject.
// هر bot که راه‌اندازی می‌شود، خود را اینجا ثبت می‌کند.
// Gateway هر webhook رسیده را به subject مربوطه forward می‌کند.
package registry

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// BotEntry اطلاعات یک bot ثبت‌شده.
type BotEntry struct {
	// Token توکن کامل تلگرام — فقط برای verify webhook path
	Token string
	// BotID عدد قبل از : در توکن
	BotID int64
	// NATSSubject subject که updates باید به آن forward شوند
	NATSSubject string
	// Type نوع ربات (botmanager, uploader, vpn, ...)
	Type string
}

// Registry نگه‌داری bot ها thread-safe.
type Registry struct {
	mu      sync.RWMutex
	// key: token_hash (نه توکن خام — امنیت)
	entries map[string]*BotEntry
	// key: bot_id برای lookup سریع
	byBotID map[int64]*BotEntry
}

func New() *Registry {
	return &Registry{
		entries: make(map[string]*BotEntry),
		byBotID: make(map[int64]*BotEntry),
	}
}

// Register یک bot را ثبت می‌کند.
// tokenHash از hash توکن ساخته می‌شود — توکن خام ذخیره نمی‌شود.
func (r *Registry) Register(entry *BotEntry) {
	h := tokenHash(entry.Token)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[h] = entry
	r.byBotID[entry.BotID] = entry
}

// Unregister یک bot را حذف می‌کند.
func (r *Registry) Unregister(token string) {
	h := tokenHash(token)
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.entries[h]; ok {
		delete(r.byBotID, e.BotID)
		delete(r.entries, h)
	}
}

// Lookup یک bot را با hash توکن پیدا می‌کند.
// path تلگرام فرمت /TOKEN/... دارد.
func (r *Registry) Lookup(token string) (*BotEntry, bool) {
	h := tokenHash(token)
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[h]
	return e, ok
}

// LookupByID یک bot را با BotID پیدا می‌کند.
func (r *Registry) LookupByID(botID int64) (*BotEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.byBotID[botID]
	return e, ok
}

// List همه bot های ثبت‌شده را برمی‌گرداند.
func (r *Registry) List() []*BotEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*BotEntry, 0, len(r.entries))
	for _, e := range r.entries {
		result = append(result, e)
	}
	return result
}

// Count تعداد bot های ثبت‌شده.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h[:16])
}
