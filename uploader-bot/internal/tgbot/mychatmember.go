// mychatmember.go — تشخیص ادمین‌شدن ربات در کانال خریدار (اجاره‌ی قفل).
//
// فقط برای instance هایی معنی دارد که RentalStatus.IsInCampaign() باشد —
// یعنی الان به یک کمپینِ اجاره‌ی قفلِ فعال در ads-bot وصل‌اند (رجوع
// core.App.RentalStatus، پرشده توسط memberclient.RunStatusLoop در main.go).
// وقتی خریدار این bot رایگان را در کانال خودش ادمین می‌کند، باید به ads-bot
// اطلاع دهیم تا قفل‌کردن برای آن تبلیغ شروع شود.
package tgbot

import (
	"context"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

func (h *Handler) onMyChatMember(c tele.Context) error {
	// فقط برای instance های اجاره‌ای این رویداد معنی دارد.
	if h.RentalStatus == nil || !h.RentalStatus.IsInCampaign() {
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

	if h.Eng.Nats == nil {
		h.Log.Warn("my_chat_member: nats unavailable, cannot confirm to ads-bot")
		return nil
	}

	mc := memberclient.New(h.Eng.Nats)
	ctx := context.Background()
	if err := mc.ConfirmChannelAdmin(ctx, h.Eng.BotID); err != nil {
		h.Log.Error("confirm channel admin failed", ports.F("err", err))
		return nil
	}

	h.Log.Info("confirmed channel admin to ads-bot — lock enforcement starting",
		ports.F("bot_id", h.Eng.BotID))
	return nil
}
