package admin

import (
	"context"
	"encoding/json"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/format"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/i18n"
	"github.com/mrjvadi/creatorbot/botmanager/internal/tgbot/state"
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
			rows = append(rows, kb.Row(
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnTest)+" "+t.Type+":"+t.ImageTag, "tmpl_test:"+t.ID.String()),
				kb.Data(h.Btn(ctx, uid, i18n.KeyBtnEditSchema), "tmpl_schema:"+t.ID.String()),
			))
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

// AdminTemplateSchemaEdit شروع ویرایش ConfigSchema یک قالب.
func (h *Admin) AdminTemplateSchemaEdit(ctx context.Context, c tele.Context, tmplID string) error {
	uid := c.Sender().ID
	defer func() { _ = c.Respond() }()
	st := h.GetState(ctx, uid)
	if st.Data == nil {
		st.Data = map[string]string{}
	}
	st.Step = state.StepTmplSchemaJSON
	st.Data["tmpl_id"] = tmplID
	h.SetState(ctx, uid, st)
	return c.Send(h.T(ctx, uid, i18n.KeyTmplAskSchema), tele.ModeHTML)
}

// AdminTemplateSchemaSet JSON ارسال‌شده را تأیید و ذخیره می‌کند.
func (h *Admin) AdminTemplateSchemaSet(ctx context.Context, c tele.Context, tmplID, jsonText string) error {
	uid := c.Sender().ID
	h.ClearState(ctx, uid)

	var fields []models.ConfigField
	if err := json.Unmarshal([]byte(jsonText), &fields); err != nil {
		return c.Send(h.T(ctx, uid, i18n.KeyTmplSchemaInvalid), tele.ModeHTML)
	}

	normalized, _ := json.Marshal(fields)
	if err := h.Store.UpdateTemplateSchema(ctx, tmplID, string(normalized)); err != nil {
		h.Log.Error("updateTemplateSchema", h.F("err", err))
		return c.Send(h.T(ctx, uid, i18n.KeyError))
	}
	return c.Send(h.T(ctx, uid, i18n.KeyTmplSchemaSet), tele.ModeHTML, h.KbAdmin(ctx, uid))
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
