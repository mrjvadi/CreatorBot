-- 0001_baseline — schema کامل سرویس image-registry (دیتابیس: imageregistry)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service image-registry -version 1


-- TABLE: allowed_callers

CREATE TABLE public.allowed_callers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    label text NOT NULL,
    c_id_r text NOT NULL,
    domain text,
    can_write boolean DEFAULT false,
    is_active boolean DEFAULT true
);

-- TABLE: registered_images

CREATE TABLE public.registered_images (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    name text NOT NULL,
    tag text NOT NULL,
    service_type text,
    description text,
    is_active boolean DEFAULT true,
    file_path text,
    file_sha256 text,
    file_size bigint
);

-- CONSTRAINT: allowed_callers allowed_callers_pkey

ALTER TABLE ONLY public.allowed_callers
    ADD CONSTRAINT allowed_callers_pkey PRIMARY KEY (id);

-- CONSTRAINT: registered_images registered_images_pkey

ALTER TABLE ONLY public.registered_images
    ADD CONSTRAINT registered_images_pkey PRIMARY KEY (id);

-- INDEX: idx_allowed_callers_is_active

CREATE INDEX idx_allowed_callers_is_active ON public.allowed_callers USING btree (is_active);

-- INDEX: idx_name_tag

CREATE UNIQUE INDEX idx_name_tag ON public.registered_images USING btree (name, tag);

-- INDEX: idx_registered_images_is_active

CREATE INDEX idx_registered_images_is_active ON public.registered_images USING btree (is_active);

-- PostgreSQL database dump complete

