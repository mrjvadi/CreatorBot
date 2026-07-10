# webhook-gateway

## این سرویس چیست
وقتی یک ربات به‌جای polling در حالت webhook اجرا می‌شود، تلگرام آپدیت‌ها را به این سرویس POST می‌کند؛ این سرویس آپدیت را به NATS subject مخصوص همان ربات forward می‌کند. هر ربات مستقل می‌تواند polling یا webhook انتخاب کند.

## مسئولیت‌ها
- `POST /webhook/:token` — دریافت آپدیت تلگرام، پیدا کردن ربات مربوطه از یک registry داخلی (بر اساس توکن)، و publish به `webhook.<bot_id>`.
- `POST /internal/register` و `/internal/unregister` — ثبت/حذف یک ربات در registry (هم از طریق HTTP، هم از طریق NATS subjects `gateway.register`/`gateway.unregister`).
- `/internal/bots`, `/internal/stats`, `/internal/health` — مشاهده‌ی وضعیت.
- Rate limiting سراسری (`GlobalRateLimit`، ۱۰۰۰ req/s) روی کل سرویس.

## ارتباطات
- HTTP روی پورت ۸۰۹۰ (پیش‌فرض).
- NATS: publish `webhook.<bot_id>` برای هر آپدیت؛ subscribe `gateway.register`/`gateway.unregister`.

## ایرادها و نکات
- **بحرانی، رفع شد در این جلسه**: روت‌های `/internal/*` قرار بود پشت یک `InternalAuth` (کلید API) باشند، ولی به‌خاطر یک باگ در نحوه‌ی استفاده از gin's `RouterGroup` (دو `Group("/internal")` جدا از هم ساخته می‌شد — یکی با middleware که هیچ route ای رویش ثبت نمی‌شد، یکی بدون middleware که route های واقعی رویش بودند)، این احراز هویت هرگز واقعاً اجرا نمی‌شد. یعنی هرکسی با دسترسی به پورت HTTP این سرویس می‌توانست webhook هر رباتی را hijack یا unregister کند — بدون هیچ کلیدی. رفع شد با ثبت route ها مستقیم روی همان گروهی که middleware رویش اعمال شده.
- **رفع شد**: یک `WebhookRateLimit` (محدودیت per-bot/per-IP، ۳۰ req/s) از قبل نوشته شده بود ولی هیچ‌جا به route واقعی وصل نشده بود — یک ربات بدرفتار می‌توانست کل بودجه‌ی نرخ‌محدودی مشترک (۱۰۰۰ req/s سراسری) را مصرف کند و webhook بقیه‌ی ربات‌ها را قحطی بدهد. الان وصل شده.
- ثبت داینامیک ربات از طریق NATS (`gateway.register`) هیچ اعتبارسنجی سرویس فرستنده ندارد — همان مشکل ریشه‌ای نبود ACL در NATS (نه چیزی خاص این سرویس).
