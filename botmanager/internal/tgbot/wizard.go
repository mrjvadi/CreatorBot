package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
)

func (h *Handler) wizardStart(ctx context.Context, c tele.Context, token string) error {
	uid := c.Sender().ID

	link, err := h.store.FindInviteLinkByToken(ctx, token)
	if err != nil {
		return c.Send(h.t(ctx, uid, i18n.KeyError))
	}
	if link == nil {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardInvalidLink))
	}
	if link.IsExpired() {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardExpiredLink))
	}
	if link.IsExhausted() {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardUsedLink))
	}

	h.setWizardPending(ctx, uid, token)

	desc := h.t(ctx, uid, botTypeDescKey(link.BotType))
	text := h.t(ctx, uid, i18n.KeyWizardConfirm,
		botTypeEmoji(link.BotType),
		strings.Title(string(link.BotType)),
		desc,
	)

	return c.Send(text, tele.ModeHTML, h.kbWizardConfirm(ctx, uid))
}

func (h *Handler) wizardFinish(ctx context.Context, c tele.Context, inviteToken, botToken string) error {
	uid := c.Sender().ID

	link, err := h.store.FindInviteLinkByToken(ctx, inviteToken)
	if err != nil || link == nil || !link.IsValid() {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyWizardInvalidLink))
	}

	// ── بررسی ظرفیت per-type (Capacity Engine) ──────────────
	hasCapacity, err := h.checkBuildCapacityForType(ctx, c, string(link.BotType))
	if err != nil {
		return err
	}
	if !hasCapacity {
		return nil // پیام فرستاده شده
	}

	botID, err := models.BotIDFromToken(botToken)
	if err != nil {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardInvalidToken), tele.ModeHTML, h.kbBack(ctx, uid))
	}

	if existing, _ := h.store.FindInstanceByBotID(ctx, botID); existing != nil {
		return c.Send(
			h.t(ctx, uid, i18n.KeyWizardAlreadyExists, existing.ID, existing.Status),
			tele.ModeHTML, h.kbUser(ctx, uid),
		)
	}

	encrypted, err := auth.Encrypt(botToken, h.encryptKey)
	if err != nil {
		h.log.Error("wizardFinish: encrypt", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyError))
	}

	user, _ := h.getOrCreateUser(ctx, c)
	if user == nil {
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyError))
	}

	server, err := h.store.FindBestOnlineServer(ctx)
	if err != nil || server == nil {
		return c.Send(h.t(ctx, uid, i18n.KeyWizardNoServer), h.kbUser(ctx, uid))
	}

	tmpl, err := h.store.FindTemplateByType(ctx, string(link.BotType))
	if err != nil || tmpl == nil {
		h.log.Error("wizardFinish: no template", h.F("type", link.BotType))
		return c.Send(h.t(ctx, uid, i18n.KeyWizardNoTemplate), h.kbUser(ctx, uid))
	}

	containerName := fmt.Sprintf("%s-%d", link.BotType, botID)
	inst := &models.BotInstance{
		OwnerID:       user.ID,
		TemplateID:    tmpl.ID,
		ServerID:      server.ID,
		BotToken:      encrypted,
		BotID:         botID,
		ContainerName: containerName,
		DBSchema:      fmt.Sprintf("inst_%d", botID),
		Status:        models.StatusPending,
	}
	if err := h.store.CreateInstance(ctx, inst); err != nil {
		h.log.Error("wizardFinish: create", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyError))
	}

	h.store.IncrementInviteUse(ctx, inviteToken, nil)

	deployErr := h.docker.Deploy(ctx, server.ID.String(), protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ServerID:      server.ID.String(),
		ContainerName: containerName,
		ImageName:     tmpl.ImageName,
		ImageTag:      tmpl.ImageTag,
		EnvVars:       map[string]string{"BOT_TOKEN": botToken},
	})

	if deployErr != nil {
		h.log.Error("wizardFinish: deploy", h.F("err", deployErr))
		h.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusError)
		return c.Send(
			h.t(ctx, uid, i18n.KeyWizardDeployError, inst.ID),
			tele.ModeHTML, h.kbUser(ctx, uid),
		)
	}

	h.log.Info("wizard: deployed",
		h.F("instance", inst.ID),
		h.F("server", server.Name))

	return c.Send(
		h.t(ctx, uid, i18n.KeyWizardSuccess,
			botTypeEmoji(link.BotType),
			strings.Title(string(link.BotType)),
			server.Name,
		),
		tele.ModeHTML, h.kbUser(ctx, uid),
	)
}

func botTypeDescKey(t models.BotType) i18n.Key {
	m := map[models.BotType]i18n.Key{
		models.BotTypeUploader: i18n.KeyBotDescUploader,
		models.BotTypeVPN:      i18n.KeyBotDescVPN,
		models.BotTypeArchive:  i18n.KeyBotDescArchive,
		models.BotTypeMember:   i18n.KeyBotDescMember,
	}
	if k, ok := m[t]; ok {
		return k
	}
	return i18n.KeyBotDescUploader
}
