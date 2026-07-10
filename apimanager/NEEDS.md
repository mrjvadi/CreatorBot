# apimanager — چه چیزی نیاز داریم

## ۰. یافته‌ی بحرانیِ بررسی ۲۰۲۶-۰۷-۰۵: proxy کردنِ image-registry با هدر اشتباه

`apimanager/internal/handler/image_registry.go` (فایل تازه — بعد از بازخورد کاربر «باید یه صفحه برای
آپلود ایمیج داشته باشم») روت‌های `/api/v1/admin/images*` را می‌سازد و همه را به `image-registry` با یک
هدر `X-Admin-Key` (از env `IMAGE_REGISTRY_ADMIN_KEY`) proxy می‌کند. **مشکل**: سمتِ `image-registry`،
`X-Admin-Key` فقط روی `/v1/callers/*` چک می‌شود؛ مسیرهای `/v1/images*` (همان چیزی که این پنل صدا
می‌زند) فقط بر اساس **IP واقعیِ فراخوان** + فیلد `CanWrite` روی آن IP در جدول `AllowedCaller` تصمیم
می‌گیرند (رجوع `image-registry/README.md`, بخش «نکته‌ی مهم برای هر سرویس دیگری که این API را proxy
می‌کند»). یعنی تا وقتی IP خروجیِ خودِ `apimanager` (نه یک کلید) به‌عنوان یک `AllowedCaller` با
`CanWrite=true` در `image-registry` ثبت نشود، **هر تلاش ثبت/آپلود/ویرایش/حذف image از پنل وب `403`
می‌گیرد** — حتی با اینکه `X-Admin-Key` درست تنظیم شده باشد. (`GET /v1/check`/`GET /v1/images` هم به
همین IP نیاز دارند، فقط این‌ها نیازی به `CanWrite` ندارند — پس یک AllowedCaller حتی read-only هم برای
همان‌ها لازم است.)

**راه‌حل‌های ممکن (یکی را انتخاب کنید):**
1. **عملیاتی، بدون تغییر کد**: IP خروجیِ container/سرور `apimanager` را به‌عنوان یک `AllowedCaller` با
   `CanWrite=true` در `image-registry` ثبت کنید (`SEED_CALLER_CIDR` در `.env` آن سرویس، یا
   `POST /v1/callers` دستی). اگر هر دو در یک `docker-compose.yml` هستند، IP شبکه‌ی داخلی Docker است
   (رجوع `image-registry/README.md` بخش «نکته‌ی عملیاتی مهم»).
2. **تغییر کد در `image-registry`**: مسیرهای `/v1/images*` هم یک راه جایگزین برای احراز با
   `X-Admin-Key` قبول کنند (علاوه بر IP، نه جایگزینش) — برای مواردی که IP خروجیِ apimanager پایدار/
   قابل‌پیش‌بینی نیست (مثلاً پشت یک load balancer با IP های چرخشی).

**نکته‌ی جانبی**: `apimanager` هیچ proxy ای برای `/v1/callers/*` (مدیریت خودِ لیست IP های مجاز) ندارد
— یعنی حتی بعد از حل مشکل بالا، اضافه‌کردن یک سرور/agentmanager جدید به `image-registry` هنوز فقط با
`curl`+`X-Admin-Key` مستقیم روی خودِ `image-registry` ممکن است، نه از پنل وب.

**کامنت داخل خودِ کد** (`image_registry.go:19-27`) هشدار می‌دهد که مشخصات این proxy از یک متنِ
«به‌شدت خراب/ناخوانا» (احتمالاً خروجی افتاده‌ی PDF/Word) برداشت شده. در این بررسی، مسیرهای واقعیِ
`image-registry` (`/v1/images`, `/v1/images/:id/file`, `/v1/check`, ...) با کد واقعیِ آن سرویس تطبیق
داده شدند و **درست از آب درآمدند** — یعنی خودِ نگاشتِ HTTP مسیرها مشکلی ندارد؛ تنها مشکلِ واقعی همان
اختلافِ مدل احراز هویت (IP در برابر کلید) در بالا بود، نه اسم/شکلِ endpoint ها.

این فایل باقیِ خودش (زیر) توسط بررسی‌کننده‌ی ۴ سرویس دیگر (`uploader-bot`, `vpn-bot`,
`archive-bot`, `member-bot`) در تاریخ ۲۰۲۶-۰۷-۰۴ نوشته شده — تمرکز بر این
سؤال: «چه CRUD عمیقی این سرویس‌ها نیاز دارند که apimanager فعلاً پوشش
نمی‌دهد؟». بررسی خود `apimanager` آن‌جا انجام نشد؛ فقط سطح فعلی API آن از
`apimanager/internal/handler/handler.go` و `apimanager/cmd/main.go` (روت‌ها)
خوانده شد تا مشخص شود چه وجود دارد.

## آنچه apimanager همین الان دارد

`apimanager/cmd/main.go:143-187` نشان می‌دهد API فعلی فقط شامل: مدیریت
instance (start/stop/restart/delete/logs)، تنظیمات عمومی
(`GET/PUT /instances/:id/settings` → `GetInstanceSettings`/
`UpdateInstanceSettings` که در برابر `BotTemplate.ConfigSchema` اعتبارسنجی
می‌شود)، پلن‌ها، پرداخت‌ها، کاربران و سرورها است. **هیچ endpoint اختصاصی
برای محتوای داخلی یک نوع ربات خاص وجود ندارد** — یعنی نمی‌شود از وب
apimanager لیست کدهای uploader-bot، پنل‌های VPN، یا قفل‌های member-bot را
دید یا مدیریت کرد؛ فقط یک `key=value` عمومی قابل تزریق است (که هر بات باید
خودش در NATS گوش بدهد و تفسیر کند — نمونه: `uploader-bot/internal/tgbot/nats_config.go:34-41`).

## درخواست‌های مشخص از ۴ سرویس

### از uploader-bot
مدل‌های `Code`, `Folder`, `ForceJoinChannel`, `Backup`
(`uploader-bot/internal/models/models.go`, `channel.go`, `billing.go`) و
متدهای store در `uploader-bot/internal/store/code.go`,
`folder.go`, `channel.go`, `backup.go` — کاندیدای یک صفحه‌ی «مدیریت محتوا»
در وب apimanager هستند: لیست کدهای فروخته‌شده، پوشه‌بندی، قفل‌های اجباری
عضویت کانال، و دانلود/آپلود بکاپ، بدون نیاز به باز کردن خود تلگرام.
در حال حاضر این‌ها فقط از طریق پنل ادمین داخل خود ربات تلگرام قابل مدیریت‌اند.

### از vpn-bot
مدل `Panel` (`vpn-bot/internal/models/models.go:36-46`) —
apimanager می‌تواند یک endpoint مثل `GET/POST /instances/:id/vpn-panels`
اضافه کند تا مالک ربات بدون ورود به تلگرام بتواند پنل Marzban/Marzneshin/
Hiddify/XUI را اضافه/غیرفعال کند (فعلاً فقط از `internal/tgbot/admin_panel.go`
داخل تلگرام ممکن است). توجه: طبق `vpn-bot/NEEDS.md`، فعلاً فقط نوع پنل
`marzban` واقعاً در `cmd/bot/main.go` وصل است؛ اگر apimanager این را قبل از
رفع آن باگ اضافه کند، باید به کاربر هشدار بدهد که انتخاب نوع دیگر باعث کرش
سرویس در استارت بعدی می‌شود.

### از archive-bot
مدل‌های `Category` و `File`
(`archive-bot/internal/models/models.go:29-47`) — کاندیدای ساده‌ای برای
CRUD وب هستند چون سرویس تک‌ادمین (owner-only) است و پیچیدگی دسترسی کمی
دارد. اولویت پایین‌تر از بقیه چون سرویس کوچک است و فعلاً از طریق خود ربات
به‌راحتی مدیریت می‌شود.

### از member-bot
مدل‌های `Lock` و `CheckBot`
(`member-bot/internal/models/models.go:40-61`) — کاندیدای مدیریت وب برای
مالکان قفل کانال. **هشدار مهم:** طبق `member-bot/NEEDS.md`، مسیر پرداخت/
تسویه‌ی اجاره‌ی قفل در خود member-bot هنوز کامل وصل نیست (`CreatePayment`
هرگز صدا زده نمی‌شود، `ApprovePayment` هرگز `UpdateBalance` را صدا نمی‌زند).
اگر apimanager بخواهد این را expose کند، فعلاً باید فقط «نمایش وضعیت قفل‌ها»
باشد، نه عملیات مالی — تا آن گپ در خود member-bot رفع شود.

## به‌روزرسانی ۲۰۲۶-۰۷-۰۶: جداسازی دیتابیس
`apimanager` عمداً همچنان دیتابیس `botmanager` را با خودِ `botmanager` مشترک نگه داشت (تنها استثنا در
این دور جداسازی) — چون این دو دقیقاً همان مدل‌های `shared-core` (`User`, `BotInstance`, `Plan`, ...) را
می‌خوانند/می‌نویسند، نه دو مالک داده‌ی مستقل. بقیه‌ی سرویس‌های مرکزی (`botpay`, `ads-bot`,
`community-service`, `revenue-service`, `license-service`, `image-registry`) هرکدام حالا دیتابیس
مخصوص خودشان را دارند (رجوع `deploy/migrations/000_create_databases.sql`, `docker-compose.yml`).

## جمع‌بندی پیشنهادی

هر ۴ سرویس یک الگوی مشترک نیاز دارند که apimanager فعلاً ندارد: توانایی
افزودن endpointهای «CRUD اختصاصی به نوع بات» (نه فقط تنظیمات عمومی
key/value). یک الگوی ممکن: هر بات یک NATS subject درخواست/پاسخ اختصاصی
expose کند (مثل `uploader.codes.list`, `vpnbot.panels.list`,
`memberbot.locks.list`) و apimanager یک لایه‌ی نازک HTTP↔NATS روی آن‌ها
بسازد — هم‌سو با نقش بلندمدت apimanager طبق ریشه‌ی CLAUDE.md بخش ۳
(«هدف نهایی: ترجمه‌ی HTTP↔NATS»).
