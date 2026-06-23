package admin

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Admin) AdminBotsList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	instances, _ := h.Store.ListAllInstances(ctx)

	if len(instances) == 0 {
		return c.Send(h.T(ctx, uid, i18n.KeyBotsEmpty), tele.ModeHTML, h.KbAdmin(ctx, uid))
	}

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

	lines := []string{
		h.T(ctx, uid, i18n.KeyBotsTitle, len(instances)),
		"",
		h.T(ctx, uid, i18n.KeyAdminBotSummary, running, stopped, pending, errored),
		"",
	}
	for _, inst := range instances {
		lines = append(lines, format.FmtInstance(inst, true))
	}

	return c.Send(format.JoinLines(lines), tele.ModeHTML, h.KbBack(ctx, uid))
}

func (h *Admin) AdminBotStop(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, _ := h.Store.FindInstance(ctx, instID)
	if inst == nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyBotNotFound))
	}
	_ = h.Docker.Stop(ctx, inst.ServerID.String(), inst.ContainerID)
	_ = h.Store.UpdateInstanceStatus(ctx, instID, models.StatusStopped)
	return c.Send(h.T(ctx, uid, i18n.KeyBotStopped, inst.ContainerName),
		tele.ModeHTML, h.KbAdmin(ctx, uid))
}

func (h *Admin) AdminBotStart(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, _ := h.Store.FindInstance(ctx, instID)
	if inst == nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyBotNotFound))
	}
	_ = h.Docker.Start(ctx, inst.ServerID.String(), inst.ContainerID)
	_ = h.Store.UpdateInstanceStatus(ctx, instID, models.StatusPending)
	return c.Send(h.T(ctx, uid, i18n.KeyBotStarted, inst.ContainerName),
		tele.ModeHTML, h.KbAdmin(ctx, uid))
}

func (h *Admin) AdminBotDelete(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, _ := h.Store.FindInstance(ctx, instID)
	if inst == nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyBotNotFound))
	}
	name := inst.ContainerName
	_ = h.Docker.Remove(ctx, inst.ServerID.String(), inst.ContainerID)
	_ = h.Store.DeleteInstance(ctx, instID)
	return c.Send(h.T(ctx, uid, i18n.KeyBotDeleted, strings.TrimSpace(name)),
		tele.ModeHTML, h.KbAdmin(ctx, uid))
}
