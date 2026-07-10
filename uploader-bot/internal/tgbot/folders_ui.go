package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// adminFolderBrowse مرور یک پوشه: زیرپوشه‌ها + رسانه‌ها + ساخت زیرپوشه.
func (h *Handler) adminFolderBrowse(ctx context.Context, c tele.Context, folderID string) error {
	parent := folderID
	if parent == "root" {
		parent = ""
	}
	subs, err := h.Store.ListFolders(ctx, parent)
	h.LogErr("adminFolderBrowse: list folders", err)
	codes, _, err := h.Store.ListCodes(ctx, parent, 1, 50)
	h.LogErr("adminFolderBrowse: list codes", err)

	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, f := range subs {
		rows = append(rows, kb.Row(
			kb.Data("📁 "+f.Name, "afolder:"+f.ID),
			kb.Data("🗑", "folder_delete:"+f.ID),
		))
	}
	for _, code := range codes {
		label := code.Caption
		if label == "" {
			label = code.Code
		}
		rows = append(rows, kb.Row(kb.Data("📄 "+label, "admin_code_edit:"+code.ID)))
	}
	rows = append(rows, kb.Row(kb.Data("➕ زیرپوشه", "folder_newsub:"+folderID)))
	rows = append(rows, kb.Row(kb.Data(btnBackLabel, "p:folders")))
	kb.Inline(rows...)

	title := "📂 ریشه"
	if parent != "" {
		title = "📂 پوشه"
	}
	return c.Edit(fmt.Sprintf("%s\n📁 زیرپوشه: %d   📄 رسانه: %d", title, len(subs), len(codes)), kb)
}

// adminNewSubfolder ساخت زیرپوشه زیر یک پوشه‌ی والد.
func (h *Handler) adminNewSubfolder(ctx context.Context, c tele.Context, parentID string) error {
	if parentID == "root" {
		parentID = ""
	}
	h.SetStepData(ctx, c.Sender().ID, stepNewSubfolder, "parent_id", parentID)
	return c.Send("📁 نام زیرپوشه‌ی جدید را بفرستید:", kbCancelOnly())
}

func (h *Handler) adminSaveSubfolder(ctx context.Context, c tele.Context, st userState, text string) error {
	h.ClearState(ctx, c.Sender().ID)
	f := &models.Folder{Name: text, ParentID: st.Data["parent_id"], IsActive: true}
	if err := h.Store.CreateFolder(ctx, f); err != nil {
		return c.Send("❌ خطا در ساخت زیرپوشه.", kbAdmin())
	}
	return c.Send("✅ زیرپوشه ساخته شد.", kbAdmin())
}

// ── انتقال کد به پوشه ─────────────────────────────────────────

func (h *Handler) adminCodeMoveMenu(ctx context.Context, c tele.Context, codeID string) error {
	folders, err := h.Store.ListFolders(ctx, "")
	h.LogErr("adminCodeMoveMenu", err)
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	rows = append(rows, kb.Row(kb.Data("📭 بدون پوشه", "code_moveto:"+codeID+":root")))
	for _, f := range folders {
		rows = append(rows, kb.Row(kb.Data("📁 "+f.Name, "code_moveto:"+codeID+":"+f.ID)))
	}
	rows = append(rows, kb.Row(kb.Data(btnBackLabel, "admin_code_edit:"+codeID)))
	kb.Inline(rows...)
	return c.Edit("📂 این رسانه به کدام پوشه منتقل شود؟", kb)
}

func (h *Handler) adminCodeMoveTo(ctx context.Context, c tele.Context, codeID, folderID string) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminCodeMoveTo: find", err)
	if code == nil {
		return c.Edit(msgNotFound)
	}
	if folderID == "root" {
		folderID = ""
	}
	code.FolderID = folderID
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminCodeMoveTo: update", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ انتقال ناموفق بود"})
	}
	return h.adminEditCodeMenu(ctx, c, codeID)
}
