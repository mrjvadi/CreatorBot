-- 0001_baseline — schema کامل سرویس license-service (دیتابیس: license)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service license-service -version 1


-- TABLE: licenses

CREATE TABLE public.licenses (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    bot_id bigint NOT NULL,
    instance_id text,
    owner_id text,
    plan_id text,
    token_hash text,
    known_server_id text,
    status text DEFAULT 'active'::text,
    revoked_reason text,
    expires_at timestamp with time zone,
    last_checkin_at timestamp with time zone,
    last_server_seen text,
    clone_flag_count bigint DEFAULT 0
);

-- CONSTRAINT: licenses licenses_pkey

ALTER TABLE ONLY public.licenses
    ADD CONSTRAINT licenses_pkey PRIMARY KEY (id);

-- INDEX: idx_licenses_bot_id

CREATE UNIQUE INDEX idx_licenses_bot_id ON public.licenses USING btree (bot_id);

-- INDEX: idx_licenses_instance_id

CREATE INDEX idx_licenses_instance_id ON public.licenses USING btree (instance_id);

-- INDEX: idx_licenses_owner_id

CREATE INDEX idx_licenses_owner_id ON public.licenses USING btree (owner_id);

-- INDEX: idx_licenses_status

CREATE INDEX idx_licenses_status ON public.licenses USING btree (status);

-- INDEX: idx_licenses_token_hash

CREATE INDEX idx_licenses_token_hash ON public.licenses USING btree (token_hash);

-- PostgreSQL database dump complete

