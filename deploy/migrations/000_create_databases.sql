-- Migration 000: جداسازی دیتابیس‌های سرویس‌های مرکزی (۲۰۲۶-۰۷-۰۶)
--
-- قبل از این، همه‌ی سرویس‌های مرکزی (botmanager, apimanager, botpay, ads-bot,
-- community-service, revenue-service, license-service, image-registry) روی
-- یک دیتابیس مشترک به نام "creatorbot" بودند — یک تصمیم آگاهانه‌ی موقتی که
-- خودِ CLAUDE.md ریشه‌ی پروژه آن را «مسیر بلندمدت: جداسازی فیزیکی» می‌نامید.
-- این فایل همان جداسازی را انجام می‌دهد — همچنان یک سرور/instance واحدِ
-- Postgres، ولی هر سرویس حالا دیتابیس مخصوص خودش را دارد.
--
-- نکته‌ی مهم: botmanager و apimanager عمداً دیتابیس‌شان را جدا نمی‌کنیم —
-- apimanager مستقیماً از shared-core/models/shared-core/store استفاده
-- می‌کند، یعنی دقیقاً همان جدول‌های botmanager (User, BotInstance, Plan, ...)
-- را می‌خواند/می‌نویسد؛ این دو، دو رابط (ربات تلگرام + HTTP API) روی یک
-- داده‌ی مشترکند، نه دو مالک داده‌ی جدا. جدا کردنشان یعنی apimanager دیگر
-- هیچ instance/کاربر/پلنی نمی‌بیند. بقیه‌ی سرویس‌ها هرکدام واقعاً جدول‌های
-- مستقل خودشان را دارند، پس جداسازی برایشان بی‌خطر و منطقی است.
--
-- این فایل فقط دیتابیس‌های خالی را می‌سازد؛ خودِ جدول‌ها را هر سرویس با
-- GORM AutoMigrate در startup خودش می‌سازد (همان الگویی که از قبل هم برای
-- botpay/ads-bot/community-service/revenue-service/license-service/
-- image-registry استفاده می‌شد) — نیازی به SQL دستی برای جدول‌ها نیست.
--
-- POSTGRES_DB این فایل خودش دیگر "creatorbot" نیست — به "botmanager" تغییر
-- کرده (رجوع docker-compose.yml)، پس این اسکریپت روی دیتابیس "botmanager"
-- اجرا می‌شود و از همان‌جا بقیه را می‌سازد.

CREATE DATABASE botpay;
CREATE DATABASE adsbot;
CREATE DATABASE community;
CREATE DATABASE revenue;
CREATE DATABASE license;
CREATE DATABASE imageregistry;
