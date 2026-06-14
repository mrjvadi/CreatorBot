package tgbot

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Handler) adminTemplatesList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	templates, _ := h.store.ListTemplates(ctx)

	lines := []string{h.t(ctx, uid, i18n.KeyTemplatesTitle), ""}
	if len(templates) == 0 {
		lines = append(lines, h.t(ctx, uid, i18n.KeyTemplatesEmpty), "")
	} else {
		for _, t := range templates {
			lines = append(lines, fmtTemplate(t))
		}
		lines = append(lines, "")
	}
	lines = append(lines, h.t(ctx, uid, i18n.KeyTemplateAskType))

	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("➕ تمپلیت جدید", "create_tmpl")))
	return c.Send(joinLines(lines), tele.ModeHTML, kb)
}

func (h *Handler) adminTemplateAdd(ctx context.Context, c tele.Context, botType, image, tag, name string) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	// بررسی اینکه آیا کاربر «رایگان» زده
	isFree := strings.Contains(strings.ToLower(name), "free") ||
		strings.Contains(name, "رایگان") ||
		image == "free" || tag == "free"

	// اگه image یا tag مقدار «free» داشت → تمپلیت رایگان بدون image واقعی
	if isFree {
		return h.adminAddFreeTemplate(ctx, c, botType, name)
	}

	t := &models.BotTemplate{
		Type:      botType,
		Name:      strings.TrimSpace(name),
		ImageName: strings.TrimSpace(image),
		ImageTag:  strings.TrimSpace(tag),
		IsActive:  true,
		IsFree:    false,
	}
	if err := h.store.CreateTemplate(ctx, t); err != nil {
		h.log.Error("adminTemplateAdd", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyTemplateAddError))
	}

	return c.Send(
		h.t(ctx, uid, i18n.KeyTemplateAdded,
			t.Name, h.botTypeLabel(ctx, uid, models.BotType(t.Type)),
			t.ImageName, t.ImageTag, t.ID),
		tele.ModeHTML, h.kbAdmin(ctx, uid),
	)
}

// adminAddFreeTemplate تمپلیت رایگان می‌سازد.
// این تمپلیت برای پلن رایگان استفاده می‌شود — image واقعی ندارد.
func (h *Handler) adminAddFreeTemplate(ctx context.Context, c tele.Context, botType, name string) error {
	uid := c.Sender().ID

	// بررسی تکراری نبودن
	existing, _ := h.store.FindTemplateByType(ctx, botType)
	if existing != nil && existing.IsFree {
		return c.Send(
			h.t(ctx, uid, i18n.KeyTmplFreeExists, botType, existing.ID),
			tele.ModeHTML, h.kbAdmin(ctx, uid),
		)
	}

	tmplName := name
	if tmplName == "" || tmplName == "رایگان" || tmplName == "free" {
		tmplName = botType + "-free"
	}

	t := &models.BotTemplate{
		Type:      botType,
		Name:      tmplName,
		ImageName: "creatorbot/" + botType + "-bot",
		ImageTag:  "latest",
		IsActive:  true,
		IsFree:    true,
	}
	if err := h.store.CreateTemplate(ctx, t); err != nil {
		h.log.Error("adminAddFreeTemplate", h.F("err", err))
		return h.sendMain(c, h.t(ctx, uid, i18n.KeyTemplateAddError))
	}

	return c.Send(
		h.t(ctx, uid, i18n.KeyTmplFreeAdded,
			t.Name, botType, t.ImageName, t.ImageTag, t.ID),
		tele.ModeHTML, h.kbAdmin(ctx, uid),
	)
}
