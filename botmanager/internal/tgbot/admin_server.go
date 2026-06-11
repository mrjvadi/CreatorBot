package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminServersList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	servers, _ := h.store.ListServers(ctx)

	lines := []string{h.t(ctx, uid, i18n.KeyServersTitle), ""}
	if len(servers) == 0 {
		lines = append(lines, h.t(ctx, uid, i18n.KeyServersEmpty), "")
	} else {
		for _, s := range servers {
			lines = append(lines, fmtServer(s))
		}
		lines = append(lines, "")
	}
	lines = append(lines, h.t(ctx, uid, i18n.KeyServerAskName))

	h.setStep(ctx, uid, stepServerName)
	return c.Send(joinLines(lines), tele.ModeHTML, h.kbBackCancel(ctx, uid))
}

func (h *Handler) adminServerAdd(ctx context.Context, c tele.Context, name, ip string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	if name == "" || ip == "" {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyError))
	}

	srv := &models.Server{
		Name:    strings.TrimSpace(name),
		IP:      strings.TrimSpace(ip),
		Channel: fmt.Sprintf("server_%s", strings.ReplaceAll(name, " ", "_")),
	}
	if err := h.store.CreateServer(ctx, srv); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return h.sendMain(c, h.t(ctx, uid, i18n.KeyServerDuplicate))
		}
		h.log.Error("adminServerAdd", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyServerAddError))
	}

	return c.Send(
		h.t(ctx, uid, i18n.KeyServerAdded, srv.Name, srv.IP, srv.ID),
		tele.ModeHTML, h.kbAdmin(ctx, uid),
	)
}
