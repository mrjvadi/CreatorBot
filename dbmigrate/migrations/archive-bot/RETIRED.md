# archive-bot — بازنشسته از dbmigrate (۲۰۲۶-۰۷-۱۷)

از این تاریخ، `archive-bot` هیچ Postgres ندارد — همه‌ی داده (users, categories,
files) به MongoDB منتقل شد (دیتابیس اختصاصی `archive_bot`، یک instance
جداگانه به‌ازای هر deploy، مطابقِ همان مدلِ per-instance-dedicated که قبلاً
برای Postgres وجود داشت).

`0001_baseline.sql` در همین پوشه تاریخچه است، دیگر روی هیچ instance واقعی
اجرا نمی‌شود — برای بازسازیِ schemaِ Postgresِ قدیمی نگه داشته شده، حذف نشده.

منبعِ حقیقتِ ایندکس‌های فعلی: `archive-bot/internal/store/store.go` تابع
`EnsureIndexes()`، در startup هر بار idempotent صدا زده می‌شود (معادلِ
AutoMigrate/CREATE UNIQUE INDEX قبلی).
