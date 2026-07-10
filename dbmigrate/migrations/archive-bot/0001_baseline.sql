-- 0001_baseline — schema کامل سرویس archive-bot (دیتابیس: archive_bot)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service archive-bot -version 1


-- EXTENSION: pg_trgm

CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public;

-- COMMENT: EXTENSION pg_trgm

COMMENT ON EXTENSION pg_trgm IS 'text similarity measurement and index searching based on trigrams';

-- TABLE: categories

CREATE TABLE public.categories (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL
);

-- TABLE: files

CREATE TABLE public.files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    file_id text NOT NULL,
    file_type text NOT NULL,
    title text NOT NULL,
    tags text,
    description text,
    category_id uuid,
    uploader_id bigint
);

-- TABLE: settings

CREATE TABLE public.settings (
    key text NOT NULL,
    value text
);

-- TABLE: users

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    telegram_id bigint NOT NULL,
    username text,
    first_name text,
    is_blocked boolean DEFAULT false
);

-- CONSTRAINT: categories categories_pkey

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_pkey PRIMARY KEY (id);

-- CONSTRAINT: files files_pkey

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_pkey PRIMARY KEY (id);

-- CONSTRAINT: settings settings_pkey

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (key);

-- CONSTRAINT: users users_pkey

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

-- INDEX: idx_categories_deleted_at

CREATE INDEX idx_categories_deleted_at ON public.categories USING btree (deleted_at);

-- INDEX: idx_categories_name

CREATE UNIQUE INDEX idx_categories_name ON public.categories USING btree (name);

-- INDEX: idx_files_category_id

CREATE INDEX idx_files_category_id ON public.files USING btree (category_id);

-- INDEX: idx_files_deleted_at

CREATE INDEX idx_files_deleted_at ON public.files USING btree (deleted_at);

-- INDEX: idx_files_trgm

CREATE INDEX idx_files_trgm ON public.files USING gin ((((((title || ' '::text) || tags) || ' '::text) || description)) public.gin_trgm_ops);

-- INDEX: idx_users_deleted_at

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);

-- INDEX: idx_users_telegram_id

CREATE UNIQUE INDEX idx_users_telegram_id ON public.users USING btree (telegram_id);

-- FK CONSTRAINT: files fk_categories_files

ALTER TABLE ONLY public.files
    ADD CONSTRAINT fk_categories_files FOREIGN KEY (category_id) REFERENCES public.categories(id);

-- PostgreSQL database dump complete

