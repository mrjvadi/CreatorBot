package i18n

import (
	"context"
	"fmt"
	"sync"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

var langs = map[Lang]map[Key]string{
	FA: fa,
	EN: en,
}

// Translator ترجمه متن‌ها و مدیریت زبان کاربر.
//
// زبان هر کاربر علاوه بر Redis در یک کش درون‌حافظه‌ای نگه‌داری می‌شود تا
// مسیر داغ ترجمه (که در هر پیام ده‌ها بار صدا زده می‌شود) به‌جای round-trip
// به Redis، از حافظه خوانده شود. چون فقط همین پروسه زبان را تغییر می‌دهد،
// کش با هر SetLang به‌روزرسانی می‌شود و نیازی به invalidation خارجی نیست.
type Translator struct {
	cache ports.Cache

	mu        sync.RWMutex
	langCache map[int64]Lang
}

func New(cache ports.Cache) *Translator {
	return &Translator{
		cache:     cache,
		langCache: make(map[int64]Lang),
	}
}

// T متن کلید داده شده را به زبان کاربر ترجمه می‌کند.
// اگه key وجود نداشت fallback به فارسی.
func (t *Translator) T(ctx context.Context, uid int64, key Key, args ...any) string {
	lang := t.GetLang(ctx, uid)
	texts, ok := langs[lang]
	if !ok {
		texts = fa
	}
	text, ok := texts[key]
	if !ok {
		// fallback به فارسی
		if text = fa[key]; text == "" {
			return string(key) // آخرین fallback
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

// Btn دکمه را به زبان کاربر ترجمه می‌کند.
func (t *Translator) Btn(ctx context.Context, uid int64, key Key) string {
	return t.T(ctx, uid, key)
}

// GetLang زبان فعلی کاربر را برمی‌گرداند.
// ابتدا کش حافظه، سپس Redis. مقدار resolve‌شده در حافظه نگه داشته می‌شود.
func (t *Translator) GetLang(ctx context.Context, uid int64) Lang {
	t.mu.RLock()
	l, ok := t.langCache[uid]
	t.mu.RUnlock()
	if ok {
		return l
	}

	resolved := Default
	if val, err := t.cache.Get(ctx, langKey(uid)); err == nil && val != "" {
		if cand := Lang(val); langs[cand] != nil {
			resolved = cand
		}
	}

	t.mu.Lock()
	t.langCache[uid] = resolved
	t.mu.Unlock()
	return resolved
}

// SetLang زبان کاربر را در Redis و کش حافظه ذخیره می‌کند.
func (t *Translator) SetLang(ctx context.Context, uid int64, lang Lang) {
	_ = t.cache.Set(ctx, langKey(uid), string(lang), 0) // بدون expire
	t.mu.Lock()
	t.langCache[uid] = lang
	t.mu.Unlock()
}

// IsValidLang بررسی می‌کند کد زبان معتبر است.
func IsValidLang(code string) bool {
	_, ok := langs[Lang(code)]
	return ok
}

// SupportedLangs لیست زبان‌های پشتیبانی‌شده.
func SupportedLangs() []Lang {
	return []Lang{FA, EN}
}

// DetectFromTelegram زبان را از language_code تلگرام تشخیص می‌دهد.
// اگه نشناخت، فارسی برمی‌گردانه.
func DetectFromTelegram(telegramLangCode string) Lang {
	switch telegramLangCode {
	case "fa", "fa-IR":
		return FA
	case "en", "en-US", "en-GB":
		return EN
	default:
		return Default
	}
}

func langKey(uid int64) string {
	return fmt.Sprintf("bm:lang:%d", uid)
}
