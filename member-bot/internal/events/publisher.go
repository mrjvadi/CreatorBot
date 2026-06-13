// Package events رویدادهای join/leave را از Telegram update ها دریافت
// و به NATS publish می‌کند تا fraud-engine و community-service مطلع شوند.
package events

import (
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
		p.publishJoin(userID, chatID, update.NewChatMember.User.Username, "organic", "")
	} else if wasInside && isOutside {
		p.publishLeave(userID, chatID)
	}

	return nil
}

// onUserJoined برای group ها (ساده‌تر از ChatMember).
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

func (p *Publisher) publishJoin(telegramID, chatID int64, username, source, campaignID string) {
	payload := map[string]any{
		"telegram_id":  telegramID,
		"community_id": chatID,
		"source":       source,
		"campaign_id":  campaignID,
		"joined_at":    time.Now().Unix(),
		"username":     username,
	}

	// به همه مصرف‌کنندگان
	p.nc.PublishCore("membership.joined", payload)

	// مستقیم به fraud-engine (fire-and-forget)
	if p.fraud != nil {
		p.fraud.ReportJoin(telegramID, chatID, source, campaignID)
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
