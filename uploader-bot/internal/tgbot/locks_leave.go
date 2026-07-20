package tgbot

import (
	"context"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// onChatMember آپدیت عضویت در کانال/گروه را می‌گیرد. اگر کاربری از یک قفلِ
// اجباری لفت بدهد و «گزارش لفت» روشن باشد، به او اطلاع و درخواست عضویت مجدد می‌دهد.
//
// نیازمندی: ربات باید در آن چت ادمین باشد و آپدیت‌های chat_member در poller
// فعال باشد (allowed_updates شامل "chat_member").
func (h *Handler) onChatMember(c tele.Context) error {
	// اول: اگر این instance رایگان به یک کمپینِ اجاره‌ی فعال وصل است، عضویتِ
	// واقعیِ کانالِ خریدار را به membership.joined/left منتشر کن (پاداشِ
	// per-join در ads-bot). این بی‌ربط به منطقِ «گزارش لفت» پایین است.
	if h.JoinPublisher != nil {
		h.LogErr("onChatMember: join publisher", h.JoinPublisher.HandleChatMember(c))
	}

	cm := c.ChatMember()
	if cm == nil || cm.NewChatMember == nil || cm.Chat == nil {
		return nil
	}
	ctx := context.Background()
	if h.Store.GetSetting(ctx, models.SettingLeaveReport) != "true" {
		return nil
	}
	if !leftRole(cm.NewChatMember.Role) || !joinedRole(roleOf(cm.OldChatMember)) {
		return nil // فقط گذار «عضو → خارج‌شده»
	}
	lock, err := h.Store.FindForceJoinByChat(ctx, cm.Chat.ID)
	h.LogErr("onChatMember: find lock", err)
	if lock == nil || !lock.IsMandatory() {
		return nil
	}
	user := cm.NewChatMember.User
	if user == nil {
		return nil
	}

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	if u := lock.LinkURL(); u != "" {
		rows = append(rows, kb.Row(kb.URL("🔒 "+lockTitle(lock), u)))
	}
	rows = append(rows, kb.Row(kb.Data("✅ عضو شدم", "check_join")))
	kb.Inline(rows...)

	if _, err := h.Bot.Send(&tele.User{ID: user.ID},
		"⚠️ شما از «"+lockTitle(lock)+"» خارج شدید.\nبرای ادامهٔ استفاده از ربات، دوباره عضو شوید 👇", kb); err != nil {
		h.LogErr("onChatMember: notify", err)
	}
	return nil
}

func roleOf(m *tele.ChatMember) tele.MemberStatus {
	if m == nil {
		return tele.Left
	}
	return m.Role
}

func leftRole(r tele.MemberStatus) bool {
	return r == tele.Left || r == tele.Kicked
}

func joinedRole(r tele.MemberStatus) bool {
	return r == tele.Member || r == tele.Administrator || r == tele.Creator || r == tele.Restricted
}
