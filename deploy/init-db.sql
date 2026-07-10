-- ⚠️ منسوخ (۲۰۲۶-۰۷-۰۶): دیگر توسط هیچ docker-compose.yml ای mount نمی‌شود.
-- محتوای این فایل به deploy/migrations/000_create_product_databases.sql منتقل
-- شد تا هر دو docker-compose (ریشه‌ی پروژه و deploy/) از یک پوشه‌ی init واحد
-- استفاده کنند. فقط برای مرجع تاریخی نگه داشته شده — این نسخه دیگر اجرا
-- نمی‌شود، فایل بالا را ویرایش کنید.
--
-- CreatorBot — PostgreSQL initialization
-- This script runs once when the postgres container is first created.
-- Creates separate databases for each service sharing the same postgres instance.

CREATE DATABASE uploader_bot  WITH OWNER botuser;
CREATE DATABASE vpn_bot       WITH OWNER botuser;
CREATE DATABASE archive_bot   WITH OWNER botuser;
CREATE DATABASE member_bot    WITH OWNER botuser;
CREATE DATABASE source_svc    WITH OWNER botuser;

-- Enable pg_trgm for archive-bot fuzzy search (runs in botmanager db by default;
-- archive-bot migration also runs this, but having it here ensures it exists.)
\c archive_bot
CREATE EXTENSION IF NOT EXISTS pg_trgm;
\c botmanager
