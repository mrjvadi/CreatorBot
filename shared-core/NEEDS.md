# shared-core — چه چیزی نیاز داریم (بررسی ۲۰۲۶-۰۷-۰۴)

> `shared/PENDING_CHANGES.md` را هم بخوانید — دو یافته‌ی مربوط به همین پکیج (کد مرده در `docstore`،
> `schema` package استفاده‌نشده) آن‌جا از قبل مستند شده‌اند و این‌جا فقط خلاصه می‌شوند.

## وضعیت فعلی
۲۷ فایل، ~۴۷۰۴ خط. مدل‌های مرکزی `botmanager`، `protocol` (مرکز تعریف NATS)، `engine` (موتور هر bot
container)، `natspayclient`، `licenseclient` (اضافه‌شده ۲۰۲۶-۰۷-۰۲).

## چیزی که واقعاً کم است
1. **کد مرده‌ی مستندشده** (طبق `PENDING_CHANGES.md`): `docstore/uploader.go`'s `CodeStore`/`FileStore`
   و `documents/uploader.go`'s `Code`/`File`/`CodeUsage` هیچ‌جا import نمی‌شوند — امن برای حذف.
2. **`schema` package استفاده‌نشده** — برای جداسازی فیزیکی DB هر instance نوشته شده ولی هنوز هیچ‌جا
   صدا زده نمی‌شود؛ این عمداً scaffolding است، نه باگ (طبق CLAUDE.md، جداسازی فیزیکی DB هدف بلندمدت
   اعلام‌شده است) — فقط تا وقتی آن مهاجرت شروع نشده، بی‌استفاده می‌ماند.
3. **`store/store.go`'s ساخت/حذف schema با string concatenation خام** (نه از طریق `schema.Create`/
   `Drop` امن) — طبق `docs/security-audit-2026-07-02.md` بخش‌های میانی، فعلاً چون `DBSchema` از کد
   داخلی ساخته می‌شود خطر فوری ندارد، ولی اگر مسیری در آینده این مقدار را از ورودی بیرونی بگیرد،
   SQL injection/DROP دلخواه خواهد بود.
4. **فقط `uploader-bot` (و `admanager-bot`) از `engine` استفاده می‌کنند** — `vpn-bot`, `archive-bot`,
   `member-bot` هرکدام DB/Redis/NATS خودشان را جدا و دستی وصل می‌کنند. یعنی هر بهبودی در `engine`
   (مثل `AttachNATS`/license loop که این دور اضافه شد) باید دستی و جدا در هر ۳ ربات دیگر هم تکرار شود
   — دقیقاً همان کاری که این دور برای `vpn-bot`/`archive-bot`/`member-bot` انجام شد. یک‌سان‌سازی این
   ۴ ربات روی یک `engine` مشترک، نگه‌داری آینده را بسیار ساده‌تر می‌کند.
5. **تنها یک فایل تست** (`store/store_test.go`) برای کل این پکیج مرکزی.
