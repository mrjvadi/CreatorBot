// Package wizard منطق ساخت ربات از طریق InviteLink را پیاده‌سازی می‌کند.
package wizard

import (
	"context"
	"fmt"

	sharedocker "github.com/mrjvadi/creatorbot/shared-core/docker"
	"github.com/mrjvadi/creatorbot/shared-core/models"
	"github.com/mrjvadi/creatorbot/shared-core/protocol"
	"github.com/mrjvadi/creatorbot/shared-core/store"
	"github.com/mrjvadi/creatorbot/shared/pkg/auth"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	tele "gopkg.in/telebot.v4"
)

// Wizard منطق چند مرحله‌ای ساخت ربات از InviteLink.
type Wizard struct {
	store      *store.Store
	docker     *sharedocker.Manager
	log        ports.Logger
	encryptKey string
}

func New(st *store.Store, docker *sharedocker.Manager, log ports.Logger, encryptKey string) *Wizard {
	return &Wizard{store: st, docker: docker, log: log, encryptKey: encryptKey}
}

// StartFromLink وقتی کاربر /start <token> می‌زند صدا می‌شود.
func (w *Wizard) StartFromLink(ctx context.Context, c tele.Context, token string) error {
	link, err := w.store.FindInviteLinkByToken(ctx, token)
	if err != nil {
		w.log.Error("wizard: find link", ports.F("err", err))
		return c.Send("خطا در بررسی لینک.")
	}
	if link == nil {
		return c.Send("این لینک معتبر نیست.")
	}
	if link.IsExpired() {
		return c.Send("این لینک منقضی شده است.")
	}
	if link.IsExhausted() {
		return c.Send("این لینک قبلاً استفاده شده است.")
	}

	msg := fmt.Sprintf(
		"🔗 <b>لینک دعوت معتبر</b>\n\nنوع: %s %s\n\n%s",
		botTypeIcon(link.BotType),
		botTypeLabel(link.BotType),
		botTypeDescription(link.BotType),
	)
	return c.Send(msg, tele.ModeHTML)
}

// ConfirmAndAskToken پس از تأیید، توکن ربات را درخواست می‌کند.
func (w *Wizard) ConfirmAndAskToken(c tele.Context, _ string) error {
	return c.Edit(
		"✅ عالی!\n\nاز <b>@BotFather</b> یک ربات جدید بسازید و توکن را اینجا ارسال کنید.\n\n⚠️ توکن را با کسی به اشتراک نگذارید.",
		tele.ModeHTML,
	)
}

// FinishWithToken توکن را دریافت، instance می‌سازد و deploy می‌کند.
func (w *Wizard) FinishWithToken(ctx context.Context, c tele.Context, inviteToken, botToken string) error {
	// ── ۱. بررسی مجدد لینک ───────────────────────────────────
	link, err := w.store.FindInviteLinkByToken(ctx, inviteToken)
	if err != nil || link == nil || !link.IsValid() {
		return c.Send("لینک نامعتبر یا منقضی شده.")
	}

	// ── ۲. استخراج Bot ID از توکن ────────────────────────────
	botID, err := models.BotIDFromToken(botToken)
	if err != nil {
		w.log.Error("wizard: invalid token", ports.F("err", err))
		return c.Send("فرمت توکن نامعتبر است. توکن باید از @BotFather باشد.")
	}

	// ── ۳. بررسی تکراری نبودن ─────────────────────────────────
	if existing, _ := w.store.FindInstanceByBotID(ctx, botID); existing != nil {
		return c.Send(
			fmt.Sprintf("این ربات قبلاً ثبت شده است.\nID: <code>%s</code>", existing.ID),
			tele.ModeHTML,
		)
	}

	// ── ۴. رمزنگاری توکن ─────────────────────────────────────
	encrypted, err := auth.Encrypt(botToken, w.encryptKey)
	if err != nil {
		w.log.Error("wizard: encrypt", ports.F("err", err))
		return c.Send("خطای داخلی.")
	}

	// ── ۵. User را پیدا یا بساز ──────────────────────────────
	user, _ := w.store.FindUserByTelegramID(ctx, c.Sender().ID)
	if user == nil {
		user = &models.User{
			TelegramID: c.Sender().ID,
			Username:   c.Sender().Username,
			FirstName:  c.Sender().FirstName,
		}
		w.store.UpsertUser(ctx, user)
	}

	// ── ۶. بهترین سرور آنلاین ────────────────────────────────
	server, err := w.store.FindBestOnlineServer(ctx)
	if err != nil || server == nil {
		return c.Send("هیچ سروری آنلاین نیست. با ادمین تماس بگیرید.")
	}

	// ── ۷. تمپلیت مناسب ──────────────────────────────────────
	tmpl, err := w.store.FindTemplateByType(ctx, string(link.BotType))
	if err != nil || tmpl == nil {
		return c.Send("تمپلیت یافت نشد. با ادمین تماس بگیرید.")
	}

	// ── ۸. ثبت instance در DB ─────────────────────────────────
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
	if err := w.store.CreateInstance(ctx, inst); err != nil {
		w.log.Error("wizard: create instance", ports.F("err", err))
		return c.Send("خطا در ثبت ربات. دوباره تلاش کنید.")
	}

	// ── ۹. ثبت استفاده از لینک ───────────────────────────────
	w.store.IncrementInviteUse(ctx, inviteToken, nil)

	// ── ۱۰. ارسال دستور Deploy از طریق NATS ──────────────────
	deployCmd := protocol.DeployCommand{
		Type:          protocol.MsgDeploy,
		ServerID:      server.ID.String(),
		ContainerName: containerName,
		ImageName:     tmpl.ImageName,
		ImageTag:      tmpl.ImageTag,
		EnvVars: map[string]string{
			"BOT_TOKEN": botToken,
		},
	}
	if err := w.docker.Deploy(ctx, server.ID.String(), deployCmd); err != nil {
		w.store.UpdateInstanceStatus(ctx, inst.ID, models.StatusError)
		w.log.Error("wizard: deploy failed",
			ports.F("instance", inst.ID), ports.F("err", err))
		return c.Send(
			fmt.Sprintf(
				"ربات ثبت شد ولی deploy با خطا مواجه شد.\nادمین بررسی خواهد کرد.\n\nID: <code>%s</code>",
				inst.ID,
			),
			tele.ModeHTML,
		)
	}

	w.log.Info("wizard: deployed",
		ports.F("instance", inst.ID),
		ports.F("server", server.Name),
		ports.F("container", containerName))

	return c.Send(fmt.Sprintf(
		"🎉 <b>ربات شما ساخته شد!</b>\n\nنوع: %s %s\nسرور: %s\nوضعیت: در حال راه‌اندازی\nID: <code>%s</code>\n\nظرف ۱-۲ دقیقه فعال می‌شود.",
		botTypeIcon(link.BotType), botTypeLabel(link.BotType),
		server.Name,
		inst.ID,
	), tele.ModeHTML)
}


func botTypeIcon(t models.BotType) string {
	switch t {
	case models.BotTypeUploader: return "📤"
	case models.BotTypeVPN:      return "🔒"
	case models.BotTypeArchive:  return "📂"
	case models.BotTypeMember:   return "👥"
	default:                      return "🤖"
	}
}

func botTypeLabel(t models.BotType) string {
	switch t {
	case models.BotTypeUploader: return "Uploader Bot"
	case models.BotTypeVPN:      return "VPN Bot"
	case models.BotTypeArchive:  return "Archive Bot"
	case models.BotTypeMember:   return "Member Bot"
	default:                      return string(t)
	}
}
func botTypeDescription(t models.BotType) string {
	switch t {
	case models.BotTypeUploader:
		return "یک ربات آپلودر فایل برای شما ساخته خواهد شد."
	case models.BotTypeVPN:
		return "یک ربات فروش VPN برای شما ساخته خواهد شد."
	case models.BotTypeArchive:
		return "یک ربات آرشیو فایل برای شما ساخته خواهد شد."
	case models.BotTypeMember:
		return "یک ربات قفل ممبر برای شما ساخته خواهد شد."
	default:
		return ""
	}
}
