// mychatmember.go — تشخیص ادمین‌شدن ربات در کانال خریدار (اجاره‌ی قفل).
//
// فقط برای instance هایی که LockMode=rented هستند معنی دارد (نگاه کنید به
// فاز ۱: eng.InstanceInfo). وقتی خریدار این bot رایگان را در کانال خودش
// ادمین می‌کند، باید به ads-bot اطلاع دهیم تا قفل‌کردن برای آن تبلیغ شروع شود.
package tgbot

import (
	"context"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

func (h *Handler) onMyChatMember(c tele.Context) error {
	// فقط برای instance های اجاره‌ای این رویداد معنی دارد.
	if h.eng == nil || h.eng.InstanceInfo == nil || !h.eng.InstanceInfo.IsRentedLock() {
		return nil
	}

	cm := c.ChatMember()
	if cm == nil || cm.NewChatMember == nil {
		return nil
	}

	role := cm.NewChatMember.Role
	isNowAdmin := role == tele.Administrator || role == tele.Creator
	if !isNowAdmin {
		return nil // فقط وقتی ادمین شد اهمیت دارد، نه ترفیع/تنزل دیگر
	}

	if h.eng.Nats == nil {
		h.log.Warn("my_chat_member: nats unavailable, cannot confirm to ads-bot")
		return nil
	}

	mc := memberclient.New(h.eng.Nats)
	ctx := context.Background()
	if err := mc.ConfirmChannelAdmin(ctx, h.eng.BotID); err != nil {
		h.log.Error("confirm channel admin failed", ports.F("err", err))
		return nil
	}

	h.log.Info("confirmed channel admin to ads-bot — lock enforcement starting",
		ports.F("bot_id", h.eng.BotID))
	return nil
}
