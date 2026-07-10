package tgbot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// settingsUpdate ساختار پیام آپدیت تنظیمات/متن‌ها از طریق NATS.
// هر دو فرم پشتیبانی می‌شود:
//
//	{"key":"welcome_text","value":"..."}
//	{"settings":{"welcome_text":"...","lbl_search":"..."}}
type settingsUpdate struct {
	Key      string            `json:"key"`
	Value    string            `json:"value"`
	Settings map[string]string `json:"settings"`
}

// configUpdatedEvent رویداد config.updated (هماهنگ با configstore).
type configUpdatedEvent struct {
	BotID string `json:"bot_id"`
	Type  string `json:"type"`
}

// startNATS اشتراک‌های NATS برای دریافت آپدیت‌ها را برقرار می‌کند.
// - config.updated : اطلاع تغییر پیکربندی (پاک‌سازی کش‌ها)
// - uploader.settings.<botID> : اعمال زندهٔ کلید/مقدار تنظیمات و متن‌ها
func (h *Handler) startNATS() {
	if h.Eng == nil || h.Eng.Nats == nil {
		return
	}

	// آپدیت کلید/مقدار تنظیمات و متن‌ها — اعمال زنده روی store.
	subj := fmt.Sprintf("uploader.settings.%d", h.Eng.BotID)
	if err := h.Eng.Nats.Subscribe(subj, func(data []byte) {
		var u settingsUpdate
		if err := json.Unmarshal(data, &u); err != nil {
			h.Log.Error("nats settings: bad payload", ports.F("err", err))
			return
		}
		ctx := context.Background()
		applied := 0
		if u.Key != "" {
			h.LogErr("nats settings: set", h.Store.SetSetting(ctx, u.Key, u.Value))
			applied++
		}
		for k, v := range u.Settings {
			h.LogErr("nats settings: set", h.Store.SetSetting(ctx, k, v))
			applied++
		}
		h.Log.Info("nats settings applied", ports.F("count", applied))
	}); err != nil {
		h.Log.Error("nats subscribe uploader.settings failed", ports.F("err", err))
	}

	// اطلاع کلیِ تغییر پیکربندی.
	if err := h.Eng.Nats.Subscribe("config.updated", func(data []byte) {
		var ev configUpdatedEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			return
		}
		// فقط برای این ربات (یا broadcast بدون bot_id)
		if ev.BotID != "" && ev.BotID != h.InstanceID && ev.BotID != fmt.Sprintf("%d", h.Eng.BotID) {
			return
		}
		h.Log.Info("config.updated received", ports.F("type", ev.Type))
	}); err != nil {
		h.Log.Error("nats subscribe config.updated failed", ports.F("err", err))
	}

	// رویداد عضویت از member-bot → شمارش دقیق حد عضوِ قفل‌ها.
	if err := h.Eng.Nats.Subscribe(protocol.SubjMembershipJoined, func(data []byte) {
		var ev protocol.MembershipJoinedEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			return
		}
		h.onMembershipJoined(ev.TelegramID, ev.CommunityID)
	}); err != nil {
		h.Log.Error("nats subscribe membership.joined failed", ports.F("err", err))
	}

	h.Log.Info("nats config listeners started", ports.F("subject", subj))
}

// onMembershipJoined با دریافت رویداد عضویت، اگر چت یکی از قفل‌های اجباریِ
// دارای حد عضوِ این ربات باشد، یک‌بار برای هر کاربر می‌شمارد و در صورت رسیدن
// به حد، قفل را غیرفعال می‌کند.
func (h *Handler) onMembershipJoined(telegramID, chatID int64) {
	if h.Cache == nil || chatID == 0 {
		return
	}
	ctx := context.Background()
	lock, err := h.Store.FindForceJoinByChat(ctx, chatID)
	h.LogErr("onMembershipJoined: find lock", err)
	if lock == nil || !lock.IsMandatory() || lock.MemberCap <= 0 {
		return
	}
	key := fmt.Sprintf("lkjoin:%s:%s:%d", h.InstanceID, lock.ID, telegramID)
	ok, err := h.Cache.SetNX(ctx, key, "1", 720*time.Hour)
	if err != nil {
		h.LogErr("onMembershipJoined: setnx", err)
		return
	}
	if !ok {
		return // قبلاً شمرده شده
	}
	if deactivated := h.Store.IncrLockJoined(ctx, lock.ID); deactivated && h.OwnerID != 0 {
		if _, sendErr := h.Bot.Send(&tele.User{ID: h.OwnerID},
			fmt.Sprintf("✅ قفل «%s» به حد عضو %d رسید و خودکار غیرفعال شد.", lockTitle(lock), lock.MemberCap)); sendErr != nil {
			h.LogErr("onMembershipJoined: notify owner", sendErr)
		}
	}
}
