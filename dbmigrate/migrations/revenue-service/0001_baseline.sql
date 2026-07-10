-- 0001_baseline — schema کامل سرویس revenue-service (دیتابیس: revenue)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service revenue-service -version 1


-- TABLE: earnings

CREATE TABLE public.earnings (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    type text NOT NULL,
    total_nano bigint NOT NULL,
    owner_telegram_id bigint NOT NULL,
    bot_id text,
    ref_id text,
    description text,
    status text DEFAULT 'pending'::text,
    owner_nano bigint,
    platform_nano bigint,
    owner_tx_id text,
    platform_tx_id text,
    processed_at timestamp with time zone,
    error text
);

-- TABLE: platform_wallets

CREATE TABLE public.platform_wallets (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    telegram_id bigint NOT NULL,
    label text,
    is_default boolean DEFAULT false,
    created_at timestamp with time zone
);

-- TABLE: revenue_rules

CREATE TABLE public.revenue_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    type text NOT NULL,
    owner_percent numeric NOT NULL,
    platform_percent numeric NOT NULL,
    is_active boolean DEFAULT true,
    description text
);

-- CONSTRAINT: earnings earnings_pkey

ALTER TABLE ONLY public.earnings
    ADD CONSTRAINT earnings_pkey PRIMARY KEY (id);

-- CONSTRAINT: platform_wallets platform_wallets_pkey

ALTER TABLE ONLY public.platform_wallets
    ADD CONSTRAINT platform_wallets_pkey PRIMARY KEY (id);

-- CONSTRAINT: revenue_rules revenue_rules_pkey

ALTER TABLE ONLY public.revenue_rules
    ADD CONSTRAINT revenue_rules_pkey PRIMARY KEY (id);

-- INDEX: idx_earnings_bot_id

CREATE INDEX idx_earnings_bot_id ON public.earnings USING btree (bot_id);

-- INDEX: idx_earnings_owner_telegram_id

CREATE INDEX idx_earnings_owner_telegram_id ON public.earnings USING btree (owner_telegram_id);

-- INDEX: idx_earnings_ref_id

CREATE INDEX idx_earnings_ref_id ON public.earnings USING btree (ref_id);

-- INDEX: idx_earnings_status

CREATE INDEX idx_earnings_status ON public.earnings USING btree (status);

-- INDEX: idx_earnings_type

CREATE INDEX idx_earnings_type ON public.earnings USING btree (type);

-- INDEX: idx_platform_wallets_telegram_id

CREATE UNIQUE INDEX idx_platform_wallets_telegram_id ON public.platform_wallets USING btree (telegram_id);

-- INDEX: idx_revenue_rules_type

CREATE UNIQUE INDEX idx_revenue_rules_type ON public.revenue_rules USING btree (type);

-- PostgreSQL database dump complete

