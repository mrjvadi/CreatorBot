# Secrets Rotation — CreatorBot V3

تاریخ: ۲۰۲۶-۰۷-۱۴.

## چرا
همه‌ی ۱۹ فایل `.env` با مقدار واقعی در git tracked و روی remote عمومی
`github.com/mrjvadi/CreatorBot.git` push شده بودند. `ENCRYPTION_KEY` ریشه از اولین کامیت
تا HEAD تغییر نکرده بود. پس هر secret که تا امروز در repo بوده **افشا شده** فرض می‌شود.

## چه چیزی خودکار انجام شد (این سشن)

### ۱. چرخش secret های اپلیکیشنی (self-contained)
مقادیر نو با `openssl rand -hex 32` تولید و در `.env` ها جایگزین شدند؛ گروه‌های مشترک
بین‌سرویسی هماهنگ ماندند (تأییدشده با مقایسه‌ی hash):
- **مشترک (یک مقدار):** `SERVICE_HMAC_SECRET` (۷ سرویس)، `ENCRYPTION_KEY` (۷ فایل)،
  `JWT_ACCESS_SECRET`، `JWT_REFRESH_SECRET`، `AGENT_API_KEY`، `INTERNAL_KEY`،
  `LOCK_API_SECRET`، `SERVICE_KEY_BOTMANAGER/UPLOADER/VPN`، `BOTPAY_API_KEY`.
- **جفت‌های admin (هر دو طرف یک مقدار):** `fraud-engine.ADMIN_KEY = ads-bot.FRAUD_ADMIN_KEY`،
  `image-registry.ADMIN_KEY = apimanager.IMAGE_REGISTRY_ADMIN_KEY`،
  `botpay.ADMIN_API_KEY = .env.BOTPAY_ADMIN_KEY`.
- **مستقل:** `LICENSE_SIGNING_SECRET`، `SESSION_ENCRYPTION_KEY`، `LOG_API_KEY`،
  `GRAFANA_PASSWORD`، `community-service.ADMIN_KEY`، `revenue-service.ADMIN_API_KEY`،
  و کلیدهای aggregate ریشه (`.env` ADMIN_KEY/ADMIN_API_KEY).

### ۲. توقف ردیابی + بازنویسی history
- همه‌ی `.env` ها از git خارج شدند (`git rm --cached`) و روی دیسک با مقدار نو ماندند.
- `.gitignore`: `**/.env` (فقط `.env.example` tracked می‌ماند).
- کل git history با `git filter-branch` بازنویسی شد تا هیچ `.env` در هیچ کامیتی نماند،
  سپس `git push --force` روی `origin/main`.
- backup کامل قبل از rewrite: `../CreatorBotV3-backup.git` (mirror).

## چه چیزی باید دستی انجام شود (نمی‌توانستم بی‌خطر انجام دهم)

### الف) پسوردهای زیرساخت — تغییر env تنها کافی نیست، سرور هم باید عوض شود
این‌ها عمداً در `.env` **تغییر نکردند** چون تغییر env-only بدون تغییر سرورِ در حال اجرا،
کل stack را می‌شکند. در یک پنجره‌ی maintenance:

| Secret | مکان | اقدام سرور |
|---|---|---|
| `POSTGRES_PASSWORD` + پسورد داخل `POSTGRES_DSN`/`MASTER_DSN` | تقریباً همه‌ی سرویس‌ها | `ALTER USER botuser WITH PASSWORD '<new>';` سپس همه‌ی DSN ها را به‌روزرسانی کن |
| `REDIS_PASSWORD` | ربات‌های redis‌دار | `requirepass <new>` در redis.conf + restart |
| `NATS_PASSWORD` (+`NATS_USERNAME`) | همه | کاربر/رمز در پیکربندی NATS + restart |
| `MONGO_PASSWORD` + `MONGO_URI` | uploader/fraud/archive/log-collector | تغییر رمز کاربر Mongo + به‌روزرسانی URI ها |

> نکته: history rewrite این مقادیر را از git **حذف** کرده، ولی چون قبلاً روی GitHub رفته‌اند،
> مقدار قدیمی ممکن است در clone/fork/cache کسی مانده باشد — پس چرخش سرور اجباری است.

### ب) توکن‌های خارجی — باید از منبع باطل شوند (revoke)
| Secret | کجا | اقدام |
|---|---|---|
| همه‌ی `BOT_TOKEN` / `BOTMANAGER_TOKEN` / `BOTPAY_TOKEN` / `TELEGRAM_BOT_TOKEN` | همه‌ی ربات‌ها + webhook-gateway + log-collector | BotFather → `/revoke` → توکن نو → در `.env` بگذار |
| `TON_API_KEY` | botpay (+ botmanager) | toncenter.com → regenerate |
| `PANEL_TOKEN` / `PANEL_PASSWORD` | vpn-bot / deploy | پنل VPN (Marzban/X-UI/…) → رمز/توکن نو |
| `ZARINPAL_MERCHANT` / `NOWPAYMENTS_KEY` | deploy | داشبورد درگاه پرداخت |
| `TG_APP_ID` / `TG_APP_HASH` / `TG_PHONE` | deploy/source-service | my.telegram.org (فعلاً خالی‌اند) |
| `SOURCE_SERVICE_KEY` | deploy | صادرشده توسط license-service |

## بعد از rotation — سلامت‌سنجی
1. `run.sh` را با `.env` های نو بالا بیاور.
2. چک کن سرویس‌ها HMAC handshake را رد نمی‌کنند (چون `SERVICE_HMAC_SECRET` مشترک هماهنگ عوض شده).
3. `git log --all -- '*.env'` (به‌جز `.env.example`) باید خالی باشد.
4. اگر داده‌ی رمزشده‌ی قبلی وجود داشت (session/رمز پنل)، به‌خاطر تغییر `ENCRYPTION_KEY`
   ناخوانا می‌شود (در dev قابل‌قبول؛ داده دورریختنی).

## هشدار force-push
history بازنویسی شده و SHA ها عوض شده‌اند. هر clone دیگری باید re-clone کند (نه `git pull`).
backup در `../CreatorBotV3-backup.git`.
