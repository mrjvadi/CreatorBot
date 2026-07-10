-- 0002_fix_users_telegram_id_unique — سرویس botmanager (دیتابیس: botmanager)
--
-- مشکل واقعی (۲۰۲۶-۰۷-۱۰): جدول users روی دیتابیس زنده، یکتاییِ telegram_id
-- را با constraint قدیمی «users_telegram_id_key» (قرارداد نام‌گذاری Postgres)
-- دارد، ولی مدل فعلی (shared-core/models: `gorm:"uniqueIndex"`) یک UNIQUE
-- INDEX با نام idx_users_telegram_id می‌خواهد. GORM در AutoMigrate تلاش
-- می‌کند constraint را با نامی که خودش انتظار دارد (uni_users_telegram_id)
-- drop کند، پیدا نمی‌کند، خطا می‌دهد و چون users اولین مدل است، کل
-- AutoMigrate همان‌جا قطع می‌شود — یعنی جدول/ستون‌های مدل‌های بعدی
-- (مثل ستون‌های tags/max_containers در servers) هرگز ساخته نمی‌شدند.
--
-- این نسخه schema زنده را به همان شکل baseline (0001) می‌رساند:
-- constraint قدیمی حذف و ایندکس non-unique قدیمی با UNIQUE INDEX همنام
-- baseline جایگزین می‌شود. بعد از این، AutoMigrate بدون خطا رد می‌شود و
-- بقیه‌ی مدل‌ها را (additive) خودش کامل می‌کند.

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_telegram_id_key;
DROP INDEX IF EXISTS idx_users_telegram_id;
CREATE UNIQUE INDEX idx_users_telegram_id ON users USING btree (telegram_id);
