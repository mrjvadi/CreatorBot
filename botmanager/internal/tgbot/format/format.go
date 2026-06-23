// Package format توابعِ محضِ قالب‌بندیِ متن و آیکن را نگه می‌دارد.
// همگی توابع آزاد (بدون وابستگی به Handler) هستند.
package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/mrjvadi/creatorbot/shared-core/models"
)

// StatusIcon آیکنِ کوتاهِ وضعیتِ instance.
func StatusIcon(s models.InstanceStatus) string {
	switch s {
	case models.StatusRunning:
		return "🟢"
	case models.StatusStopped:
		return "🔴"
	case models.StatusPending:
		return "🟡"
	case models.StatusError:
		return "⚠️"
	}
	return "⚪️"
}

// StatusEmoji مشابه StatusIcon (سازگاری با کدِ قدیمی).
func StatusEmoji(s models.InstanceStatus) string { return StatusIcon(s) }

// BotTypeEmoji آیکنِ نوعِ سرویس؛ برای نوعِ ناشناخته 🤖 (graceful).
func BotTypeEmoji(t models.BotType) string {
	switch t {
	case models.BotTypeUploader:
		return "📤"
	case models.BotTypeVPN:
		return "🔒"
	case models.BotTypeArchive:
		return "📂"
	case models.BotTypeMember:
		return "👥"
	}
	return "🤖"
}

func FmtInstance(inst models.BotInstance, _ bool) string {
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
		StatusEmoji(inst.Status), inst.ContainerName, expiry, inst.Status, inst.ID)
}

func FmtServer(s models.Server) string {
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

func FmtTemplate(t models.BotTemplate) string {
	active := "✅"
	if !t.IsActive {
		active = "❌"
	}
	return fmt.Sprintf("• %s <b>%s</b> [%s]\n  <code>%s:%s</code>\n  ID: <code>%s</code>",
		active, t.Name, t.Type, t.ImageName, t.ImageTag, t.ID)
}

func FmtPlan(p models.Plan) string {
	return fmt.Sprintf("• <b>%s</b> — %d days — <b>%.0f</b>\n  ID: <code>%s</code>",
		p.Name, p.DurationDay, p.Price, p.ID)
}

func FmtUser(u models.User) string {
	var role string
	switch u.Role {
	case models.RoleOwner:
		role = "👑"
	case models.RoleAdmin:
		role = "🛡"
	default:
		role = "👤"
	}
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

func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}
