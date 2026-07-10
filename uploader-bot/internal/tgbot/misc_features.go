package tgbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// ── ضد اسپم ───────────────────────────────────────────────────

func (h *Handler) spamKey(uid int64) string {
	return fmt.Sprintf("upl:spam:%s:%d", h.InstanceID, uid)
}

// spamOK اگر کاربر در بازه‌ی ضد اسپم نباشد true برمی‌گرداند.
// ادمین‌ها معاف‌اند.
func (h *Handler) spamOK(ctx context.Context, c tele.Context) bool {
	if h.isAdmin(c) || h.Cache == nil {
		return true
	}
	delay := h.GetSettingInt(ctx, models.SettingSpamDelay, 0)
	if delay <= 0 {
		return true
	}
	ok, err := h.Cache.SetNX(ctx, h.spamKey(c.Sender().ID), "1", time.Duration(delay)*time.Second)
	if err != nil {
		h.LogErr("spamOK", err)
		return true // اگر کش در دسترس نبود، کاربر را مسدود نکنیم
	}
	return ok
}

// ── دکمه‌های شروع پیشرفته ─────────────────────────────────────

// startButtons دکمه‌های inline شروع را از تنظیم start_buttons می‌سازد.
// هر خط: «برچسب|https://link». اگر خالی باشد nil برمی‌گرداند.
func (h *Handler) startButtons(ctx context.Context) *tele.ReplyMarkup {
	raw := strings.TrimSpace(h.Store.GetSetting(ctx, models.SettingStartButtons))
	if raw == "" {
		return nil
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		label := strings.TrimSpace(parts[0])
		url := strings.TrimSpace(parts[1])
		if label == "" || url == "" {
			continue
		}
		rows = append(rows, kb.Row(kb.URL(label, url)))
	}
	if len(rows) == 0 {
		return nil
	}
	kb.Inline(rows...)
	return kb
}
