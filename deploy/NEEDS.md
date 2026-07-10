# deploy/ — چه چیزی نیاز داریم (بررسی و رفع ۲۰۲۶-۰۷-۰۶)

## چه چیزی خراب/ناقص بود

این پوشه دو راه مستقل برای بالا آوردن کل استک داشت که سال‌ها از هم عقب افتاده بودند:

1. **`docker-compose.yml` ریشه‌ی پروژه** — همیشه به‌روز نگه داشته می‌شد (هر سرویس جدیدی که این دور
   ساخته شد، مثل license-service/log-collector/image-registry، همان‌جا هم اضافه می‌شد).
2. **`deploy/docker-compose.yml`** — نسخه‌ی قدیمی‌تر با Traefik/دامنه/TLS (که کاربر واقعاً از همین
   استفاده می‌کند)، ولی از ۸ سرویس واقعی‌اش (botmanager, apimanager, agentmanager, uploader-bot,
   vpn-bot, archive-bot, member-bot, source-service) همه کامنت بودند، ۱۰ سرویس جدیدتر
   (botpay, community-service, revenue-service, license-service, log-collector, image-registry,
   webhook-gateway, fraud-engine, docker-socket-proxy) اصلاً وجود نداشتند، و anchor ی که همه‌ی
   بلاک‌های build به آن اشاره می‌کردند (`x-build`) خودش هم کامنت بود — یعنی حتی اگر یکی از آن ۸
   بلاک را دستی از کامنت درمی‌آوردید، فایل از اول invalid بود.

علاوه بر این، دو مشکل واقعی جدا کشف و رفع شد:
- **نسخه‌ی Go در ۱۳ Dockerfile از ۱۹ تا اشتباه بود** — `go.work` نیازمند `go 1.25.0` است، ولی اکثر
  Dockerfile ها (هم در ریشه‌ی هر سرویس، هم در `<service>/deploy/`) هنوز `golang:1.22-alpine` داشتند؛
  یعنی build یا شکست می‌خورد یا مجبور به دانلود خودکار toolchain از اینترنت می‌شد. همه به
  `golang:1.25-alpine` یکسان شدند.
- **دیتابیس‌های نوع‌های ربات محصول اصلاً استفاده نمی‌شدند** — `deploy/init-db.sql` از قبل دیتابیس‌های
  `uploader_bot`/`vpn_bot`/`archive_bot`/`member_bot`/`source_svc` را می‌ساخت، ولی خودِ
  `uploader-bot/.env`/`archive-bot/.env` هنوز به `botmanager` وصل بودند (نه دیتابیس اختصاصی خودشان)،
  و `vpn-bot`/`member-bot`/`source-service`'s `.env.example` ها اصلاً یا نام env متغیر را اشتباه
  داشتند (`POSTGRES_DSN` به‌جای `MASTER_DSN` که کد واقعی vpn-bot/member-bot می‌خواند) یا
  کاربر/دیتابیسِ ساختگی (`vpnbot`, `memberbot`, `sourceservice`, `app`) که هیچ‌جا واقعاً ساخته
  نمی‌شد. همه به `botuser` + نام دیتابیس واقعی (`uploader_bot`, `vpn_bot`, ...) اصلاح شدند.

## چیزی که انجام شد

1. `deploy/docker-compose.yml` کامل بازنویسی شد: `x-build` anchor درست شد، هر ۸ بلاک قدیمی
   uncomment+fix شد، ۹ سرویس جدید + `docker-socket-proxy` اضافه شد، Traefik label برای
   `apimanager` (API عمومی) و `webhook-gateway` (اگر حالت webhook فعال باشد) اضافه شد.
2. `deploy/migrations/000_create_databases.sql` (سرویس‌های مرکزی) و
   `000_create_product_databases.sql` (نوع‌های ربات محصول، جایگزین `deploy/init-db.sql`) — هر دو
   docker-compose (ریشه و این پوشه) حالا از همین یک پوشه‌ی `migrations` می‌خوانند.
   `deploy/init-db.sql` دیگر mount نمی‌شود؛ فقط برای مرجع تاریخی نگه داشته شد.
3. هر ۱۹ Dockerfile به `golang:1.25-alpine` یکسان شد.
4. `.env`/`.env.example` هر ۵ ربات محصول (uploader-bot, vpn-bot, archive-bot, member-bot,
   source-service) به دیتابیس اختصاصیِ خودشان + کاربر واقعی (`botuser`) + نام env متغیر درست اصلاح شد.
   `source-service/.env` هم از صفر ساخته شد (وجود نداشت، `deploy/docker-compose.yml` به آن نیاز دارد).
5. `deploy/.env.example` بازطراحی شد — فقط متغیرهایی که واقعاً با `${VAR}` در
   `deploy/docker-compose.yml` جایگزین می‌شوند این‌جا ماندند؛ رمزهای مخصوص هر سرویس حذف شدند چون از
   `env_file` مخصوص همان سرویس خوانده می‌شوند.
6. `deploy/Makefile`'s `build-all` سه سرویس جاافتاده (license-service, log-collector,
   image-registry) را هم اضافه کرد.

## چیزی که هنوز باقی است

1. **تست واقعی نشده** — این یک بازنویسی کامل بدون Docker daemon در دسترس (محیط sandbox) بود؛ قبل از
   اجرای واقعی روی سرور، حتماً `docker compose --env-file .env config` را برای چک نحو نهایی و بعد
   `docker compose up -d` را با دقت (و لاگ کامل هر سرویس) اجرا کنید.
2. **ads-bot عمداً در این فایل نیست** — هنوز هیچ Dockerfile ای ندارد (نه در ریشه، نه در
   `ads-bot/deploy/`)؛ رجوع CLAUDE.md — فعلاً با `go run cmd/*.go` دستی تست می‌شود. وقتی
   Dockerfile آن ساخته شد، این‌جا هم باید اضافه شود.
3. **مدل «یک DB به‌ازای نوع ربات، فیلتر با instance_id» تازه برای اولین‌بار این‌جا صریح شد** — قبلاً
   فقط در کد (`instance_id` در `uploader-bot/internal/store/*.go`) و یک کامنت پراکنده در `.env` دیده
   می‌شد. اگر عملاً بیش از یک instance واقعی از یک نوع ربات هم‌زمان روی همین یک `docker-compose`
   دیده شود، ارزش تأیید مجدد این فرض را دارد (یا با خواندن دقیق‌تر `agentmanager`'s deploy flow، یا
   با یک تست دستی).
4. **Traefik/ACME هنوز تست نشده** — این فایل از قبل هم Traefik داشت، ولی چون هیچ سرویس واقعی‌ای
   روشن نبود، مسیر HTTPS واقعی (`api.${DOMAIN}`) هرگز end-to-end چک نشده.
5. **`agentmanager` در این فایل به `docker-socket-proxy` وصل است، نه مستقیم به `docker.sock`** —
   همان سخت‌گیری امنیتی‌ای که در `docker-compose.yml` ریشه هست، این‌جا هم اعمال شد؛ ولی چون
   `agentmanager` طبق معماری معمولاً روی سرورهای دیگر (نه همین استک مرکزی) اجرا می‌شود، این بلاک در
   عمل بیشتر برای تست/dev محلی کاربرد دارد تا یک عضو همیشگی این compose در production واقعی.

## این‌ها را در سرویس‌های دیگر هم نوشتم
- `docker-compose.yml` ریشه‌ی پروژه هم اکنون از `deploy/migrations/000_create_product_databases.sql`
  تأثیر می‌گیرد (چون همان پوشه را mount می‌کند) — یعنی اگر آن استک هم بالا بیاید، دیتابیس‌های نوع‌های
  ربات محصول را هم می‌سازد، حتی اگر خودِ آن compose این ربات‌ها را اجرا نکند. بی‌ضرر، ولی عمدی مستند شد.
