package tgbot

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminStats(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	instances, _ := h.store.ListAllInstances(ctx)
	users, _ := h.store.ListUsers(ctx)
	servers, _ := h.store.ListServers(ctx)
	templates, _ := h.store.ListTemplates(ctx)
	plans, _ := h.store.ListPlans(ctx)

	running, stopped, pending, errored := 0, 0, 0, 0
	for _, inst := range instances {
		switch inst.Status {
		case models.StatusRunning: running++
		case models.StatusStopped: stopped++
		case models.StatusPending: pending++
		case models.StatusError:   errored++
		}
	}
	onlineSrv, admins, blocked := 0, 0, 0
	for _, s := range servers { if s.IsOnline { onlineSrv++ } }
	for _, u := range users {
		if u.Role == models.RoleAdmin || u.Role == models.RoleOwner { admins++ }
		if u.IsBlocked { blocked++ }
	}

	text := fmt.Sprintf(
		"%s\n⏰ %s\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"<b>📦</b> %d  |  <b>💰</b> %d",
		h.t(ctx, uid, i18n.KeyStatsTitle),
		time.Now().Format("2006-01-02 | 15:04"),
		h.t(ctx, uid, i18n.KeyStatsBotsLine,
			len(instances), running, stopped, pending, errored),
		h.t(ctx, uid, i18n.KeyStatsServersLine,
			len(servers), onlineSrv, len(servers)-onlineSrv),
		h.t(ctx, uid, i18n.KeyStatsUsersLine,
			len(users), admins, blocked),
		len(templates), len(plans),
	)

	return c.Send(text, tele.ModeHTML, h.kbAdmin(ctx, uid))
}
