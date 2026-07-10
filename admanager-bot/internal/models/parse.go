// parse.go — تحلیلِ خالصِ ورودیِ متنیِ زمان‌بندی (بدون وابستگی به تلگرام)،
// تا هم در ویزارد ساخت/ویرایشِ کمپین (internal/tgbot) و هم هرجای دیگری که
// لازم شود استفاده و تست شود — نگاه کنید به parse_test.go.
package models

import (
	"strconv"
	"strings"
)

// NormalizeDigits ارقام فارسی/عربی را به لاتین تبدیل می‌کند تا ادمین
// بتواند هرکدام را تایپ کند.
func NormalizeDigits(s string) string {
	repl := map[rune]rune{
		'۰': '0', '۱': '1', '۲': '2', '۳': '3', '۴': '4',
		'۵': '5', '۶': '6', '۷': '7', '۸': '8', '۹': '9',
		'٠': '0', '١': '1', '٢': '2', '٣': '3', '٤': '4',
		'٥': '5', '٦': '6', '٧': '7', '٨': '8', '٩': '9',
	}
	out := []rune(s)
	for i, r := range out {
		if v, ok := repl[r]; ok {
			out[i] = v
		}
	}
	return string(out)
}

// ParseClock "23:08" یا "23" را به ساعت/دقیقه تبدیل می‌کند.
func ParseClock(s string) (int, int, bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	h, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || h < 0 || h > 23 {
		return 0, 0, false
	}
	m := 0
	if len(parts) > 1 {
		m, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || m < 0 || m > 59 {
			return 0, 0, false
		}
	}
	return h, m, true
}

// ParseClockRange "23:08-03:00" را به ساعت/دقیقه‌ی شروع و پایان تبدیل
// می‌کند. اگر فقط یک زمان بدون خط‌تیره بیاید، پایان همان شروع در نظر
// گرفته می‌شود (یعنی کل شبانه‌روز، طبق InDailyWindow/DailyWindowBounds).
func ParseClockRange(s string) (int, int, int, int, bool) {
	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, "-", 2)
	sh, sm, ok := ParseClock(parts[0])
	if !ok {
		return 0, 0, 0, 0, false
	}
	if len(parts) == 1 {
		return sh, sm, sh, sm, true
	}
	eh, em, ok := ParseClock(parts[1])
	if !ok {
		return 0, 0, 0, 0, false
	}
	return sh, sm, eh, em, true
}

// ParseSchedule ۴ خط زمان‌بندیِ ویزاردِ کمپین را تحلیل می‌کند:
//
//	خط۱: بازه‌ی روزانه "HH:MM-HH:MM"
//	خط۲: فاصله بین پست‌ها (دقیقه)
//	خط۳: عمر کل چرخه (دقیقه)
//	خط۴: چرخش (دقیقه)
//
// خروجی: startHour, startMinute, endHour, endMinute, interval, deleteAfter, rotation, ok
func ParseSchedule(text string) (int, int, int, int, int, int, int, bool) {
	raw := strings.ReplaceAll(NormalizeDigits(text), "\r", "")
	var lines []string
	for _, l := range strings.Split(raw, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			lines = append(lines, t)
		}
	}
	if len(lines) != 4 {
		return 0, 0, 0, 0, 0, 0, 0, false
	}

	sh, sm, eh, em, ok := ParseClockRange(lines[0])
	if !ok {
		return 0, 0, 0, 0, 0, 0, 0, false
	}
	interval, e1 := strconv.Atoi(lines[1])
	del, e2 := strconv.Atoi(lines[2])
	rot, e3 := strconv.Atoi(lines[3])
	if e1 != nil || e2 != nil || e3 != nil || interval < 1 || del < 0 || rot < 0 {
		return 0, 0, 0, 0, 0, 0, 0, false
	}
	return sh, sm, eh, em, interval, del, rot, true
}
