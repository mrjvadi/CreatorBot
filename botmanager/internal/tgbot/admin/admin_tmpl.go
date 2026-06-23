package admin

import (
	"context"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/shared-core/models"
)

func (h *Admin) AdminTemplatesList(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	templates, _ := h.Store.ListTemplates(ctx)

	lines := []string{h.T(ctx, uid, i18n.KeyTemplatesTitle), ""}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	if len(templates) == 0 {
		lines = append(lines, h.T(ctx, uid, i18n.KeyTemplatesEmpty))
	} else {
		for _, t := range templates {
			lines = append(lines, format.FmtTemplate(t))
			// دکمه‌ی دپلوی تستی برای هر تمپلیت (سرویس+تگ)
			rows = append(rows, kb.Row(kb.Data(
				h.Btn(ctx, uid, i18n.KeyBtnTest)+" "+t.Type+":"+t.ImageTag,
				"tmpl_test:"+t.ID.String())))
		}
	}
	lines = append(lines, "")

	rows = append(rows, kb.Row(kb.Data(h.T(ctx, uid, i18n.KeyBtnNewTemplate), "create_tmpl")))
	kb.Inline(rows...)
	return c.Send(format.JoinLines(lines), tele.ModeHTML, kb)
}

func (h *Admin) AdminTemplateAdd(ctx context.Context, c tele.Context, botType, image, tag, name string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	// بررسی اینکه آیا کاربر «رایگان» زده
	isFree := strings.Contains(strings.ToLower(name), "free") ||
		strings.Contains(name, "رایگان") ||
		image == "free" || tag == "free"

	// اگه image یا tag مقدار «free» داشت → تمپلیت رایگان بدون image واقعی
	if isFree {
		return h.AdminAddFreeTemplate(ctx, c, botType, name)
	}

	t := &models.BotTemplate{
		Type:      botType,
		Name:      strings.TrimSpace(name),
		ImageName: strings.TrimSpace(image),
		ImageTag:  strings.TrimSpace(tag),
		IsActive:  true,
		IsFree:    false,
	}
	if err := h.Store.CreateTemplate(ctx, t); err != nil {
		h.Log.Error("adminTemplateAdd", h.F("err", err))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyTemplateAddError))
	}
	// config.updated publish
	if h.NC != nil {
		_ = h.NC.PublishCore("config.updated", map[string]any{"type": "template"})
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyTemplateAdded,
			t.Name, h.BotTypeLabel(ctx, uid, models.BotType(t.Type)),
			t.ImageName, t.ImageTag, t.ID),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}

// adminAddFreeTemplate تمپلیت رایگان می‌سازد.
// این تمپلیت برای پلن رایگان استفاده می‌شود — image واقعی ندارد.
func (h *Admin) AdminAddFreeTemplate(ctx context.Context, c tele.Context, botType, name string) error {
	uid := c.Sender().ID

	// بررسی تکراری نبودن
	existing, _ := h.Store.FindTemplateByType(ctx, botType)
	if existing != nil && existing.IsFree {
		return c.Send(
			h.T(ctx, uid, i18n.KeyTmplFreeExists, botType, existing.ID),
			tele.ModeHTML, h.KbAdmin(ctx, uid),
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
	if err := h.Store.CreateTemplate(ctx, t); err != nil {
		h.Log.Error("adminAddFreeTemplate", h.F("err", err))
		return h.SendMain(c, h.T(ctx, uid, i18n.KeyTemplateAddError))
	}

	return c.Send(
		h.T(ctx, uid, i18n.KeyTmplFreeAdded,
			t.Name, botType, t.ImageName, t.ImageTag, t.ID),
		tele.ModeHTML, h.KbAdmin(ctx, uid),
	)
}
