package tgbot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// ── Auth helpers ───────────────────────────────────────────

func (h *Handler) isOwner(c tele.Context) bool {
	return c.Sender().ID == h.ownerID
}

func (h *Handler) isAdmin(c tele.Context) bool {
	if c.Sender().ID == h.ownerID {
		return true
	}
	ctx := context.Background()
	u, _ := h.store.FindUserByTelegramID(ctx, c.Sender().ID)
	return u != nil && (u.Role == models.RoleAdmin || u.Role == models.RoleOwner)
}

func (h *Handler) getOrCreateUser(ctx context.Context, c tele.Context) (*models.User, error) {
	u, err := h.store.FindUserByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		return nil, err
	}
	if u != nil {
		return u, nil
	}
	role := models.RoleUser
	if c.Sender().ID == h.ownerID {
		role = models.RoleOwner
	}
	u = &models.User{
		TelegramID: c.Sender().ID,
		Username:   c.Sender().Username,
		FirstName:  c.Sender().FirstName,
		Role:       role,
	}
	return u, h.store.UpsertUser(ctx, u)
}

// ── i18n helpers ───────────────────────────────────────────

func (h *Handler) F(key string, val any) ports.Field {
	return ports.F(key, val)
}

func (h *Handler) botTypeLabel(ctx context.Context, uid int64, t models.BotType) string {
	m := map[models.BotType]i18n.Key{
		models.BotTypeUploader: i18n.KeyBotTypeUploader,
		models.BotTypeVPN:      i18n.KeyBotTypeVPN,
		models.BotTypeArchive:  i18n.KeyBotTypeArchive,
		models.BotTypeMember:   i18n.KeyBotTypeMember,
	}
	if k, ok := m[t]; ok {
		return h.t(ctx, uid, k)
	}
	return string(t)
}

// ── Format helpers ─────────────────────────────────────────

func fmtInstance(inst models.BotInstance, _ bool) string {
	expiry := ""
	if inst.ExpiresAt != nil {
		rem := time.Until(*inst.ExpiresAt)
		if rem < 0 {
			expiry = "\n  ❌ Expired"
		} else {
			expiry = fmt.Sprintf("\n  ⏰ %d days", int(rem.Hours()/24))
		}
	}
	return fmt.Sprintf("%s <b>%s</b>%s\n  Status: %s\n  ID: <code>%s</code>",
		statusEmoji(inst.Status), inst.ContainerName, expiry, inst.Status, inst.ID)
}

func fmtServer(s models.Server) string {
	online := "🔴"
	if s.IsOnline {
		online = "🟢"
	}
	lastSeen := ""
	if !s.LastSeen.IsZero() {
		diff := time.Since(s.LastSeen)
		if diff < time.Minute {
			lastSeen = " (just now)"
		} else {
			lastSeen = fmt.Sprintf(" (%dm ago)", int(diff.Minutes()))
		}
	}
	return fmt.Sprintf("• %s <b>%s</b>%s\n  IP: <code>%s</code>\n  ID: <code>%s</code>",
		online, s.Name, lastSeen, s.IP, s.ID)
}

func fmtTemplate(t models.BotTemplate) string {
	active := "✅"
	if !t.IsActive {
		active = "❌"
	}
	return fmt.Sprintf("• %s <b>%s</b> [%s]\n  <code>%s:%s</code>\n  ID: <code>%s</code>",
		active, t.Name, t.Type, t.ImageName, t.ImageTag, t.ID)
}

func fmtPlan(p models.Plan) string {
	return fmt.Sprintf("• <b>%s</b> — %d days — <b>%.0f</b>\n  ID: <code>%s</code>",
		p.Name, p.DurationDay, p.Price, p.ID)
}

func fmtUser(u models.User) string {
	role := map[models.UserRole]string{
		models.RoleOwner: "👑",
		models.RoleAdmin: "🛡",
		models.RoleUser:  "👤",
	}[u.Role]
	blocked := ""
	if u.IsBlocked {
		blocked = " 🚫"
	}
	uname := ""
	if u.Username != "" {
		uname = " @" + u.Username
	}
	return fmt.Sprintf("• %s <b>%s</b>%s%s\n  <code>%d</code>",
		role, u.FirstName, uname, blocked, u.TelegramID)
}

func fmtLink(l models.InviteLink, botUsername string) string {
	valid := "✅"
	if !l.IsValid() {
		valid = "❌"
	}
	limit := "∞"
	if l.MaxUse > 0 {
		limit = fmt.Sprintf("%d/%d", l.UsedCount, l.MaxUse)
	}
	label := l.Label
	if label == "" {
		label = "—"
	}
	return fmt.Sprintf("%s [%s] <b>%s</b> — %s\n  <code>https://t.me/%s?start=%s</code>",
		valid, l.BotType, label, limit, botUsername, l.Token)
}

func statusEmoji(s models.InstanceStatus) string {
	m := map[models.InstanceStatus]string{
		models.StatusRunning: "🟢",
		models.StatusStopped: "🔴",
		models.StatusPending: "🟡",
		models.StatusError:   "⚠️",
	}
	if e, ok := m[s]; ok {
		return e
	}
	return "⚪️"
}

func botTypeEmoji(t models.BotType) string {
	m := map[models.BotType]string{
		models.BotTypeUploader: "📤",
		models.BotTypeVPN:      "🔒",
		models.BotTypeArchive:  "📂",
		models.BotTypeMember:   "👥",
	}
	if e, ok := m[t]; ok {
		return e
	}
	return "🤖"
}

func genToken() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}
