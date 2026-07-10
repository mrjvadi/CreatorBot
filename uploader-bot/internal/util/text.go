// Package util — توابع کمکیِ خالص و مستقل از Handler (قابل استفاده در همه‌ی پکیج‌ها).
package util

import (
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// EscapeHTML کاراکترهای خاص HTML را امن می‌کند.
func EscapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// entityTags تگ باز و بستهٔ HTML برای یک entity را برمی‌گرداند.
func entityTags(e models.Entity) (string, string) {
	switch e.Type {
	case "bold":
		return "<b>", "</b>"
	case "italic":
		return "<i>", "</i>"
	case "underline":
		return "<u>", "</u>"
	case "strikethrough":
		return "<s>", "</s>"
	case "spoiler":
		return "<tg-spoiler>", "</tg-spoiler>"
	case "code":
		return "<code>", "</code>"
	case "pre":
		return "<pre>", "</pre>"
	case "blockquote":
		return "<blockquote>", "</blockquote>"
	case "text_link":
		if e.URL != "" {
			return `<a href="` + EscapeHTML(e.URL) + `">`, "</a>"
		}
	}
	return "", ""
}

// EntitiesToHTML متن + entities تلگرام را به HTML معتبر تبدیل می‌کند
// (با احتساب درستِ offsetهای UTF-16).
func EntitiesToHTML(text string, ents []models.Entity) string {
	if len(ents) == 0 {
		return EscapeHTML(text)
	}
	runes := []rune(text)
	toRune := make(map[int]int, len(runes)+1)
	u16 := 0
	for ri := 0; ri <= len(runes); ri++ {
		toRune[u16] = ri
		if ri < len(runes) {
			if runes[ri] > 0xFFFF {
				u16 += 2
			} else {
				u16++
			}
		}
	}
	n := len(runes)
	opens := make([][]string, n+1)
	closes := make([][]string, n+1)
	for _, e := range ents {
		o, cl := entityTags(e)
		if o == "" {
			continue
		}
		rs, ok1 := toRune[e.Offset]
		re, ok2 := toRune[e.Offset+e.Length]
		if !ok1 || !ok2 || rs >= re || rs < 0 || re > n {
			continue
		}
		opens[rs] = append(opens[rs], o)
		closes[re] = append([]string{cl}, closes[re]...)
	}
	var b strings.Builder
	for i := 0; i <= n; i++ {
		for _, cl := range closes[i] {
			b.WriteString(cl)
		}
		for _, o := range opens[i] {
			b.WriteString(o)
		}
		if i < n {
			b.WriteString(EscapeHTML(string(runes[i])))
		}
	}
	return b.String()
}

// ToModelEntities entities تلگرام را به مدل ذخیره‌سازی تبدیل می‌کند.
func ToModelEntities(ents tele.Entities) []models.Entity {
	if len(ents) == 0 {
		return nil
	}
	out := make([]models.Entity, 0, len(ents))
	for _, e := range ents {
		out = append(out, models.Entity{
			Type: string(e.Type), Offset: e.Offset, Length: e.Length,
			URL: e.URL, Language: e.Language,
		})
	}
	return out
}

// ToTeleEntities مدل ذخیره‌سازی را به entities تلگرام برمی‌گرداند.
func ToTeleEntities(ents []models.Entity) tele.Entities {
	if len(ents) == 0 {
		return nil
	}
	out := make(tele.Entities, 0, len(ents))
	for _, e := range ents {
		out = append(out, tele.MessageEntity{
			Type: tele.EntityType(e.Type), Offset: e.Offset, Length: e.Length,
			URL: e.URL, Language: e.Language,
		})
	}
	return out
}
