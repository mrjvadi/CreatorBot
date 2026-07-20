// Package search نرمال‌سازی متن فارسی/عربی + جستجوی فازیِ n-gram (معادلِ
// app-side برای pg_trgm که در MongoDB خودِ‌سرور معادل ندارد — بدون Atlas
// Search self-hosted، هیچ trigram/fuzzy واقعی‌ای موجود نیست).
//
// الگوریتم: هر متن (هنگام نوشتن و هنگام جستجو) به مجموعه‌ی trigramهای
// کاراکتری تبدیل می‌شود (دقیقاً الگوریتمِ خودِ pg_trgm: رشته با یک space
// پیش/پس padding می‌شود، سپس تمام زیررشته‌های ۳کاراکتریِ متوالی گرفته
// می‌شوند). امتیازِ شباهت هم دقیقاً همان فرمولِ pg_trgm.similarity() است:
// Jaccard = |A∩B| / |A∪B| روی مجموعه‌ی trigramها.
package search

import (
	"strings"
	"unicode"
)

// Normalize متنِ فارسی/عربی را برای تطبیقِ یکدست نرمال می‌کند:
//   - عربی ي  → فارسی ی
//   - عربی ك  → فارسی ک
//   - حذف اعراب (harakat U+064B–U+065F)
//   - حذف نیم‌فاصله (ZWNJ، U+200C)
//   - فشرده‌سازی فاصله‌های اضافی
func Normalize(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case 'ي':
			sb.WriteRune('ی')
		case 'ك':
			sb.WriteRune('ک')
		case '‌': // ZWNJ — skip
		default:
			if r >= 0x064B && r <= 0x065F { // diacritics
				continue
			}
			if unicode.IsMark(r) {
				continue
			}
			sb.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// Trigrams مجموعه‌ی یکتای trigramهای کاراکتریِ یک متنِ نرمال‌شده را می‌سازد —
// معادلِ دقیقِ الگوریتمِ pg_trgm (padding با یک space هر طرف، پنجره‌ی
// لغزنده‌ی ۳کاراکتری). رشته‌ی کوتاه‌تر از یک trigram، آرایه‌ی خالی می‌دهد.
func Trigrams(normalized string) []string {
	if normalized == "" {
		return nil
	}
	padded := " " + normalized + " "
	r := []rune(padded)
	if len(r) < 3 {
		return nil
	}
	seen := make(map[string]bool, len(r))
	out := make([]string, 0, len(r))
	for i := 0; i+3 <= len(r); i++ {
		tg := string(r[i : i+3])
		if !seen[tg] {
			seen[tg] = true
			out = append(out, tg)
		}
	}
	return out
}

// Similarity امتیازِ شباهتِ Jaccard بین دو مجموعه‌ی trigram را برمی‌گرداند —
// همان فرمولِ pg_trgm.similarity(): |A∩B| / |A∪B|، بینِ ۰ و ۱.
func Similarity(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setB := make(map[string]bool, len(b))
	for _, tg := range b {
		setB[tg] = true
	}
	inter := 0
	for _, tg := range a {
		if setB[tg] {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
