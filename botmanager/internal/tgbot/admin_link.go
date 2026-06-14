package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminLinksList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	links, _ := h.store.ListInviteLinks(ctx, uid)

	lines := []string{h.t(ctx, uid, i18n.KeyLinksTitle), ""}

	if len(links) == 0 {
		lines = append(lines, h.t(ctx, uid, i18n.KeyLinksEmpty))
	} else {
		active, expired := 0, 0
		for _, l := range links {
			if l.IsValid() { active++ } else { expired++ }
		}
		lines = append(lines, h.t(ctx, uid, i18n.KeyAdminLinkStats, active, expired), "")
		for _, l := range links {
			lines = append(lines, fmtLink(l, h.botUsername))
		}
	}

	lines = append(lines, "", h.t(ctx, uid, i18n.KeyLinkAskType))
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("➕ لینک جدید", "create_link")))
	return c.Send(joinLines(lines), tele.ModeHTML, kb)
}

func (h *Handler) adminLinkCreate(ctx context.Context, c tele.Context, botType string, maxUse int, label string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	token := genToken()
	link := &models.InviteLink{
		Token:     token,
		BotType:   models.BotType(botType),
		Label:     label,
		MaxUse:    maxUse,
		CreatedBy: uid,
	}
	if err := h.store.CreateInviteLink(ctx, link); err != nil {
		h.log.Error("adminLinkCreate", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyLinkCreateError))
	}

	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", h.botUsername, token)
	limitText := h.t(ctx, uid, i18n.KeyBtnLimitNo)
	if maxUse > 0 {
		limitText = h.t(ctx, uid, i18n.KeyAdminLinkLimitX, maxUse)
	}
	labelLine := ""
	if label != "" {
		labelLine = "\n" + label
	}

	return c.Send(
		h.t(ctx, uid, i18n.KeyLinkCreated,
			botTypeEmoji(models.BotType(botType)), botType, labelLine,
			limitText, deepLink),
		tele.ModeHTML, h.kbAdmin(ctx, uid),
	)
}
