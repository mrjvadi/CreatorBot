package tgbot

import (
	"context"
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/search"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
)

const maxResults = 10

// doSearch جستجوی متنی و ارسال نتایج.
func (h *Handler) doSearch(ctx context.Context, c tele.Context, query string) error {
	if len(strings.TrimSpace(query)) < 2 {
		return c.Send("حداقل ۲ کاراکتر وارد کنید.")
	}

	normalized := search.Normalize(query)
	files, err := search.Search(ctx, h.db.Conn(), normalized, maxResults)
	if err != nil {
		h.log.Error("doSearch", ports.F("err", err))
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

// onInlineQuery جستجوی inline — از ArticleResult با دستور /file_{id} استفاده می‌کند.
// کاربر نتیجه را انتخاب می‌کند و bot فایل را به chat می‌فرستد.
func (h *Handler) onInlineQuery(c tele.Context) error {
	ctx := context.Background()
	query := c.Query().Text
	if len(strings.TrimSpace(query)) < 2 {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	normalized := search.Normalize(query)
	files, err := search.Search(ctx, h.db.Conn(), normalized, 10)
	if err != nil || len(files) == 0 {
		return c.Answer(&tele.QueryResponse{Results: tele.Results{}})
	}

	var results tele.Results
	for i, f := range files {
		desc := f.Description
		if desc == "" {
			desc = f.Tags
		}

		typeIcon := "📄"
		switch f.FileType {
		case "photo":
			typeIcon = "🖼"
		case "video":
			typeIcon = "🎬"
		case "audio":
			typeIcon = "🎵"
		}

		r := &tele.ArticleResult{
			ResultBase:  tele.ResultBase{},
			Title:       typeIcon + " " + f.Title,
			Description: desc,
			// وقتی انتخاب میشه این متن ارسال میشه
			Text: fmt.Sprintf("/file_%s", f.ID),
		}
		r.SetResultID(fmt.Sprintf("%d", i))
		results = append(results, r)
	}

	return c.Answer(&tele.QueryResponse{
		Results:   results,
		CacheTime: 30,
	})
}
