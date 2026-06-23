// Package tgbot - reminders.go
// یادآورِ انقضای سرویس‌ها: یک job پس‌زمینه که سرویس‌های نزدیک به انقضا را پیدا
// کرده و به صاحبشان پیام تمدید می‌فرستد. با dedupe در Redis از اسپم جلوگیری می‌شود.
package user

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
)

const (
	// expiryWindow بازه‌ی هشدار قبل از انقضا.
	expiryWindow = 72 * time.Hour
	// reminderInterval فاصله‌ی اسکن‌ها.
	reminderInterval = 6 * time.Hour
)

// StartExpiryReminders job یادآور را در پس‌زمینه اجرا می‌کند (با ctx لغو می‌شود).
func (h *User) StartExpiryReminders(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(reminderInterval)
		defer ticker.Stop()
		h.RunExpiryScan(ctx) // اجرای اولیه
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.RunExpiryScan(ctx)
			}
		}
	}()
}

// runExpiryScan یک‌بار سرویس‌های نزدیک‌انقضا را اسکن و یادآوری می‌فرستد.
func (h *User) RunExpiryScan(ctx context.Context) {
	now := time.Now()
	insts, err := h.Store.ListInstancesExpiringBetween(ctx, now, now.Add(expiryWindow))
	if err != nil {
		h.Log.Error("expiry scan", h.F("err", err))
		return
	}

	for _, inst := range insts {
		if inst.ExpiresAt == nil {
			continue
		}
		// dedupe: یک یادآور در هر بازه‌ی انقضا
		dedupeKey := fmt.Sprintf("bm:exprem:%s", inst.ID.String())
		if v, _ := h.Cache.Get(ctx, dedupeKey); v != "" {
			continue
		}

		owner, _ := h.Store.FindUserByID(ctx, inst.OwnerID)
		if owner == nil || owner.TelegramID == 0 {
			continue
		}

		days := int(time.Until(*inst.ExpiresAt).Hours() / 24)
		if days < 0 {
			days = 0
		}

		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data(
			h.Btn(ctx, owner.TelegramID, i18n.KeyBtnRenew),
			"svc_renew:"+inst.ID.String())))

		if _, err := h.Bot.Send(
			tele.ChatID(owner.TelegramID),
			h.T(ctx, owner.TelegramID, i18n.KeyExpiryReminder, inst.ContainerName, days),
			tele.ModeHTML, kb,
		); err != nil {
			continue // در صورت خطا، dedupe ست نمی‌شود تا دفعه‌ی بعد دوباره تلاش شود
		}

		ttl := time.Until(*inst.ExpiresAt)
		if ttl < time.Hour {
			ttl = time.Hour
		}
		_ = h.Cache.Set(ctx, dedupeKey, "1", ttl)
	}
}
