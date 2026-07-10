package tgbot

import (
	"context"
	"fmt"
	"time"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

// گیت سین/ری‌اکشن اجباری فیک: قبل از تحویل، کاربر باید روی یک دکمه کلیک کند.

func (h *Handler) gateKey(uid int64, code string) string {
	return fmt.Sprintf("upl:gate:%s:%d:%s", h.InstanceID, uid, code)
}

func (h *Handler) gatePassed(ctx context.Context, uid int64, code string) bool {
	if h.Cache == nil {
		return true // بدون کش نمی‌توان گیت را اعمال کرد؛ مانع تحویل نشویم
	}
	v, err := h.Cache.Get(ctx, h.gateKey(uid, code))
	h.LogErr("gatePassed", err)
	return v == "1"
}

// sendGate پیام گیت را با دکمه‌های فیک نشان می‌دهد.
func (h *Handler) sendGate(c tele.Context, code *models.Code) error {
	kb := &tele.ReplyMarkup{}
	var btns []tele.Btn
	if code.ForceReact {
		btns = append(btns,
			kb.Data("👍", "gate:"+code.Code),
			kb.Data("❤️", "gate:"+code.Code),
			kb.Data("🔥", "gate:"+code.Code),
		)
	}
	if code.ForceSeen {
		btns = append(btns, kb.Data("👁 مشاهده کردم", "gate:"+code.Code))
	}
	kb.Inline(kb.Row(btns...))
	msg := "🙌 برای باز شدن قفل رسانه، فقط کافیه یکی از دکمه‌های زیر رو بزنی 👇"
	return c.Send(msg, kb)
}

// gatePass پس از کلیک کاربر، گیت را عبور می‌دهد و رسانه را تحویل می‌دهد.
func (h *Handler) gatePass(ctx context.Context, c tele.Context, codeStr string) error {
	uid := c.Sender().ID
	if h.Cache != nil {
		h.LogErr("gatePass: cache set", h.Cache.Set(ctx, h.gateKey(uid, codeStr), "1", 10*time.Minute))
	}
	h.LogErr("gatePass: respond", c.Respond(&tele.CallbackResponse{Text: "✅ ممنون!"}))
	h.LogErr("gatePass: delete gate message", c.Delete())
	user, err := h.Store.GetUser(ctx, uid)
	h.LogErr("gatePass: get user", err)
	return h.userDeliverCode(ctx, c, user, codeStr)
}
