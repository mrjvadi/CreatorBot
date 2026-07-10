-- 0003_drop_legacy_unique_constraints — سرویس botmanager (دیتابیس: botmanager)
--
-- ادامه‌ی 0002: همان مشکل روی بقیه‌ی جدول‌ها. schema قدیمیِ زنده یکتایی را
-- با UNIQUE CONSTRAINT های قرارداد قدیمی Postgres (*_key) دارد، ولی مدل‌های
-- فعلی UNIQUE INDEX های نام‌دار می‌خواهند — و برای bot_instances ایندکسِ
-- partial با «WHERE deleted_at IS NULL» (باگ واقعی ۲۰۲۶-۰۷-۰۵: ردیف
-- soft-delete شده نباید bot_id/container_name را برای همیشه اشغال کند —
-- رجوع کامنت DBSchema در shared-core/models/models.go که صریحاً می‌گوید
-- این تغییر باید دستی migrate شود چون AutoMigrate ایندکس موجود را عوض
-- نمی‌کند). GORM هم موقع AutoMigrate سعی می‌کند این constraint ها را با
-- نامی که خودش انتظار دارد (uni_*) حذف کند، پیدا نمی‌کند و کل migration
-- همان‌جا قطع می‌شود.
--
-- تعریف‌های CREATE دقیقاً از baseline (0001) کپی شده‌اند.

-- servers: name دیگر unique نیست؛ ip با ایندکس نام‌دار unique می‌شود.
ALTER TABLE servers DROP CONSTRAINT IF EXISTS servers_name_key;
ALTER TABLE servers DROP CONSTRAINT IF EXISTS servers_ip_key;
DROP INDEX IF EXISTS idx_servers_ip;
CREATE UNIQUE INDEX idx_servers_ip ON servers USING btree (ip);

-- bot_instances: یکتایی فقط روی ردیف‌های زنده (partial، ضد soft-delete).
ALTER TABLE bot_instances DROP CONSTRAINT IF EXISTS bot_instances_bot_id_key;
ALTER TABLE bot_instances DROP CONSTRAINT IF EXISTS bot_instances_container_name_key;
DROP INDEX IF EXISTS idx_bot_instances_bot_id;
DROP INDEX IF EXISTS idx_bot_instances_container_name;
CREATE UNIQUE INDEX idx_bot_instances_bot_id ON bot_instances USING btree (bot_id) WHERE (deleted_at IS NULL);
CREATE UNIQUE INDEX idx_bot_instances_container_name ON bot_instances USING btree (container_name) WHERE (deleted_at IS NULL);

-- invite_links
ALTER TABLE invite_links DROP CONSTRAINT IF EXISTS invite_links_token_key;
DROP INDEX IF EXISTS idx_invite_links_token;
CREATE UNIQUE INDEX idx_invite_links_token ON invite_links USING btree (token);
