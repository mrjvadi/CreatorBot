# botmanager

## این سرویس چیست
ربات فروش اصلی پلتفرم — پنل کاربر (خرید پلن، ساخت ربات، مدیریت سرویس‌های خودش، کیف‌پول) و پنل ادمین کامل (کاربران، پلن‌ها، سرورها، تمپلیت‌ها، تنظیمات سیستم). تنها سرویسی که مدل‌هایش مستقیم از `shared-core/models` می‌آید، نه مدل اختصاصی خودش.

## مسئولیت‌ها
- ثبت/ورود کاربر (`GetOrCreateUser`)، نقش‌ها (`RoleUser`/`RoleAdmin`/`RoleOwner`).
- خرید پلن و ساخت instance (`internal/tgbot/user/wizard.go`'s `Provision`): انتخاب کمترین‌بارِ سرور، ساخت رکورد `BotInstance`، کسر پول از طریق `botpay` (با `natspayclient`)، صدور لایسنس از طریق `license-service` (اضافه‌شده ۲۰۲۶-۰۷-۰۲)، و ارسال `DeployCommand` به `agentmanager`. اگر deploy شکست بخورد، پول خودکار برمی‌گردد (`RefundOnFailure`).
- پنل ادمین: مسدود/آزادکردن کاربر، ارتقا به ادمین، افزودن اعتبار دستی، مدیریت سرور/تمپلیت/پلن با دکمه‌های ➕➖، broadcast، تست دیپلوی داخلی (`admin_svctest.go`).
- گوش‌دادن به heartbeat/نتیجه‌ی دستورات از `agentmanager` روی NATS.

## ارتباطات
- NATS: publish `deploy.<server_id>`, `service.creation.requested`, `freebot.created`؛ subscribe `agent.*.heartbeat`, `agent.*.result`.
- `natspayclient` برای `pay.*` — یکی از سه سرویس مرکزی مورد اعتماد `botpay` (`service_id="botmanager"`, کلید از `SERVICE_HMAC_SECRET` مشتق می‌شود — رجوع به گزارش امنیتی).
- `licenseclient` برای `license.*` — یکی از دو سرویس مجاز به `license.issue`/`license.revoke`.
- PostgreSQL مشترک پلتفرم، از طریق `shared-core/store`.

## ایرادها و نکات
- **بررسی و تأیید شد (نه ایراد)**: یک عامل تحقیقاتی قبلی ادعا کرده بود `onCallback` در `internal/tgbot/router.go` هیچ چک ادمینی ندارد. این ادعا **غلط** بود — کد از قبل یک گیت deny-by-default دارد: خط ۳۱ `isAdminOnlyAction(action) && !h.IsAdmin(c)` قبل از کل switch چک می‌شود، و `isAdminOnlyAction` هم پیشوند `admin_` را خودکار می‌گیرد هم یک map صریح (`block_user`, `make_admin`, `add_credit`, `plan_edit`, ...) دارد که همه‌ی اکشن‌های حساس را می‌پوشاند. نیازی به تغییر نیست.
- migration این سرویس از یک لیست متمرکز (`shared-core/models.AllModels()`) استفاده می‌کند — تنها سرویسی در کل پلتفرم که این الگوی خوب را دارد؛ بقیه (`botpay`, `ads-bot`, ...) هرکدام `AutoMigrate` جدا و دستی دارند.
- `Provision` حالا وابسته به سه سرویس مرکزی است (botpay + agentmanager + license-service)؛ صدور لایسنس عمداً fail-open نوشته شده (اگر `license-service` در دسترس نباشد، deploy را متوقف نمی‌کند، فقط `LICENSE_TOKEN` خالی می‌ماند) — یعنی این وابستگی جدید ریسک قطعی سرویس را افزایش نمی‌دهد.
