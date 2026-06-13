package tgbot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/vpn-bot/internal/models"
)

// ── membership check ──────────────────────────────────────

func (h *Handler) checkMembership(ctx context.Context, c tele.Context) error {
	if h.channelID == 0 {
		return nil
	}
	status, err := h.sender.GetChatMember(ctx, h.channelID, c.Sender().ID)
	if err != nil || !status.IsActive() {
		kb := &tele.ReplyMarkup{}
		kb.Inline(kb.Row(kb.URL("📢 عضویت در کانال", fmt.Sprintf("https://t.me/c/%d", h.channelID))))
		return c.Send("⛔️ برای استفاده از ربات باید در کانال عضو باشید.", kb)
	}
	return nil
}

// ── getOrCreateUser ───────────────────────────────────────

func (h *Handler) getOrCreate(ctx context.Context, c tele.Context) (*models.User, error) {
	u, err := h.store.FindUserByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		return nil, err
	}
	if u != nil {
		return u, nil
	}
	u = &models.User{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
	}
	return u, h.store.UpsertUser(ctx, u)
}

func (h *Handler) isAdmin(c tele.Context) bool {
	return c.Sender().ID == h.ownerID
}

// ── format helpers ────────────────────────────────────────

func fmtSub(sub models.Subscription) string {
	statusIcon := map[models.SubscriptionStatus]string{
		models.SubActive:   "🟢",
		models.SubExpired:  "🔴",
		models.SubDisabled: "🟡",
	}[sub.Status]

	expiry := ""
	rem := time.Until(sub.ExpiresAt)
	if sub.ExpiresAt.IsZero() {
		expiry = "♾ ابدی"
	} else if rem < 0 {
		expiry = "❌ منقضی شده"
	} else if rem < 24*time.Hour {
		expiry = fmt.Sprintf("⚠️ %d ساعت مانده", int(rem.Hours()))
	} else {
		expiry = fmt.Sprintf("⏰ %d روز مانده", int(rem.Hours()/24))
	}

	traffic := ""
	if sub.DataLimit > 0 {
		usedGB := float64(sub.UsedData) / 1e9
		totalGB := float64(sub.DataLimit) / 1e9
		pct := int(float64(sub.UsedData) / float64(sub.DataLimit) * 100)
		traffic = fmt.Sprintf("\n📊 ترافیک: %.1f/%.1f GB (%d%%)", usedGB, totalGB, pct)
	} else {
		traffic = "\n📊 ترافیک: نامحدود"
	}

	return fmt.Sprintf(
		"%s <b>%s</b>\n%s%s",
		statusIcon, sub.Username, expiry, traffic,
	)
}

func fmtPlan(p models.Plan) string {
	traffic := "نامحدود"
	if p.DataGB > 0 {
		traffic = fmt.Sprintf("%.0f GB", p.DataGB)
	}
	return fmt.Sprintf(
		"📦 <b>%s</b>\n"+
			"⏳ %s\n"+
			"📶 %s\n"+
			"💰 %.0f تومان",
		p.Name, fmtDuration(p.DurationDay), traffic, p.Price,
	)
}

// ── vpn username generator ────────────────────────────────

// genVPNUsername یک username یکتا برای VPN panel می‌سازد.
// فرمت: u<telegramID>_<4 char random>
func genVPNUsername(telegramID int64) string {
	b := make([]byte, 2)
	rand.Read(b)
	suffix := hex.EncodeToString(b)
	return fmt.Sprintf("u%d_%s", telegramID, suffix)
}

// ── QR code helper ────────────────────────────────────────

// qrURL یک URL برای تولید QR code از subscription link.
func qrURL(link string) string {
	return "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=" +
		strings.ReplaceAll(link, "//", "%2F%2F")
}
