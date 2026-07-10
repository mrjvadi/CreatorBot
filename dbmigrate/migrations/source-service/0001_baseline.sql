-- 0001_baseline — schema کامل سرویس source-service (دیتابیس: source_svc)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service source-service -version 1


-- TABLE: archive_files

CREATE TABLE public.archive_files (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    message_id bigint NOT NULL,
    file_type text NOT NULL,
    file_name text,
    mime_type text,
    file_size bigint,
    caption text
);

-- TABLE: bot_file_caches

CREATE TABLE public.bot_file_caches (
    archive_file_id uuid NOT NULL,
    bot_token_hash text NOT NULL,
    file_id text NOT NULL,
    cached_at timestamp with time zone
);

-- TABLE: channel_watches

CREATE TABLE public.channel_watches (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    phone text NOT NULL,
    source_channel text NOT NULL,
    dest_channel text NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- TABLE: nats_watches

CREATE TABLE public.nats_watches (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    phone text NOT NULL,
    subject text NOT NULL,
    dest_channel text NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- TABLE: rules

CREATE TABLE public.rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    phone text NOT NULL,
    trigger_type text NOT NULL,
    trigger jsonb NOT NULL,
    conditions jsonb,
    action_type text NOT NULL,
    action jsonb NOT NULL,
    active boolean DEFAULT true NOT NULL
);

-- TABLE: telegram_sessions

CREATE TABLE public.telegram_sessions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    phone text NOT NULL,
    encrypted bytea NOT NULL
);

-- CONSTRAINT: archive_files archive_files_pkey

ALTER TABLE ONLY public.archive_files
    ADD CONSTRAINT archive_files_pkey PRIMARY KEY (id);

-- CONSTRAINT: bot_file_caches bot_file_caches_pkey

ALTER TABLE ONLY public.bot_file_caches
    ADD CONSTRAINT bot_file_caches_pkey PRIMARY KEY (archive_file_id, bot_token_hash);

-- CONSTRAINT: channel_watches channel_watches_pkey

ALTER TABLE ONLY public.channel_watches
    ADD CONSTRAINT channel_watches_pkey PRIMARY KEY (id);

-- CONSTRAINT: nats_watches nats_watches_pkey

ALTER TABLE ONLY public.nats_watches
    ADD CONSTRAINT nats_watches_pkey PRIMARY KEY (id);

-- CONSTRAINT: rules rules_pkey

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_pkey PRIMARY KEY (id);

-- CONSTRAINT: telegram_sessions telegram_sessions_pkey

ALTER TABLE ONLY public.telegram_sessions
    ADD CONSTRAINT telegram_sessions_pkey PRIMARY KEY (id);

-- INDEX: idx_archive_files_deleted_at

CREATE INDEX idx_archive_files_deleted_at ON public.archive_files USING btree (deleted_at);

-- INDEX: idx_channel_watches_deleted_at

CREATE INDEX idx_channel_watches_deleted_at ON public.channel_watches USING btree (deleted_at);

-- INDEX: idx_channel_watches_phone

CREATE INDEX idx_channel_watches_phone ON public.channel_watches USING btree (phone);

-- INDEX: idx_nats_watches_deleted_at

CREATE INDEX idx_nats_watches_deleted_at ON public.nats_watches USING btree (deleted_at);

-- INDEX: idx_nats_watches_phone

CREATE INDEX idx_nats_watches_phone ON public.nats_watches USING btree (phone);

-- INDEX: idx_rules_deleted_at

CREATE INDEX idx_rules_deleted_at ON public.rules USING btree (deleted_at);

-- INDEX: idx_rules_phone

CREATE INDEX idx_rules_phone ON public.rules USING btree (phone);

-- INDEX: idx_telegram_sessions_deleted_at

CREATE INDEX idx_telegram_sessions_deleted_at ON public.telegram_sessions USING btree (deleted_at);

-- INDEX: idx_telegram_sessions_phone

CREATE UNIQUE INDEX idx_telegram_sessions_phone ON public.telegram_sessions USING btree (phone);

-- PostgreSQL database dump complete

