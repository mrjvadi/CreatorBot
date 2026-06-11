package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminBotsList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	instances, _ := h.store.ListAllInstances(ctx)

	if len(instances) == 0 {
		return c.Send(h.t(ctx, uid, i18n.KeyBotsEmpty), tele.ModeHTML, h.kbAdmin(ctx, uid))
	}

	running, stopped, pending, errored := 0, 0, 0, 0
	for _, inst := range instances {
		switch inst.Status {
		case models.StatusRunning: running++
		case models.StatusStopped: stopped++
		case models.StatusPending: pending++
		case models.StatusError:   errored++
		}
	}

	lines := []string{
		h.t(ctx, uid, i18n.KeyBotsTitle, len(instances)),
		"",
		h.t(ctx, uid, i18n.KeyAdminBotSummary, running, stopped, pending, errored),
		"",
	}
	for _, inst := range instances {
		lines = append(lines, fmtInstance(inst, true))
	}

	return c.Send(joinLines(lines), tele.ModeHTML, h.kbBack(ctx, uid))
}

func (h *Handler) adminBotStop(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, _ := h.store.FindInstance(ctx, instID)
	if inst == nil {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyBotNotFound))
	}
	h.docker.Stop(ctx, inst.ServerID.String(), inst.ContainerID)
	h.store.UpdateInstanceStatus(ctx, instID, models.StatusStopped)
	return c.Send(h.t(ctx, uid, i18n.KeyBotStopped, inst.ContainerName),
		tele.ModeHTML, h.kbAdmin(ctx, uid))
}

func (h *Handler) adminBotStart(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, _ := h.store.FindInstance(ctx, instID)
	if inst == nil {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyBotNotFound))
	}
	h.docker.Start(ctx, inst.ServerID.String(), inst.ContainerID)
	h.store.UpdateInstanceStatus(ctx, instID, models.StatusPending)
	return c.Send(h.t(ctx, uid, i18n.KeyBotStarted, inst.ContainerName),
		tele.ModeHTML, h.kbAdmin(ctx, uid))
}

func (h *Handler) adminBotDelete(ctx context.Context, c tele.Context, instID string) error {
	uid := c.Sender().ID
	inst, _ := h.store.FindInstance(ctx, instID)
	if inst == nil {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyBotNotFound))
	}
	name := inst.ContainerName
	h.docker.Remove(ctx, inst.ServerID.String(), inst.ContainerID)
	h.store.DeleteInstance(ctx, instID)
	return c.Send(h.t(ctx, uid, i18n.KeyBotDeleted, strings.TrimSpace(name)),
		tele.ModeHTML, h.kbAdmin(ctx, uid))
}
