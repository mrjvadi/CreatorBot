# archive-bot

## این سرویس چیست
آرشیو فایل با جستجوی فارسی fuzzy — ساده‌ترین ربات محصول پلتفرم از نظر حجم قابلیت.

## مسئولیت‌ها
- ذخیره‌ی فایل‌ها به‌همراه دسته‌بندی (`Category`) و متادیتا.
- جستجوی fuzzy فارسی با `pg_trgm` (اکستنشن PostgreSQL) — یک GIN index روی `title || ' ' || tags || ' ' || description` ساخته می‌شود.
- مدیریت ادمین ساده (حذف فایل).

## ارتباطات
- `shared-core/engine` استفاده نمی‌شود — Postgres/Redis مستقیم.
- NATS فقط برای heartbeat/license check-in (قبلاً فقط در webhook mode بدون auth — رجوع به ایرادها).

## ایرادها و نکات
- **بدون یافته‌ی امنیتی**: بررسی دقیق `internal/tgbot/handler.go`'s `onCallback` نشان داد چک `h.isAdmin(c)` برای اکشن `del` به‌درستی و به‌صورت inline انجام می‌شود. جستجوی fuzzy در `internal/search/search.go` از placeholder های پارامتری GORM استفاده می‌کند (نه string concatenation) — در برابر SQL injection ایمن است.
- **رفع شد (۲۰۲۶-۰۷-۰۲، جانبی)**: مثل vpn-bot و member-bot، اتصال NATS این ربات هم قبلاً فقط در حالت webhook و بدون username/password ساخته می‌شد. الان هر وقت `NATS_URL` تنظیم باشد با auth کامل وصل می‌شود — لازم برای license check-in دوره‌ای.
- ساده‌ترین ربات از نظر سطح حمله هم هست — بدون پرداخت، بدون قفل کانال، بدون توکن ثانویه؛ به همین دلیل کمترین ریسک را در کل پلتفرم دارد.
