# مستندسازی سرویس‌ها — CreatorBotV3

بررسی اولیه‌ی هر سرویس (۲۰۲۶-۰۷-۰۲)، هرکدام در یک فایل جدا: چه کاری می‌کند، با چه چیزی حرف می‌زند، و چه
ایرادی (اگر بود) پیدا شد. ⚠️ **این فایل‌ها snapshot یک لحظه‌اند و بعضی جاها قدیمی شده‌اند** (مثلاً
`apimanager` و `source-service` بعد از این تاریخ خیلی رشد کردند) — برای وضعیت به‌روز و نیازهای واقعیِ هر
سرویس، همیشه اول `<نام‌سرویس>/NEEDS.md` داخل خودِ پوشه‌ی آن سرویس را نگاه کنید، نه این‌جا. خلاصه‌ی
جمع‌بندی‌شده‌ی همه‌ی این نیازها هم در [`docs/gap-analysis-2026-07-04.md`](../gap-analysis-2026-07-04.md).

## لایه‌ی مرکزی
- [botmanager](botmanager.md) — ربات فروش اصلی، پنل کاربر/ادمین
- [apimanager](apimanager.md) — دروازه‌ی HTTP بیرونی (⚠️ قدیمی — الان یک API+وب کامل دارد، رجوع
  `apimanager/PROJECT_UNDERSTANDING.md` و `apimanager/NEEDS.md`)
- [agentmanager](agentmanager.md) — اجرای واقعی Docker container ها
- [image-registry](../../image-registry/README.md) — استخر ثبت‌شده‌ی image مجاز + whitelist بر اساس
  IP/CIDR (سرویس جدید، ۲۰۲۶-۰۷-۰۴؛ تنها سرویس HTTP-first این پلتفرم)
- [webhook-gateway](webhook-gateway.md) — دریافت و forward webhook تلگرام
- [botpay](botpay.md) — کیف‌پول مرکزی TON
- [license-service](license-service.md) — لایسنس/ضدکپی هر instance (سرویس جدید)
- [log-collector](log-collector.md) — جمع‌آوری لاگ Warn+ روی NATS، Mongo + تلگرام (سرویس جدید)

## ربات‌های محصول
- [uploader-bot](uploader-bot.md) — فروش فایل با کد
- [vpn-bot](vpn-bot.md) — فروش اشتراک VPN
- [archive-bot](archive-bot.md) — آرشیو فایل با جستجوی فارسی
- [member-bot](member-bot.md) — زیرساخت داخلی چک عضویت

## لایه‌ی تبلیغات/درآمد
- [ads-bot](ads-bot.md) — تبلیغات CPJ + اجاره‌ی قفل کانال
- [admanager-bot](admanager-bot.md) — ابزار ادمین‌محور مدیریت تبلیغ (خارج از CLAUDE.md اصلی)
- [community-service](community-service.md) — تقسیم درآمد بین گروه‌ها
- [fraud-engine](fraud-engine.md) — امتیازدهی کیفیت/تقلب
- [revenue-service](revenue-service.md) — قوانین کمیسیون و واریز نهایی

## ناتمام
- [source-service](source-service.md) — ⚠️ قدیمی؛ دیگر stub نیست، رجوع `source-service/NEEDS.md`

## کتابخانه‌های مشترک (نه سرویس مستقل)
- [shared](shared.md) — ابزار پایه‌ی مشترک همه‌ی سرویس‌ها
- [shared-core](shared-core.md) — مدل/protocol/engine مخصوص لایه‌ی پلتفرم

---

برای گزارش کامل امنیتی (شبیه‌سازی حمله، مسیر دقیق هر باگ رفع‌شده) رجوع کنید به [`docs/security-audit-2026-07-02.md`](../security-audit-2026-07-02.md).
