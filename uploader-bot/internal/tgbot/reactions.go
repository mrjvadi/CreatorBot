package tgbot

import (
	"context"

	tele "gopkg.in/telebot.v4"
)

// reactToggle واکنش کاربر را ثبت/برمی‌دارد و کیبورد را با شمارش جدید به‌روز می‌کند.
func (h *Handler) reactToggle(ctx context.Context, c tele.Context, codeStr string, val int) error {
	code, err := h.Store.FindCode(ctx, codeStr)
	h.LogErr("reactToggle: find", err)
	if code == nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ یافت نشد"})
	}

	newVal := h.Store.SetReaction(ctx, codeStr, c.Sender().ID, val)

	// همان کیبوردِ زیر فایل را با شمارش جدید بازسازی می‌کنیم.
	if kb, ok := h.buildFileKb(ctx, code); ok {
		h.LogErr("reactToggle: edit keyboard", c.Edit(kb))
	}

	txt := "👍 لایک شد"
	if val < 0 {
		txt = "👎 دیسلایک شد"
	}
	if newVal == 0 {
		txt = "واکنش برداشته شد"
	}
	return c.Respond(&tele.CallbackResponse{Text: txt})
}
