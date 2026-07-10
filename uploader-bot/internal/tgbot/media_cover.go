package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"
)

// adminCodeAskCover از ادمین می‌خواهد عکس کاور را بفرستد.
func (h *Handler) adminCodeAskCover(ctx context.Context, c tele.Context, codeID string) error {
	h.SetStepData(ctx, c.Sender().ID, stepSetCover, "code_id", codeID)
	return c.Send("🖼 عکس کاور را بفرستید (روی ویدیوهای این کد اعمال می‌شود):", kbCancelOnly())
}

// adminSaveCover عکس ارسالی را به‌عنوان کاور ویدیوهای کد ذخیره می‌کند.
// از onMedia هنگام step=stepSetCover فراخوانی می‌شود.
func (h *Handler) adminSaveCover(ctx context.Context, c tele.Context) error {
	uid := c.Sender().ID
	st := h.GetState(ctx, uid)
	codeID := st.Data["code_id"]
	h.ClearState(ctx, uid)

	if c.Message().Photo == nil {
		return c.Send("❌ لطفاً یک عکس بفرستید.", kbAdmin())
	}
	thumb := c.Message().Photo.FileID

	code, err := h.Store.FindCodeByID(ctx, codeID)
	h.LogErr("adminSaveCover: find code", err)
	if code == nil {
		return c.Send(msgCodeNF, kbAdmin())
	}
	files, err := h.Store.GetFilesForCode(ctx, code.ID)
	h.LogErr("adminSaveCover: get files", err)
	cnt := 0
	for _, f := range files {
		if f.FileType == "video" {
			h.LogErr("adminSaveCover: set thumbnail", h.Store.SetFileThumbnail(ctx, f.ID, thumb))
			cnt++
		}
	}
	h.Store.InvalidateCode(ctx, code.Code)
	if cnt == 0 {
		return c.Send("⚠️ این کد ویدیویی ندارد که کاور بگیرد.", kbAdmin())
	}
	return c.Send(fmt.Sprintf("✅ کاور روی %d ویدیو اعمال شد.", cnt), kbAdmin())
}
