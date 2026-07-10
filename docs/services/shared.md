# shared

## این سرویس چیست
`shared` یک سرویس مستقل نیست — یک ماژول Go کتابخانه‌ای است که همه‌ی سرویس‌های دیگر برای عملیات پایه و مشترک (اتصال به DB/Redis/NATS، رمزنگاری، JWT، تنظیمات، لاگ، متریک) از آن استفاده می‌کنند. هدف این ماژول جلوگیری از تکرار کد بین ۱۸+ سرویس است.

## بسته‌های کلیدی (`shared/pkg/...`)
- `auth` — تولید/اعتبارسنجی JWT (`GenerateAccessToken`/`ParseAccessToken`)، رمزنگاری AES-256-GCM (`Encrypt`/`Decrypt`)، و (اضافه‌شده در این جلسه) `ComputeServiceKey`/`ValidateServiceKey` برای احراز هویت سرویس-به-سرویس مبتنی بر HMAC.
- `adapters/postgres`, `adapters/mongodb`, `adapters/redis`, `adapters/nats` — wrapper های اتصال به هر یک از این زیرساخت‌ها با یک `Config` ساده.
- `adapters/webhook` — تشخیص حالت polling/webhook هر ربات و ساخت poller مناسب؛ استخراج BotID از توکن.
- `adapters/marzban`, `adapters/zarinpal`, `adapters/nowpayments` — کلاینت‌های درگاه‌های خاص vpn-bot.
- `config` — بارگذاری تنظیمات از env با `mapstructure` (`config.MustLoad`).
- `logger` — wrapper روی zap.
- `metrics` — متریک‌های Prometheus مشترک (deploy، پرداخت، fraud score، ...) + سرور `/metrics` و `/health`.
- `ports` — اینترفیس‌های انتزاعی (`Cache`, `DB`, `Logger`, `Notifier`, `VPNPanel`, `PaymentGateway`) که هر سرویس پیاده‌سازی می‌کند؛ این‌ها اجازه می‌دهند منطق کسب‌وکار به یک پیاده‌سازی خاص وابسته نباشد.
- `rotation` — کمک‌کننده برای rotation کلید/راز (کد آماده هست، ولی هیچ سرویسی فعلاً از آن استفاده نمی‌کند — رجوع به بخش ایرادها).

## ایرادها و نکات
- **کلید رمزنگاری بین سرویس‌ها ناهماهنگ بود**: مقدار `ENCRYPTION_KEY` در `.env` ریشه با مقدار استفاده‌شده در `botmanager/.env`, `vpn-bot/.env`, `member-bot/.env`, `apimanager/.env` یکی نیست (دو مقدار مختلف در گردش است). چون `auth.Encrypt`/`auth.Decrypt` کلید را از env هر سرویس می‌خوانند، اگر جایی این مقدار اشتباه ست شود، decrypt آن سرویس روی داده‌ی رمزنگاری‌شده توسط سرویس دیگر شکست می‌خورد. باید یکی از این دو مقدار انتخاب و در همه‌جا یکسان شود.
- بسته‌ی `rotation` نوشته شده ولی به هیچ سرویسی وصل نیست — یعنی چرخش کلید امروز عملاً هیچ‌جا اتفاق نمی‌افتد؛ اگر `ENCRYPTION_KEY` یا `SERVICE_HMAC_SECRET` نشت کند، تنها راه واقعی، rotate دستی و redeploy همه‌ی سرویس‌هاست.
- (اضافه‌شده ۲۰۲۶-۰۷-۰۲) `ComputeServiceKey`/`ValidateServiceKey` در `auth.go` — این‌ها بخشی از رفع یک باگ امنیتی critical در botpay هستند؛ رجوع کنید به `docs/security-audit-2026-07-02.md` بخش ۱.
