package admin

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Admin) AdminStats(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	instances, _ := h.Store.ListAllInstances(ctx)
	users, _ := h.Store.ListUsers(ctx)
	servers, _ := h.Store.ListServers(ctx)
	templates, _ := h.Store.ListTemplates(ctx)
	plans, _ := h.Store.ListPlans(ctx)

	running, stopped, pending, errored := 0, 0, 0, 0
	for _, inst := range instances {
		switch inst.Status {
		case models.StatusRunning:
			running++
		case models.StatusStopped:
			stopped++
		case models.StatusPending:
			pending++
		case models.StatusError:
			errored++
		}
	}
	onlineSrv, admins, blocked := 0, 0, 0
	for _, s := range servers {
		if s.IsOnline {
			onlineSrv++
		}
	}
	for _, u := range users {
		if u.Role == models.RoleAdmin || u.Role == models.RoleOwner {
			admins++
		}
		if u.IsBlocked {
			blocked++
		}
	}

	text := fmt.Sprintf(`📈 <b>آمار سیستم</b>
⏰ %s

🤖 <b>ربات‌ها</b> (%d کل)
🟢 فعال: %d  |  🔴 متوقف: %d  |  🟡 در انتظار: %d  |  ⚠️ خطا: %d

🖥 <b>سرورها</b> (%d کل)
🟢 آنلاین: %d  |  🔴 آفلاین: %d

👥 <b>کاربران</b> (%d نفر)
🛡 ادمین: %d  |  🚫 مسدود: %d

📦 تمپلیت‌ها: %d  |  💎 پلن‌ها: %d`,
		time.Now().Format("2006-01-02 | 15:04"),
		len(instances), running, stopped, pending, errored,
		len(servers), onlineSrv, len(servers)-onlineSrv,
		len(users), admins, blocked,
		len(templates), len(plans),
	)

	return c.Send(text, tele.ModeHTML, h.KbAdmin(ctx, uid))
}
