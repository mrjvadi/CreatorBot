# Changelog — CreatorBot V3

## [2026-07-20] — botprofile.Sync حالا در dev هم پروفایلِ ربات را عادی می‌کند

قبلاً `shared/pkg/botprofile.Sync` (نام/description/bio/عکسِ ربات در تلگرام) فقط در
`APP_ENV=production` کاری می‌کرد — در dev هیچ‌کاری نمی‌کرد و پروفایلِ دستی/قدیمی هرچه بود
می‌ماند. کاربر خواست dev هم نرمالایز شود تا هیچ instance تستی با پروفایلی شبیهِ production
دیده نشود. رفع: `Sync` حالا همیشه description/bio/عکس را پاک می‌کند؛ در production نام برابرِ
نامِ سرویس می‌شود، در هر محیطِ غیرِproduction (development, staging, یا حتی خالی) نام با
برچسبِ محیط پسوند می‌خورد — مثلاً `"Uploader Bot (development)"`. چون این پکیجِ مشترک است و
هر ۸ سرویسِ رباتی (ads-bot, uploader-bot, botmanager, vpn-bot, member-bot, botpay,
admanager-bot, archive-bot) از قبل `botprofile.Sync` را در startup صدا می‌زنند، هیچ تغییرِ
دیگری در هیچ `main.go` ای لازم نبود — رفتارِ جدید خودکار روی همه اعمال می‌شود. تست‌ها
(`profile_test.go`) به‌روزرسانی شدند.

## [2026-07-20] — داشبوردِ زنده‌ی وضعیتِ سرویس‌ها در تلگرام (heartbeat + edit پیام)

قابلیتِ جدید: `log-collector` حالا یک پیامِ تکی در یک topic اختصاصی (`📊 وضعیت سرویس‌های اصلی`)
می‌سازد که وضعیتِ ۱۳ سرویسِ اصلیِ پلتفرم را نشان می‌دهد و **هر ۳۰ ثانیه به‌جای فرستادنِ پیامِ
جدید، همان پیام را edit می‌کند** — دقیقاً همان چیزی که کاربر خواست.

- **`shared/pkg/logger`**: subject جدید `service.heartbeat` — هر سرویسی که از قبل
  `log.AttachNATS(nc, name)` را صدا می‌زند (یعنی همه‌ی سرویس‌های مرکزی، بدون نیاز به تغییرِ حتی
  یک خط در `main.go` هرکدام) حالا خودکار هر ۲۰ ثانیه یک پیامِ حضور منتشر می‌کند. `HeartbeatEvent`
  علاوه بر timestamp، `StartedAt` (زمانِ ساختِ logger، نزدیک‌ترین لحظه به شروعِ واقعیِ پروسه) را
  هم حمل می‌کند — تا اگر خودِ log-collector ری‌استارت شود، آپ‌تایمِ نمایش‌داده‌شده درست بماند.
- **`log-collector/internal/status/`** (پکیجِ جدید): `Monitor` نقشه‌ی سرویس→آخرین‌حضور/زمانِ شروع
  را از روی heartbeatها نگه می‌دارد؛ `Reporter` هر ۳۰ ثانیه متن را رندر و ارسال/edit می‌کند
  (✅/⛔/❔ به‌ازای هر سرویس + آپ‌تایمِ انسانی‌خوان مثلِ «۲ روز و ۳ ساعت»، threshold برای down
  = ۳ برابرِ فاصله‌ی heartbeat).
- **`log-collector/internal/telegram`**: `EditMessage` (editMessageText) و
  `SendToTopicGetID` (sendMessage با برگرداندنِ message_id) اضافه شدند.
- **`log-collector/internal/store`**: کالکشنِ singleton جدید `log_status_dashboard` — topic/
  message_id پیامِ داشبورد را نگه می‌دارد تا با هر ری‌استارتِ log-collector همان پیام edit شود،
  نه یک پیامِ تکراریِ جدید.
- **رفعِ باگ: خودِ log-collector در داشبوردِ خودش «هنوز دیده نشده» می‌ماند** — چون
  `log-collector/cmd/main.go` هیچ‌وقت `log.AttachNATS(nc, "log-collector")` را روی
  loggerِ خودش صدا نمی‌زد (فقط برای مصرف‌کردنِ لاگ/heartbeatِ بقیه‌ی سرویس‌ها از NATS استفاده
  می‌شد، نه انتشارِ heartbeatِ خودش). رفع شد — یک خط اضافه شد.
- **پشتیبانیِ ربات‌های محصولِ چندنسخه‌ای** (پیگیریِ سؤالِ کاربر «سرویس‌های کاربران هم وضعیت
  می‌فرستن؟»): `uploader-bot`/`vpn-bot`/`archive-bot`/`member-bot` از قبل `AttachNATS` صدا
  می‌زدند، پس heartbeat جدید را خودکار می‌فرستادند — ولی چون این چهارتا برخلافِ ۱۳ سرویسِ
  مرکزی می‌توانند هم‌زمان چند instance (برای چند مشتری) داشته باشند، و heartbeat فقط با نامِ
  نوع شناسایی می‌شد، چند instance زیرِ یک کلید تداخل می‌کردند. رفع: `HeartbeatEvent` یک
  `InstanceID` اختیاری گرفت، `AttachNATS` یک پارامترِ variadic جدید برایش گرفت (امضای قبلی
  نشکست)، و هر چهار ربات همان `instanceID` (`"bot_" + botID`) که از قبل برای جداسازیِ
  tenant در Mongo دارند را پاس می‌دهند. داشبورد حالا این‌ها را جدا، به‌صورتِ «۴ از ۵ instance
  آنلاین» می‌شمارد (کاربر این گزینه را از بینِ چند طرح انتخاب کرد).

### باگ‌های واقعیِ کشف‌شده حینِ تست این قابلیت

- **`botpay`/`ads-bot` مقدارِ `LOCAL_BOT_API` را اصلاً از Config نمی‌خواندند** — یک IP قدیمی
  مستقیم در کدِ `tele.NewBot(...)` هاردکد شده بود. رفع شد (فیلدِ `LocalBotAPI` به هر دو Config
  اضافه شد، همان الگویی که بقیه‌ی ربات‌ها داشتند).
- **آلودگیِ محیطِ اجرا (مهم‌ترین کشف)**: یک نسخه‌ی کاملِ `.env` ریشه‌یِ *قدیمی* (قبل از بازنویسیِ
  همین روز — با هاست‌نیم‌های شبکه‌ی داکر مثلِ `mongo`/`postgres`/`nats` به‌جای `localhost`) در
  یک shell والدِ ناشناخته export شده بود و چون `viper.AutomaticEnv()` به متغیرهای محیطی
  اولویت می‌دهد نه فایلِ `.env.` واقعیِ هر سرویس، همه‌چیز — از جمله اتصالِ Mongo — بی‌صدا از این
  مقادیرِ قدیمی می‌خواند، نه از فایل‌های واقعاً ویرایش‌شده. راهِ حل در سطحِ کد: `directConnection=true`
  به همه‌ی ۸ مقدارِ `MONGO_URI` اضافه شد (چون سرورِ Mongo با هاست‌نیمِ داخلیِ داکرِ خودش
  (`mongo`) در پاسخِ hello جواب می‌دهد و بدونِ این flag، driver به آن سوییچ می‌کند). راهِ حلِ
  واقعی این است که کاربر منبعِ export را در shell/profile خودش پیدا و پاک کند — من نمی‌توانم آن
  را از داخلِ ابزارهای sandboxed خودم پاک کنم چون هر فراخوانی یک shell کاملاً تازه می‌گیرد.

## [2026-07-20] — بازنویسیِ کاملِ .env/.env.example همه‌ی ۲۱ سرویس

بررسیِ سیستماتیکِ کدِ واقعی (`mapstructure`/`os.Getenv` هر سرویس) در برابرِ `.env`/`.env.example`
فعلی — به‌جای فرض‌کردن، مستقیم از کد استخراج شد که هر سرویس واقعاً چه env ای می‌خواند، بعد با
فایل‌های موجود مقایسه شد. نتیجه: ۶ سرویس اصلاً `.env.example` نداشتند
(community-service, dbmigrate, fraud-engine, revenue-service, webhook-gateway — به‌علاوه‌ی ads-bot
که فقط از قبل‌موجود‌بودنش مطمئن نبودیم)، و اکثرِ بقیه بین ۲ تا ۹ متغیرِ واقعاً لازم را کم داشتند یا
متغیرهای مرده (خوانده‌نشده توسط هیچ کدی) را همچنان نگه داشته بودند.

**یافته‌های امنیتی/عملیاتیِ واقعی (نه فقط مستندسازی):**

- **member-bot Lock HTTP API فعلاً fail-open بود**: `LOCK_API_SECRET` اصلاً در `member-bot/.env`
  نبود — و `authMiddleware` وقتی `apiKey` خالی باشد، درخواستِ بدونِ هدرِ `X-API-Key` را هم قبول
  می‌کند (`"" != ""` یعنی false یعنی رد نمی‌شود). مقدارِ درست از `.env` ریشه‌ی legacy پروژه (که
  از قبل همین مقدار را داشت ولی هیچ‌وقت به `member-bot/.env` کپی نشده بود) بازیابی و اعمال شد.
- **`license-service/TEST_LICENSE_SECRET` یک secret واقعیِ کامیت‌شده در `.env.example` بود** (نه
  placeholder) و دقیقاً همان مقدار در `uploader-bot/.env.example` هم زیرِ نامِ اشتباهِ
  `LICENSE_SIGNING_SECRET` (یک متغیرِ کاملاً مرده — uploader-bot اصلاً این نام را نمی‌خواند)
  کپی شده بود. رفع: `TEST_LICENSE_SECRET` در `license-service` rotate شد (مقدارِ جدید با
  `openssl rand -hex 32`)؛ چون `uploader-bot/.env`'s واقعیِ `LICENSE_TOKEN` عمداً همان مقدار را
  به‌عنوانِ «لایسنسِ تستیِ سراسری» برای dev استفاده می‌کرد (نه یک اشتباه، طراحیِ عمدیِ bypass)،
  همان‌جا هم به مقدارِ جدید به‌روز شد تا هماهنگ بمانند؛ نامِ اشتباهِ `LICENSE_SIGNING_SECRET` از
  `uploader-bot/.env.example` حذف و با `LICENSE_TOKEN` (نامِ درست) جایگزین شد.
- **`apimanager/.env.example` و `botmanager/.env.example` هر دو مقادیرِ واقعیِ hex-shaped (نه
  `change_me`) داشتند** برای `POSTGRES_PASSWORD`/`ENCRYPTION_KEY`/`CENTRIFUGO_*`/`AGENT_API_KEY`
  — با placeholder جایگزین شدند.
- **`GRAFANA_PASSWORD` اصلاً در `.env` ریشه نبود** — یعنی Grafana بی‌صدا با پسوردِ پیش‌فرضِ
  ناامنِ `admin` بالا می‌آمد (`GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD:-admin}` در
  `docker-compose.yml`). یک پسوردِ تصادفیِ جدید اضافه شد.
- **`source-service` اصلاً `SERVICE_HMAC_SECRET` نداشت** — سرویس با `log.Fatal` بالا نمی‌آمد.
  مقدارِ مشترکِ پلتفرم (که در ۵ سرویسِ دیگر هم عیناً یکی است، تأیید شد با fingerprint، نه با
  چاپِ مقدار) به آن اضافه شد.
- **`vpn-bot`/`archive-bot` اصلاً `LICENSE_TOKEN` نداشتند** — با چکِ fail-closedِ
  `licenseclient.RequireValid` که هر دو دارند، این یعنی این دو سرویس با `.env` فعلی‌شان اصلاً
  بالا نمی‌آمدند. فیلد اضافه شد (خالی — در deploy واقعی توسط botmanager تزریق می‌شود).
- **`vpn-bot/.env`'s `PANEL_TOKEN` کاملاً مرده بود** (کد با `PANEL_USERNAME`+`PANEL_PASSWORD`
  احراز هویت می‌کند، نه یک توکنِ تکی) — جایگزین شد.
- **متغیرهای کاملاً مرده حذف شدند** (خوانده‌نشده توسط هیچ کدی، تأیید با grep سراسریِ ریپو):
  `ads-bot` (`LOCAL_BOT_API`, `FRAUD_ENGINE_URL`, `FRAUD_ADMIN_KEY`), `botpay`
  (`LOCAL_BOT_API`, `API_PORT`, `ADMIN_API_KEY`, `SERVICE_KEY_BOTMANAGER/UPLOADER/VPN`),
  `botmanager/.env.example` (`CENTRIFUGO_*` ×۴, `POSTGRES_PASSWORD`), `community-service`
  (`BOTPAY_URL`, `BOTPAY_API_KEY`, `FRAUD_ENGINE_URL` ×۲ تکراری, `MONGO_URI` تزریقی‌ولی‌بی‌مصرف)،
  `license-service/.env.example` (`APP_ENV`)، `apimanager/.env.example` (`CENTRIFUGO_URL`,
  `CENTRIFUGO_TOKEN`). چند فایل هم بلوکِ `NATS_URL/USERNAME/PASSWORD` تکراری داشتند
  (`botmanager/.env`, `member-bot/.env`) — یکی‌سازی شد.
- **`.env`/`.env.example` ریشه‌ی پروژه بازطراحی شد** — از یک «master env» با ~۵۰ متغیرِ
  legacy (که فقط ۶ تا از آن‌ها واقعاً توسط `docker-compose.yml` ریشه با `${VAR}` جایگزین
  می‌شوند؛ بقیه از وقتی هر سرویس `.env` مخصوصِ خودش را گرفت، بلااستفاده ماندند) به همان الگویی
  که `deploy/.env.example` از قبل در ۲۰۲۶-۰۷-۰۶ داشت.
- **۶ سرویس `.env.example` جدید گرفتند**: `ads-bot`, `community-service`, `dbmigrate`,
  `fraud-engine`, `revenue-service`, `webhook-gateway`.
- **باگِ واقعیِ کشف‌شده در ادامه‌ی همین کار (نه فقط env)**: `botpay` و `ads-bot` مقدارِ
  `LOCAL_BOT_API` را اصلاً از Config نمی‌خواندند — یک IP قدیمی (`141.95.210.17:8081`) مستقیم
  در کدِ `tele.NewBot(...)` هاردکد شده بود، یعنی هرچه در `.env` می‌گذاشتی نادیده گرفته می‌شد.
  رفع: فیلدِ `LocalBotAPI string \`mapstructure:"LOCAL_BOT_API"\`` به Config هر دو اضافه شد و
  همان الگویی که botmanager/uploader-bot/member-bot/admanager-bot/log-collector از قبل داشتند
  (`if cfg.LocalBotAPI != "" { settings.URL = cfg.LocalBotAPI }`) پیاده شد. IP سرور هم در همه‌ی
  ۱۰ سرویسِ رباتی (`.env` واقعی) به آدرسِ جدید (`65.109.221.21:8081`) به‌روزرسانی شد.
- **`run.sh` تکمیل شد**: دو سرویسِ مرکزیِ همیشه‌روشن که در local test runner جا افتاده بودند
  اضافه شدند — `log-collector` (زودتر از همه، تا لاگِ startup بقیه هم جمع شود) و
  `webhook-gateway` (کنارِ سرویس‌های پشتیبان). هر دو build شدند و سالم‌اند.
- **کشفِ جانبی (مستندسازی‌نشده، رفع نشد)**: `deploy/docker-compose.yml` — `deploy/NEEDS.md`
  ادعا می‌کند در ۲۰۲۶-۰۷-۰۶ همه‌ی بلاک‌های سرویسِ برنامه (botmanager, apimanager, ...) از حالتِ
  کامنت خارج و سیم‌کشی شدند؛ فایلِ فعلی روی دیسک این ادعا را تأیید نمی‌کند — anchor
  `x-build` و همه‌ی بلاک‌های برنامه هنوز کامنت‌اند. یا این تغییر بعداً revert شده یا هیچ‌وقت
  commit نشده. جدا از env-var scope همین کار است؛ نیاز به بررسیِ دستیِ بعدی دارد.

## [2026-07-19] — رفعِ رگرسیونِ LockMode + اتصالِ واقعیِ ربات‌های رایگان به ads-bot

بررسیِ مکالمه درباره‌ی «چطور ربات رایگان به سیستمِ تبلیغات وصل می‌شود» یک رگرسیونِ واقعی از
همین سشن را کشف کرد: وقتی Postgres کاملاً از uploader-bot حذف شد (رجوع ورودیِ migration
پایین‌تر)، تابعِ `onMyChatMember` (تشخیصِ «ربات در کانالِ خریدار ادمین شد» برای شروعِ
enforcement قفلِ اجاره‌ای) از کار افتاد — چون تنها منبعِ `LockMode` همان کوئریِ Postgریِ حذف‌شده
بود (`bot_instances.lock_mode`)، و `IsRentedLock()` بدونِ آن همیشه `false` برمی‌گشت.

**رفع + طراحیِ جدید (به‌جای برگردوندنِ Postgres):** به‌جای این‌که ربات از botmanager/Postgres
بپرسد «LockMode من چیه؟»، معماری برعکس شد — **ads-bot** (که خودش مالکِ واقعیِ
`FreeBotSlot`/`LockRentalCampaign` است) پاسخ می‌دهد. هر ربات رایگان (uploader-bot/vpn-bot/
archive-bot — **نه member-bot**، که زیرساختِ داخلی است و هیچ‌وقت به مشتری داده نمی‌شود) موقعِ
start و هر ۵ دقیقه یک‌بار روی NATS می‌پرسد «الان به کمپینی وصل‌ام؟»، دقیقاً مثلِ الگویِ
`licenseclient.RunLicenseLoop` موجود.

- **`shared-core/protocol/subjects.go`**: subject جدید `ads.bot_status_check` +
  `BotStatusRequest`/`BotStatusResponse` (شاملِ `CampaignID` برای attribution دقیقِ گزارش‌های
  fraud-engine).
- **`shared-core/memberclient/rental_status.go`** (فایل جدید): `Client.CheckBotStatus`،
  `RentalStatus` (نگه‌دارنده‌ی thread-safe وضعیت)، `RunStatusLoop` (چک فوری + هر ۵ دقیقه، fail-open
  — خطای شبکه یعنی آخرین وضعیتِ شناخته‌شده حفظ می‌شود، نه ریست به false).
- **`ads-bot`**: `Handler.GetBotStatus` (کوئریِ `FreeBotSlot`→`LockRentalCampaign.IsActive()`) +
  NATS responder جدید در `cmd/main.go`.
- **`shared/pkg/joinevents/` (پکیجِ جدید)**: منطقِ `member-bot/internal/events/publisher.go`
  (انتشارِ `membership.joined`/`membership.left`/`community.activity.updated` از رویِ Telegram
  chat-member update ها) به این‌جا منتقل شد تا هر ربات دیگری هم بتواند بدونِ کپی همین رفتار را
  داشته باشد — با یک `Gate func() bool` جدید (member-bot همیشه فعال، رباتِ رایگان فقط وقتی
  `RentalStatus.IsInCampaign()`). `member-bot/internal/events` حذف شد (جایگزینش این پکیج است).
- **uploader-bot/vpn-bot/archive-bot**: هرکدام حالا `RentalStatus`+`JoinPublisher` دارند؛
  `onMyChatMember` (تأییدِ ادمین‌شدن به ads-bot) از `RentalStatus.IsInCampaign()` استفاده می‌کند
  (نه دیگر `Eng.InstanceInfo`)؛ عضویتِ واقعیِ کانالِ خریدار (وقتی در کمپین‌اند) مستقیم به
  `membership.joined`/`left` منتشر می‌شود — این همان حلقه‌ای بود که قبلاً اصلاً وصل نبود (رجوع
  تحلیلِ قبلی: نه قفلِ رایگان به member-bot وصل بود، نه چیزِ دیگری join واقعی را تشخیص می‌داد).
- **پاکسازیِ کدِ کاملاً مرده**: `shared-core/engine`: `InstanceInfo`/`IsFreeLock`/`IsRentedLock`/
  `loadInstanceInfo`/`PostgresDSN`/فیلدِ `DB` کاملاً حذف شدند (تنها مصرف‌کننده‌شان همان
  `onMyChatMember` بود که حالا از مکانیزمِ جدید استفاده می‌کند). `admanager-bot` هم همین
  `engine.Config.PostgresDSN` را پاس می‌داد بدونِ این‌که هیچ‌جا از `InstanceInfo` استفاده کند —
  همان‌جا هم پاک شد (`POSTGRES_DSN` از `.env`/`.env.example`ش حذف شد). `agentmanager/botenv.env`
  و `agentmanager/botenv/uploader.env` هم از `POSTGRES_DSN`ی که دیگر هیچ‌کدام از ۴ ربات محصول
  نمی‌خوانند پاک شدند.
- **تأیید:** `go build/vet/test` سبز روی هر ۸ ماژولِ تغییریافته + `test-all.sh` کاملِ workspace،
  `gofmt` تمیز.

## [2026-07-17] — vpn-bot/archive-bot/member-bot: رفعِ نبودِ جداسازیِ tenant (instance_id)

پیگیریِ سؤالِ «الان هر رباتی که می‌سازیم یک دیتابیس دارد؟» نشان داد جوابِ واقعی نه بود: بعد از
migration کاملِ Postgres→MongoDB (رجوع ورودی‌های پایین‌تر همین فایل)، هر سه ربات یک دیتابیسِ
Mongo مشترک به‌ازای نوع‌شان دارند (`MONGO_DB=vpn_bot`/`archive_bot`/`member_bot`، مقدارِ ثابت در
`agentmanager/botenv/<type>.env`) ولی store‌ای که در همان migration نوشته شد **هیچ فیلدِ
instance_id ای نداشت** — یعنی همه‌ی instanceهای یک نوع ربات دیتای هم را می‌دیدند/می‌شکستند. این
باگِ واقعی و از قبل هم موجود بود (همان مشکل زیرِ Postgres هم بود، DSN همیشه یک مقدارِ ثابت به‌ازای
نوعِ سرویس بود)، فقط تا حالا کشف نشده بود.

تصمیمِ معماریِ نهایی (به انتخابِ صریحِ کاربر): **یک دیتابیسِ Mongo مشترک به‌ازای هر نوع ربات** (برای
بهترین عملکرد، نه یک دیتابیسِ جدا به‌ازای هر instance) + **جداسازیِ منطقی با فیلدِ instance_id** روی
هر سند و هر کوئری — دقیقاً همان الگویی که `uploader-bot` با `shared-core/docstore`
(`Base.baseFilter()`/`DocBase.InstanceID`) از قبل درست پیاده می‌کرد. `docstore` مستقیماً reuse
نشد چون `ports.DocumentStore`/`ports.Collection` هیچ `FindOneAndUpdate`/upsert/matched-count ای
expose نمی‌کند و این migration دقیقاً برای همین این‌ها را با درایورِ خامِ Mongo بازنویسی کرده بود
(`UpsertUser`، `ClaimPendingPayment`، `FindOrCreateCategory`) — به‌جایش همان الگوی فیلترکردن در
همان storeهای محلی تکرار شد.

- **`instanceID` بدونِ env جدید**: از `BOT_TOKEN` مشتق می‌شود (`fmt.Sprintf("bot_%d", botID)`،
  همان مقداری که `webhook.BotIDFromToken(cfg.BotToken)` از قبل در هر سه `main.go` محاسبه می‌کرد —
  فقط جابه‌جا شد تا قبل از ساختِ store اجرا شود، دقیقاً همان مکانیزمی که
  `shared-core/engine.go` برای uploader-bot استفاده می‌کند).
- **هر سه `Store`**: فیلدِ `instanceID string` اضافه شد؛ متدِ کمکیِ `scoped(extra bson.M) bson.M`
  (معادلِ `docstore.Base.baseFilter()`) که `instance_id` را به هر فیلتر اضافه می‌کند — همه‌ی
  متدهای خواندن/نوشتن از این عبور می‌کنند. هر `*Doc` هم فیلدِ `InstanceID` گرفت.
- **ایندکس‌های یکتا compound شدند** (`instance_id` کلیدِ پیشرو): `vpn-bot`
  (`users`, `discount_codes`, `payments` partial-unique)، `member-bot` (`owners`, `locks`)،
  `archive-bot` (`users`, `categories`).
- **دو باگِ واقعیِ cross-tenant که این migration کشف/رفع کرد:**
  - **archive-bot `Search`**: بدونِ `instance_id` در فیلترِ candidate-fetch، جستجوی یک مشتری
    فایل‌های آپلودشده در instanceِ کاملاً متفاوتِ archive-bot را هم برمی‌گرداند — نشتِ دیتای
    واقعی بینِ دو مشتری، نه فقط یک نقصِ correctness.
  - **member-bot `ClearBotMemberships`**: قبلاً `UpdateMany(bson.M{}, ...)` بدونِ هیچ فیلتری بود —
    sync دوره‌ایِ dispatcherِ یک instance می‌توانست memberships همه‌ی instanceهای دیگرِ
    member-bot را هم پاک کند.
- **تست:** هر سه `store_integration_test.go` بازنویسی شدند — `testStore` حالا `instanceID`
  می‌گیرد، و `testDB` جدید اضافه شد تا چند `Store` با `instanceID`های متفاوت روی **همان یک
  دیتابیس** ساخته شوند (دقیقاً شبیه‌سازیِ رفتارِ واقعیِ production). تست‌های جدیدِ ایزولیشن در
  هر سه: `TestInstanceIsolation_*` — شاملِ تأییدِ صریح که `archive-bot.Search` هرگز فایلِ
  instanceِ دیگر را برنمی‌گرداند حتی با عنوانِ کاملاً یکسان، و که
  `member-bot.ClearBotMemberships` رویِ instanceِ دیگر اثر نمی‌گذارد. همه با Mongo واقعی سبز
  (`go test -tags=integration ./internal/store/...`).
- **دیتابیس‌های dev**: `vpn_bot`/`archive_bot`/`member_bot` (که خالی بودند — بدون دیتای واقعی)
  drop شدند تا `EnsureIndexes()` ایندکس‌های compound را از صفر و تمیز بسازد.

## [2026-07-17] — Cross-cutting: جمع‌بندیِ deployment migrationِ ۴فازی به MongoDB

بعد از تکمیل هر ۴ فاز (uploader-bot/vpn-bot/member-bot/archive-bot)، سیم‌کشیِ deployment که
مشترک بین همه بود در یک مرحله جمع شد:

- **`dbmigrate/migrations/{vpn-bot,archive-bot,member-bot}/RETIRED.md`** (فایل‌های جدید) —
  `0001_baseline.sql` هرکدام تاریخچه ماند (حذف نشد)، ولی این‌ها دیگر «فعال» نیستند؛ منبعِ
  حقیقتِ ایندکس‌های فعلی `EnsureIndexes()` خودِ هر سرویس است.
- **`dbmigrate/internal/migrate/registry.go`**: این سه سرویس از `Registry` حذف شدند —
  `dbmigrate up -service=vpn-bot` و مشابه حالا با خطای «سرویس ناشناخته» fail می‌کند (fail-fast
  عمدی، به‌جای تلاشِ بی‌اثر روی یک DB که دیگر لازم نیست وجود داشته باشد). `dbmigrate/README.md`
  هم به‌روزرسانی شد (۱۱ سرویس Postgres قبلی → ۸ سرویس).
- **`deploy/migrations/000_create_product_databases.sql`**: `CREATE DATABASE vpn_bot/archive_bot/
  member_bot` و بلوکِ `CREATE EXTENSION pg_trgm` (که فقط برای archive-bot لازم بود) حذف شدند —
  فقط `uploader_bot` (همچنان best-effort/optional، رجوع Phase 1) و `source_svc` باقی ماندند.
  `deploy/init-db.sql` عمداً دست‌نخورده ماند — از قبل با یک هدر «منسوخ، دیگر اجرا نمی‌شود»
  علامت‌گذاری شده بود؛ ویرایش‌های واقعی از ۲۰۲۶-۰۷-۰۶ به بعد فقط در `000_create_product_databases.sql`
  انجام می‌شوند.

## [2026-07-17] — archive-bot: migration کامل Postgres → MongoDB + بازطراحیِ جستجوی فازی (Phase 4/4)

آخرین فاز از migration چهارگانه؛ سخت‌ترین بخش نه خودِ CRUD بلکه جایگزینیِ `pg_trgm.similarity()`
بود که در MongoDB self-hosted هیچ معادلی ندارد (بدون Atlas Search).

- **`internal/models/models.go`**: بازنویسیِ کامل به value objectهای خالص. **`Setting` حذف شد**
  (grep کامل تأیید کرد هیچ‌جا خوانده/نوشته نمی‌شد — دقیقاً همان الگویی که در member-bot هم دیده
  شد). **`File.Category` (نسخه‌ی embed‌شده از `Preload("Category")` قبلی) هم حذف شد** — در کل
  کدِ handler فقط `CategoryID` خوانده می‌شد، هرگز خودِ `Category` تودرتو.
- **`internal/search/search.go`**: `Normalize()` (فولدینگِ عربی→فارسی، حذف اعراب/ZWNJ) عیناً
  حفظ شد. دو تابعِ جدید اضافه شد: `Trigrams(s string) []string` — دقیقاً همان الگوریتمِ خودِ
  pg_trgm (padding با یک space هر طرف + پنجره‌ی لغزنده‌ی ۳کاراکتری)، و `Similarity(a, b []string)
  float64` — همان فرمولِ واقعیِ `pg_trgm.similarity()`: Jaccard = |A∩B|/|A∪B|.
- **`internal/store/store.go`**: بازنویسیِ کامل روی درایور رسمی Mongo. تصمیمِ کلیدیِ طراحیِ
  جستجو: هر `File` هنگام نوشتن یک فیلدِ `ngrams []string` (trigramهای متنِ نرمال‌شده‌ی
  title+tags+description) ذخیره می‌کند؛ روی همین فیلد یک ایندکسِ چندکلیدیِ معمولی ساخته شد.
  `Store.Search(query, limit)`: کوئری هم trigram می‌شود → با `$in` روی `ngrams` یک candidate set
  ارزانِ index-assisted گرفته می‌شود → در Go امتیازِ Jaccardِ دقیق محاسبه، فیلترِ `>0.1` (همان
  آستانه‌ی قبلی) اعمال، نزولی مرتب و به `limit` محدود می‌شود. `RelatedFiles` هم migrate شد ولی
  مثل قبل در هیچ handler ای سیم‌کشی نشد (بلااستفاده‌ی شناخته‌شده، نه یک regression).
  **بدون soft-delete**: مدلِ قبلی از `gorm.DeletedAt` استفاده می‌کرد ولی هیچ مسیرِ «بازیابی» در
  کل کدبیس نبود — پس `DeleteFile` حالا حذفِ واقعی (hard delete) است؛ دقیقاً همان رفتارِ
  قابل‌مشاهده‌ی قبلی، بدون پیچیدگیِ اضافه‌ی نگه‌داشتنِ یک فیلدِ بلااستفاده.
  `FindOrCreateCategory` هم اتمیک شد (`findOneAndUpdate`+`$setOnInsert`) — قبلاً `FirstOrCreate`ی
  GORM بود که race داشت (دو کلیکِ هم‌زمان روی «دسته جدید» می‌توانست دو رکورد با یک نام بسازد).
- **`internal/tgbot/handler.go`/`search_handler.go`**: پارامترِ `db ports.DB` از `NewHandler`
  حذف شد (جستجو دیگر مستقیم `h.db.Conn()` را بایپس نمی‌کند)؛ `doSearch`/`onInlineQuery` حالا
  `h.store.Search(ctx, query, limit)` صدا می‌زنند — فراخوانیِ صریحِ `search.Normalize(query)` هم
  حذف شد چون `Store.Search` خودش نرمال‌سازی می‌کند.
- **تست:**
  - `internal/search/search_test.go` (بدون نیاز به Mongo) — `Normalize`/`Trigrams`/`Similarity`
    را با ورودی‌های شناخته‌شده تأیید می‌کند.
  - `internal/store/store_integration_test.go` (تگ `integration`، Mongo واقعی) — **۵ تست سبز**:
    رتبه‌بندیِ صحیحِ جستجو با فیلترِ آستانه (فایلِ کاملاً بی‌ربط حذف می‌شود)، بی‌تفاوتیِ جستجو به
    اعراب/نیم‌فاصله (query و doc نرمال‌سازیِ یکسان می‌گیرند)، عدمِ ساختِ دستهٔ تکراری با
    ۱۰ فراخوانیِ هم‌زمانِ `FindOrCreateCategory`، یکتاییِ واقعیِ `categories.name` در سطحِ
    دیتابیس، و رفتارِ `UpsertUser` (ID یکسان و بروزرسانیِ صحیحِ فیلدها در فراخوانیِ دوم).
- **Config:** `.env`/`.env.example`/`agentmanager/botenv/archive-bot.env` — `POSTGRES_DSN` و
  بلوکِ `CREATE EXTENSION pg_trgm`/`CREATE INDEX ... GIN` از `main.go` حذف شدند؛
  `MONGO_URI`/`MONGO_DB=archive_bot` جایگزین شد.
- **تأیید:** `go build/vet/test` سبز (شاملِ integration tests روی Mongo واقعیِ لوکال)، صفر
  ارجاعِ gorm/postgres باقی‌مانده در کدِ واقعی (فقط در کامنت‌های تاریخی).

## [2026-07-17] — member-bot: migration کامل Postgres → MongoDB (Phase 3/4)

- **`internal/models/models.go`**: بازنویسیِ کامل به value objectهای خالص. **`MemberVerification`
  و `Setting` حذف شدند** — با grep کاملِ کدبیس تأیید شد هیچ‌کدام هرگز خوانده/نوشته نمی‌شدند
  (جدول‌هایی که فقط migrate می‌شدند و همیشه خالی می‌ماندند).
- **`internal/store/store.go`**: بازنویسیِ کامل روی درایور رسمی Mongo. تصمیمِ کلیدیِ طراحی:
  `CheckBot.Memberships` (رابطه‌ی قبلاً join-table با composite PK) به‌صورت **آرایه‌ی embed‌شده
  داخل خودِ سندِ CheckBot** درآمد — چون همیشه با هم خوانده می‌شدند (`Preload`) و هرگز join
  معکوس نداشتند. `AddBotMembership` حالا: تلاش برای آپدیتِ عنصرِ موجود با positional operator
  (`memberships.$.last_verified`)، و اگر عضویتی برای آن کانال نبود (`matchedCount==0`)، عنصر
  تازه با `$push` اضافه می‌شود — معادلِ دقیقِ رفتار قبلیِ `FirstOrCreate`+`Assign` روی composite PK.
- **رفعِ کدِ تکراری real:** `dispatcher.go` `loadBots` یک کوئریِ خامِ Postgres جدا از store داشت
  (`d.db.Conn().Preload(...)`) — با حذفِ کاملِ `ports.DB` از `Dispatcher`، حالا از
  `st.FindActiveBots(ctx)` استفاده می‌کند (تکرارِ منطق حذف شد، نه فقط migrate).
- **بدونِ تغییرِ رفتار (عمدی):** مسیرِ پرداختِ محلیِ `Payment`→`Owner.Balance` که در Postgres
  هم orphan بود (`CreatePayment` هرگز از `tgbot` صدا زده نمی‌شد، `ApprovePayment` هرگز
  `UpdateBalance` را صدا نمی‌زد) عیناً با همان وضعیت منتقل شد — فعال‌سازیِ این جریان یک
  فیچرِ جدید است، نه کارِ migration.
- **مسیر پرترافیک چک عضویت دست‌نخورده:** `internal/lock/server.go`/`internal/worker/*`/
  `internal/memberresponder` صددرصد روی Redis هستند (Stream+SETNX+cache) — تأییدشده در تحقیقِ
  این فاز، بدون هیچ وابستگی به Postgres/Mongo؛ این migration هیچ ریسکی برای hot-path نداشت.
- **دو باگِ واقعی که فقط با تستِ زنده کشف شدند** (رجوع تست‌ها پایین‌تر):
  - partial unique index با `$ne` در MongoDB پذیرفته نمی‌شود (فقط `$eq/$gt/$gte/$lt/$lte/
    $exists/$type`) — همان مشکلی که در vpn-bot (Phase 2) هم دیده شد؛ اینجا لازم نبود چون
    member-bot ایندکسِ partial ندارد، ولی الگو مستند شد.
  - `CreateCheckBot` فیلدِ `memberships` را `nil` می‌گذاشت که BSON آن را `null` می‌سازد (نه
    آرایه‌ی خالی) — اولین `$push` با خطای «must be an array but is of type null» شکست
    می‌خورد. رفع: مقداردهیِ صریح `Memberships: []membershipDoc{}` در insert اول.
- **تست:** `internal/store/store_integration_test.go` (تگ `integration`، Mongo واقعی) —
  **۴ تست سبز**: عدمِ تکراریِ عضویت با دو فراخوانیِ همان (bot,channel)، چند کانالِ متفاوت روی
  یک bot درست append می‌شوند، `ClearBotMemberships` همه‌ی آرایه‌ها را خالی می‌کند، و یکتاییِ
  `owners.telegram_id` واقعاً اعمال می‌شود (duplicate-key روی تلاش دوم).
- **Config:** `.env`/`.env.example`/`agentmanager/botenv/member-bot.env` (فایل جدید) —
  `MASTER_DSN` حذف، `MONGO_URI`/`MONGO_DB=member_bot` جایگزین شد.
- **تأیید:** `go build/vet/test` سبز (شامل رفعِ `balancer_test.go` که به شکلِ قدیمیِ
  `models.Base{ID:...}` وابسته بود)، gofmt تمیز، صفر ارجاعِ gorm/postgres باقی‌مانده.

## [2026-07-17] — vpn-bot: migration کامل Postgres → MongoDB (Phase 2/4)

هسته‌ی داده‌ی vpn-bot (کاربران، پنل‌ها، پلن‌ها، اشتراک‌ها، کدهای تخفیف، پرداخت‌ها) از Postgres/GORM
به MongoDB منتقل شد؛ دیتابیس اختصاصیِ per-instance حفظ شد (نه multi-tenant).

- **`internal/models/models.go`**: بازنویسیِ کامل به value objectهای خالص (بدون gorm tag). شکل/نام
  فیلدها با نسخه‌ی قبلی یکی نگه داشته شد تا لایه‌ی `tgbot`/`scheduler` بدون تغییر کامپایل شود.
  `Setting` (که در کل کد بلااستفاده بود) حذف شد.
- **`internal/store/store.go`**: بازنویسیِ کامل روی درایور رسمی Mongo (نه wrapper محدودِ
  `ports.Collection` — چون به `FindOneAndUpdate`/upsert/matched-count نیاز داشت که آن wrapper
  expose نمی‌کند). هر عملیاتِ اتمیکِ قبلی معادلِ دقیق گرفت:
  - `DeductBalanceIfEnough` → `findOneAndUpdate({_id, balance:{$gte:amount}}, {$inc})`.
  - `ClaimOnlinePayment` → `insertOne` + catch خطای duplicate-key (۱۱۰۰۰) به‌جای `ON CONFLICT`.
  - `UpsertUser` → `findOneAndUpdate` با `$setOnInsert` برای فیلدهای پیش‌فرض (balance/is_blocked)
    تا هیچ‌وقت overwrite نشوند؛ سندِ نهایی در `*u` پر می‌شود (دقیقاً رفتار قبلیِ
    `FirstOrCreate` که `getOrCreate` در `helpers.go` بدون خواندنِ دوباره به آن متکی بود).
  - `Subscription.User` (فقط `TelegramID`ی که `scheduler.go` می‌خواند) denormalize شد به‌جای
    `$lookup` — تنها فیلدی از رابطه که در کل کد خوانده می‌شد.
- **باگ واقعی رفع شد:** `internal/tgbot/admin.go` `approvePayment` — چک `status!="pending"` و
  `UpdateBalance` در دو مرحله‌ی جدا بودند (بدون قفل)؛ دو کلیک هم‌زمان روی «تأیید» هر دو رد
  می‌شدند و موجودی **دو بار** اضافه می‌شد. رفع با متد جدید `ClaimPendingPayment`
  (findOneAndUpdate اتمیک روی `status:"pending"→"confirmed"`).
- **ایندکس‌ها:** `EnsureIndexes()` در startup — `users.telegram_id` و `discount_codes.code`
  یکتا، و یکتاییِ partial `(gateway, ref_code)` روی `payments` (معادل partial unique index قبلی).
  **نکته‌ی فنی مهم که در تستِ زنده کشف شد:** MongoDB در partial index فقط زیرمجموعه‌ای از
  عملگرها (`$eq/$gt/$gte/$lt/$lte/$exists/$type`) را می‌پذیرد — `$ne` رد می‌شود. به‌جای
  `ref_code:{$ne:""}` از `ref_code:{$gt:""}` استفاده شد (معادل، چون رشته‌ی خالی کوچک‌ترین
  مقدارِ لغوی است).
- **تست:** `internal/store/store_integration_test.go` (تگ build جدا `integration`، نیازمند
  Mongo واقعی) — **۴ تست با DB واقعی اجرا و سبز شدند**: عدم double-spend همزمان
  (`DeductBalanceIfEnough`)، عدم فعال‌سازی دوباره با کلیک تکراریِ پرداخت آنلاین
  (`ClaimOnlinePayment`)، عدم duplicate-credit با دو کلیکِ هم‌زمانِ تأیید ادمین
  (`ClaimPendingPayment`)، و وجودِ ایندکسِ یکتا. اجرا: `go test -tags=integration ./internal/store/...`.
- **Config:** `vpn-bot/.env`+`.env.example`+`agentmanager/botenv/vpn-bot.env` — `MASTER_DSN` حذف،
  `MONGO_URI`/`MONGO_DB=vpn_bot` جایگزین شد.
- **تأیید:** `go build/vet/test` سبز، gofmt تمیز، هیچ ارجاع gorm/postgres باقی نماند (grep تأیید شد).
  اجرای کاملِ زنده‌ی startup تا انتها تست نشد (نیازمند پنل VPN واقعی؛ `panel.Login` بدون
  context timeout روی URL تستی hang می‌کند — این یک نقصِ از‌پیش‌موجود و خارج از scope این
  migration است، نه چیزی که این تغییر ایجاد کرده باشد).

## [2026-07-17] — ربات‌های کاربر: قطع دسترسی Postgres (Phase 1/4 — uploader-bot)

هدف کلی (۴ فاز): uploader-bot، vpn-bot، archive-bot، member-bot هیچ‌کدام نباید Postgres داشته
باشند — همه‌ی داده روی MongoDB. این ورودی فقط **فاز ۱** (uploader-bot، کم‌ریسک‌ترین) است.

- **`shared-core/engine/engine.go`**: اتصال Postgres در `New()` اختیاری شد. قبلاً حتی برای
  uploader-bot (که صددرصد داده‌اش روی MongoDB است) اتصال Postgres fail-fatal اجباری بود، درحالی‌که
  تنها مصرفش یک کوئری best-effort (`loadInstanceInfo` روی `bot_instances.plan_id/lock_mode`) بود
  که خودش از قبل fail-open بود (`log.Warn` نه fatal). حالا `PostgresDSN` خالی یعنی اصلاً تلاشی
  برای اتصال نمی‌شود (`db ports.DB` = nil)، نه فقط نادیده‌گرفتنِ خطای اتصال.
- **`uploader-bot/cmd/bot/main.go`**: فیلد `PostgresDSN`/`POSTGRES_DSN` کاملاً حذف شد — دیگر جایی
  برای ست‌کردنش نیست.
- **`uploader-bot/.env` و `.env.example`**: خط `POSTGRES_DSN` حذف شد.
- **تأیید:** `go build/vet/test` هم `shared-core` هم `uploader-bot` سبز؛ اجرای زنده
  (`go run cmd/bot/*.go`) خطای قبلی Postgres (`role "javad" does not exist`) کاملاً از بین رفت —
  اجرا تا مرحله‌ی اتصال MongoDB پیش رفت (که یک مسئله‌ی جدا و محیطی است: DNS داخلی داکر برای
  hostname `mongo`، نه چیزی که این تغییر ایجاد کرده باشد).
- **باقی‌مانده:** فاز ۲ (vpn-bot)، فاز ۳ (member-bot)، فاز ۴ (archive-bot + بازطراحی جستجوی فازی) —
  این سه واقعاً داده‌ی کسب‌وکاری روی Postgres دارند؛ migration کامل در حال انجام.

## [2026-07-17] — license-service: لایسنسِ تستیِ سراسری برای اجرای دستیِ ربات‌ها

نیاز: اجرای دستیِ ربات‌های محصول (uploader/vpn/archive/member) برای تست، بدون طیِ چرخه‌ی
واقعیِ خرید → `license.issue` برای هر instance.

- **`license-service/internal/licensing/service.go`**: فیلد `testSecret` + helperِ خالصِ
  `isTestToken(secret, token)` (مقایسه با `crypto/subtle.ConstantTimeCompare`، fail-closed —
  secret یا token خالی همیشه false). `Verify()` پیش از هر کوئریِ DB این را چک می‌کند؛ اگر
  token دقیقاً برابرِ secret باشد، برای **هر BotID دلخواه** (بدون نیاز به رکورد License واقعی)
  `valid=true, status=active` برمی‌گرداند و یک `Warn` با bot_id/server_id لاگ می‌کند (هر مصرف،
  چون این یک دورزدنِ کنترل‌شده‌ی حفاظتِ ضدِ کپی است، نه لایسنسِ واقعی).
- **`license-service/cmd/main.go`**: `Config.TestLicenseSecret` (`TEST_LICENSE_SECRET`، پیش‌فرض
  خالی = غیرفعال)؛ اگر تنظیم شود، در startup یک `Warn` صریح چاپ می‌شود (ریسک را در لاگِ هر بار
  بالاآمدنِ سرویس هم نشان می‌دهد، نه فقط per-request).
- **سمتِ کلاینت بدون تغییر:** `shared-core/licenseclient.RequireValid`/`RunLicenseLoop` فقط
  `Valid`/`Status` سرور را می‌خوانند — کافی است روی هر container دستیِ تست، env
  `LICENSE_TOKEN=<TEST_LICENSE_SECRET>` گذاشته شود؛ هیچ ربات محصولی تغییر نکرد.
- **تست:** `license-service/internal/licensing/service_test.go` (`isTestToken`، بدون DB).
- **تأیید:** `go build/vet/test` license-service سبز، `scripts/test-all.sh` سبز، gofmt تمیز.
- **⚠️ نکته‌ی امنیتی:** فقط در محیط تست/dev فعال شود؛ هرگز در deploymentی که مشتریِ واقعی هم
  دارد، مگر با پذیرشِ آگاهانه‌ی ریسک. مثل بقیه‌ی secretهای پلتفرم rotate شود.

## [2026-07-15] — رفع 401 ورود Telegram در apimanager

- قرارداد backend و React با payload رسمی Telegram Login Widget هم‌سو شد: فیلد امضاشده `id`
  پذیرفته و verify می‌شود؛ `telegram_id` برای سازگاری درخواست‌های قدیمی همچنان پشتیبانی می‌شود.
- فرم dev اکنون `id` را امضا/ارسال، `BOT_TOKEN` ورودی را trim و کلیدهای data-check-string را
  با ترتیب ASCII قطعی مرتب می‌کند.
- timestamp بیش از ۲۴ ساعت قدیمی یا بیشتر از ۵ دقیقه در آینده رد می‌شود؛ مسیر آینده که قبلاً به
  دلیل `time.Since` منفی پذیرفته می‌شد بسته شد.
- پیام signature mismatch روشن می‌کند که توکن dev باید دقیقاً همان `BOT_TOKEN` سرویس apimanager
  باشد، نه توکن botmanager یا یک ربات دیگر؛ راهنمای فارسی/انگلیسی فرم نیز اصلاح شد.
- تست‌های backend امضای رسمی `id`، فرمت legacy، رد توکن متفاوت و مرزهای زمانی را پوشش می‌دهند.
- اعتبارسنجی: `go test ./...` در apimanager و `npm run lint`/`npm run build` در apimanager/web موفق.

## [2026-07-15] — همگام‌سازی خودکار پروفایل همه ربات‌ها در production

- helper مشترک `shared/pkg/botprofile` اضافه شد؛ فقط وقتی `APP_ENV=production|prod` باشد اجرا می‌شود.
- در startup هشت ربات اصلی `botmanager`، `botpay`، `ads-bot`، `uploader-bot`، `vpn-bot`،
  `archive-bot`، `member-bot` و `admanager-bot` نام نمایشی از `BOT_SERVICE_NAME` گرفته می‌شود.
- نام پیش‌فرض و localizationهای فارسی/انگلیسی همگام، description و short-description/bio پاک و
  عکس پروفایل فعلی با `removeMyProfilePhoto` حذف می‌شود. عملیات idempotent است و در هر restart
  production قابل تکرار است.
- dev/test/staging به‌صورت fail-closed هیچ profile mutationی انجام نمی‌دهند؛ خطای شبکه/API در
  production ثبت می‌شود ولی startup ربات را متوقف نمی‌کند.
- provisioning از هر دو مسیر botmanager و apimanager اکنون `APP_ENV=production` و
  `BOT_SERVICE_NAME=<template name>` را به containerهای ربات کاربر تزریق می‌کند.
- `.env.example` همه ربات‌ها با دو متغیر جدید به‌روز و برای ads-bot نمونه تنظیمات ساخته شد.
- تست `shared/pkg/botprofile` عدم mutation در development و ۱۰ فراخوانی مورد انتظار production
  را کنترل می‌کند؛ `go test ./...` در shared و هر هشت ربات اصلی و نیز apimanager/agentmanager سبز است.

## [2026-07-15] — botpay: یکپارچه‌سازی کاملِ تراکنش‌ها + دفترِ هش + ذخیره‌ی کاملِ TON

هر عملیاتِ تغییرِ موجودی حالا اتمیک و یکسان: یک `Transaction` می‌سازد، یک بلوکِ هش‌زنجیره‌ای
(`LedgerEntry`) append می‌کند، و کاربرِ متأثر را notify می‌کند.

- **دفترِ هش‌زنجیره‌ای زنده شد:** `RecordDeposit`/`RecordTransfer` قبلاً **dead code** بودند (هیچ‌جا
  صدا زده نمی‌شدند) → زنجیره خالی و «بررسی سلامت زنجیره» بی‌معنا بود. جایگزین: helperِ مشترکِ
  `Store.appendLedger` که **درونِ همان transactionِ عملیات** یک بلوک درج می‌کند. حالا
  `Deposit`/`Deduct`/`AddCredit`/`Transfer`/`CompleteWithdraw` هر کدام بلوکِ دفتر می‌سازند.
  invariant: `Σcredit − Σdebit == ton_balance + credit` (VerifyLedgerBalance اصلاح شد).
- **برداشت حالا Transaction می‌سازد:** `CompleteWithdraw` یک `TxWithdraw` (با هشِ on-chain و مقصد)
  + بلوکِ debit؛ `RejectWithdraw` یک `TxRefund`ِ اطلاع‌رسانی (آزادسازیِ hold، بدون بلوکِ دفتر چون
  موجودیِ realized تغییر نمی‌کند). قبلاً برداشت هیچ Transactionی در تاریخچه نداشت.
- **ذخیره‌ی کاملِ TON:** watcher حالا `utime`/`lt`/`fee`/`destination`/`comment` را از toncenter
  پارس می‌کند؛ ستون‌های جدید `tx_lt`/`tx_utime` (+ `fee`/`from`/`to`/`ref=comment` موجود) روی
  `Transaction` ذخیره می‌شوند. `store.Deposit` به ساختارِ `DepositRecord` تبدیل شد. migration:
  `dbmigrate/migrations/botpay/0003_transaction_ton_fields.sql` (AutoMigrate هم additive می‌سازد).
- **اعلانِ پرداخت:** `Pay` هم به کاربر اعلان می‌دهد (کلید `notify.payment`)، دقیقاً یک‌بار؛ برای این
  `store.Deduct` نیز مثل `AddCredit` یک flag `created` برمی‌گرداند تا روی retryِ idempotent اعلان
  دوباره نرود.
- **تست:** `chain_test.go` (تشخیصِ دستکاریِ هش، خالص/بدون DB، هم‌سو با سبکِ تست‌های موجود).
  `go build/vet/test` botpay و `scripts/test-all.sh` سبز. هسته‌ی رمزنگاری/consensus دست‌نخورده.

## [2026-07-15] — botpay: اعلان به کاربر هنگام برداشت/اعتبار

عملیاتِ تغییردهنده‌ی موجودی که ادمین یا سرویس انجام می‌داد فقط به اپراتور نتیجه می‌داد و به
**کاربرِ متأثر** اطلاع نمی‌داد (برخلاف واریز on-chain و انتقال که اطلاع می‌دادند).

- **متمرکز در `wallet.Service`** (که Notifier دارد): سه متد جدید که store را می‌پوشانند و اعلان
  best-effort می‌فرستند — `SettleWithdraw` (تأیید برداشت → کاربر مطلع)، `RejectWithdraw`
  (رد + بازگشت وجه → کاربر مطلع)، `Credit` (افزودن اعتبار → کاربر مطلع، **دقیقاً یک‌بار**).
- **`store.AddCredit`** حالا `(*Transaction, bool created, error)` برمی‌گرداند تا اعلانِ اعتبار
  فقط روی creditِ تازه فایر شود و روی retryِ idempotent نه. (تصمیمِ create/existing داخل همان
  تراکنشِ قفل‌دار است → race-free.) `store.GetWithdraw` (فقط‌خواندنی) برای خواندن مبلغ/کیف پول.
- **rewire:** پنل ادمین (`handlers_admin.go`) و `payresponder` (مسیر `pay.credit` سرویس‌ها) حالا
  از این متدهای سرویس می‌روند؛ پس هم تأیید/رد برداشت ادمین و هم creditهای سرویسی (پاداش/استرداد)
  به کاربر اعلان می‌دهند.
- i18n: کلیدهای جدید `notify.withdraw_done`/`notify.withdraw_rejected`/`notify.credit_added` (fa/en).
- هسته‌ی مالی دست‌نخورده؛ `go build/vet/test` و `scripts/test-all.sh` سبز.

## [2026-07-15] — بازطراحی کامل ربات تلگرام botpay (UX + پنل مدیریت)

رابط ربات botpay از نو ساخته شد؛ **هسته‌ی مالی دست‌نخورده** ماند (لجر دوطرفه
`store/ledger.go`، زنجیره‌ی هش `store/chain.go`، consensus `internal/consensus/`، و ریاضیِ
`internal/wallet/wallet.go`). فقط لایه‌ی رابط/orchestration و i18n بازنویسی و چند کوئری
فقط‌خواندنی اضافه شد. امضای عمومی (`tgbot.New/Register/SetBot`) تغییر نکرد؛ `cmd/main.go` دست‌نخورده.

### UX کاربر (inline-first)
- ناوبری edit-in-place به‌جای فقط reply-keyboard؛ کارت کیف پول با تفکیک قابل‌استفاده/اعتبار/بلوک‌شده،
  آدرس واریز و pay handle.
- واریز: دکمه‌های مبلغ پیش‌فرض (۱/۵/۱۰/۵۰) + مبلغ دلخواه + «هر مبلغ» + فاکتور با کد Comment + «بررسی واریز».
- برداشت و انتقال: **مرحله‌ی تأیید** با کارت خلاصه پیش از اجرا؛ انتقال با **آیدی عددی یا pay handle**.
- تاریخچه‌ی **صفحه‌بندی‌شده** (قبلی/بعدی)، منوی دستورات تلگرام (`/`) با `SetCommands`.

### پنل مدیریت (کامل‌تر)
- داشبورد: تعداد کیف پول، مجموع واریز/پرداخت، برداشت‌های منتظر، **کل موجودی پلتفرم** و **بلوک‌شده**.
- برداشت‌ها: فهرست صفحه‌بندی‌شده → جزئیات → تأیید/رد.
- افزودن اعتبار با مرحله‌ی تأیید (userID → مبلغ → تأیید).
- **جستجوی کاربر** (با آیدی/handle) → کارت کیف پول + تراکنش‌های اخیر (جدید).
- **بررسی سلامت زنجیره‌ی هش** با `store.VerifyChain` — تشخیص دستکاری دیتابیس (جدید).

### فایل‌ها
- بازنویسی: `internal/tgbot/{bot,keyboards,handlers_user,handlers_conv,handlers_admin,callbacks,lang,state}.go`
  + `render.go` (جدید)؛ `internal/i18n/{keys.go,locales/fa.json,locales/en.json}` کامل بازنویسی.
- جدید: `internal/store/admin_queries.go` (فقط‌خواندنی: `SumWalletBalances`/`SumFrozen`/`CountLedgerEntries`)،
  `internal/i18n/i18n_test.go` (گارد کامل‌بودن ترجمه + تقارن verb).
- اصلاح: `internal/wallet/wallet.go` — دو فراخوانی push (`KDepositConfirmed`/`KTransferReceived`) حالا
  رشته‌ی قالب‌بندی‌شده می‌فرستند (helper `fmtTONStr`) تا با قالب‌های `%s` جدید بخوانند (بدون تغییر منطق).
- **تأیید:** `go build`/`go vet`/`go test` سبز؛ ۱۰۷ کلید i18n کامل و متقارن fa/en؛ `scripts/test-all.sh` سبز.
  اجرای زنده نیازمند bot token و Postgres است (موکول، مثل بقیه‌ی E2E).

## [2026-07-15] — احراز اصالت envelope دستور deploy (Plan 001)

خروجی اجرای `plans/001-agentmanager-deploy-envelope-auth.md` (skill `improve`).

- **مشکل:** `agentmanager/internal/queue/worker.go` هر `DeployCommand` روی
  `deploy.<serverID>` را بدون هیچ چک اصالت/تازگی/replay اجرا می‌کرد؛ هر publisherِ NATS
  می‌توانست container دلخواه با env/network/rootfs/capability انتخابی اجرا کند یا یک deploy
  قدیمی را replay کند.
- **رفع (همان الگوی `source-service/internal/bus.authorize`):** چهار فیلد envelope
  (`service_id/service_key/issued_at/nonce`) به `DeployCommand` اضافه شد
  (`shared-core/protocol/subjects.go`). `agentmanager/internal/queue/authz.go` (جدید) یک
  `Verifier` با HMAC (`auth.ValidateServiceKey`) + پنجره‌ی تازگی ۵دقیقه/۳۰ثانیه + nonce
  store درون‌پردازه‌ای دارد؛ `worker.go` پیش از enqueue `Check` می‌کند و دستور نامعتبر/replay
  را بی‌صدا رد و log می‌کند. `agentmanager/cmd/main.go` حالا `SERVICE_HMAC_SECRET` را
  می‌خواند و در نبودش fail-closed (`log.Fatal`) می‌شود.
- **سمت فرستنده:** `shared-core/docker/docker.go` متد `NewSignedManager` + `Manager.Send`
  گرفت که هر دستور را امضا می‌کند. **همه‌ی** publisher های deploy از مسیر امضاشده رفتند:
  botmanager (`wizard.go` مسیر اصلی provision + `admin_svctest.go`)، apimanager
  (`handler.go` ×۲)، و `tools/e2e-provision`. (اگر فقط Manager امضا می‌شد، publish مستقیمِ
  wizard رد می‌شد و provisioning می‌شکست — STOP condition پلن که مدیریت شد.)
- **تأیید:** build/vet همه‌ی moduleهای متأثر سبز، ۸ تست جدید `authz_test.go`، `scripts/test-all.sh`
  سبز، gofmt تمیز. **باقی‌مانده (پلن جدا):** سخت‌سازی سمت سرور تا override های امتیازساز
  `DeploySettings` (CapAdd/writable rootfs) توسط `SecurityPolicy` سرور محدود شوند.
- **عملیاتی:** باید `SERVICE_HMAC_SECRET` در `agentmanager/.env` واقعی (gitignored) برابر
  botmanager/apimanager تنظیم شود وگرنه agentmanager بالا نمی‌آید.

## [2026-07-15] — هم‌سان‌سازی botmanager و apimanager

### Lifecycle و امنیت

- subjectهای بدون subscriber یعنی `instance.<action>` از مسیر کاربر botmanager حذف و همه عملیات start/stop/restart/delete روی قرارداد `deploy.{server_id}` یکسان شدند.
- `agentmanager` اجرای واقعی `MsgStart` را با managed-container guard دریافت کرد و commandهای API اکنون `ContainerID` را ارسال می‌کنند.
- IDOR endpointهای start/stop/restart/delete/logs بسته شد؛ همه پیش از عملیات مالکیت instance را بررسی می‌کنند.
- middleware جدید `UserState` وضعیت block و role را در هر درخواست از DB تازه می‌کند؛ login blocked رد و refresh بر اساس user/role جاری صادر می‌شود.
- botmanager نیز BotToken را مانند API با AES-GCM ذخیره می‌کند؛ migration تلگرام رکورد plaintext قدیمی را هنگام استفاده دوباره رمز می‌کند.

### Provisioning و مالی

- provisioning وب با مسیر تلگرام همسان شد: PlanID، LockMode، رویدادهای free/service creation، license token و envهای owner/plan/JWT/server اضافه شدند.
- allowlist امضاشده `license-service`، شناسه `apimanager` را نیز کنار `agentmanager` و `botmanager` می‌پذیرد و با تست مجاز/غیرمجاز پوشش داده شد.
- `Store.ActivateSubscription` تعویض اشتراک را در transaction انجام می‌دهد و از چند subscription فعال هم‌زمان جلوگیری می‌کند.
- خرید API در شکست activation refund می‌کند، `plan.upgraded` و audit می‌فرستد و Payment را فقط پس از activation موفق ثبت می‌کند.
- خرید تلگرام از attempt ID مستقل، activation اتمیک و Payment history مشترک استفاده می‌کند؛ retry و خرید مستقل از هم جدا هستند.

### انتقال قابلیت‌ها

- به apimanager اضافه شد: renew instance، redeem کد، audit user/admin، CRUD Source Worker و broadcast متنی فیلترشده.
- به وب اضافه شد: فرم promo در Plans، action تمدید و صفحه `/admin/operations` برای broadcast، worker و audit.
- به botmanager اضافه شد: نمایش payment/audit ادمین و migration انتخابی instance به سرور آنلاین دیگر.
- API migration و botmanager provisioning برای رکوردهای قدیمی plaintext مسیر upgrade سازگار دارند.

### مستندات و تست

- `prog/MANAGER_PARITY.md` به‌عنوان ماتریس و قرارداد مرجع اضافه و PROJECT/WEB/SECURITY/TESTING/README به‌روز شدند.
- تست‌های helper برای apimanager اضافه شد.
- `go test ./...` در shared-core، agentmanager، apimanager، botmanager و license-service موفق بود.
- `npm run lint` و `npm run build` در apimanager/web موفق بود؛ Vite تعداد ۲۳۱۹ module را transform کرد و exit code صفر بود.
- محدودیت باز: remote log transport هنوز placeholder است؛ broadcast وب queue پایدار ندارد و E2E زنده NATS/Docker/botpay/license اجرا نشده است.

---

## [2026-07-14] — بهبود تجربه پنل وب در موبایل و دسترس‌پذیری

- در apimanager/web برای پنل کاربر bottom navigation ثابت چهارقسمتی شامل داشبورد، ربات‌ها، پلن‌ها و پرداخت‌ها اضافه شد؛ route فعال هم در ظاهر و هم با aria-current مشخص می‌شود.
- safe-area دستگاه‌های دارای home indicator در جای نوار و padding انتهای محتوا محاسبه شد تا کنترل‌ها و آخرین ردیف صفحه زیر ناوبری نمانند.
- skip link به محتوای اصلی، focus-visible سراسری و پشتیبانی از prefers-reduced-motion اضافه شد.
- محتوای routeها به عرض حداکثر ۱۶۰۰ پیکسل محدود و animation کوتاه جابه‌جایی route با pathname هماهنگ شد.
- labelهای جدید در ترجمه فارسی و انگلیسی ثبت شدند و README وب با routeها و endpointهای واقعی service-types و templates اصلاح شد.
- مرجع prog/WEB.md برای معماری فرانت، UX واکنش‌گرا، قرارداد API، auth، build و محدودیت‌های امنیتی اضافه شد.
- اعتبارسنجی: npm run lint و npm run build در apimanager/web موفق بودند.

---

---
## [2026-07-14] — بازبینی کامل کد و همگام‌سازی مستندات

- تمام ۳۹۸ فایل Go بدون vendor/generated در ۲۲ module و مسیرهای تست خوانده و بر اساس wiring واقعی inventory شدند.
- README برای botpay مبتنی بر NATS، فرمان صحیح migration/test، وضعیت واقعی VPN providerها و تفاوت Core NATS/JetStream اصلاح شد.
- prog/services/SERVICE_REVIEW.md بر اساس file count، datastore، startup، API/NATS، state machine، تست و شکاف هر module بازنویسی شد.
- prog/PROJECT.md وضعیت فعال apimanager، Docker SDK agentmanager، تعداد moduleها و نتایج audit کامل را ثبت کرد.
- prog/SECURITY.md بدهی‌های deploy envelope، Telegram webhook secret، tenant cache source و eventهای اقتصادی core را اولویت‌بندی کرد.
- prog/TESTING.md تفاوت test production با specification simulation، E2E خارج CI و اولویت concurrency/contract test را شفاف کرد.
- service docs برای count و اصطلاحات stale shared-core، botpay و agentmanager اصلاح شدند.
- این مرحله docs-only است و هیچ source code یا migration جدیدی تغییر نکرد.

---

## [2026-07-14] — Sprint: audit fixes + E2E اجاره‌ی قفل + runbook + rotation

### Reliability & Security — اجرای برنامه‌ی audit (مرحله‌ی اول)

- **source tenant enforcement:** migration `source-service/0003` پس از backfill، `tenant_id` را `NOT NULL` می‌کند.

- **botpay:** `pay.credit` با ref اجباری و constraint یکتای `(wallet, service, ref, type)` idempotent شد؛ retry همان transaction را برمی‌گرداند. migration: `botpay/0002`.
- **provisioning:** هر خرید plan یک `attempt_id` پایدار دارد؛ خرید مستقل همان plan دوباره کسر می‌شود و retry همان attempt دوباره کسر نمی‌شود. REST از `Idempotency-Key` پشتیبانی می‌کند.
- **revenue-service:** earning قبل از payout اتمیک claim می‌شود، ref غیرخالی unique است و payoutهای owner/platform ref قطعی دارند. migration: `revenue-service/0002`.
- **ads-bot:** تخصیص FreeBotSlot با `FOR UPDATE SKIP LOCKED` انجام می‌شود.
- **source-service:** task و file API با HMAC، tenant scope، freshness و nonce یک‌بارمصرف fail-closed شدند؛ heartbeat نیز HMAC و freshness دارد. migration tenant: `source-service/0002` و backfill رکوردهای قدیمی قبل از NOT NULL لازم است.
- **webhook-gateway:** `SERVICE_HMAC_SECRET` در startup اجباری و register/unregister هر دو authenticated شدند.
- **agentmanager:** صف deploy به worker pool ثابت با backlog محدود ۳۰تایی تبدیل شد.
- **Verification:** تست قدیمی `shared-core/store` اصلاح شد؛ `scripts/test-all.sh` و GitHub CI برای جلوگیری از بدهی جدید gofmt و اجرای vet/test همه moduleها اضافه و Go baseline روی 1.25 هماهنگ شد. اجرای aggregate همه moduleها سبز است.
- **CI formatting baseline:** فهرست `.gofmt-baseline` بدهی قالب‌بندی legacy را ثبت می‌کند؛ CI هر فایل بدفرمت جدید را رد می‌کند بدون اینکه این sprint را با rewrite نامرتبط آلوده کند.
- **vpn-bot:** خط unreachable تکراری در route پنل ادمین حذف شد تا `go vet` کل workspace سبز باشد.

### Security — رفع باگ‌های audit (کارگاه A)

- **fraud-engine: رفع fail-open در احراز هویت admin.**
  - `internal/api/api.go` — `authMiddleware` اگر `adminKey` خالی باشد همیشه `401` می‌دهد
    (قبلاً هدر غایب `""` با کلید خالی برابر می‌شد → همه‌ی `/admin/*` باز). مقایسه با
    `crypto/subtle.ConstantTimeCompare`.
  - `cmd/main.go` — اگر `ADMIN_KEY` تنظیم نشده باشد `log.Fatal` (fail-closed در startup).
- **vpn-bot: رفع double-spend race در خرید با موجودی.**
  - `internal/store/store.go` — متد اتمیک `DeductBalanceIfEnough`
    (`UPDATE ... WHERE balance >= amount` + بررسی `RowsAffected`).
  - `internal/tgbot/user.go:confirmBuyWithBalance` — استفاده از متد اتمیک به‌جای
    check-then-`UpdateBalance`؛ دو کلیک هم‌زمان دیگر دو اشتراک نمی‌سازند.
- **vpn-bot: رفع نبود dedup در تأیید پرداخت آنلاین.**
  - `internal/store/store.go` — متد `ClaimOnlinePayment` با `ON CONFLICT DO NOTHING`.
  - `cmd/bot/main.go` — ایندکس partial یکتا `uq_payment_online_ref` روی
    `payments(gateway, ref_code)` برای گیت‌وی‌های آنلاین.
  - `internal/tgbot/user.go:verifyOnlinePayment` — پرداخت را بر اساس `refID` «claim» می‌کند؛
    کلیک تکراری «پرداخت کردم» دیگر اشتراک دوم نمی‌سازد.
- **archive-bot: رفع باگ کوچک `botUsername`.**
  - `internal/tgbot/handler.go` — `NewHandler` حالا `botUsername` را به struct ست می‌کند.
- (فقط مستند، بدون تغییر کد) **source-service hotspots** برای audit بعدی: مرز authorization در
  `internal/userbot/run_bot_command.go`، ذخیره‌ی session کامل MTProto در DB، لاگ‌شدن شماره تلفن.

### Added — تست E2E اجاره‌ی قفل کانال (کارگاه B)

- **`ads-bot/tools/e2e-lockrental/`** — ابزار تست end-to-end مدل اقتصادی lock-rental بدون تلگرام،
  با متدهای واقعی `ads-bot/internal/store` + botpay واقعی (NATS/HMAC): seed → approve → join →
  idempotency → fraud reversal → settlement → completion. داخل ماژول ads-bot و زیر `tools/`
  (نه `cmd/`) تا run.sh دست‌نخورده بماند. + `README.md`.

### Docs — runbook و آماده‌سازی مسیر A (کارگاه D)

- **`E2E_RUNBOOK.md`** — راهنمای سه سطح تست + وضعیت زیرساخت. زیرساخت داده بالاست ولی
  سرویس‌های go خاموش و local-bot-api (`141.95.210.17:8081`) در دسترس نیست → botpay startup
  را کامل نمی‌کند، پس اجرای زنده به bot API قابل‌دسترس موکول است (هر دو تست `go build` سبز).

### Security — چرخش secret ها + بازنویسی history (کارگاه C)

- **چرخش secret های اپلیکیشنی** در همه‌ی `.env` ها با `openssl rand -hex 32`، گروه‌های مشترک
  بین‌سرویسی هماهنگ (تأییدشده با hash): `SERVICE_HMAC_SECRET`، `ENCRYPTION_KEY`، `JWT_*`،
  `AGENT_API_KEY`، `INTERNAL_KEY`، `LOCK_API_SECRET`، `SERVICE_KEY_*`، `BOTPAY_API_KEY`،
  جفت‌های admin (fraud/image-registry/botpay)، `LICENSE_SIGNING_SECRET`،
  `SESSION_ENCRYPTION_KEY`، `LOG_API_KEY`، `GRAFANA_PASSWORD` و کلیدهای admin مستقل.
- **توقف ردیابی:** همه‌ی ۱۹ فایل `.env` با `git rm --cached` از git خارج شدند (روی دیسک با
  مقدار نو ماندند)؛ `.gitignore` → `**/.env` (فقط `.env.example` tracked).
- **بازنویسی history:** با `git filter-branch` همه‌ی `.env` از کل history حذف و
  `git push --force` روی `origin/main`. backup: `../CreatorBotV3-backup.git` (mirror).
- **دستی (در `SECRETS_ROTATION.md`):** پسوردهای زیرساخت (Postgres/Redis/NATS/Mongo — نیاز به
  تغییر سرور) و توکن‌های خارجی (BotFather/toncenter/پنل VPN — نیاز به revoke) عمداً env-only
  عوض نشدند چون stack را می‌شکست؛ با دستور دقیق مستند شدند.
- **`SECRETS_ROTATION.md`** (جدید) — runbook کامل چرخش.

### Process

- **قانون دائمی ثبت تغییرات:** از این تاریخ هر تغییر کد، تنظیمات، migration، deployment، تست یا مستندات باید هم‌زمان در `CHANGELOG.md` (تاریخچه و دلیل) و `prog/` (وضعیت نهایی و اثر معماری) ثبت شود؛ ثبت در فقط یکی از این دو ناقص محسوب می‌شود.
- از این سشن هر تغییر در **CHANGELOG.md** و کل پروژه با جزئیات در **`prog/`** ثبت می‌شود
  (خواسته‌ی کاربر ۲۰۲۶-۰۷-۱۴).

---
## [2026-07-10] — Sprint: یکپارچه‌سازی botmanager+apimanager + Env Schema wizard

### فاز ۱: اصلاح تداخل heartbeat/result بین botmanager و apimanager

#### shared-core/agentlistener/ — پکیج جدید
- `HandleHeartbeat`: RecordHeartbeat (CPU/RAM/containers JSON) + loop container status
- `HandleResult`: Deploy→Running / Failure→Error / Stop→Stopped / Remove→DeleteInstance
- **هر دو** botmanager و apimanager از این پکیج استفاده می‌کنند (منطق مشترک)

#### botmanager/cmd/main.go
- QueueSubscribe queue group از `"botmanager"` به `"managers"` تغییر کرد
- heartbeat/result inline handlers حذف شدند → `agentlistener.Handle*` جایگزین
- `MarkStaleServersOffline` ticker اضافه شد (هر ۲۰ ثانیه، آستانه ۶۰ ثانیه)
- import: `encoding/json` و `protocol` و `models` حذف شدند، `agentlistener` اضافه شد

#### apimanager/cmd/main.go
- `nc.Subscribe("agent.*.heartbeat", ...)` و `nc.Subscribe("agent.*.result", ...)` حذف شدند
- توابع `handleHeartbeat`/`handleResult` → delegate به `agentlistener.Handle*`
- HTTP fallback endpoints (`/agent/heartbeat`, `/agent/result`) دست‌نخورده ماندند

**باگ رفع‌شده:** Remove موفق → botmanager آن را `StatusRunning` می‌کرد (اشتباه). حالا `DeleteInstance` صدا می‌شود.

---
### فاز ۲: Env Schema — تعریف فیلد برای image + ویرایش توسط کاربر

#### shared-core/models/models.go
- `ConfigField struct` اضافه شد (`Key/Label/Default/Required`)
- `BotTemplate.ParseConfigSchema() []ConfigField`
- `BotInstance.ParseEnvOverrides() map[string]string`
- `BotInstance.SetEnvOverrides(map[string]string)`
- import: `encoding/json` اضافه شد

#### shared-core/store/store.go
- `UpdateTemplateSchema(ctx, id, schema string) error` — فقط config_schema را آپدیت می‌کند

#### botmanager/internal/tgbot/state/state.go
- `StepWizardConfig = "wiz:config"` — کاربر فیلدهای ConfigSchema را پر می‌کند
- `StepTmplSchemaJSON = "tmpl:schema:json"` — ادمین JSON schema می‌فرستد

#### botmanager/internal/tgbot/i18n/
- `keys.go`: `KeyWizardConfigField`, `KeyWizardConfigDone`, `KeyTmplAskSchema`, `KeyTmplSchemaSet`, `KeyTmplSchemaInvalid`, `KeyBtnEditSchema` اضافه شدند
- `fa.go` و `en.go`: ترجمه‌های متناظر

#### botmanager/internal/tgbot/admin/admin_tmpl.go
- `AdminTemplatesList`: دکمه `KeyBtnEditSchema` برای هر template اضافه شد
- `AdminTemplateSchemaEdit(ctx, c, tmplID)` — شروع ویرایش schema
- `AdminTemplateSchemaSet(ctx, c, tmplID, jsonText)` — تأیید + ذخیره schema

#### botmanager/internal/tgbot/user/wizard.go
- constant: `wkCfgIdx`, `wkCfgValues` اضافه شدند
- `WizardFinish`: اگر template دارای ConfigSchema باشد → `WizardConfigStart` به‌جای confirm
- `wizardShowConfigField(ctx, c, uid, fields, idx)` — نمایش فیلد با label + default
- `wizardShowConfirm(ctx, c, uid, data, plan)` — تأیید نهایی (extracted از WizardFinish)
- `WizardConfigValue(ctx, c, st, text)` — handler state `wiz:config`: ذخیره مقدار + پیشروی
- `parseCfgValues(jsonStr)` — parse helper
- `Provision(...)`: اضافه شد `extraEnv map[string]string` — مقادیر ConfigSchema به EnvVars merge می‌شوند
- `WizardPay`/`WizardCreateFree`: `parseCfgValues(data[wkCfgValues])` → `Provision`

#### botmanager/internal/tgbot/router.go
- callback `tmpl_schema` → `AdminTemplateSchemaEdit` اضافه شد (admin-only)

#### botmanager/internal/tgbot/router_text.go
- `case stepWizardConfig:` → `WizardConfigValue`
- `case stepTmplSchemaJSON:` → `AdminTemplateSchemaSet`

#### botmanager/internal/tgbot/state.go (alias file)
- `stepWizardConfig` و `stepTmplSchemaJSON` alias اضافه شدند



---
## [2026-07-10] — Sprint: fail-closed license برای همه ربات‌ها + per-service-type base env

### vpn-bot/cmd/bot/main.go
- اضافه شد `EncryptKey string` به Config (فیلد گمشده‌ای که NewHandler نیاز داشت)
- اضافه شد `licenseclient.RequireValid` در startup — fail-closed

### archive-bot/cmd/bot/main.go
- اضافه شد `licenseclient.RequireValid` در startup — fail-closed

### agentmanager — per-service-type base env
- `DeployDefaults`: فیلد جدید `TypeEnvDir` — دایرکتوری با فایل‌های per-type
- `Deploy()`: merge ۳ لایه: BaseEnv → TypeEnv (`<dir>/<image>.env`) → cmd.EnvVars
- `envfile.go`: اضافه شد `mergeEnvMaps()` و `parseEnvFileIfExists()`
- `cmd/main.go`: Config جدید `BOT_ENV_DIR`
- `.env`: `BOT_ENV_DIR=./botenv` اضافه شد
- `botenv/uploader.env`, `botenv/vpn-bot.env`, `botenv/archive-bot.env` ساخته شدند
- `botenv/` در `.gitignore` (روی سرور جدید دستی ساخته شود)

---
## [2026-07-10] — Sprint: botpay allowlist + revenue-service NATS migration

### botpay/internal/store/store.go
- `ValidateServiceID` allowlist: اضافه شد `community-service`, `fraud-engine`, `revenue-service`
- قبلاً این سه سرویس اگر pay.credit/deduct می‌زدند، botpay آن‌ها را رد می‌کرد

### revenue-service — HTTP → NATS
- `cmd/main.go`: `botpayClient` HTTP struct حذف شد → `natspayAdapter` با `natspayclient.Client`
- Config: حذف `BOTPAY_URL/BOTPAY_API_KEY/BOTPAY_ADMIN_KEY` → اضافه `SERVICE_HMAC_SECRET`
- `go.mod`: `shared-core` به عنوان dependency اضافه شد
- `.env`: آدرس HTTP botpay حذف شد، `SERVICE_HMAC_SECRET` اضافه شد

---
## [2026-07-10] — Sprint: License Fail-Closed + E2E Integration

### تغییرات این سشن (کامیت fdfb693)

#### shared-core/licenseclient/client.go — تابع جدید
- `RequireValid(ctx, nc, botID, token, serverID) error` اضافه شد
- fail-closed: token خالی / NATS قطع / verify رد شده → error (نه warning)
- `RunLicenseLoop` (هر ۶ ساعت) همچنان fail-open باقی ماند

#### uploader-bot/cmd/bot/main.go
- بعد از `engine.New()` و قبل از `rawBot.Start()`:
  `licenseclient.RequireValid` با timeout 20s صدا زده می‌شود
- اگر fail کند → `log.Fatal` → container اجرا نمی‌شود

---
## [2026-07-10] — Sprint: E2E Integration Chain (7 Services)

### سرویس‌های تغییرداده‌شده

#### shared/pkg/config/config.go — باگ بحرانی fix شد
- `bindEnvs()` اضافه شد تا همه فیلدهای mapstructure صریحاً BindEnv شوند
- **بدون این fix:** container بدون .env همه config را خالی می‌گرفت (BOT_TOKEN خالی → crash-loop)

#### agentmanager/internal/docker/client.go + envfile.go
- `DeployDefaults{BaseEnv, DefaultNetwork}` اضافه شد
- `Deploy()`: env نهایی = merge(BaseEnv از botenv.env، cmd.EnvVars) — overlay برنده
- اگر `cmd.NetworkName == ""` → DefaultNetwork (`deploy_backend`)

#### agentmanager/cmd/main.go + .env
- Config: `BOT_BASE_ENV_FILE=./botenv.env`, `DEFAULT_NETWORK=deploy_backend`
- `agentmanager/botenv.env` در .gitignore — روی سرور جدید دستی بساز

#### botmanager/internal/tgbot/user/wizard.go و admin_svctest.go
- `"OWNER_ID": fmt.Sprint(u.TelegramID)` به EnvVars اضافه شد
- (قبلاً فقط OWNER_TELEGRAM بود؛ uploader-bot OWNER_ID می‌خواند)

#### shared-core/engine/engine.go
- `loadInstanceInfo()` — PlanID/LockMode از جدول bot_instances خوانده می‌شود
- `InstanceInfo` struct با `IsFreeLock()` / `IsRentedLock()`

#### tools/e2e-provision/main.go — ابزار جدید
- E2E بدون تلگرام: pay.credit/balance/deduct → license.issue → DeployCommand → ResultMsg
- اجرا: `go run . -hmac <SECRET> -bot-token <TOKEN> -server-id <UUID>`

#### run.sh
- اضافه: image-registry، license-service، apimanager
- حذف: standalone uploader-bot (حالا فقط container داینامیک)

#### botpay/.env
- اضافه: `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`

#### image-registry/.env
- `SEED_CALLER_CIDR`: `172.16.0.0/12` → `127.0.0.1/32`

### شکاف‌های باقی‌مانده
- botpay allowlist هاردکد: community-service و fraud-engine هنوز در آن نیستند
- revenue-service هنوز HTTP قدیمی botpay را صدا می‌زند (باید NATS شود)
- botenv.env فعلاً DSN فقط برای uploader_bot دارد (vpn-bot نیاز به DSN جدا دارد)

---
## [2026-07-10] — Sprint: dbmigrate (migration ورژن‌دار)

### dbmigrate/ — ماژول جدید
- CLI: `up / status / mark / new / list` با `-dsn` و `-service`
- جدول `schema_migrations` با checksum در هر دیتابیس
- baseline SQL از AutoMigrate واقعی هر ۱۱ سرویس Postgres‌دار
- `migrations/botmanager/0001` تا `0004`:
  - 0002: fix `users_telegram_id_key` → `idx_users_telegram_id`
  - 0003: حذف legacy unique constraints روی servers/bot_instances/invite_links
  - 0004: تبدیل uuid→text برای FK ستون‌های bot_instances/plans/subscriptions

---
## [Unreleased] — Sprint 1-9

### Added
- سرویس `dbmigrate` — migration ورژن‌دار SQL برای هر ۱۱ سرویس Postgres دار
  (baseline از AutoMigrate واقعی هر سرویس؛ up/status/mark/new؛
  جدول schema_migrations با checksum؛ ساخت خودکار دیتابیس) — رجوع dbmigrate/README.md
- Self-service bot provisioning — کاربر بدون دخالت ادمین ربات می‌سازد
- Double-entry ledger در botpay
- Prometheus metrics در همه سرویس‌ها
- Grafana + Loki در docker-compose
- Audit log برای همه عملیات حساس
- Secret rotation با dual-key grace period
- Rate limiting (token bucket) در webhook-gateway
- Health check endpoints در همه سرویس‌ها
- Kubernetes manifests (16 manifest)
- Config versioning با rollback
- VPN adapters: Hiddify، X-UI، MarzNeshin
- E2E test suite
- Migration system (golang-migrate)
- Panic recovery در همه handler ها
- Context timeout در عملیات طولانی
- DB connection pool با lifecycle management

### Fixed
- uuid=text bug در PostgreSQL JOIN queries
- telebot v4 API: Photo/ForwardFrom/InlineResult
- NATS Authorization Violation در startup
- Duplicate ApprovePayment در member-bot
- Bot auto-state در admin list handlers
- Format string مشکلات در i18n

### Architecture
- NATS JetStream جایگزین Centrifugo WebSocket
- Bot engines مستقیم به DB وصل می‌شوند (نه apimanager)
- instance_id = bot_<BotID> برای persistence
- fraud-engine از request/reply NATS
