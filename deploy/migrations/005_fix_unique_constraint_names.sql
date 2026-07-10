-- Migration 005: هماهنگ‌کردن نامِ سه UNIQUE constraint با قراردادِ GORM (۲۰۲۶-۰۷-۰۶)
--
-- این migration برای پایگاه‌داده‌هایی است که از قبل با نسخه‌ی قدیمیِ 001_initial.sql
-- ساخته شده‌اند — یعنی users.telegram_id / servers.ip / invite_links.token با یک
-- UNIQUE بی‌نام ساخته شده‌اند و Postgres خودش نامِ پیش‌فرض (<table>_<column>_key) را
-- گذاشته. روی چنین دیتابیسی، اولین بار که apimanager/botmanager بالا می‌آید و
-- AutoMigrate اجرا می‌شود، این خطای واقعی رخ می‌دهد (که در تولید هم دیده شد):
--
--   ERROR: constraint "uni_users_telegram_id" of relation "users" does not exist (SQLSTATE 42704)
--
-- علت: GORM's AutoMigrate برای فیلدهایی که `gorm:"uniqueIndex"` دارند (بدون اسم صریح)،
-- علاوه بر خودِ index، انتظار دارد یک UNIQUE CONSTRAINT با نامِ قراردادی خودش
-- (uni_<table>_<column>) هم مدیریت کند — و در تلاش برای هماهنگ‌کردنِ آن، یک
-- DROP CONSTRAINT بدون IF EXISTS صادر می‌کند که چون آن نام هرگز وجود نداشته، با خطا
-- متوقف می‌شود (FATAL، کل migrate/startup را می‌کشد).
--
-- این migration نام‌های موجود را idempotent (قابل‌اجرای چندباره، بدون خطا چه constraint
-- با نام قدیمی باشد چه از قبل هم renamed شده باشد) به نامِ موردنظرِ GORM تغییر می‌دهد،
-- تا AutoMigrate دیگر این ناسازگاری را نبیند. رجوع 001_initial.sql برای اینکه از این
-- به بعد جدول‌های تازه از همان ابتدا با نامِ درست ساخته می‌شوند.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_telegram_id_key'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uni_users_telegram_id'
    ) THEN
        ALTER TABLE users RENAME CONSTRAINT users_telegram_id_key TO uni_users_telegram_id;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'servers_ip_key'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uni_servers_ip'
    ) THEN
        ALTER TABLE servers RENAME CONSTRAINT servers_ip_key TO uni_servers_ip;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'invite_links_token_key'
    ) AND NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uni_invite_links_token'
    ) THEN
        ALTER TABLE invite_links RENAME CONSTRAINT invite_links_token_key TO uni_invite_links_token;
    END IF;
END $$;
