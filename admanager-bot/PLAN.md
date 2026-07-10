# AdManager Bot — نقشه‌ی پیاده‌سازی

این فایل راهنمای قدم‌به‌قدم توسعه است. هر فاز یک واحد مستقل است و می‌توان
آن را جداگانه تحویل داد.

---

## ماهیت پروژه (مهم)

این ربات یک **ابزار ادمین‌محور** است:

- فقط **صاحب/ادمین کانال‌ها** به ربات دسترسی دارد (تشخیص با `OWNER_ID`).
- ادمین ربات را به کانال‌های خودش اضافه می‌کند تا **مدیریت تبلیغات،
  زمان‌بندی پست‌ها و آمار** را راحت‌تر و حرفه‌ای‌تر انجام دهد.
- **هیچ نقش «مشتری/تبلیغ‌دهنده»، کیف پول، بودجه‌ی مالی یا workflow تأیید
  وجود ندارد.** کمپین صرفاً یک گروه‌بندی از تبلیغ‌ها + زمان‌بندی + هدف‌گذاری
  روی کانال‌هاست.

> این نسخه بازنویسی‌شده است؛ مفهوم مشتری و لایه‌ی پرداخت به‌طور کامل حذف شد.

---

## وضعیت فعلی

> ✅ فازهای ۱ تا ۸ پیاده‌سازی شده‌اند. فایل‌های هندلر اضافه‌شده:
> `keyboards.go`، `state.go`، `handler_channel.go`، `handler_campaign.go`،
> `handler_ad.go`، `handler_reply.go`، `handler_template.go`،
> `handler_stats.go` و scheduler واقعی. تست end-to-end با ربات و دیتابیس
> واقعی باقی مانده است.

اسکلت کامل پروژه ساخته شده:

```
admanager-bot/
├── cmd/bot/main.go              ✅ ساخته شده
├── internal/
│   ├── models/models.go         ✅ مدل‌های MongoDB (بدون Customer/مالی)
│   ├── store/
│   │   ├── store.go             ✅ پایه + helpers
│   │   ├── channel.go           ✅ Channel + Tag CRUD
│   │   ├── campaign.go          ✅ Campaign + Advertisement CRUD
│   │   ├── job.go               ✅ ScheduledJob CRUD
│   │   ├── reservation.go       ✅ Reservation CRUD
│   │   ├── reply.go             ✅ Reply CRUD
│   │   ├── setting.go           ✅ BotSettings
│   │   ├── template.go          ✅ CampaignTemplate CRUD
│   │   └── audit.go             ✅ AuditLog
│   ├── tgbot/
│   │   ├── bot.go               ✅ Handler + Deps + State types
│   │   └── router.go            ✅ onCallback + stub handlers
│   └── scheduler/
│       └── scheduler.go         ✅ Job loop + stub executors
├── go.mod                       ✅
└── PLAN.md                      ✅ (این فایل)
```

---

## فاز ۱ — احراز هویت و منوی اصلی

**هدف**: ادمین `/start` بزند، با `OWNER_ID` تأیید شود و منوی اصلی را ببیند.
کاربران غیرادمین پاسخ کوتاه «دسترسی ندارید» بگیرند.

### کارها

#### `internal/tgbot/bot.go`
- [ ] `onStart` واقعی: اگر `isAdmin` نبود → پیام رد دسترسی؛ در غیر این صورت
  نمایش پنل اصلی ادمین
- [ ] `onText` با state machine واقعی: `clearState` + switch بر اساس متن منو

#### `internal/tgbot/keyboards.go` (فایل جدید)
- [ ] `kbAdminMain()` — منوی اصلی (Reply Keyboard)
  - 📡 کانال‌ها | 📋 کمپین‌ها
  - ➕ کمپین جدید | 📈 آمار
  - ⚙️ تنظیمات

#### `internal/tgbot/state.go` (فایل جدید)
- [ ] `setState(ctx, uid, s state)` — ذخیره در Redis (JSON)
- [ ] `getState(ctx, uid) state`
- [ ] `clearState(ctx, uid)`
- [ ] `handleStep(ctx, c, st, text)` — switch مراحل

---

## فاز ۲ — مدیریت کانال

**هدف**: ادمین می‌تواند کانال‌ها را اضافه/فعال/غیرفعال/مشاهده کند.

### کارها

#### `internal/tgbot/handler_channel.go` (فایل جدید)
- [ ] `adminChannelsList(c)` — لیست کانال‌ها با صفحه‌بندی
- [ ] `adminChannelView(c, channelID)` — جزئیات + دکمه‌های فعال/غیرفعال
- [ ] `adminChannelSetStatus(c, channelID, action)` — تغییر وضعیت
- [ ] `adminChannelAddStart(c)` — شروع افزودن کانال (state: `stepChannelAdd`)
- [ ] در `handleStep`:
  - `stepChannelAdd` → دریافت username → اعتبارسنجی → `CreateChannel`
- [ ] `adminTagsList` / `adminTagAdd` — مدیریت برچسب‌ها

#### نکات فنی
- username کانال باید با `@` شروع شود یا عدد باشد
- `TelegramID` کانال از username با `bot.ChatByUsername`
- `member_count` با `bot.ChatMemberCount`
- ربات باید ادمینِ خودِ کانال باشد تا بتواند پست بفرستد

---

## فاز ۳ — ساخت و مدیریت کمپین

**هدف**: ادمین کمپین بسازد و وضعیتش را مدیریت کند.

### wizard ساخت کمپین

مراحل state machine:
1. `stepCampaignName` → دریافت نام
2. `stepCampaignMaxSends` → تعداد دفعات ارسال در هر کانال (0 = نامحدود)
3. انتخاب برچسب‌های هدف یا کانال‌های خاص (inline multi-select)
4. انتخاب پنجره‌ی زمانی (`start_hour`–`end_hour`) و بازه‌ی تاریخ
5. خلاصه و تأیید → `CreateCampaign` (وضعیت `draft`)
6. دکمه‌ی «شروع» → وضعیت `running` و ساخت رزروها/jobها

#### `internal/tgbot/handler_campaign.go` (فایل جدید)
- [ ] `campaignNew(c)` — شروع wizard
- [ ] `campaignsList(c, arg)` — لیست کمپین‌ها (فیلتر وضعیت)
- [ ] `campaignView(c, campaignID)` — جزئیات کمپین
- [ ] `campaignSetStatus(c, id, action)` — pause / resume / cancel

#### نکات فنی
- بعد از `running` شدن، jobهای `send_ad` بر اساس زمان‌بندی ساخته شوند
- لغو کمپین → `CancelCampaignJobs` + `CancelCampaignReservations`

---

## فاز ۴ — مدیریت محتوای تبلیغ

**هدف**: ادمین محتوای تبلیغ (متن/عکس/ویدئو/فوروارد) را برای کمپین بسازد.

### wizard ساخت تبلیغ

1. `stepAdName` → نام داخلی
2. انتخاب نوع (text / photo / video / forward)
3. `stepAdContent` → کاربر محتوا می‌فرستد
   - متن: مستقیم ذخیره
   - عکس/ویدئو: استخراج `file_id`
   - forward: ذخیره `source_channel_id` + `source_message_id`
4. `stepAdCaption` → (اختیاری، فقط برای رسانه)
5. افزودن دکمه‌های inline (اختیاری)
6. پیش‌نمایش + تأیید → `CreateAd`

#### `internal/tgbot/handler_ad.go` (فایل جدید)
- [ ] `adNew(c, campaignID)`
- [ ] `adView(c, adID)`
- [ ] `adDelete(c, adID)`
- [ ] `adPreview(c, adID)` — ارسال پیام نمونه به ادمین

---

## فاز ۵ — زمان‌بندی و ارسال خودکار (scheduler واقعی)

**هدف**: جایگزینی stubهای scheduler با منطق واقعی.

#### `internal/scheduler/scheduler.go`
- [ ] `sendAd(ctx, job)` واقعی:
  1. بارگذاری `Reservation` و `Advertisement`
  2. ارسال محتوا به کانال با `bot.Send(channel, content)`
  3. ذخیره `TelegramMessageID` در Reservation (`MarkReservationSent`)
  4. `IncrCampaignImpressions`
  5. اگر سقف `MaxSendsPerChannel` یا `EndAt` رسید → `endCampaign`
- [ ] `updateStats(ctx, job)`:
  1. برای هر کانال active: `bot.ChatMemberCount` → `UpdateChannelStats`
  2. محاسبه میانگین views
- [ ] `buildReservations(ctx, campaign)` — ساخت jobهای ارسال بر اساس زمان‌بندی

#### منطق زمان‌بندی
- هر بازه، scheduler کمپین‌های `running` را بررسی می‌کند (`ListActiveCampaigns`)
- برای هر کمپین، کانال‌های هدف انتخاب می‌شوند (`ListChannelsByTag` + `MinMemberCount`)
- job با `run_at` داخل پنجره‌ی `start_hour`–`end_hour` ساخته می‌شود
- `MaxSendsPerChannel` سقف تعداد ارسال در هر کانال را کنترل می‌کند

---

## فاز ۶ — پاسخ خودکار (Reply System)

**هدف**: ادمین قوانین پاسخ خودکار تعریف کند.

#### `internal/tgbot/handler_reply.go` (فایل جدید)
- [ ] `adminRepliesList(c)`
- [ ] `adminReplyNew(c)` → wizard: نوع → trigger → محتوا
- [ ] `adminReplyView(c, replyID)`
- [ ] `adminReplyDelete(c, replyID)`

#### `internal/tgbot/bot.go`
- [ ] `onText`: بعد از بررسی state machine، `FindReplyByKeyword` → ارسال پاسخ

---

## فاز ۷ — قالب‌های کمپین

**هدف**: ادمین قالب‌های پیش‌فرض بسازد تا کمپین‌ها سریع‌تر ساخته شوند.

#### `internal/tgbot/handler_template.go` (فایل جدید)
- [ ] `adminTemplateNew(c)` — ساخت قالب
- [ ] `adminTemplatesList(c)` — `ListTemplates`
- [ ] در wizard کمپین جدید: گزینه‌ی «از قالب» اضافه شود

---

## فاز ۸ — آمار و گزارش

**هدف**: داشبورد آماری برای ادمین.

#### `internal/tgbot/handler_stats.go` (فایل جدید)
- [ ] `adminStats(c)` واقعی:
  - تعداد کانال‌های فعال / کل
  - تعداد کمپین‌های در حال اجرا
  - جمع نمایش‌ها (impressions) و کلیک‌ها
  - موثرترین کمپین
  - نمودار روزانه (با Inline Keyboard صفحه‌بندی تاریخ)

---

## وابستگی‌های خارجی

| وابستگی | کجا استفاده می‌شود | نحوه‌ی اتصال |
|---|---|---|
| MongoDB | همه‌ی لایه‌ی store | از `shared-core/engine` |
| Redis | state machine کاربر | از `engine.Cache` |
| Telegram Bot API | ارسال پست، آمار کانال | `telebot.v4` |

> دیگر نیازی به `botpay`/کیف پول نیست. اگر بعداً نیاز به اتصال به سرویس‌های
> دیگر (مثل `member-bot` برای چک عضویت یا `fraud-engine`) بود، در همین جدول
> اضافه شود.

---

## نکات معماری

1. **instance_id** — همه‌ی query های MongoDB حتماً با `s.f(...)` فیلتر شوند.
2. **تک‌نقشه بودن** — هر ورودی ابتدا با `h.isAdmin` محافظت شود؛ هیچ مسیر مشتری نیست.
3. **state machine** — هر ورودی متنی ابتدا cancel چک شود، بعد state machine.
4. **scheduler idempotency** — job قبل از اجرا به `running` تغییر کند تا در
   restart دوباره اجرا نشود (`MarkJobRunning`).
5. **error logging** — همه‌ی خطاها با `zap.Error(err)` و context (campaign_id,
   channel_id) لاگ شوند.
