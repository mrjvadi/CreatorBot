# member-bot — بازنشسته از dbmigrate (۲۰۲۶-۰۷-۱۷)

از این تاریخ، `member-bot` هیچ Postgres ندارد — همه‌ی داده (owners, locks,
check_bots + memberships تودرتو, payments) به MongoDB منتقل شد (دیتابیس
اختصاصی `member_bot`، یک instance جداگانه به‌ازای هر deploy، مطابقِ همان
مدلِ per-instance-dedicated که قبلاً برای Postgres وجود داشت).
`member_verifications` و `settings` هم مهاجرت نشدند — با grep کاملِ کدبیس
تأیید شد هیچ‌جا خوانده/نوشته نمی‌شدند (schema مرده).

`0001_baseline.sql` در همین پوشه تاریخچه است، دیگر روی هیچ instance واقعی
اجرا نمی‌شود — برای بازسازیِ schemaِ Postgresِ قدیمی نگه داشته شده، حذف نشده.

منبعِ حقیقتِ ایندکس‌های فعلی: `member-bot/internal/store/store.go` تابع
`EnsureIndexes()`، در startup هر بار idempotent صدا زده می‌شود (معادلِ
AutoMigrate/CREATE UNIQUE INDEX قبلی).
