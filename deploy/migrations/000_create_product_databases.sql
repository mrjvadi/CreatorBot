-- Migration 000 (product bots): جداسازی دیتابیسِ نوع‌های ربات محصول (۲۰۲۶-۰۷-۰۶)
--
-- این فایل جایگزین deploy/init-db.sql است — همان محتوا، فقط منتقل‌شده به
-- deploy/migrations تا هر دو docker-compose.yml (ریشه‌ی پروژه و deploy/) از
-- یک پوشه‌ی init مشترک استفاده کنند، نه دو مسیر جدا که با هم هماهنگ نبودند.
-- deploy/init-db.sql دیگر توسط هیچ docker-compose.yml ای mount نمی‌شود —
-- برای مرجع تاریخی نگه داشته شده، ولی از این به بعد بی‌اثر است.
--
-- uploader-bot/source-service «سرویس‌های مرکزی» نیستند (برخلاف
-- botmanager/botpay/...) — این‌ها نوع‌های ربات محصول‌اند که agentmanager
-- به‌صورت داینامیک برای هر مشتری container می‌سازد. طبق کد واقعی (رجوع
-- `instance_id`/`InstanceID` در uploader-bot/internal/store/*.go)، چند
-- instance از یک نوع ربات می‌توانند این یک دیتابیس را با هم شریک شوند و با
-- ستون instance_id از هم جدا بمانند — پس هرکدام یک دیتابیس (نه یک دیتابیس
-- به‌ازای هر instance)، ولی همچنان جدا از botmanager (که یک مالک داده‌ی
-- کاملاً متفاوت است: User/BotInstance/Plan خودِ پلتفرم).
--
-- (۲۰۲۶-۰۷-۱۷) vpn_bot/archive_bot/member_bot از اینجا حذف شدند — این سه نوع
-- ربات دیگر هیچ Postgres ندارند، همه‌ی داده‌شان روی MongoDB (دیتابیس اختصاصی
-- به‌ازای هر instance) است. رجوع dbmigrate/migrations/{vpn-bot,archive-bot,
-- member-bot}/RETIRED.md برای جزئیات.

CREATE DATABASE uploader_bot WITH OWNER botuser;
CREATE DATABASE source_svc   WITH OWNER botuser;
