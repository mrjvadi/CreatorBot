package tgbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/ads-bot/internal/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ── ثبت کانال ────────────────────────────────────────────

func (h *Handler) onAddChannel(c tele.Context) error {
	ctx := context.Background()
	h.setStep(ctx, c.Sender().ID, stepChannelFwd)
	return c.Send(
		"<b>📢 افزودن کانال</b>\n\n"+
			"یک پیام از کانال مورد نظر را forward کنید.\n\n"+
			"⚠️ ربات باید ادمین کانال باشد.",
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleChannelForward(ctx context.Context, c tele.Context, st wizardState) error {
	uid := c.Sender().ID
	msg := c.Message()

	if msg.ForwardFromChat == nil {
		return c.Send("لطفاً یک پیام از کانال forward کنید.")
	}

	ch := msg.ForwardFromChat
	if ch.Type != "channel" {
		return c.Send("پیام باید از یک کانال باشد.")
	}

	// بررسی تکراری نبودن
	existing, _ := h.store.FindChannelByTelegramID(ctx, ch.ID)
	if existing != nil {
		return c.Send(
			fmt.Sprintf("این کانال قبلاً ثبت شده است.\n🆔 <code>%s</code>", existing.ID),
			tele.ModeHTML, kbMain(),
		)
	}

	// بررسی ادمین بودن ربات
	botMember, err := h.bot.ChatMemberOf(&tele.Chat{ID: ch.ID}, h.bot.Me)
	if err != nil || (botMember.Role != tele.Administrator && botMember.Role != tele.Creator) {
		return c.Send("❌ ربات ادمین این کانال نیست.\n\nابتدا ربات را به کانال اضافه و ادمین کنید.")
	}

	h.setStep(ctx, uid, stepChannelCPJ,
		"channel_id", strconv.FormatInt(ch.ID, 10),
		"channel_name", ch.Title,
		"member_count", strconv.Itoa(ch.MembersCount),
	)

	return c.Send(
		fmt.Sprintf(
			"کانال: <b>%s</b>\n"+
				"اعضا: %d\n\n"+
				"حداقل CPJ را وارد کنید (TON):\n"+
				"تبلیغاتی که CPJ کمتر از این دارند پذیرفته نمی‌شوند.\n"+
				"مثال: <code>0.005</code>",
			ch.Title, ch.MembersCount,
		),
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleChannelCPJ(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	cpj, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || cpj < 0 {
		return c.Send("❌ مقدار نامعتبر.")
	}

	pub, _ := h.store.FindPublisher(ctx, uid)
	if pub == nil {
		return c.Send(h.t("error"))
	}

	channelID, _ := strconv.ParseInt(st.Data["channel_id"], 10, 64)
	memberCount, _ := strconv.Atoi(st.Data["member_count"])

	ch := &store.AdChannel{
		OwnerID:     pub.ID,
		ChannelID:   channelID,
		ChannelName: st.Data["channel_name"],
		MemberCount: memberCount,
		CPJRate:     cpj,
		IsVerified:  false,
		IsActive:    true,
	}

	if err := h.store.CreateChannel(ctx, ch); err != nil {
		h.log.Error("createChannel", ports.F("err", err))
		return c.Send("❌ خطا در ثبت کانال.")
	}

	// اطلاع به ادمین برای verify
	if h.ownerID != 0 {
		admin := &tele.Chat{ID: h.ownerID}
		verifyKB := &tele.ReplyMarkup{}
		verifyKB.Inline(verifyKB.Row(
			verifyKB.Data("✅ تأیید", "verify_ch:"+ch.ID.String()),
			verifyKB.Data("❌ رد", "reject_ch:"+ch.ID.String()),
		))
		h.bot.Send(admin,
			fmt.Sprintf(
				"📢 <b>کانال جدید</b>\n\n"+
					"نام: %s\n"+
					"اعضا: %d\n"+
					"CPJ: %.3f TON\n"+
					"صاحب: <code>%d</code>\n"+
					"🆔 <code>%s</code>",
				ch.ChannelName, ch.MemberCount, ch.CPJRate,
				uid, ch.ID,
			),
			tele.ModeHTML, verifyKB,
		)
	}

	return c.Send(
		fmt.Sprintf(
			"✅ <b>کانال ثبت شد</b>\n\n"+
				"📢 %s\n"+
				"👥 %d عضو\n"+
				"💰 حداقل CPJ: %.3f TON\n\n"+
				"⏳ منتظر تأیید ادمین...",
			ch.ChannelName, ch.MemberCount, ch.CPJRate,
		),
		tele.ModeHTML, kbMain(),
	)
}

// ── لیست کانال‌ها ─────────────────────────────────────────

func (h *Handler) onMyChannels(c tele.Context) error {
	ctx := context.Background()
	pub, _ := h.store.FindPublisher(ctx, c.Sender().ID)
	if pub == nil {
		return c.Send("ابتدا /start بزنید.")
	}

	channels, _ := h.store.ListChannelsByOwner(ctx, pub.ID)
	if len(channels) == 0 {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.Data("➕ افزودن کانال", "add_ch")))
		return c.Send("هیچ کانالی ثبت نکرده‌اید.", kb)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>📢 کانال‌های شما (%d)</b>\n\n", len(channels)))
	for _, ch := range channels {
		verified := "⏳ در انتظار تأیید"
		if ch.IsVerified {
			verified = "✅ تأیید شده"
		}
		active := "🟢"
		if !ch.IsActive {
			active = "🔴"
		}
		sb.WriteString(fmt.Sprintf(
			"%s <b>%s</b> %s\n"+
				"👥 %d اعضا | 💰 CPJ: %.3f TON\n"+
				"🆔 <code>%s</code>\n\n",
			active, ch.ChannelName, verified,
			ch.MemberCount, ch.CPJRate, ch.ID,
		))
	}

	return c.Send(sb.String(), tele.ModeHTML, kbMain())
}
