package tgbot

import (
	"fmt"

	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/member-bot/internal/models"
)

const (
	btnMyLocks  = "🔒 قفل‌های من"
	btnNewLock  = "➕ قفل جدید"
	btnMyBots   = "🤖 check bot ها"
	btnAddBot   = "➕ افزودن bot"
	btnBalance  = "💰 موجودی"
	btnHelp     = "❓ راهنما"
	btnCancel   = "❌ انصراف"
	btnBack     = "🔙 بازگشت"
)

func kbMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text(btnMyLocks), kb.Text(btnNewLock)),
		kb.Row(kb.Text(btnMyBots), kb.Text(btnAddBot)),
		kb.Row(kb.Text(btnBalance), kb.Text(btnHelp)),
	)
	return kb
}

func kbCancel() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(kb.Row(kb.Text(btnCancel)))
	return kb
}

func kbLockActions(lockID string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(kb.Data("⏸ توقف موقت", "lock_pause:"+lockID)),
		kb.Row(kb.Data("🗑 حذف قفل", "lock_delete:"+lockID)),
	)
	return kb
}

func kbAdminMain() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}
	kb.Reply(
		kb.Row(kb.Text("📊 آمار"), kb.Text("👥 مالکان")),
		kb.Row(kb.Text("🔒 همه قفل‌ها"), kb.Text("📣 broadcast")),
		kb.Row(kb.Text(btnBack)),
	)
	return kb
}

func fmtLock(l models.Lock) string {
	status := "🟢 فعال"
	if l.Status == models.LockExpired {
		status = "🔴 منقضی"
	}
	return fmt.Sprintf(
		"%s <b>%s</b>\n"+
			"📢 کانال: <code>%d</code>\n"+
			"👥 اعضا: %d\n"+
			"💰 %.0f تومان/روز",
		status, l.ChannelTitle, l.ChannelID,
		l.CurrentCount, l.PricePerDay,
	)
}
