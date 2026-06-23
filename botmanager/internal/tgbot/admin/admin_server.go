package admin

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Admin) AdminServersList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	servers, _ := h.Store.ListServers(ctx)

	lines := []string{h.T(ctx, uid, i18n.KeyServersTitle), ""}
	if len(servers) == 0 {
		lines = append(lines, h.T(ctx, uid, i18n.KeyServersEmpty), "")
	} else {
		for _, s := range servers {
			lines = append(lines, format.FmtServer(s))
		}
		lines = append(lines, "")
	}
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data(h.Btn(ctx, uid, i18n.KeyBtnAddServer), "add_server")),
	)
	return c.Send(format.JoinLines(lines), tele.ModeHTML, kb)
}

func (h *Admin) AdminServerStart(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	h.SetStep(ctx, uid, state.StepServerName)
	return c.Edit(h.T(ctx, uid, i18n.KeyServerAskName), tele.ModeHTML, h.KbBackCancel(ctx, uid))
}

func (h *Admin) AdminServerAdd(ctx context.Context, c tele.Context, name, ip string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	if name == "" || ip == "" {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyError))
	}

	srv := &models.Server{
		Name:    strings.TrimSpace(name),
		IP:      strings.TrimSpace(ip),
		Channel: fmt.Sprintf("server_%s", strings.ReplaceAll(name, " ", "_")),
	}
	if err := h.Store.CreateServer(ctx, srv); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return h.SendMain(c, h.T(ctx, uid, i18n.KeyServerDuplicate))
		}
		h.Log.Error("adminServerAdd", h.F("err", err))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyServerAddError))
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyServerAdded, srv.Name),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}
