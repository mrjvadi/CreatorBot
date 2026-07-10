-- 0001_baseline — schema کامل سرویس vpn-bot (دیتابیس: vpn_bot)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service vpn-bot -version 1


-- TABLE: discount_codes

CREATE TABLE public.discount_codes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    code text NOT NULL,
    percent numeric NOT NULL,
    max_use bigint DEFAULT 1,
    used_count bigint DEFAULT 0,
    is_active boolean DEFAULT true
);

-- TABLE: panels

CREATE TABLE public.panels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL,
    type text NOT NULL,
    base_url text NOT NULL,
    username text,
    password text,
    capacity bigint DEFAULT 0,
    active_count bigint DEFAULT 0,
    is_active boolean DEFAULT true
);

-- TABLE: payments

CREATE TABLE public.payments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    user_id text NOT NULL,
    amount numeric,
    gateway text,
    status text DEFAULT 'pending'::text,
    ref_code text,
    receipt text,
    plan_id text
);

-- TABLE: plans

CREATE TABLE public.plans (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL,
    duration_day bigint NOT NULL,
    data_gb numeric DEFAULT 0,
    price numeric NOT NULL,
    is_active boolean DEFAULT true
);

-- TABLE: settings

CREATE TABLE public.settings (
    key text NOT NULL,
    value text
);

-- TABLE: subscriptions

CREATE TABLE public.subscriptions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    user_id uuid NOT NULL,
    panel_id text NOT NULL,
    plan_id text NOT NULL,
    username text,
    status text DEFAULT 'active'::text,
    expires_at timestamp with time zone,
    data_limit numeric,
    used_data numeric
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
    balance numeric DEFAULT 0,
    is_blocked boolean DEFAULT false,
    reseller_id uuid,
    discount numeric DEFAULT 0
);

-- CONSTRAINT: discount_codes discount_codes_pkey

ALTER TABLE ONLY public.discount_codes
    ADD CONSTRAINT discount_codes_pkey PRIMARY KEY (id);

-- CONSTRAINT: panels panels_pkey

ALTER TABLE ONLY public.panels
    ADD CONSTRAINT panels_pkey PRIMARY KEY (id);

-- CONSTRAINT: payments payments_pkey

ALTER TABLE ONLY public.payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (id);

-- CONSTRAINT: plans plans_pkey

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT plans_pkey PRIMARY KEY (id);

-- CONSTRAINT: settings settings_pkey

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (key);

-- CONSTRAINT: subscriptions subscriptions_pkey

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_pkey PRIMARY KEY (id);

-- CONSTRAINT: users users_pkey

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

-- INDEX: idx_discount_codes_code

CREATE UNIQUE INDEX idx_discount_codes_code ON public.discount_codes USING btree (code);

-- INDEX: idx_discount_codes_deleted_at

CREATE INDEX idx_discount_codes_deleted_at ON public.discount_codes USING btree (deleted_at);

-- INDEX: idx_panels_deleted_at

CREATE INDEX idx_panels_deleted_at ON public.panels USING btree (deleted_at);

-- INDEX: idx_payments_deleted_at

CREATE INDEX idx_payments_deleted_at ON public.payments USING btree (deleted_at);

-- INDEX: idx_payments_user_id

CREATE INDEX idx_payments_user_id ON public.payments USING btree (user_id);

-- INDEX: idx_plans_deleted_at

CREATE INDEX idx_plans_deleted_at ON public.plans USING btree (deleted_at);

-- INDEX: idx_subscriptions_deleted_at

CREATE INDEX idx_subscriptions_deleted_at ON public.subscriptions USING btree (deleted_at);

-- INDEX: idx_subscriptions_panel_id

CREATE INDEX idx_subscriptions_panel_id ON public.subscriptions USING btree (panel_id);

-- INDEX: idx_subscriptions_user_id

CREATE INDEX idx_subscriptions_user_id ON public.subscriptions USING btree (user_id);

-- INDEX: idx_users_deleted_at

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);

-- INDEX: idx_users_reseller_id

CREATE INDEX idx_users_reseller_id ON public.users USING btree (reseller_id);

-- INDEX: idx_users_telegram_id

CREATE UNIQUE INDEX idx_users_telegram_id ON public.users USING btree (telegram_id);

-- FK CONSTRAINT: subscriptions fk_subscriptions_user

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT fk_subscriptions_user FOREIGN KEY (user_id) REFERENCES public.users(id);

-- PostgreSQL database dump complete

