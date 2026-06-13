package tgbot

import (
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/archive-bot/internal/models"
)

const (
	btnSearch     = "🔍 جستجو"
	btnCategories = "📂 دسته‌بندی‌ها"
	btnHelp       = "❓ راهنما"
	btnCancel     = "❌ انصراف"
	btnBack       = "🔙 بازگشت"
	btnSkip       = "⏭ رد کردن"
	btnConfirm    = "✅ تأیید و آپلود"
)

func kbMain(isAdmin bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	if isAdmin {
		kb.Reply(
			kb.Row(kb.Text(btnSearch), kb.Text(btnCategories)),
			kb.Row(kb.Text("➕ فایل جدید"), kb.Text("📂 دسته جدید")),
			kb.Row(kb.Text(btnHelp)),
		)
	} else {
		kb.Reply(
			kb.Row(kb.Text(btnSearch), kb.Text(btnCategories)),
			kb.Row(kb.Text(btnHelp)),
		)
	}
	return kb
}

func kbCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnCancel)))
	return kb
}

func kbSkipCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnSkip), kb.Text(btnCancel)))
	return kb
}

func kbCategories(cats []models.Category) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, cat := range cats {
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("📂 %s", cat.Name), "cat:"+cat.ID.String()),
		))
	}
	rows = append(rows, kb.Row(kb.Data(btnBack, "back")))
	kb.Inline(rows...)
	return kb
}

func kbConfirmUpload() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnConfirm)),
		kb.Row(kb.Text(btnCancel)),
	)
	return kb
}

func kbFileActions(fileID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(kb.Row(kb.Data("🗑 حذف", "del:"+fileID)))
	return kb
}
