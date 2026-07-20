# prog/ — مستند کامل و دقیق پروژه CreatorBot V3

این پوشه منبع واحدِ همیشه‌به‌روزِ اطلاعات کل پروژه است (خواسته‌ی کاربر ۲۰۲۶-۰۷-۱۴).
با هر تغییر کد باید این‌جا هم به‌روز شود، در کنار `CHANGELOG.md` و `CLAUDE.md`.

## قانون اجباری ثبت تغییرات

هر تغییر در کد، تنظیمات، migration، deployment، تست یا مستندات باید در همان نوبت در هر دو محل ثبت شود:

1. `CHANGELOG.md` — چه چیزی، چه زمانی و چرا تغییر کرد.
2. `prog/` — وضعیت نهایی سیستم، اثر معماری، قراردادها و محدودیت‌های باقی‌مانده.

تغییری که فقط در یکی از این دو محل ثبت شده باشد از نظر فرایند پروژه ناقص است.

## فهرست
- [PROJECT.md](PROJECT.md) — نمای کامل معماری، ۱۸+ سرویس، مدل داده، NATS، جریان‌ها،
  وضعیت امنیتی، تست، deployment، و شکاف‌ها. با جزئیات فایل و متد.
- [SECURITY.md](SECURITY.md) — وضعیت امنیتی سرویس‌به‌سرویس: چه audit شده، چه رفع شده،
  چه hotspot هایی برای audit بعدی مانده.
- [TESTING.md](TESTING.md) — همه‌ی ابزارها و مسیرهای تست (harness ها، e2e-provision، مسیر A).
- [services/SERVICE_REVIEW.md](services/SERVICE_REVIEW.md) — بازبینی کامل ۳۹۸ فایل Go؛ wiring واقعی، datastore، ارتباطات، تست و شکاف‌های هر module.
- [AUDIT_IMPLEMENTATION_2026-07-14.md](AUDIT_IMPLEMENTATION_2026-07-14.md) — تغییرات audit، قراردادهای جدید، migration و rollout.
- [WEB.md](WEB.md) — معماری، مسیرها، UX، دسترس‌پذیری، قرارداد API، build و محدودیت‌های پنل React.
- [MANAGER_PARITY.md](MANAGER_PARITY.md) — قرارداد و ماتریس هم‌سان‌سازی botmanager/apimanager، routeها، امنیت و محدودیت‌ها.
- [BOT_PROFILE.md](BOT_PROFILE.md) — قرارداد پاک‌سازی و نام‌گذاری خودکار پروفایل همه ربات‌ها در production.

## رابطه با بقیه‌ی مستندات
- **`CLAUDE.md`** — خلاصه‌ی معماری برای همکاری روزمره (فارسی، مختصرتر).
- **`CHANGELOG.md`** — تاریخچه‌ی زمانیِ تغییرات.
- **`prog/`** — مرجع کامل و عمیق (این پوشه).
- **memory** — یادداشت‌های بین‌سشنی (index در `MEMORY.md`).
