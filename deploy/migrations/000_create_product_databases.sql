-- Migration 000 (product bots): جداسازی دیتابیسِ نوع‌های ربات محصول (۲۰۲۶-۰۷-۰۶)
--
-- این فایل جایگزین deploy/init-db.sql است — همان محتوا، فقط منتقل‌شده به
-- deploy/migrations تا هر دو docker-compose.yml (ریشه‌ی پروژه و deploy/) از
-- یک پوشه‌ی init مشترک استفاده کنند، نه دو مسیر جدا که با هم هماهنگ نبودند.
-- deploy/init-db.sql دیگر توسط هیچ docker-compose.yml ای mount نمی‌شود —
-- برای مرجع تاریخی نگه داشته شده، ولی از این به بعد بی‌اثر است.
--
-- uploader-bot/vpn-bot/archive-bot/member-bot/source-service «سرویس‌های
-- مرکزی» نیستند (برخلاف botmanager/botpay/...) — این‌ها نوع‌های ربات محصول‌اند
-- که agentmanager به‌صورت داینامیک برای هر مشتری container می‌سازد. طبق کد
-- واقعی (رجوع `instance_id`/`InstanceID` در uploader-bot/internal/store/*.go و
-- مشابه در بقیه)، چند instance از یک نوع ربات می‌توانند این یک دیتابیس را با
-- هم شریک شوند و با ستون instance_id از هم جدا بمانند — پس هرکدام یک
-- دیتابیس (نه یک دیتابیس به‌ازای هر instance)، ولی همچنان جدا از botmanager
-- (که یک مالک داده‌ی کاملاً متفاوت است: User/BotInstance/Plan خودِ پلتفرم).

CREATE DATABASE uploader_bot WITH OWNER botuser;
CREATE DATABASE vpn_bot      WITH OWNER botuser;
CREATE DATABASE archive_bot  WITH OWNER botuser;
CREATE DATABASE member_bot   WITH OWNER botuser;
CREATE DATABASE source_svc   WITH OWNER botuser;

-- pg_trgm برای جستجوی فارسی fuzzy در archive-bot (رجوع CLAUDE.md بخش ۳).
-- خودِ migration داخلی archive-bot هم این را چک می‌کند؛ این‌جا فقط برای
-- اطمینان از وجودش از روز اول تکرار شده.
\c archive_bot
CREATE EXTENSION IF NOT EXISTS pg_trgm;
