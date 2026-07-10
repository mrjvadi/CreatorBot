package tgbot

import (
	"context"
	"time"

	tele "gopkg.in/telebot.v4"
)

// sendActiveAd یک تبلیغ فعال را هنگام تحویل رسانه نمایش می‌دهد.
// extraKb (در صورت غیر nil) دکمه‌های رسانه را به همان پیام تبلیغ می‌چسباند
// (تا وقتی تبلیغ آخرین پیام است، دکمه‌ها زیرش باشند).
// خروجی: id پیام تبلیغ (0 اگر چیزی فرستاده نشد).
func (h *Handler) sendActiveAd(ctx context.Context, c tele.Context, extraRows []tele.Row) int {
	ads, err := h.Store.ListAds(ctx)
	h.LogErr("ListAds", err)
	if len(ads) == 0 {
		return 0
	}
	ad := ads[int(time.Now().UnixNano())%len(ads)]

	text := ad.Text
	if ad.Title != "" {
		text = "<b>" + ad.Title + "</b>\n\n" + ad.Text
	}
	if text == "" {
		return 0
	}

	// ترکیب دکمه‌ی تبلیغ (URL) با دکمه‌های رسانه در یک کیبورد
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	if ad.ButtonText != "" && ad.ButtonURL != "" {
		rows = append(rows, kb.Row(kb.URL(ad.ButtonText, ad.ButtonURL)))
	}
	rows = append(rows, extraRows...)

	opts := []any{tele.ModeHTML}
	if len(rows) > 0 {
		kb.Inline(rows...)
		opts = append(opts, kb)
	}
	m, err := c.Bot().Send(c.Recipient(), text, opts...)
	if err != nil || m == nil {
		return 0
	}
	return m.ID
}

// adsCount تعداد تبلیغ‌های فعال.
func (h *Handler) adsCount(ctx context.Context) int {
	ads, err := h.Store.ListAds(ctx)
	h.LogErr("ListAds", err)
	return len(ads)
}
