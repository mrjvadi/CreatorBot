-- 0001_baseline — schema کامل سرویس community-service (دیتابیس: community)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service community-service -version 1


-- TABLE: campaign_participants

CREATE TABLE public.campaign_participants (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    campaign_id text NOT NULL,
    community_id text NOT NULL,
    telegram_id bigint NOT NULL,
    joined_at timestamp with time zone,
    validated_at timestamp with time zone,
    left_at timestamp with time zone,
    status text DEFAULT 'pending'::text,
    revenue_earned numeric DEFAULT 0
);

-- TABLE: communities

CREATE TABLE public.communities (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    owner_id text NOT NULL,
    telegram_id bigint NOT NULL,
    type text NOT NULL,
    name text NOT NULL,
    username text,
    status text DEFAULT 'pending'::text,
    invite_link text,
    invite_hash text,
    owner_percent numeric DEFAULT 0,
    members_percent numeric DEFAULT 0,
    platform_percent numeric DEFAULT 0,
    member_count bigint DEFAULT 0,
    quality_score bigint DEFAULT 50,
    validation_window_sec bigint DEFAULT 86400,
    verified_at timestamp with time zone
);

-- TABLE: community_distributions

CREATE TABLE public.community_distributions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    revenue_id text NOT NULL,
    community_id text NOT NULL,
    telegram_id bigint NOT NULL,
    amount numeric,
    activity_score bigint,
    tx_id text,
    status text DEFAULT 'pending'::text
);

-- TABLE: community_revenues

CREATE TABLE public.community_revenues (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    community_id text NOT NULL,
    campaign_id text NOT NULL,
    total_amount numeric,
    owner_amount numeric,
    members_amount numeric,
    platform_amount numeric,
    valid_joins bigint,
    status text DEFAULT 'pending'::text,
    distributed_at timestamp with time zone
);

-- CONSTRAINT: campaign_participants campaign_participants_pkey

ALTER TABLE ONLY public.campaign_participants
    ADD CONSTRAINT campaign_participants_pkey PRIMARY KEY (id);

-- CONSTRAINT: communities communities_pkey

ALTER TABLE ONLY public.communities
    ADD CONSTRAINT communities_pkey PRIMARY KEY (id);

-- CONSTRAINT: community_distributions community_distributions_pkey

ALTER TABLE ONLY public.community_distributions
    ADD CONSTRAINT community_distributions_pkey PRIMARY KEY (id);

-- CONSTRAINT: community_revenues community_revenues_pkey

ALTER TABLE ONLY public.community_revenues
    ADD CONSTRAINT community_revenues_pkey PRIMARY KEY (id);

-- INDEX: idx_camp_user

CREATE UNIQUE INDEX idx_camp_user ON public.campaign_participants USING btree (campaign_id, telegram_id);

-- INDEX: idx_campaign_participants_campaign_id

CREATE INDEX idx_campaign_participants_campaign_id ON public.campaign_participants USING btree (campaign_id);

-- INDEX: idx_campaign_participants_community_id

CREATE INDEX idx_campaign_participants_community_id ON public.campaign_participants USING btree (community_id);

-- INDEX: idx_campaign_participants_telegram_id

CREATE INDEX idx_campaign_participants_telegram_id ON public.campaign_participants USING btree (telegram_id);

-- INDEX: idx_communities_deleted_at

CREATE INDEX idx_communities_deleted_at ON public.communities USING btree (deleted_at);

-- INDEX: idx_communities_invite_hash

CREATE UNIQUE INDEX idx_communities_invite_hash ON public.communities USING btree (invite_hash);

-- INDEX: idx_communities_invite_link

CREATE UNIQUE INDEX idx_communities_invite_link ON public.communities USING btree (invite_link);

-- INDEX: idx_communities_owner_id

CREATE INDEX idx_communities_owner_id ON public.communities USING btree (owner_id);

-- INDEX: idx_communities_status

CREATE INDEX idx_communities_status ON public.communities USING btree (status);

-- INDEX: idx_communities_telegram_id

CREATE UNIQUE INDEX idx_communities_telegram_id ON public.communities USING btree (telegram_id);

-- INDEX: idx_community_distributions_community_id

CREATE INDEX idx_community_distributions_community_id ON public.community_distributions USING btree (community_id);

-- INDEX: idx_community_distributions_revenue_id

CREATE INDEX idx_community_distributions_revenue_id ON public.community_distributions USING btree (revenue_id);

-- INDEX: idx_community_distributions_telegram_id

CREATE INDEX idx_community_distributions_telegram_id ON public.community_distributions USING btree (telegram_id);

-- INDEX: idx_community_revenues_campaign_id

CREATE INDEX idx_community_revenues_campaign_id ON public.community_revenues USING btree (campaign_id);

-- INDEX: idx_community_revenues_community_id

CREATE INDEX idx_community_revenues_community_id ON public.community_revenues USING btree (community_id);

-- PostgreSQL database dump complete

