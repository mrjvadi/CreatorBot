package i18n

import (
	"context"
	"fmt"

	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

var langs = map[Lang]map[Key]string{
	FA: fa,
	EN: en,
}

// Translator ترجمه متن‌ها و مدیریت زبان کاربر.
type Translator struct {
	cache ports.Cache
}

func New(cache ports.Cache) *Translator {
	return &Translator{cache: cache}
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
func (t *Translator) GetLang(ctx context.Context, uid int64) Lang {
	val, err := t.cache.Get(ctx, langKey(uid))
	if err != nil || val == "" {
		return Default
	}
	l := Lang(val)
	if _, ok := langs[l]; !ok {
		return Default
	}
	return l
}

// SetLang زبان کاربر را ذخیره می‌کند.
func (t *Translator) SetLang(ctx context.Context, uid int64, lang Lang) {
	t.cache.Set(ctx, langKey(uid), string(lang), 0) // بدون expire
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
