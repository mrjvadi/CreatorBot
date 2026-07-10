package tgbot

import (
	"context"
	"fmt"
	"strconv"

	tele "gopkg.in/telebot.v4"
)

// adminFilesOrder منوی مرتب‌سازی فایل‌های یک کد.
func (h *Handler) adminFilesOrder(ctx context.Context, c tele.Context, codeID string) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminFilesOrder: find", err)
	if code == nil {
		return c.Edit(msgNotFound)
	}
	files, err := h.Store.GetFilesForCode(ctx, codeID)
	h.LogErr("adminFilesOrder: files", err)
	if len(files) == 0 {
		return c.Edit("📭 این کد فایلی ندارد.", kbBackHome())
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, f := range files {
		label := fmt.Sprintf("%d. %s %s", i+1, fileTypeIcon(f.FileType), shortCap(f.Caption))
		rows = append(rows, kb.Row(
			kb.Data("⬆️", "fmoveup:"+codeID+":"+strconv.Itoa(i)),
			kb.Data(label, "noop"),
			kb.Data("⬇️", "fmovedown:"+codeID+":"+strconv.Itoa(i)),
		))
	}
	rows = append(rows, kb.Row(kb.Data(btnBackLabel, "admin_code_edit:"+codeID)))
	kb.Inline(rows...)
	return c.Edit("🔀 ترتیب فایل‌ها را با ⬆️/⬇️ تنظیم کنید:", kb)
}

// adminFileMove یک فایل را در ترتیب جابه‌جا می‌کند (dir = -1 بالا، +1 پایین).
func (h *Handler) adminFileMove(ctx context.Context, c tele.Context, codeID, idxStr string, dir int) error {
	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminFileMove: find", err)
	if code == nil {
		return c.Edit(msgNotFound)
	}
	i, _ := strconv.Atoi(idxStr) // از دکمه‌ی خودمان می‌آید، همیشه عددی معتبر است
	j := i + dir
	ids := code.FileIDs
	if i < 0 || i >= len(ids) || j < 0 || j >= len(ids) {
		return h.adminFilesOrder(ctx, c, codeID)
	}
	ids[i], ids[j] = ids[j], ids[i]
	code.FileIDs = ids
	if err := h.Store.UpdateCode(ctx, code); err != nil {
		h.LogErr("adminFileMove: update", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ ذخیره نشد"})
	}
	return h.adminFilesOrder(ctx, c, codeID)
}

func shortCap(s string) string {
	if len(s) > 25 {
		return s[:25] + "…"
	}
	return s
}
