-- 0004_align_fk_columns_with_models — سرویس botmanager (دیتابیس: botmanager)
--
-- ادامه‌ی 0002/0003 (پاک‌سازی schema قدیمی زنده تا AutoMigrate دیگر وسط راه
-- قطع نشود): schema قدیمی ستون‌های رابطه را uuid + FOREIGN KEY واقعی ساخته
-- بود، ولی مدل‌های فعلی (shared-core/models) این ستون‌ها را string (text)
-- بدون FK تعریف می‌کنند (همان «uuid=text bug» که در CHANGELOG هم آمده؛
-- تنها FK ای که مدل‌ها می‌خواهند fk_plans_limits روی plan_bot_limits است
-- که AutoMigrate خودش می‌سازد). GORM موقع تلاش برای هم‌راستا کردن نوع،
-- روی FK قدیمی با خطای 42804 می‌شکست و AutoMigrate همان‌جا می‌ایستاد.
--
-- انواع هدف دقیقاً از baseline (0001، تولیدشده از AutoMigrate واقعی مدل‌ها)
-- کپی شده‌اند. همه‌ی این جدول‌ها الان خالی‌اند، پس تبدیل نوع بدون ریسک داده است.

-- FK های قدیمی — مدل‌های فعلی هیچ‌کدام را ندارند.
ALTER TABLE plans         DROP CONSTRAINT IF EXISTS plans_template_id_fkey;
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_user_id_fkey;
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_fkey;
ALTER TABLE bot_instances DROP CONSTRAINT IF EXISTS bot_instances_owner_id_fkey;
ALTER TABLE bot_instances DROP CONSTRAINT IF EXISTS bot_instances_template_id_fkey;
ALTER TABLE bot_instances DROP CONSTRAINT IF EXISTS bot_instances_server_id_fkey;
ALTER TABLE invite_links  DROP CONSTRAINT IF EXISTS invite_links_created_by_fkey;

-- uuid → text (مطابق baseline)
ALTER TABLE bot_instances
    ALTER COLUMN owner_id    TYPE text USING owner_id::text,
    ALTER COLUMN template_id TYPE text USING template_id::text,
    ALTER COLUMN server_id   TYPE text USING server_id::text;
ALTER TABLE plans         ALTER COLUMN template_id TYPE text USING template_id::text;
ALTER TABLE subscriptions
    ALTER COLUMN user_id TYPE text USING user_id::text,
    ALTER COLUMN plan_id TYPE text USING plan_id::text;

-- invite_links.created_by در مدل فعلی TelegramID ادمین است (bigint)، نه
-- uuid کاربر — تبدیل مستقیم معنی ندارد؛ جدول خالی است پس NULL می‌شود.
ALTER TABLE invite_links ALTER COLUMN created_by TYPE bigint USING NULL;
