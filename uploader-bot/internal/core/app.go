// Package core وابستگی‌های مشترک ربات (App) و دسترسی‌های پایه را نگه می‌دارد.
// زیرپکیج‌های فیچر (در گام‌های بعد) روی *core.App کار خواهند کرد.
package core

import (
	tele "gopkg.in/telebot.v4"

	"github.com/mrjvadi/creatorbot/shared-core/engine"
	"github.com/mrjvadi/creatorbot/shared-core/memberclient"
	"github.com/mrjvadi/creatorbot/shared/pkg/joinevents"
	"github.com/mrjvadi/creatorbot/shared/pkg/ports"
	"github.com/mrjvadi/creatorbot/uploader-bot/internal/store"
)

// App وابستگی‌های مشترک یک نمونه‌ی ربات.
type App struct {
	Bot        *tele.Bot
	Store      *store.Store
	Cache      ports.Cache
	Log        ports.Logger
	OwnerID    int64
	ChannelID  int64
	InstanceID string // "bot_<botID>"
	Eng        *engine.Engine

	// EncryptKey برای رمزنگاری AES-256-GCM مقادیر حساس قبل از ذخیره در
	// Mongo — مثلاً BotToken یک قفل نوع «ربات» (ForceJoinChannel.BotToken)،
	// که قبلاً به‌صورت متن‌خام ذخیره می‌شد (رجوع کنید به گزارش امنیتی).
	EncryptKey string

	// RentalStatus وضعیتِ «آیا این instance (اگر رایگان است) الان به یک
	// کمپینِ اجاره‌ی قفلِ فعال در ads-bot وصل است» — nil یعنی این چک اصلاً
	// راه‌اندازی نشده (مثلاً NATS در دسترس نبوده).
	RentalStatus *memberclient.RentalStatus

	// JoinPublisher وقتی RentalStatus.IsInCampaign() باشد، عضویت‌های واقعیِ
	// کانالِ خریدار را به membership.joined/left منتشر می‌کند (برای پاداشِ
	// per-join در ads-bot). nil یعنی راه‌اندازی نشده.
	JoinPublisher *joinevents.Publisher
}

// LogErr خطای عملیات‌هایی که در هندلرها معمولاً «best effort» صدا زده می‌شوند
// (یک Get/List که نتیجه‌اش با nil-check ادامه پیدا می‌کند) را به‌جای نادیده
// گرفتنِ کامل با `_`، لاگ می‌کند. جریان کاربر تغییر نمی‌کند — فقط دیگر یک
// خطای واقعیِ DB/Cache در لاگ‌ها بی‌اثر و نامرئی نمی‌ماند.
func (a *App) LogErr(op string, err error) {
	if err == nil {
		return
	}
	if a.Log != nil {
		a.Log.Error("tgbot: "+op, ports.F("err", err))
	}
}
