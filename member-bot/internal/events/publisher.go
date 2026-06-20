// Package events رویدادهای join/leave را از Telegram update ها دریافت
// و به NATS publish می‌کند تا fraud-engine و community-service مطلع شوند.
package events

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	natsclient "github.com/mrjvadi/creatorbot/shared/pkg/adapters/nats"
	"github.com/mrjvadi/creatorbot/shared/pkg/fraudclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// Publisher Telegram chat_member update ها را به NATS می‌فرستد.
type Publisher struct {
	nc    *natsclient.Client
	fraud *fraudclient.Client
	log   ports.Logger
}

func NewPublisher(nc *natsclient.Client, fraud *fraudclient.Client, log ports.Logger) *Publisher {
	return &Publisher{nc: nc, fraud: fraud, log: log}
}

// Register handler های مربوط به عضویت را ثبت می‌کند.
func (p *Publisher) Register(b *tele.Bot) {
	b.Handle(tele.OnChatMember, p.onChatMember)
	b.Handle(tele.OnUserJoined, p.onUserJoined)
	b.Handle(tele.OnUserLeft, p.onUserLeft)
}

// onChatMember برای channel/group update های عضویت.
// این اصلی‌ترین handler است — وقتی bot ادمین کانال/گروه است.
func (p *Publisher) onChatMember(c tele.Context) error {
	update := c.ChatMember()
	if update == nil {
		return nil
	}

	userID := update.NewChatMember.User.ID
	chatID := c.Chat().ID

	oldStatus := update.OldChatMember.Role
	newStatus := update.NewChatMember.Role

	wasOutside := oldStatus == tele.Left || oldStatus == tele.Kicked || oldStatus == ""
	isInside := newStatus == tele.Member || newStatus == tele.Administrator || newStatus == tele.Creator || newStatus == tele.Restricted

	wasInside := oldStatus == tele.Member || oldStatus == tele.Administrator || oldStatus == tele.Creator || oldStatus == tele.Restricted
	isOutside := newStatus == tele.Left || newStatus == tele.Kicked

	if wasOutside && isInside {
		source, inviteHash := "organic", ""
		// تلگرام وقتی کاربر از طریق یک invite link مشخص join کند، آن لینک را
		// در update.InviteLink برمی‌گرداند (فقط وقتی bot ادمین کانال/گروه است).
		// از این، attribution واقعی (به‌جای "organic" هاردکد) ساخته می‌شود.
		if update.InviteLink != nil && update.InviteLink.InviteLink != "" {
			source = "invite_link"
			inviteHash = extractInviteHash(update.InviteLink.InviteLink)
		}
		p.publishJoin(userID, chatID, update.NewChatMember.User.Username, source, inviteHash)
	} else if wasInside && isOutside {
		p.publishLeave(userID, chatID)
	}

	return nil
}

// extractInviteHash از URL کامل لینک دعوت تلگرام (مثلا
// "https://t.me/+AbCdEfGhIjK" یا ".../joinchat/AbCdEfGhIjK") فقط بخش
// hash را استخراج می‌کند — همان مقداری که community-service موقع ساخت
// لینک ذخیره کرده (InviteHash).
func extractInviteHash(url string) string {
	for _, sep := range []string{"/+", "/joinchat/"} {
		if idx := strings.LastIndex(url, sep); idx != -1 {
			return url[idx+len(sep):]
		}
	}
	// لینک عمومی (t.me/channelusername) — hash اختصاصی ندارد
	return ""
}

// onUserJoined برای group ها (ساده‌تر از ChatMember).
// نکته: برخلاف onChatMember، این رویداد هیچ اطلاعات invite-link ندارد
// (تلگرام آن را در new_chat_members نمی‌فرستد) — پس "organic" واقعاً
// درست است، نه یک مقدار هاردکدشده‌ی جایگزین.
func (p *Publisher) onUserJoined(c tele.Context) error {
	for _, user := range c.Message().UsersJoined {
		p.publishJoin(user.ID, c.Chat().ID, user.Username, "organic", "")
	}
	return nil
}

func (p *Publisher) onUserLeft(c tele.Context) error {
	if c.Message().UserLeft == nil {
		return nil
	}
	p.publishLeave(c.Message().UserLeft.ID, c.Chat().ID)
	return nil
}

// ── publish helpers ────────────────────────────────────────

func (p *Publisher) publishJoin(telegramID, chatID int64, username, source, inviteHash string) {
	payload := map[string]any{
		"telegram_id":  telegramID,
		"community_id": chatID,
		"source":       source,
		"invite_hash":  inviteHash,
		"joined_at":    time.Now().Unix(),
		"username":     username,
	}

	// به همه مصرف‌کنندگان
	p.nc.PublishCore("membership.joined", payload)

	// مستقیم به fraud-engine (fire-and-forget).
	// نکته: ReportJoin پارامتر چهارم را campaign_id می‌نامد ولی فعلاً
	// تنها attribution که داریم invite_hash (گروه) است، نه campaign_id
	// (تبلیغ ads-bot) — این دو مفهوم متفاوتند و فاز بعد باید campaign_id
	// واقعی را هم به این مسیر متصل کند.
	if p.fraud != nil {
		p.fraud.ReportJoin(telegramID, chatID, source, inviteHash)
	}

	p.log.Info("membership.joined published",
		ports.F("user", telegramID),
		ports.F("chat", chatID))
}

func (p *Publisher) publishLeave(telegramID, chatID int64) {
	payload := map[string]any{
		"telegram_id":  telegramID,
		"community_id": chatID,
		"left_at":      time.Now().Unix(),
	}

	p.nc.PublishCore("membership.left", payload)

	if p.fraud != nil {
		p.fraud.ReportLeave(telegramID, chatID)
	}

	p.log.Info("membership.left published",
		ports.F("user", telegramID),
		ports.F("chat", chatID))
}

// PublishActivity فعالیت یک کاربر در گروه را publish می‌کند.
// از message handler صدا زده می‌شود.
func (p *Publisher) PublishActivity(telegramID, chatID int64, messages, replies, reactions int) {
	payload := map[string]any{
		"telegram_id":  telegramID,
		"community_id": chatID,
		"messages":     messages,
		"replies":      replies,
		"reactions":    reactions,
	}
	p.nc.PublishCore("community.activity.updated", payload)

	if p.fraud != nil {
		p.fraud.ReportActivity(telegramID, chatID, messages, replies, reactions)
	}
}
