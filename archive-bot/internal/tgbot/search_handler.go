package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/search"
)

const maxResults = 10

// doSearch جستجوی متنی و ارسال نتایج.
func (h *Handler) doSearch(ctx context.Context, c tele.Context, query string) error {
	if len(strings.TrimSpace(query)) < 2 {
		return c.Send("حداقل ۲ کاراکتر وارد کنید.")
	}

	normalized := search.Normalize(query)
	files, err := search.Search(h.db.Conn(), normalized, maxResults)
	if err != nil {
		h.log.Error("doSearch", nil)
		return c.Send("❌ خطا در جستجو.")
	}

	if len(files) == 0 {
		return c.Send(
			fmt.Sprintf("🔍 نتیجه‌ای برای «%s» یافت نشد.", query),
			kbMain(h.isAdmin(c)),
		)
	}

	_ = c.Send(
		fmt.Sprintf("🔍 %d نتیجه برای «%s»:", len(files), query),
		kbMain(h.isAdmin(c)),
	)

	for _, f := range files {
		sendArchiveFile(c, f, h.isAdmin(c))
	}
	return nil
}

// onAdd شروع wizard آپلود.
func (h *Handler) onAdd(c tele.Context) error {
	if !h.isAdmin(c) {
		return nil
	}
	return c.Send(
		"<b>📤 آپلود فایل جدید</b>\n\nفایل مورد نظر را ارسال کنید:",
		tele.ModeHTML, kbCancel(),
	)
}

// onInlineQuery جستجوی inline در هر chat.
func (h *Handler) onInlineQuery(c tele.Context) error {
	query := c.Query().Text
	if len(strings.TrimSpace(query)) < 2 {
		return c.Answer(&tele.QueryResponse{
			Results: tele.Results{},
		})
	}

	ctx := context.Background()
	normalized := search.Normalize(query)
	files, err := search.Search(h.db.Conn(), normalized, 10)
	if err != nil || len(files) == 0 {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	var results tele.Results
	for i, f := range files {
		var result tele.Result
		caption := "<b>" + f.Title + "</b>"
		if f.Tags != "" {
			caption += "\n🏷 " + f.Tags
		}

		switch f.FileType {
		case "photo":
			result = &tele.PhotoResult{
				FileID:  f.FileID,
				Caption: caption,
			}
		case "video":
			result = &tele.VideoResult{
				FileID:  f.FileID,
				Caption: caption,
				Title:   f.Title,
			}
		default:
			result = &tele.DocumentResult{
				FileID:  f.FileID,
				Caption: caption,
				Title:   f.Title,
			}
		}

		result.SetResultID(fmt.Sprintf("%d", i))
		result.SetParseMode(tele.ModeHTML)
		results = append(results, result)
	}

	return c.Answer(&tele.QueryResponse{
		Results:    results,
		CacheTime:  30,
	})
}
