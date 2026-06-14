-- Migration 001: Initial schema
-- اجرا: psql $POSTGRES_DSN -f 001_initial.sql

BEGIN;

-- ── Extensions ────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ── Users ─────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    telegram_id  BIGINT UNIQUE NOT NULL,
    username     TEXT,
    first_name   TEXT,
    role         TEXT NOT NULL DEFAULT 'user',
    is_blocked   BOOLEAN NOT NULL DEFAULT FALSE,
    language     TEXT NOT NULL DEFAULT 'fa'
);
CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- ── Servers ───────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS servers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    name       TEXT NOT NULL,
    ip         TEXT NOT NULL,
    channel    TEXT,
    is_online  BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_servers_deleted_at ON servers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_servers_online ON servers(is_online);

-- ── Bot Templates ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS bot_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    image_name  TEXT NOT NULL,
    image_tag   TEXT NOT NULL,
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    is_free     BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_bot_templates_type ON bot_templates(type);

-- ── Plans ─────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS plans (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    template_id  UUID REFERENCES bot_templates(id),
    name         TEXT NOT NULL,
    duration_day INTEGER NOT NULL DEFAULT 30,
    price        DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_bots     INTEGER NOT NULL DEFAULT 1,
    is_free      BOOLEAN NOT NULL DEFAULT FALSE,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_plans_is_active ON plans(is_active);

-- ── Plan Bot Limits ───────────────────────────────────────

CREATE TABLE IF NOT EXISTS plan_bot_limits (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id  UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    bot_type TEXT NOT NULL,
    max_bots INTEGER NOT NULL DEFAULT 1,
    UNIQUE(plan_id, bot_type)
);
CREATE INDEX IF NOT EXISTS idx_plan_bot_limits_plan_id ON plan_bot_limits(plan_id);

-- ── Subscriptions ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS subscriptions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id    UUID NOT NULL REFERENCES users(id),
    plan_id    UUID NOT NULL REFERENCES plans(id),
    bot_count  INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_is_active ON subscriptions(is_active);

-- ── Bot Instances ─────────────────────────────────────────

CREATE TABLE IF NOT EXISTS bot_instances (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,
    owner_id       UUID NOT NULL REFERENCES users(id),
    template_id    UUID NOT NULL REFERENCES bot_templates(id),
    server_id      UUID REFERENCES servers(id),
    bot_token      TEXT NOT NULL,
    bot_id         BIGINT,
    container_id   TEXT,
    container_name TEXT,
    db_schema      TEXT,
    status         TEXT NOT NULL DEFAULT 'pending',
    expires_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_bot_instances_owner_id ON bot_instances(owner_id);
CREATE INDEX IF NOT EXISTS idx_bot_instances_status ON bot_instances(status);
CREATE INDEX IF NOT EXISTS idx_bot_instances_bot_id ON bot_instances(bot_id);

-- ── Invite Links ──────────────────────────────────────────

CREATE TABLE IF NOT EXISTS invite_links (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    code       TEXT UNIQUE NOT NULL,
    bot_type   TEXT NOT NULL,
    label      TEXT,
    max_use    INTEGER NOT NULL DEFAULT 1,
    use_count  INTEGER NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_invite_links_code ON invite_links(code);

-- ── Payments ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS payments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id      UUID NOT NULL REFERENCES users(id),
    plan_id      UUID REFERENCES plans(id),
    invoice_id   TEXT,
    tx_hash      TEXT,
    amount       DOUBLE PRECISION NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    confirmed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);

-- ── Audit Log ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_id    UUID,
    actor_role  TEXT,
    action      TEXT NOT NULL,
    target_id   TEXT,
    target_type TEXT,
    description TEXT,
    ip_address  TEXT,
    extra       JSONB
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- ── Deploy Jobs ───────────────────────────────────────────

CREATE TABLE IF NOT EXISTS deploy_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    instance_id  UUID REFERENCES bot_instances(id),
    type         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    error        TEXT,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_deploy_jobs_instance_id ON deploy_jobs(instance_id);
CREATE INDEX IF NOT EXISTS idx_deploy_jobs_status ON deploy_jobs(status);

COMMIT;

SELECT 'Migration 001 completed successfully' AS result;
