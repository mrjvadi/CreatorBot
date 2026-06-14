package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

// startUploadWizard ادمین یک فایل فرستاده — wizard شروع می‌شود.
func (h *Handler) startUploadWizard(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	fi := extractFileInfo(c)
	if fi == nil {
		return nil
	}

	h.setStep(ctx, uid, stepUploadTitle,
		"file_id",   fi.fileID,
		"file_type", fi.fileType,
		"caption",   fi.caption,
	)

	return c.Send(
		"<b>📤 آپلود فایل</b>\n\n"+
			"عنوان فایل را وارد کنید:",
		tele.ModeHTML, kbCancel(),
	)
}

func (h *Handler) handleTitle(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	if len(text) < 2 {
		return c.Send("عنوان خیلی کوتاه است.")
	}
	h.setStep(ctx, uid, stepUploadTags, "title", text,
		"file_id", st.Data["file_id"],
		"file_type", st.Data["file_type"],
		"caption", st.Data["caption"],
	)
	return c.Send(
		"تگ‌ها را وارد کنید (با کاما جدا کنید):\nمثال: <code>golang,backend,tutorial</code>\n\nیا رد کنید:",
		tele.ModeHTML, kbSkipCancel(),
	)
}

func (h *Handler) handleTags(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	tags := ""
	if text != btnSkip {
		tags = cleanTags(text)
	}
	h.setStep(ctx, uid, stepUploadDesc, "tags", tags,
		"title", st.Data["title"],
		"file_id", st.Data["file_id"],
		"file_type", st.Data["file_type"],
		"caption", st.Data["caption"],
	)
	return c.Send("توضیحات (اختیاری):", kbSkipCancel())
}

func (h *Handler) handleDesc(ctx context.Context, c tele.Context, st wizardState, text string) error {
	uid := c.Sender().ID
	desc := ""
	if text != btnSkip {
		desc = text
	}

	cats, _ := h.store.ListCategories(ctx)
	h.setStep(ctx, uid, stepUploadCategory, "desc", desc,
		"title", st.Data["title"],
		"tags", st.Data["tags"],
		"file_id", st.Data["file_id"],
		"file_type", st.Data["file_type"],
		"caption", st.Data["caption"],
	)

	if len(cats) == 0 {
		// بدون دسته مستقیم تأیید
		return h.showUploadPreview(ctx, c, st)
	}
	return c.Send("دسته‌بندی را انتخاب کنید:", kbCategories(cats))
}

func (h *Handler) handleCategory(ctx context.Context, c tele.Context, st wizardState, text string) error {
	// از callback هندل می‌شه — اگه متن رسید یعنی keyboard reply
	if text == btnSkip {
		return h.showUploadPreview(ctx, c, st)
	}
	// پیدا کردن دسته با نام
	cat, err := h.store.FindOrCreateCategory(ctx, text)
	if err == nil && cat != nil {
		h.setStep(ctx, c.Sender().ID, stepUploadConfirm,
			"cat_id", cat.ID.String(),
			"title", st.Data["title"],
			"tags", st.Data["tags"],
			"desc", st.Data["desc"],
			"file_id", st.Data["file_id"],
			"file_type", st.Data["file_type"],
		)
	}
	return h.showUploadPreview(ctx, c, st)
}

func (h *Handler) showUploadPreview(ctx context.Context, c tele.Context, st wizardState) error {
	uid := c.Sender().ID
	h.setStep(ctx, uid, stepUploadConfirm,
		"title", st.Data["title"],
		"tags", st.Data["tags"],
		"desc", st.Data["desc"],
		"file_id", st.Data["file_id"],
		"file_type", st.Data["file_type"],
		"cat_id", st.Data["cat_id"],
	)

	tagsStr := st.Data["tags"]
	if tagsStr == "" {
		tagsStr = "—"
	}
	descStr := st.Data["desc"]
	if descStr == "" {
		descStr = "—"
	}

	return c.Send(
		fmt.Sprintf(
			"<b>پیش‌نمایش</b>\n\n"+
				"📌 عنوان: %s\n"+
				"🏷 تگ‌ها: %s\n"+
				"📝 توضیح: %s\n\n"+
				"تأیید می‌کنید؟",
			st.Data["title"], tagsStr, descStr,
		),
		tele.ModeHTML, kbConfirmUpload(),
	)
}

func (h *Handler) confirmUpload(ctx context.Context, c tele.Context, st wizardState) error {
	uid := c.Sender().ID
	h.clearState(ctx, uid)

	file := &models.File{
		FileID:      st.Data["file_id"],
		FileType:    st.Data["file_type"],
		Title:       st.Data["title"],
		Tags:        st.Data["tags"],
		Description: st.Data["desc"],
		UploaderID:  uid,
	}

	if catIDStr := st.Data["cat_id"]; catIDStr != "" {
		cat, _ := h.store.FindCategoryByID(ctx, catIDStr)
		if cat != nil {
			file.CategoryID = &cat.ID
		}
	}

	if err := h.store.CreateFile(ctx, file); err != nil {
		h.log.Error("confirmUpload", ports.F("err", err))
		return c.Send("❌ خطا در ذخیره فایل.")
	}

	return c.Send(
		fmt.Sprintf(
			"✅ <b>فایل ذخیره شد</b>\n\n"+
				"📌 %s\n🆔 <code>%s</code>",
			file.Title, file.ID,
		),
		tele.ModeHTML, kbMain(true),
	)
}

// ── helpers ──────────────────────────────────────────────

type fileInfo struct {
	fileID   string
	fileType string
	caption  string
}

func extractFileInfo(c tele.Context) *fileInfo {
	m := c.Message()
	switch {
	case m.Document != nil:
		return &fileInfo{m.Document.FileID, "document", m.Caption}
	case m.Video != nil:
		return &fileInfo{m.Video.FileID, "video", m.Caption}
	case m.Audio != nil:
		return &fileInfo{m.Audio.FileID, "audio", m.Caption}
	case m.Photo != nil:
		return &fileInfo{m.Photo.FileID, "photo", m.Caption}
	}
	return nil
}

func sendArchiveFile(c tele.Context, f models.File, isAdmin bool) {
	file := tele.File{FileID: f.FileID}
	caption := "<b>" + f.Title + "</b>"
	if f.Description != "" {
		caption += "\n" + f.Description
	}
	if f.Tags != "" {
		caption += "\n🏷 " + strings.ReplaceAll(f.Tags, ",", " #")
	}

	var kb *tele.ReplyMarkup
	if isAdmin {
		kb = kbFileActions(f.ID.String())
	}

	var err error
	switch f.FileType {
	case "video":
		if kb != nil {
			err = c.Send(&tele.Video{File: file, Caption: caption}, tele.ModeHTML, kb)
		} else {
			err = c.Send(&tele.Video{File: file, Caption: caption}, tele.ModeHTML)
		}
	case "audio":
		if kb != nil {
			err = c.Send(&tele.Audio{File: file, Caption: caption}, tele.ModeHTML, kb)
		} else {
			err = c.Send(&tele.Audio{File: file, Caption: caption}, tele.ModeHTML)
		}
	case "photo":
		if kb != nil {
			err = c.Send(&tele.Photo{File: file, Caption: caption}, tele.ModeHTML, kb)
		} else {
			err = c.Send(&tele.Photo{File: file, Caption: caption}, tele.ModeHTML)
		}
	default:
		if kb != nil {
			err = c.Send(&tele.Document{File: file, Caption: caption}, tele.ModeHTML, kb)
		} else {
			err = c.Send(&tele.Document{File: file, Caption: caption}, tele.ModeHTML)
		}
	}
	_ = err
}

func cleanTags(text string) string {
	parts := strings.Split(text, ",")
	var clean []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p), "#"))
		if p != "" {
			clean = append(clean, p)
		}
	}
	return strings.Join(clean, ",")
}
