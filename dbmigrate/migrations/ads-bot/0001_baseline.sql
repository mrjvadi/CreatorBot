-- 0001_baseline — schema کامل سرویس ads-bot (دیتابیس: adsbot)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service ads-bot -version 1


-- TABLE: ad_channels

CREATE TABLE public.ad_channels (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    owner_id text NOT NULL,
    category_id uuid,
    channel_id bigint NOT NULL,
    channel_name text,
    channel_username text,
    member_count bigint DEFAULT 0,
    status text DEFAULT 'pending'::text,
    is_active boolean DEFAULT true,
    score bigint DEFAULT 0,
    fake_percent numeric DEFAULT 0,
    real_members bigint DEFAULT 0,
    effective_cpj numeric DEFAULT 0,
    last_analyzed_at timestamp with time zone,
    total_impressions bigint DEFAULT 0,
    total_earned numeric DEFAULT 0
);

-- TABLE: ad_configs

CREATE TABLE public.ad_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    updated_at timestamp with time zone,
    base_cpj numeric DEFAULT 0.005 NOT NULL,
    min_channel_score bigint DEFAULT 30,
    max_fake_percent numeric DEFAULT 30,
    platform_commission numeric DEFAULT 20,
    is_active boolean DEFAULT true
);

-- TABLE: campaigns

CREATE TABLE public.campaigns (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    publisher_id text NOT NULL,
    name text NOT NULL,
    status text DEFAULT 'draft'::text,
    target_category_id uuid,
    min_channel_score bigint,
    media_file_id text,
    media_type text,
    caption text,
    button_text text,
    button_url text,
    budget numeric NOT NULL,
    spent numeric DEFAULT 0,
    cpj numeric NOT NULL,
    total_joins bigint DEFAULT 0,
    real_joins bigint DEFAULT 0,
    target_count bigint DEFAULT 0,
    start_at timestamp with time zone,
    end_at timestamp with time zone,
    review_note text,
    reviewed_at timestamp with time zone,
    reviewer_id bigint
);

-- TABLE: channel_categories

CREATE TABLE public.channel_categories (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    name text NOT NULL,
    label text,
    cpj_multiplier numeric DEFAULT 1,
    is_active boolean DEFAULT true
);

-- TABLE: free_bot_owner_rewards

CREATE TABLE public.free_bot_owner_rewards (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    rental_id text,
    slot_id text,
    owner_telegram_id bigint,
    amount_ton numeric,
    status text DEFAULT 'pending'::text,
    settle_at timestamp with time zone,
    settled_at timestamp with time zone
);

-- TABLE: free_bot_slots

CREATE TABLE public.free_bot_slots (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    bot_instance_id text NOT NULL,
    bot_id bigint NOT NULL,
    rental_id text,
    assigned_owner_telegram_id bigint,
    is_channel_admin_confirmed boolean DEFAULT false
);

-- TABLE: impressions

CREATE TABLE public.impressions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    campaign_id text NOT NULL,
    channel_id bigint NOT NULL,
    message_id bigint,
    total_joins bigint DEFAULT 0,
    real_joins bigint DEFAULT 0,
    fake_joins bigint DEFAULT 0,
    cost numeric DEFAULT 0
);

-- TABLE: lock_rental_campaigns

CREATE TABLE public.lock_rental_campaigns (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    buyer_telegram_id bigint NOT NULL,
    target_channel_id bigint NOT NULL,
    target_channel_username text,
    status text DEFAULT 'pending_review'::text,
    review_note text,
    reviewed_at timestamp with time zone,
    reviewer_id bigint,
    reward_per_join_ton numeric NOT NULL,
    budget numeric NOT NULL,
    spent numeric DEFAULT 0,
    free_bot_owner_reward_percent numeric DEFAULT 5,
    total_joins bigint DEFAULT 0,
    real_joins bigint DEFAULT 0,
    start_at timestamp with time zone,
    end_at timestamp with time zone
);

-- TABLE: member_analyses

CREATE TABLE public.member_analyses (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    channel_id bigint NOT NULL,
    telegram_id bigint NOT NULL,
    has_username boolean DEFAULT false,
    has_profile_photo boolean DEFAULT false,
    account_age bigint,
    is_bot boolean DEFAULT false,
    real_score bigint DEFAULT 0,
    is_fake boolean DEFAULT false,
    analyzed_at timestamp with time zone
);

-- TABLE: publishers

CREATE TABLE public.publishers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    telegram_id bigint NOT NULL,
    username text,
    first_name text,
    balance numeric DEFAULT 0,
    is_blocked boolean DEFAULT false
);

-- TABLE: rental_join_rewards

CREATE TABLE public.rental_join_rewards (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    rental_id text,
    telegram_id bigint,
    amount_ton numeric,
    status text DEFAULT 'pending'::text,
    settle_at timestamp with time zone,
    settled_at timestamp with time zone
);

-- CONSTRAINT: ad_channels ad_channels_pkey

ALTER TABLE ONLY public.ad_channels
    ADD CONSTRAINT ad_channels_pkey PRIMARY KEY (id);

-- CONSTRAINT: ad_configs ad_configs_pkey

ALTER TABLE ONLY public.ad_configs
    ADD CONSTRAINT ad_configs_pkey PRIMARY KEY (id);

-- CONSTRAINT: campaigns campaigns_pkey

ALTER TABLE ONLY public.campaigns
    ADD CONSTRAINT campaigns_pkey PRIMARY KEY (id);

-- CONSTRAINT: channel_categories channel_categories_pkey

ALTER TABLE ONLY public.channel_categories
    ADD CONSTRAINT channel_categories_pkey PRIMARY KEY (id);

-- CONSTRAINT: free_bot_owner_rewards free_bot_owner_rewards_pkey

ALTER TABLE ONLY public.free_bot_owner_rewards
    ADD CONSTRAINT free_bot_owner_rewards_pkey PRIMARY KEY (id);

-- CONSTRAINT: free_bot_slots free_bot_slots_pkey

ALTER TABLE ONLY public.free_bot_slots
    ADD CONSTRAINT free_bot_slots_pkey PRIMARY KEY (id);

-- CONSTRAINT: impressions impressions_pkey

ALTER TABLE ONLY public.impressions
    ADD CONSTRAINT impressions_pkey PRIMARY KEY (id);

-- CONSTRAINT: lock_rental_campaigns lock_rental_campaigns_pkey

ALTER TABLE ONLY public.lock_rental_campaigns
    ADD CONSTRAINT lock_rental_campaigns_pkey PRIMARY KEY (id);

-- CONSTRAINT: member_analyses member_analyses_pkey

ALTER TABLE ONLY public.member_analyses
    ADD CONSTRAINT member_analyses_pkey PRIMARY KEY (id);

-- CONSTRAINT: publishers publishers_pkey

ALTER TABLE ONLY public.publishers
    ADD CONSTRAINT publishers_pkey PRIMARY KEY (id);

-- CONSTRAINT: rental_join_rewards rental_join_rewards_pkey

ALTER TABLE ONLY public.rental_join_rewards
    ADD CONSTRAINT rental_join_rewards_pkey PRIMARY KEY (id);

-- INDEX: idx_ad_channels_category_id

CREATE INDEX idx_ad_channels_category_id ON public.ad_channels USING btree (category_id);

-- INDEX: idx_ad_channels_channel_id

CREATE UNIQUE INDEX idx_ad_channels_channel_id ON public.ad_channels USING btree (channel_id);

-- INDEX: idx_ad_channels_owner_id

CREATE INDEX idx_ad_channels_owner_id ON public.ad_channels USING btree (owner_id);

-- INDEX: idx_ad_channels_status

CREATE INDEX idx_ad_channels_status ON public.ad_channels USING btree (status);

-- INDEX: idx_campaigns_publisher_id

CREATE INDEX idx_campaigns_publisher_id ON public.campaigns USING btree (publisher_id);

-- INDEX: idx_campaigns_status

CREATE INDEX idx_campaigns_status ON public.campaigns USING btree (status);

-- INDEX: idx_channel_categories_name

CREATE UNIQUE INDEX idx_channel_categories_name ON public.channel_categories USING btree (name);

-- INDEX: idx_free_bot_owner_rewards_settle_at

CREATE INDEX idx_free_bot_owner_rewards_settle_at ON public.free_bot_owner_rewards USING btree (settle_at);

-- INDEX: idx_free_bot_owner_rewards_status

CREATE INDEX idx_free_bot_owner_rewards_status ON public.free_bot_owner_rewards USING btree (status);

-- INDEX: idx_free_bot_slots_bot_id

CREATE UNIQUE INDEX idx_free_bot_slots_bot_id ON public.free_bot_slots USING btree (bot_id);

-- INDEX: idx_free_bot_slots_bot_instance_id

CREATE UNIQUE INDEX idx_free_bot_slots_bot_instance_id ON public.free_bot_slots USING btree (bot_instance_id);

-- INDEX: idx_free_bot_slots_rental_id

CREATE INDEX idx_free_bot_slots_rental_id ON public.free_bot_slots USING btree (rental_id);

-- INDEX: idx_impressions_campaign_id

CREATE INDEX idx_impressions_campaign_id ON public.impressions USING btree (campaign_id);

-- INDEX: idx_impressions_channel_id

CREATE INDEX idx_impressions_channel_id ON public.impressions USING btree (channel_id);

-- INDEX: idx_lock_rental_campaigns_buyer_telegram_id

CREATE INDEX idx_lock_rental_campaigns_buyer_telegram_id ON public.lock_rental_campaigns USING btree (buyer_telegram_id);

-- INDEX: idx_lock_rental_campaigns_status

CREATE INDEX idx_lock_rental_campaigns_status ON public.lock_rental_campaigns USING btree (status);

-- INDEX: idx_lock_rental_campaigns_target_channel_id

CREATE INDEX idx_lock_rental_campaigns_target_channel_id ON public.lock_rental_campaigns USING btree (target_channel_id);

-- INDEX: idx_member_analyses_channel_id

CREATE INDEX idx_member_analyses_channel_id ON public.member_analyses USING btree (channel_id);

-- INDEX: idx_member_analyses_telegram_id

CREATE INDEX idx_member_analyses_telegram_id ON public.member_analyses USING btree (telegram_id);

-- INDEX: idx_publishers_telegram_id

CREATE UNIQUE INDEX idx_publishers_telegram_id ON public.publishers USING btree (telegram_id);

-- INDEX: idx_rental_join_rewards_settle_at

CREATE INDEX idx_rental_join_rewards_settle_at ON public.rental_join_rewards USING btree (settle_at);

-- INDEX: idx_rental_join_rewards_status

CREATE INDEX idx_rental_join_rewards_status ON public.rental_join_rewards USING btree (status);

-- INDEX: idx_rental_slot

CREATE UNIQUE INDEX idx_rental_slot ON public.free_bot_owner_rewards USING btree (rental_id, slot_id);

-- INDEX: idx_rental_user

CREATE UNIQUE INDEX idx_rental_user ON public.rental_join_rewards USING btree (rental_id, telegram_id);

-- FK CONSTRAINT: ad_channels fk_ad_channels_category

ALTER TABLE ONLY public.ad_channels
    ADD CONSTRAINT fk_ad_channels_category FOREIGN KEY (category_id) REFERENCES public.channel_categories(id);

-- FK CONSTRAINT: campaigns fk_campaigns_target_category

ALTER TABLE ONLY public.campaigns
    ADD CONSTRAINT fk_campaigns_target_category FOREIGN KEY (target_category_id) REFERENCES public.channel_categories(id);

-- PostgreSQL database dump complete

