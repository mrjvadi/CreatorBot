package tgbot

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// notifyAdminsPending به ادمین‌ها اطلاع می‌دهد یک فایل در انتظار تایید است.
func (h *Handler) notifyAdminsPending(ctx context.Context, code *models.Code, from *tele.User) {
	msg := fmt.Sprintf("🆕 فایل جدید در انتظار تایید\n👤 از: %d (@%s)\n🔑 کد: %s",
		from.ID, from.Username, code.Code)
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("✅ تایید", "code_approve:"+code.ID),
			kb.Data("🗑 رد", "code_reject:"+code.ID),
		),
	)
	admins, err := h.Store.ListAdmins(ctx)
	h.LogErr("notifyAdminsPending: list admins", err)
	recipients := map[int64]bool{}
	for _, a := range admins {
		recipients[a.TelegramID] = true
	}
	if h.OwnerID != 0 {
		recipients[h.OwnerID] = true
	}
	for id := range recipients {
		if _, sendErr := h.Bot.Send(&tele.User{ID: id}, msg, kb); sendErr != nil {
			h.LogErr("notifyAdminsPending: send", sendErr)
		}
	}
}

// adminPendingList صف تایید رسانه‌های کاربران.
func (h *Handler) adminPendingList(ctx context.Context, c tele.Context) error {
	codes, err := h.Store.ListPendingCodes(ctx, 30)
	h.LogErr("adminPendingList", err)
	if len(codes) == 0 {
		return c.Edit("📭 صف تایید خالی است.", kbBackHome())
	}
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, code := range codes {
		label := code.Caption
		if label == "" {
			label = code.Code
		}
		rows = append(rows, kb.Row(
			kb.Data("👁 "+label, "code_resend:"+code.Code),
			kb.Data("✅", "code_approve:"+code.ID),
			kb.Data("🗑", "code_reject:"+code.ID),
		))
	}
	rows = append(rows, kb.Row(kb.Data(btnBackLabel, "p:home")))
	kb.Inline(rows...)
	return c.Edit(fmt.Sprintf("⏳ صف تایید (%d):", len(codes)), kb)
}

func (h *Handler) adminApproveCode(ctx context.Context, c tele.Context, id string) error {
	if err := h.Store.ApproveCode(ctx, id); err != nil {
		h.LogErr("adminApproveCode", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا"})
	}
	code, err := h.Store.FindCodeByID(ctx, id)
	h.LogErr("adminApproveCode: find", err)
	if code != nil && code.UploaderID != 0 {
		if _, sendErr := h.Bot.Send(&tele.User{ID: code.UploaderID},
			fmt.Sprintf("✅ فایل شما تایید شد.\n🔑 کد: %s", code.Code)); sendErr != nil {
			h.LogErr("adminApproveCode: notify uploader", sendErr)
		}
	}
	h.LogErr("adminApproveCode: respond", c.Respond(&tele.CallbackResponse{Text: "✅ تایید شد"}))
	return c.Edit("✅ تایید شد.")
}

func (h *Handler) adminRejectCode(ctx context.Context, c tele.Context, id string) error {
	// قبل از حذف، آپلودکننده را پیدا می‌کنیم تا بتوانیم به او هم اطلاع بدهیم
	// (قبلاً فقط تاییدشده‌ها اطلاع‌رسانی می‌شدند؛ رد شدن بی‌سروصدا بود).
	code, err := h.Store.FindCodeByID(ctx, id)
	h.LogErr("adminRejectCode: find", err)
	if err := h.Store.DeleteCode(ctx, id); err != nil {
		h.LogErr("adminRejectCode: delete", err)
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در حذف"})
	}
	if code != nil && code.UploaderID != 0 {
		if _, sendErr := h.Bot.Send(&tele.User{ID: code.UploaderID}, "❌ فایل ارسالی شما رد شد."); sendErr != nil {
			h.LogErr("adminRejectCode: notify uploader", sendErr)
		}
	}
	h.LogErr("adminRejectCode: respond", c.Respond(&tele.CallbackResponse{Text: "🗑 رد شد"}))
	return c.Edit("🗑 رد و حذف شد.")
}

// kbBackHome دکمه‌ی بازگشت به خانه.
func kbBackHome() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data(btnBackLabel, "p:home")))
	return kb
}
