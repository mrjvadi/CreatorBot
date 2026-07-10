-- 0001_baseline — schema کامل سرویس botmanager (دیتابیس: botmanager)
--
-- تولیدشده ۲۰۲۶-۰۷-۱۰ از اجرای واقعیِ AutoMigrate خود سرویس روی یک
-- دیتابیس خالی و سپس pg_dump --schema-only — یعنی دقیقاً همان schema ای
-- که GORM از مدل‌های فعلی می‌سازد، نه بازنویسی دستی.
--
-- این baseline برای دیتابیس تازه است. اگر دیتابیس از قبل schema دارد
-- (مثل botmanager/botpay فعلی)، به‌جای اجرا با دستور mark ثبتش کنید:
--   dbmigrate mark -service botmanager -version 1


-- TABLE: audit_logs

CREATE TABLE public.audit_logs (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    actor_id uuid,
    actor_role text,
    action text NOT NULL,
    target_id text,
    target_type text,
    description text,
    ip_address text,
    extra text
);

-- SEQUENCE: audit_logs_id_seq

CREATE SEQUENCE public.audit_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

-- SEQUENCE: audit_logs_id_seq

ALTER SEQUENCE public.audit_logs_id_seq OWNED BY public.audit_logs.id;

-- TABLE: bot_instances

CREATE TABLE public.bot_instances (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    owner_id text NOT NULL,
    template_id text NOT NULL,
    server_id text NOT NULL,
    container_id text,
    container_name text,
    bot_token text,
    bot_id bigint NOT NULL,
    plan_id text,
    lock_mode text DEFAULT 'none'::text,
    status text DEFAULT 'pending'::text,
    expires_at timestamp with time zone,
    db_schema text,
    env_overrides text
);

-- TABLE: bot_templates

CREATE TABLE public.bot_templates (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL,
    type text NOT NULL,
    image_name text NOT NULL,
    image_tag text NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    is_free boolean DEFAULT false,
    config_schema text
);

-- TABLE: deploy_jobs

CREATE TABLE public.deploy_jobs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    instance_id text NOT NULL,
    server_id text NOT NULL,
    status text DEFAULT 'pending'::text,
    priority bigint DEFAULT 0,
    attempts bigint DEFAULT 0,
    max_attempts bigint DEFAULT 3,
    scheduled_at timestamp with time zone,
    started_at timestamp with time zone,
    finished_at timestamp with time zone,
    error text
);

-- TABLE: invite_links

CREATE TABLE public.invite_links (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    token text NOT NULL,
    bot_type text NOT NULL,
    label text,
    max_use bigint DEFAULT 1,
    used_count bigint DEFAULT 0,
    expires_at timestamp with time zone,
    created_by bigint,
    instance_id uuid
);

-- TABLE: payments

CREATE TABLE public.payments (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    user_id text NOT NULL,
    plan_id text,
    amount numeric,
    currency text DEFAULT 'TON'::text,
    status text DEFAULT 'pending'::text,
    tx_hash text,
    from_wallet text,
    payment_url text,
    invoice_id text,
    confirmed_at timestamp with time zone,
    instance_id text
);

-- TABLE: plan_bot_limits

CREATE TABLE public.plan_bot_limits (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    plan_id uuid NOT NULL,
    bot_type text NOT NULL,
    max_bots bigint DEFAULT 1 NOT NULL
);

-- TABLE: plans

CREATE TABLE public.plans (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    template_id text,
    name text,
    duration_day bigint,
    price numeric,
    max_bots bigint DEFAULT 1,
    is_free boolean DEFAULT false,
    is_active boolean DEFAULT true
);

-- TABLE: promo_codes

CREATE TABLE public.promo_codes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    code text NOT NULL,
    amount_ton numeric NOT NULL,
    max_uses bigint DEFAULT 0,
    used_count bigint DEFAULT 0,
    expires_at timestamp with time zone,
    is_active boolean DEFAULT true,
    created_by bigint
);

-- TABLE: promo_redemptions

CREATE TABLE public.promo_redemptions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    promo_id text NOT NULL,
    user_id text NOT NULL
);

-- TABLE: servers

CREATE TABLE public.servers (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL,
    ip text NOT NULL,
    is_online boolean DEFAULT false,
    last_seen timestamp with time zone,
    channel text,
    online_since timestamp with time zone,
    cpu_percent numeric,
    memory_used_mb bigint,
    memory_total_mb bigint,
    last_containers text,
    tags text,
    max_containers bigint DEFAULT 0
);

-- TABLE: source_worker_configs

CREATE TABLE public.source_worker_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    label text,
    license_key text NOT NULL,
    worker_id text NOT NULL,
    app_id bigint NOT NULL,
    app_hash text NOT NULL,
    phone text NOT NULL,
    session_key text NOT NULL,
    is_active boolean DEFAULT true,
    last_heartbeat_at timestamp with time zone,
    last_status text
);

-- TABLE: subscriptions

CREATE TABLE public.subscriptions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    user_id text NOT NULL,
    plan_id text NOT NULL,
    started_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone,
    is_active boolean DEFAULT true,
    bot_count bigint DEFAULT 0
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
    role text DEFAULT 'user'::text,
    balance numeric DEFAULT 0,
    is_blocked boolean DEFAULT false
);

-- DEFAULT: audit_logs id

ALTER TABLE ONLY public.audit_logs ALTER COLUMN id SET DEFAULT nextval('public.audit_logs_id_seq'::regclass);

-- CONSTRAINT: audit_logs audit_logs_pkey

ALTER TABLE ONLY public.audit_logs
    ADD CONSTRAINT audit_logs_pkey PRIMARY KEY (id);

-- CONSTRAINT: bot_instances bot_instances_pkey

ALTER TABLE ONLY public.bot_instances
    ADD CONSTRAINT bot_instances_pkey PRIMARY KEY (id);

-- CONSTRAINT: bot_templates bot_templates_pkey

ALTER TABLE ONLY public.bot_templates
    ADD CONSTRAINT bot_templates_pkey PRIMARY KEY (id);

-- CONSTRAINT: deploy_jobs deploy_jobs_pkey

ALTER TABLE ONLY public.deploy_jobs
    ADD CONSTRAINT deploy_jobs_pkey PRIMARY KEY (id);

-- CONSTRAINT: invite_links invite_links_pkey

ALTER TABLE ONLY public.invite_links
    ADD CONSTRAINT invite_links_pkey PRIMARY KEY (id);

-- CONSTRAINT: payments payments_pkey

ALTER TABLE ONLY public.payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (id);

-- CONSTRAINT: plan_bot_limits plan_bot_limits_pkey

ALTER TABLE ONLY public.plan_bot_limits
    ADD CONSTRAINT plan_bot_limits_pkey PRIMARY KEY (id);

-- CONSTRAINT: plans plans_pkey

ALTER TABLE ONLY public.plans
    ADD CONSTRAINT plans_pkey PRIMARY KEY (id);

-- CONSTRAINT: promo_codes promo_codes_pkey

ALTER TABLE ONLY public.promo_codes
    ADD CONSTRAINT promo_codes_pkey PRIMARY KEY (id);

-- CONSTRAINT: promo_redemptions promo_redemptions_pkey

ALTER TABLE ONLY public.promo_redemptions
    ADD CONSTRAINT promo_redemptions_pkey PRIMARY KEY (id);

-- CONSTRAINT: servers servers_pkey

ALTER TABLE ONLY public.servers
    ADD CONSTRAINT servers_pkey PRIMARY KEY (id);

-- CONSTRAINT: source_worker_configs source_worker_configs_pkey

ALTER TABLE ONLY public.source_worker_configs
    ADD CONSTRAINT source_worker_configs_pkey PRIMARY KEY (id);

-- CONSTRAINT: subscriptions subscriptions_pkey

ALTER TABLE ONLY public.subscriptions
    ADD CONSTRAINT subscriptions_pkey PRIMARY KEY (id);

-- CONSTRAINT: users users_pkey

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

-- INDEX: idx_audit_logs_action

CREATE INDEX idx_audit_logs_action ON public.audit_logs USING btree (action);

-- INDEX: idx_audit_logs_actor_id

CREATE INDEX idx_audit_logs_actor_id ON public.audit_logs USING btree (actor_id);

-- INDEX: idx_audit_logs_created_at

CREATE INDEX idx_audit_logs_created_at ON public.audit_logs USING btree (created_at);

-- INDEX: idx_audit_logs_target_id

CREATE INDEX idx_audit_logs_target_id ON public.audit_logs USING btree (target_id);

-- INDEX: idx_bot_instances_bot_id

CREATE UNIQUE INDEX idx_bot_instances_bot_id ON public.bot_instances USING btree (bot_id) WHERE (deleted_at IS NULL);

-- INDEX: idx_bot_instances_container_name

CREATE UNIQUE INDEX idx_bot_instances_container_name ON public.bot_instances USING btree (container_name) WHERE (deleted_at IS NULL);

-- INDEX: idx_bot_instances_db_schema

CREATE UNIQUE INDEX idx_bot_instances_db_schema ON public.bot_instances USING btree (db_schema) WHERE (deleted_at IS NULL);

-- INDEX: idx_bot_instances_deleted_at

CREATE INDEX idx_bot_instances_deleted_at ON public.bot_instances USING btree (deleted_at);

-- INDEX: idx_bot_instances_owner_id

CREATE INDEX idx_bot_instances_owner_id ON public.bot_instances USING btree (owner_id);

-- INDEX: idx_bot_instances_plan_id

CREATE INDEX idx_bot_instances_plan_id ON public.bot_instances USING btree (plan_id);

-- INDEX: idx_bot_instances_server_id

CREATE INDEX idx_bot_instances_server_id ON public.bot_instances USING btree (server_id);

-- INDEX: idx_bot_instances_template_id

CREATE INDEX idx_bot_instances_template_id ON public.bot_instances USING btree (template_id);

-- INDEX: idx_bot_templates_deleted_at

CREATE INDEX idx_bot_templates_deleted_at ON public.bot_templates USING btree (deleted_at);

-- INDEX: idx_deploy_jobs_deleted_at

CREATE INDEX idx_deploy_jobs_deleted_at ON public.deploy_jobs USING btree (deleted_at);

-- INDEX: idx_deploy_jobs_instance_id

CREATE INDEX idx_deploy_jobs_instance_id ON public.deploy_jobs USING btree (instance_id);

-- INDEX: idx_deploy_jobs_server_id

CREATE INDEX idx_deploy_jobs_server_id ON public.deploy_jobs USING btree (server_id);

-- INDEX: idx_deploy_jobs_status

CREATE INDEX idx_deploy_jobs_status ON public.deploy_jobs USING btree (status);

-- INDEX: idx_invite_links_deleted_at

CREATE INDEX idx_invite_links_deleted_at ON public.invite_links USING btree (deleted_at);

-- INDEX: idx_invite_links_token

CREATE UNIQUE INDEX idx_invite_links_token ON public.invite_links USING btree (token);

-- INDEX: idx_payments_deleted_at

CREATE INDEX idx_payments_deleted_at ON public.payments USING btree (deleted_at);

-- INDEX: idx_payments_invoice_id

CREATE UNIQUE INDEX idx_payments_invoice_id ON public.payments USING btree (invoice_id);

-- INDEX: idx_payments_plan_id

CREATE INDEX idx_payments_plan_id ON public.payments USING btree (plan_id);

-- INDEX: idx_payments_tx_hash

CREATE UNIQUE INDEX idx_payments_tx_hash ON public.payments USING btree (tx_hash);

-- INDEX: idx_payments_user_id

CREATE INDEX idx_payments_user_id ON public.payments USING btree (user_id);

-- INDEX: idx_plan_bot_limits_deleted_at

CREATE INDEX idx_plan_bot_limits_deleted_at ON public.plan_bot_limits USING btree (deleted_at);

-- INDEX: idx_plan_bot_limits_plan_id

CREATE INDEX idx_plan_bot_limits_plan_id ON public.plan_bot_limits USING btree (plan_id);

-- INDEX: idx_plan_bottype

CREATE UNIQUE INDEX idx_plan_bottype ON public.plan_bot_limits USING btree (plan_id, bot_type);

-- INDEX: idx_plans_deleted_at

CREATE INDEX idx_plans_deleted_at ON public.plans USING btree (deleted_at);

-- INDEX: idx_plans_template_id

CREATE INDEX idx_plans_template_id ON public.plans USING btree (template_id);

-- INDEX: idx_promo_codes_code

CREATE UNIQUE INDEX idx_promo_codes_code ON public.promo_codes USING btree (code);

-- INDEX: idx_promo_codes_deleted_at

CREATE INDEX idx_promo_codes_deleted_at ON public.promo_codes USING btree (deleted_at);

-- INDEX: idx_promo_redemptions_deleted_at

CREATE INDEX idx_promo_redemptions_deleted_at ON public.promo_redemptions USING btree (deleted_at);

-- INDEX: idx_promo_redemptions_promo_id

CREATE INDEX idx_promo_redemptions_promo_id ON public.promo_redemptions USING btree (promo_id);

-- INDEX: idx_promo_redemptions_user_id

CREATE INDEX idx_promo_redemptions_user_id ON public.promo_redemptions USING btree (user_id);

-- INDEX: idx_promo_user

CREATE UNIQUE INDEX idx_promo_user ON public.promo_redemptions USING btree (promo_id, user_id);

-- INDEX: idx_servers_deleted_at

CREATE INDEX idx_servers_deleted_at ON public.servers USING btree (deleted_at);

-- INDEX: idx_servers_ip

CREATE UNIQUE INDEX idx_servers_ip ON public.servers USING btree (ip);

-- INDEX: idx_source_worker_configs_deleted_at

CREATE INDEX idx_source_worker_configs_deleted_at ON public.source_worker_configs USING btree (deleted_at);

-- INDEX: idx_source_worker_configs_license_key

CREATE UNIQUE INDEX idx_source_worker_configs_license_key ON public.source_worker_configs USING btree (license_key);

-- INDEX: idx_source_worker_configs_worker_id

CREATE UNIQUE INDEX idx_source_worker_configs_worker_id ON public.source_worker_configs USING btree (worker_id);

-- INDEX: idx_subscriptions_deleted_at

CREATE INDEX idx_subscriptions_deleted_at ON public.subscriptions USING btree (deleted_at);

-- INDEX: idx_subscriptions_is_active

CREATE INDEX idx_subscriptions_is_active ON public.subscriptions USING btree (is_active);

-- INDEX: idx_subscriptions_plan_id

CREATE INDEX idx_subscriptions_plan_id ON public.subscriptions USING btree (plan_id);

-- INDEX: idx_subscriptions_user_id

CREATE INDEX idx_subscriptions_user_id ON public.subscriptions USING btree (user_id);

-- INDEX: idx_users_deleted_at

CREATE INDEX idx_users_deleted_at ON public.users USING btree (deleted_at);

-- INDEX: idx_users_telegram_id

CREATE UNIQUE INDEX idx_users_telegram_id ON public.users USING btree (telegram_id);

-- FK CONSTRAINT: plan_bot_limits fk_plans_limits

ALTER TABLE ONLY public.plan_bot_limits
    ADD CONSTRAINT fk_plans_limits FOREIGN KEY (plan_id) REFERENCES public.plans(id);

-- PostgreSQL database dump complete

