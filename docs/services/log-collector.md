# log-collector (سرویس جدید — ساخته‌شده در ۲۰۲۶-۰۷-۰۲)

## این سرویس چیست
یک سرویس مرکزی تازه که لاگ‌های سطح Warn به بالا (Warn/Error/Fatal) از همه‌ی ۱۹ سرویس دیگر را روی NATS جمع می‌کند، در MongoDB ذخیره می‌کند، قابل کوئری است، و (اگر تنظیم شده باشد) به یک سوپرگروه فوروم تلگرام هشدار می‌فرستد — هر سرویس یک topic اختصاصی خودش را می‌گیرد.

## مسئولیت‌ها
1. **دریافت**: subscribe به `logs.events` (NATS) — همه‌ی سرویس‌ها این را منتشر می‌کنند (رجوع به بخش «چطور کار می‌کند»).
2. **ذخیره**: هر لاگ در MongoDB (`log_entries`) با ایندکس روی `timestamp`, `service`, `level`.
3. **کوئری**: `GET /logs?service=&level=&q=&from=&to=&limit=&skip=` (پشت `X-API-Key`، fail-closed اگر کلید تنظیم نشده باشد).
4. **هشدار تلگرام**: اولین لاگ هر سرویس یک forum topic با نام همان سرویس می‌سازد (`createForumTopic`)؛ نگاشت سرویس→topic در Mongo (`log_topics`) ذخیره می‌شود تا دوباره ساخته نشود؛ پیام‌های بعدی با `sendMessage`+`message_thread_id` به همان topic می‌روند. حداقل سطح ارسال به تلگرام با `MIN_TELEGRAM_LEVEL` قابل تنظیم است (پیش‌فرض warn).

## چطور کار می‌کند (تغییر در `shared/pkg/logger`)
لاگر مشترک (`shared/pkg/logger`, استفاده‌شده در همه‌ی سرویس‌ها) یک متد تازه گرفت: `log.AttachNATS(nc, "نام-سرویس")`. بعد از این فراخوانی، هر `log.Warn(...)`/`log.Error(...)`/`log.Fatal(...)` علاوه بر خروجی محلی معمول (stdout/فایل — بدون تغییر)، یک `LogEvent` JSON هم روی `logs.events` منتشر می‌کند. `Debug`/`Info` هرگز منتشر نمی‌شوند — فیلتر همان لحظه‌ی تولید لاگ انجام می‌شود تا حجم NATS زیاد نشود. این مسیر عمداً **best-effort** است: اگر `AttachNATS` صدا زده نشود کاری تغییر نمی‌کند؛ اگر NATS در دسترس نباشد یا publish شکست بخورد، هیچ‌وقت باعث panic/توقف سرویس اصلی نمی‌شود (`recover()` دور آن هست).

همه‌ی ۱۵ سرویس دارای NATS (به‌جز `source-service` که اصلاً NATS ندارد، و خودِ `log-collector`) به این مسیر وصل شده‌اند: botmanager, apimanager, agentmanager, webhook-gateway, botpay, license-service, uploader-bot, vpn-bot, archive-bot, member-bot, ads-bot, admanager-bot, community-service, fraud-engine, revenue-service.

## ارتباطات
- NATS: subscribe `logs.events`.
- MongoDB: `log_entries`, `log_topics`.
- HTTP: `GET /logs` (پشت API key)، `GET /health`. پورت پیش‌فرض ۸۰۹۹.
- تلگرام Bot API (یا `LOCAL_BOT_API`) برای `createForumTopic`/`sendMessage`.

## راه‌اندازی تلگرام
۱) یک ربات تازه از BotFather بسازید. ۲) یک سوپرگروه بسازید و از تنظیمات گروه، «Topics» (فوروم) را فعال کنید. ۳) ربات را ادمین گروه کنید با اجازه‌ی «Manage Topics». ۴) `TELEGRAM_BOT_TOKEN` و `TELEGRAM_CHAT_ID` (آی‌دی عددی منفی سوپرگروه) را در `.env` این سرویس پر کنید. بدون این دو، سرویس فقط در Mongo ذخیره می‌کند و کاملاً سالم کار می‌کند (fail-soft، نه fail-closed).

## ایرادها و نکات
- **fail-closed درست**: بدون `LOG_API_KEY`، `GET /logs` همه‌ی درخواست‌ها را رد می‌کند (نه اینکه پیش‌فرض باز بماند) — طبق همان الگویی که در رفع باگ‌های امنیتی قبلی این پروژه استفاده شد.
- **محدودیت شناخته‌شده**: چون این هم یک subject معمولی NATS بدون ACL است (مثل همه‌جای دیگر این پلتفرم)، هر کلاینتی با دسترسی NATS نظری می‌تواند `LogEvent` جعلی به `logs.events` بفرستد و آن را در Mongo/تلگرام تزریق کند (نه یک سطح ریسک بالا — نهایتاً یک لاگ جعلی ذخیره می‌شود، دسترسی به پول/داده‌ی دیگری نمی‌دهد — ولی می‌تواند spam کند). بخشی از همان ضعف عمومی نبودِ ACL در NATS.
- **تازه‌ساخته‌شده، تست end-to-end نشده**: ساخت topic فوروم، ارسال پیام، و مسیر کامل logger→NATS→Mongo→Telegram هنوز با یک محیط واقعی امتحان نشده‌اند — قبل از تکیه‌کردن روی این برای هشدار production، یک تست دستی توصیه می‌شود.
- `source-service` به این مسیر وصل نشده چون اصلاً کلاینت NATS ندارد (فقط Postgres/Redis/HTTP) — لاگ‌های Warn/Error آن فقط محلی (stdout) باقی می‌مانند، مگر اینکه در آینده یک اتصال NATS به آن اضافه شود.
