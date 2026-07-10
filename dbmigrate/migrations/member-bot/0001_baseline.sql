-- 0001_baseline — schema کامل سرویس member-bot (دیتابیس: member_bot)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service member-bot -version 1


-- TABLE: bot_channel_memberships

CREATE TABLE public.bot_channel_memberships (
    bot_id uuid NOT NULL,
    channel_id bigint NOT NULL,
    joined_at timestamp with time zone,
    last_verified timestamp with time zone
);

-- TABLE: check_bots

CREATE TABLE public.check_bots (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    token text NOT NULL,
    username text,
    is_active boolean DEFAULT true,
    rate_limit bigint DEFAULT 20
);

-- TABLE: locks

CREATE TABLE public.locks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    owner_id text NOT NULL,
    channel_id bigint NOT NULL,
    channel_title text,
    max_members bigint DEFAULT 0,
    current_count bigint DEFAULT 0,
    duration_day bigint NOT NULL,
    price_per_day numeric NOT NULL,
    status text DEFAULT 'active'::text,
    expires_at timestamp with time zone
);

-- TABLE: member_verifications

CREATE TABLE public.member_verifications (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    lock_id text NOT NULL,
    user_id bigint NOT NULL,
    checked_by uuid,
    is_member boolean,
    checked_at timestamp with time zone
);

-- TABLE: owners

CREATE TABLE public.owners (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    telegram_id bigint NOT NULL,
    username text,
    first_name text,
    wallet_addr text,
    balance numeric DEFAULT 0,
    is_blocked boolean DEFAULT false
);

-- TABLE: payments

CREATE TABLE public.payments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    owner_id text NOT NULL,
    lock_id text NOT NULL,
    amount numeric,
    tx_hash text,
    status text DEFAULT 'pending'::text
);

-- TABLE: settings

CREATE TABLE public.settings (
    key text NOT NULL,
    value text
);

-- CONSTRAINT: bot_channel_memberships bot_channel_memberships_pkey

ALTER TABLE ONLY public.bot_channel_memberships
    ADD CONSTRAINT bot_channel_memberships_pkey PRIMARY KEY (bot_id, channel_id);

-- CONSTRAINT: check_bots check_bots_pkey

ALTER TABLE ONLY public.check_bots
    ADD CONSTRAINT check_bots_pkey PRIMARY KEY (id);

-- CONSTRAINT: locks locks_pkey

ALTER TABLE ONLY public.locks
    ADD CONSTRAINT locks_pkey PRIMARY KEY (id);

-- CONSTRAINT: member_verifications member_verifications_pkey

ALTER TABLE ONLY public.member_verifications
    ADD CONSTRAINT member_verifications_pkey PRIMARY KEY (id);

-- CONSTRAINT: owners owners_pkey

ALTER TABLE ONLY public.owners
    ADD CONSTRAINT owners_pkey PRIMARY KEY (id);

-- CONSTRAINT: payments payments_pkey

ALTER TABLE ONLY public.payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (id);

-- CONSTRAINT: settings settings_pkey

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (key);

-- INDEX: idx_bot_channel_memberships_bot_id

CREATE INDEX idx_bot_channel_memberships_bot_id ON public.bot_channel_memberships USING btree (bot_id);

-- INDEX: idx_bot_channel_memberships_channel_id

CREATE INDEX idx_bot_channel_memberships_channel_id ON public.bot_channel_memberships USING btree (channel_id);

-- INDEX: idx_check_bots_deleted_at

CREATE INDEX idx_check_bots_deleted_at ON public.check_bots USING btree (deleted_at);

-- INDEX: idx_locks_channel_id

CREATE UNIQUE INDEX idx_locks_channel_id ON public.locks USING btree (channel_id);

-- INDEX: idx_locks_deleted_at

CREATE INDEX idx_locks_deleted_at ON public.locks USING btree (deleted_at);

-- INDEX: idx_locks_owner_id

CREATE INDEX idx_locks_owner_id ON public.locks USING btree (owner_id);

-- INDEX: idx_member_verifications_deleted_at

CREATE INDEX idx_member_verifications_deleted_at ON public.member_verifications USING btree (deleted_at);

-- INDEX: idx_member_verifications_lock_id

CREATE INDEX idx_member_verifications_lock_id ON public.member_verifications USING btree (lock_id);

-- INDEX: idx_member_verifications_user_id

CREATE INDEX idx_member_verifications_user_id ON public.member_verifications USING btree (user_id);

-- INDEX: idx_owners_deleted_at

CREATE INDEX idx_owners_deleted_at ON public.owners USING btree (deleted_at);

-- INDEX: idx_owners_telegram_id

CREATE UNIQUE INDEX idx_owners_telegram_id ON public.owners USING btree (telegram_id);

-- INDEX: idx_payments_deleted_at

CREATE INDEX idx_payments_deleted_at ON public.payments USING btree (deleted_at);

-- INDEX: idx_payments_lock_id

CREATE INDEX idx_payments_lock_id ON public.payments USING btree (lock_id);

-- INDEX: idx_payments_owner_id

CREATE INDEX idx_payments_owner_id ON public.payments USING btree (owner_id);

-- FK CONSTRAINT: bot_channel_memberships fk_check_bots_memberships

ALTER TABLE ONLY public.bot_channel_memberships
    ADD CONSTRAINT fk_check_bots_memberships FOREIGN KEY (bot_id) REFERENCES public.check_bots(id);

-- PostgreSQL database dump complete

