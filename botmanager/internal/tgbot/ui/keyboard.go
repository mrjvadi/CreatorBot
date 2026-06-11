// Package ui تمام inline keyboard های ربات را تعریف می‌کند.
// هر تابع یک *tele.ReplyMarkup برمی‌گرداند — هیچ منطق تجاری اینجا نیست.
package ui

import (
	"fmt"

	"github.com/mrjvadi/creatorbot/shared-core/models"
	tele "gopkg.in/telebot.v4"
)

// ── Main Menu ──────────────────────────────────────────────

func MainMenuOwner() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🖥 سرورها", "menu:servers"),
			kb.Data("📦 تمپلیت‌ها", "menu:templates"),
		),
		kb.Row(
			kb.Data("💰 پلن‌ها", "menu:plans"),
			kb.Data("🔗 لینک‌های دعوت", "menu:links"),
		),
		kb.Row(
			kb.Data("🤖 همه ربات‌ها", "menu:all_instances"),
			kb.Data("👥 کاربران", "menu:users"),
		),
	)
	return kb
}

func MainMenuUser() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🤖 ربات‌های من", "menu:my_instances")),
		kb.Row(kb.Data("👤 حساب من", "menu:me")),
	)
	return kb
}

// ── Server List ────────────────────────────────────────────

func ServerList(servers []models.Server) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range servers {
		icon := "🔴"
		if s.IsOnline {
			icon = "🟢"
		}
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("%s %s", icon, s.Name), "server:view:"+s.ID.String()),
		))
	}
	rows = append(rows,
		kb.Row(kb.Data("➕ افزودن سرور", "server:add")),
		kb.Row(kb.Data("🔙 بازگشت", "menu:main")),
	)
	kb.Inline(rows...)
	return kb
}

func ServerDetail(s models.Server) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🗑 حذف سرور", "server:delete:"+s.ID.String())),
		kb.Row(kb.Data("🔙 بازگشت", "menu:servers")),
	)
	return kb
}

func ConfirmDelete(action, id string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("✅ بله، حذف شود", action+":"+id),
			kb.Data("❌ خیر", "menu:main"),
		),
	)
	return kb
}

// ── Template List ──────────────────────────────────────────

func TemplateList(templates []models.BotTemplate) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, t := range templates {
		icon := botTypeIcon(models.BotType(t.Type))
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("%s %s", icon, t.Name), "tmpl:view:"+t.ID.String()),
		))
	}
	rows = append(rows,
		kb.Row(kb.Data("➕ افزودن تمپلیت", "tmpl:add")),
		kb.Row(kb.Data("🔙 بازگشت", "menu:main")),
	)
	kb.Inline(rows...)
	return kb
}

func TemplateTypeSelect() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("📤 Uploader", "tmpl:type:uploader"),
			kb.Data("🔒 VPN", "tmpl:type:vpn"),
		),
		kb.Row(
			kb.Data("📂 Archive", "tmpl:type:archive"),
			kb.Data("👥 Member", "tmpl:type:member"),
		),
		kb.Row(kb.Data("🔙 بازگشت", "menu:templates")),
	)
	return kb
}

// ── Plan List ─────────────────────────────────────────────

func PlanList(plans []models.Plan) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, p := range plans {
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("💰 %s — %d روز", p.Name, p.DurationDay),
				"plan:view:"+p.ID.String()),
		))
	}
	rows = append(rows,
		kb.Row(kb.Data("➕ افزودن پلن", "plan:add")),
		kb.Row(kb.Data("🔙 بازگشت", "menu:main")),
	)
	kb.Inline(rows...)
	return kb
}

// ── InviteLink ────────────────────────────────────────────

// InviteLinkTypeSelect انتخاب نوع ربات برای ساخت لینک دعوت.
func InviteLinkTypeSelect() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("📤 Uploader Bot", "link:type:uploader"),
			kb.Data("🔒 VPN Bot", "link:type:vpn"),
		),
		kb.Row(
			kb.Data("📂 Archive Bot", "link:type:archive"),
			kb.Data("👥 Member Bot", "link:type:member"),
		),
		kb.Row(kb.Data("🔙 بازگشت", "menu:links")),
	)
	return kb
}

// InviteLinkUseLimit انتخاب محدودیت استفاده از لینک.
func InviteLinkUseLimit(botType string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	prefix := "link:limit:" + botType + ":"
	kb.Inline(
		kb.Row(
			kb.Data("1️⃣ یک‌بار", prefix+"1"),
			kb.Data("3️⃣ سه‌بار", prefix+"3"),
		),
		kb.Row(
			kb.Data("5️⃣ پنج‌بار", prefix+"5"),
			kb.Data("♾ نامحدود", prefix+"0"),
		),
		kb.Row(kb.Data("🔙 بازگشت", "link:type:"+botType)),
	)
	return kb
}

// InviteLinkList لیست لینک‌های ساخته‌شده توسط owner.
func InviteLinkList(links []models.InviteLink, botUsername string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, l := range links {
		icon := "✅"
		if !l.IsValid() {
			icon = "❌"
		}
		label := fmt.Sprintf("%s %s [%s] %d/%d", icon, botTypeIcon(l.BotType), l.BotType, l.UsedCount, l.MaxUse)
		rows = append(rows, kb.Row(
			kb.Data(label, "link:view:"+l.Token),
		))
	}
	rows = append(rows,
		kb.Row(kb.Data("➕ لینک جدید", "link:new")),
		kb.Row(kb.Data("🔙 بازگشت", "menu:main")),
	)
	kb.Inline(rows...)
	return kb
}

func InviteLinkDetail(link models.InviteLink) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("🗑 حذف لینک", "link:delete:"+link.Token)),
		kb.Row(kb.Data("🔙 بازگشت", "menu:links")),
	)
	return kb
}

// ── Instance List ─────────────────────────────────────────

func InstanceList(instances []models.BotInstance, isOwner bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, inst := range instances {
		icon := instanceStatusIcon(inst.Status)
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("%s %s", icon, inst.ContainerName),
				"inst:view:"+inst.ID.String()),
		))
	}
	if isOwner {
		rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "menu:main")))
	} else {
		rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "menu:main")))
	}
	kb.Inline(rows...)
	return kb
}

func InstanceDetail(inst models.BotInstance, isOwner bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row

	if inst.Status == models.StatusRunning {
		rows = append(rows, kb.Row(kb.Data("⏹ توقف", "inst:stop:"+inst.ID.String())))
	} else {
		rows = append(rows, kb.Row(kb.Data("▶️ شروع", "inst:start:"+inst.ID.String())))
	}
	if isOwner {
		rows = append(rows, kb.Row(kb.Data("🗑 حذف", "inst:delete_confirm:"+inst.ID.String())))
	}
	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "menu:all_instances")))
	kb.Inline(rows...)
	return kb
}

// ── User List ─────────────────────────────────────────────

func UserList(users []models.User) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, u := range users {
		icon := "👤"
		if u.IsBlocked {
			icon = "🚫"
		}
		if u.Role == models.RoleOwner {
			icon = "👑"
		} else if u.Role == models.RoleAdmin {
			icon = "🛡"
		}
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("%s %s", icon, u.FirstName),
				"user:view:"+u.ID.String()),
		))
	}
	rows = append(rows, kb.Row(kb.Data("🔙 بازگشت", "menu:main")))
	kb.Inline(rows...)
	return kb
}

func UserDetail(u models.User) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	blockLabel := "🚫 بلاک"
	if u.IsBlocked {
		blockLabel = "✅ آنبلاک"
	}
	kb.Inline(
		kb.Row(
			kb.Data(blockLabel, "user:toggle_block:"+u.ID.String()),
			kb.Data("🎭 نقش", "user:role_menu:"+u.ID.String()),
		),
		kb.Row(kb.Data("🔙 بازگشت", "menu:users")),
	)
	return kb
}

func UserRoleMenu(userID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("👤 User", "user:setrole:"+userID+":user"),
			kb.Data("🛡 Admin", "user:setrole:"+userID+":admin"),
		),
		kb.Row(kb.Data("🔙 بازگشت", "user:view:"+userID)),
	)
	return kb
}

// ── Wizard: ساخت ربات از InviteLink ──────────────────────

// WizardStep1 اولین قدم wizard — تأیید نوع ربات.
func WizardConfirm(botType models.BotType, token string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(
			fmt.Sprintf("✅ بله، %s بسازید", botTypeLabel(botType)),
			"wizard:confirm:"+token,
		)),
		kb.Row(kb.Data("❌ انصراف", "wizard:cancel")),
	)
	return kb
}

// WizardServerSelect انتخاب سرور در wizard.
func WizardServerSelect(servers []models.Server, token string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, s := range servers {
		if !s.IsOnline {
			continue
		}
		rows = append(rows, kb.Row(
			kb.Data("🟢 "+s.Name, "wizard:server:"+token+":"+s.ID.String()),
		))
	}
	if len(rows) == 0 {
		rows = append(rows, kb.Row(kb.Data("❌ سروری آنلاین نیست", "wizard:cancel")))
	}
	rows = append(rows, kb.Row(kb.Data("❌ انصراف", "wizard:cancel")))
	kb.Inline(rows...)
	return kb
}

// ── helpers ───────────────────────────────────────────────

func botTypeIcon(t models.BotType) string {
	switch t {
	case models.BotTypeUploader:
		return "📤"
	case models.BotTypeVPN:
		return "🔒"
	case models.BotTypeArchive:
		return "📂"
	case models.BotTypeMember:
		return "👥"
	default:
		return "🤖"
	}
}

func botTypeLabel(t models.BotType) string {
	switch t {
	case models.BotTypeUploader:
		return "Uploader Bot"
	case models.BotTypeVPN:
		return "VPN Bot"
	case models.BotTypeArchive:
		return "Archive Bot"
	case models.BotTypeMember:
		return "Member Bot"
	default:
		return string(t)
	}
}

func instanceStatusIcon(s models.InstanceStatus) string {
	switch s {
	case models.StatusRunning:
		return "🟢"
	case models.StatusStopped:
		return "🔴"
	case models.StatusPending:
		return "🟡"
	case models.StatusError:
		return "⚠️"
	default:
		return "⚪️"
	}
}

// BotTypeIcon و BotTypeLabel برای استفاده در wizard export شدند.
func BotTypeIcon(t models.BotType) string { return botTypeIcon(t) }
func BotTypeLabel(t models.BotType) string { return botTypeLabel(t) }
