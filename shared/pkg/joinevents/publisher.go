// Package joinevents رویدادهای Telegram chat_member/user_joined/user_left
// را می‌گیرد و membership.joined/membership.left را روی NATS منتشر می‌کند.
//
// این منطق در اصل داخل member-bot/internal/events بود؛ به این‌جا منتقل شد
// تا هر ربات دیگری هم (uploader-bot/vpn-bot/archive-bot، وقتی instance
// رایگان‌شان به یک کمپینِ اجاره‌ی قفلِ فعال در ads-bot وصل است) بتواند
// بدونِ کپی همین رفتار را داشته باشد — با Gate برای فعال/غیرفعال‌کردنِ
// شرطی (member-bot همیشه فعال است، ربات‌های رایگان فقط وقتی در کمپین‌اند).
package joinevents

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

	// Gate اگر ست شده باشد و false برگرداند، هیچ رویدادی منتشر نمی‌شود —
	// برای ربات‌های رایگان که فقط وقتی به کمپینِ اجاره‌ی فعالی وصل‌اند باید
	// این کار را انجام بدهند. nil یعنی همیشه فعال (رفتار قبلیِ member-bot).
	Gate func() bool

	// CampaignID (اختیاری) برای attribution دقیق‌ترِ گزارش‌های fraud-engine
	// وقتی این publisher برای یک ربات رایگانِ متصل به یک کمپینِ خاص است.
	CampaignID func() string
}

func NewPublisher(nc *natsclient.Client, fraud *fraudclient.Client, log ports.Logger) *Publisher {
	return &Publisher{nc: nc, fraud: fraud, log: log}
}

// Register handلر های مربوط به عضویت را مستقیم ثبت می‌کند — فقط برای
// بات‌هایی که از قبل خودشان روی این سه event هندلر ندارند (مثلاً
// member-bot). اگر bot شما از قبل روی یکی از این‌ها (مثلاً OnChatMember)
// هندلر دارد، به‌جایش HandleChatMember/HandleUserJoined/HandleUserLeft را
// از داخل همان هندلرِ موجودتان صدا بزنید (رجوع uploader-bot).
func (p *Publisher) Register(b *tele.Bot) {
	b.Handle(tele.OnChatMember, p.HandleChatMember)
	b.Handle(tele.OnUserJoined, p.HandleUserJoined)
	b.Handle(tele.OnUserLeft, p.HandleUserLeft)
}

func (p *Publisher) enabled() bool {
	return p.Gate == nil || p.Gate()
}

// HandleChatMember برای channel/group update های عضویت.
// این اصلی‌ترین handler است — وقتی bot ادمین کانال/گروه است.
func (p *Publisher) HandleChatMember(c tele.Context) error {
	if !p.enabled() {
		return nil
	}
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

// HandleUserJoined برای group ها (ساده‌تر از ChatMember).
// نکته: برخلاف HandleChatMember، این رویداد هیچ اطلاعات invite-link ندارد
// (تلگرام آن را در new_chat_members نمی‌فرستد) — پس "organic" واقعاً
// درست است، نه یک مقدار هاردکدشده‌ی جایگزین.
func (p *Publisher) HandleUserJoined(c tele.Context) error {
	if !p.enabled() {
		return nil
	}
	for _, user := range c.Message().UsersJoined {
		p.publishJoin(user.ID, c.Chat().ID, user.Username, "organic", "")
	}
	return nil
}

func (p *Publisher) HandleUserLeft(c tele.Context) error {
	if !p.enabled() {
		return nil
	}
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

	// مستقیم به fraud-engine (fire-and-forget) — campaign_id (اگر ست شده
	// باشد) attribution دقیقِ کمپینِ اجاره‌ی مربوطه را ممکن می‌کند.
	if p.fraud != nil {
		campaignID := ""
		if p.CampaignID != nil {
			campaignID = p.CampaignID()
		}
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
	if !p.enabled() {
		return
	}
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
