// Package tgbot - admin_test.go
// دپلوی تستیِ سرویس‌ها توسط ادمین (بدون پلن و پرداخت).
package admin

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/core"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// adminTestStart شروعِ دپلوی تستی یک تمپلیت (سرویس+تگ) — درخواست توکن.
func (h *Admin) AdminTestStart(ctx context.Context, c tele.Context, tmplID string) error {
	uid := c.Sender().ID
	_ = c.Respond()
	if !h.IsAdmin(c) {
		return c.Edit(h.T(ctx, uid, i18n.KeyNoAccess))
	}

	tmpl, err := h.Store.FindTemplate(ctx, tmplID)
	if err != nil || tmpl == nil {
		return c.Edit(h.T(ctx, uid, i18n.KeyNotFound))
	}

	h.SetStep(ctx, uid, state.StepAdminTestToken, "tmpl_id", tmplID)
	return c.Edit(
		h.T(ctx, uid, i18n.KeyAdminTestAskToken, tmpl.Type, tmpl.ImageTag),
		tele.ModeHTML, h.KbCancel(ctx, uid),
	)
}

// adminTestDeploy توکن را می‌گیرد و instance تستی را بدون پلن/پرداخت دپلوی می‌کند.
func (h *Admin) AdminTestDeploy(ctx context.Context, c tele.Context, tmplID, token string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)
	if !h.IsAdmin(c) {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyNoAccess))
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tmpl, err := h.Store.FindTemplate(ctx, tmplID)
	if err != nil || tmpl == nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyNotFound))
	}

	botID, err := core.ExtractBotID(token)
	if err != nil {
		return c.Send(h.T(ctx, uid, i18n.KeyServiceInvalidToken), tele.ModeHTML)
	}
	if existing, _ := h.Store.FindInstanceByBotID(ctx, botID); existing != nil {
		return c.Send(h.T(ctx, uid, i18n.KeyServiceDuplicate), tele.ModeHTML)
	}

	u, _ := h.GetOrCreateUser(ctx, c)
	if u == nil {
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyError))
	}

	// تست دیپلوی ادمین به هیچ تگ خاصی محدود نیست (برخلاف Provision در wizard.go
	// که پلن رایگان را به سرورهای تگ "free" محدود می‌کند) — ادمین باید بتواند
	// روی هر سرور آنلاینی تست بزند.
	server, err := h.Store.SelectLeastLoadedServer(ctx, "")
	if err != nil || server == nil {
		return c.Send(h.T(ctx, uid, i18n.KeyWizardNoServer), tele.ModeHTML)
	}

	containerName := fmt.Sprintf("%s_%d", tmpl.Type, botID)
	instance := &models.BotInstance{
		OwnerID:       u.ID,
		TemplateID:    tmpl.ID,
		ServerID:      server.ID,
		BotToken:      token,
		ContainerName: containerName,
		BotID:         botID,
		DBSchema:      fmt.Sprintf("inst_%d", botID),
		Status:        "pending",
		PlanID:        nil, // تستی — بدون پلن
		LockMode:      models.LockModeNone,
	}
	if err := h.Store.CreateInstance(ctx, instance); err != nil {
		return c.Send(h.T(ctx, uid, i18n.KeyWizardCreateError), tele.ModeHTML)
	}

	jwtToken, _ := auth.GenerateAccessToken(
		u.ID.String(), "user",
		auth.JWTConfig{AccessSecret: h.EncryptKey},
	)

	licenseToken := ""
	if h.License != nil {
		lt, lerr := h.License.Issue(ctx, botID, "bot_"+fmt.Sprint(botID), u.ID.String(), server.ID.String(), "")
		if lerr != nil {
			h.Log.Error("license issue failed — deploying without LICENSE_TOKEN", ports.F("err", lerr), ports.F("bot_id", botID))
		} else {
			licenseToken = lt
		}
	}

	cmd := protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ContainerName: containerName,
		ImageName:     tmpl.ImageName,
		ImageTag:      tmpl.ImageTag,
		EnvVars: map[string]string{
			"BOT_TOKEN":      token,
			"INSTANCE_ID":    "bot_" + fmt.Sprint(botID),
			"OWNER_TELEGRAM": fmt.Sprint(u.TelegramID),
			// مثل wizard.go: ربات‌های محصول مالک را از OWNER_ID می‌خوانند.
			"OWNER_ID":      fmt.Sprint(u.TelegramID),
			"PLAN_ID":       "",
			"JWT_TOKEN":     jwtToken,
			"LICENSE_TOKEN": licenseToken,
			"SERVER_ID":     server.ID.String(),
		},
	}

	if h.NC == nil {
		_ = h.Store.UpdateInstanceStatus(ctx, instance.ID.String(), "failed")
		return c.Send(h.T(ctx, uid, i18n.KeyWizardDeployError), tele.ModeHTML)
	}
	if err := h.Docker.Send(ctx, server.ID.String(), cmd); err != nil {
		h.Log.Error("admin test deploy failed", ports.F("err", err))
		_ = h.Store.UpdateInstanceStatus(ctx, instance.ID.String(), "failed")
		return c.Send(h.T(ctx, uid, i18n.KeyWizardDeployError), tele.ModeHTML)
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyAdminTestDeployed, containerName),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}
