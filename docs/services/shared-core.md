# shared-core

## این سرویس چیست
مثل `shared`، این هم یک سرویس مستقل نیست بلکه یک ماژول کتابخانه‌ای است — ولی برخلاف `shared` (که ابزارهای عمومی هر نوع سرویس Go است)، `shared-core` مخصوص لایه‌ی «پلتفرم/فروش» است: مدل‌های داده‌ی `botmanager`، NATS protocol مشترک همه‌ی سرویس‌ها، و مهم‌تر از همه `engine` — موتوری که هر ربات ساخته‌شده (uploader-bot و از این جلسه به بعد سایر ربات‌ها هم) داخل خودش اجرا می‌کند.

## بسته‌های کلیدی
- `models` — تمام مدل‌های `botmanager`: `User`, `Server`, `BotTemplate`, `BotInstance`, `Plan`, `PlanBotLimit`, `Subscription`, `Payment`, `InviteLink`, `DeployJob`, `AuditLog`. تنها جایی در کل پروژه که یک `AllModels()` متمرکز برای migration دارد (بقیه‌ی سرویس‌ها این را ندارند).
- `protocol` — **مرکز تعریف همه‌ی NATS subjects و ساختار پیام‌ها** در کل پلتفرم: `deploy.*`, `agent.*`, `pay.*`, `member.check`, `ads.*`, و (اضافه‌شده در این جلسه) `license.*`. هر تغییر در قرارداد بین‌سرویسی باید این‌جا اضافه شود.
- `natspayclient` — کلاینت مشترک برای صحبت با `botpay` (کش Redis + NATS request/reply). فقط `botmanager` و `ads-bot` از این استفاده می‌کنند.
- `licenseclient` (اضافه‌شده در این جلسه) — کلاینت مشترک برای صحبت با `license-service`؛ شامل `RunLicenseLoop` که مستقیم داخل هر ربات (چه از طریق `engine` چه مستقل) اجرا می‌شود.
- `engine` — «موتور» کامل هر bot container: اتصال مستقیم Postgres (فیلتر با bot_id)، Mongo (فیلتر با instance_id)، Redis (پیشوند bot_id)، و یک heartbeat loop + license-check loop روی NATS. طبق کامنت خودِ کد، `apimanager` دیگر در مسیر hot-path نیست — هر بات مستقیم به DB وصل می‌شود.
- `docstore`/`configstore` — دسترسی به تنظیمات/آمار هر بات در MongoDB با ایزوله‌سازی خودکار instance_id.
- `store`/`docker`/`ton`/`schema` — لایه‌ی دیتابیس، مدیریت Docker (برای اجرای local/test)، کلاینت TON، و مدیریت schema (`schema.Create`/`Drop` با اعتبارسنجی نام — نسخه‌ی امن‌شده).

## ایرادها و نکات
- **فقط `uploader-bot` (و `admanager-bot`) از `engine` استفاده می‌کنند** — `vpn-bot`, `archive-bot`, `member-bot` هرکدام DB/Redis/NATS خودشان را جدا و دستی وصل می‌کنند، بدون این انتزاع مشترک. یعنی چهار پیاده‌سازی مشابه ولی غیر یکسان از یک منطق وجود دارد (دشوار برای نگه‌داری آینده).
- `store/store.go`'s `CreateInstanceWithSchema`/`DropInstanceSchema` مستقیماً SQL خام با string concatenation می‌سازند (`"CREATE SCHEMA IF NOT EXISTS " + inst.DBSchema`) به‌جای استفاده از تابع امنِ موجود در `schema/manager.go` (که همین کار را با اعتبارسنجی نام انجام می‌دهد). فعلاً چون `DBSchema` از کد داخلی ساخته می‌شود (نه ورودی کاربر) خطر فوری ندارد، ولی اگر مسیری در آینده اجازه دهد این مقدار از بیرون تنظیم شود، یک SQL injection/DROP دلخواه خواهد بود.
- (رفع‌شده در این جلسه) `engine.Start()` حالا علاوه بر heartbeat، یک license-check loop هم اجرا می‌کند (`licenseclient.RunLicenseLoop`) — به‌صورت fail-open، یعنی قطعی license-service هرگز ربات را متوقف نمی‌کند.
