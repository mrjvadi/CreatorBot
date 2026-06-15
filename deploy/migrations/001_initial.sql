-- Migration 001: Initial schema
-- اجرا: migrate -path deploy/migrations -database $POSTGRES_DSN up

-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Users
CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    telegram_id BIGINT UNIQUE NOT NULL,
    username    TEXT,
    first_name  TEXT,
    last_name   TEXT,
    lang        TEXT NOT NULL DEFAULT 'fa',
    role        TEXT NOT NULL DEFAULT 'user',
    is_blocked  BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at  ON users(deleted_at);

-- Servers
CREATE TABLE IF NOT EXISTS servers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    name        TEXT UNIQUE NOT NULL,
    ip          TEXT UNIQUE NOT NULL,
    is_online   BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen   TIMESTAMPTZ
);

-- Bot Templates
CREATE TABLE IF NOT EXISTS bot_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL,
    image_name  TEXT NOT NULL,
    image_tag   TEXT NOT NULL DEFAULT 'latest',
    description TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    is_free     BOOLEAN NOT NULL DEFAULT FALSE
);

-- Plans
CREATE TABLE IF NOT EXISTS plans (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    template_id  UUID REFERENCES bot_templates(id),
    name         TEXT NOT NULL,
    price        DECIMAL(18,8) NOT NULL DEFAULT 0,
    duration_day INT NOT NULL DEFAULT 30,
    max_bots     INT NOT NULL DEFAULT 1,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    is_free      BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_plans_name_tmpl ON plans(name, template_id) WHERE deleted_at IS NULL;

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    user_id    UUID NOT NULL REFERENCES users(id),
    plan_id    UUID NOT NULL REFERENCES plans(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_subs_user_active ON subscriptions(user_id, is_active);

-- Bot Instances
CREATE TABLE IF NOT EXISTS bot_instances (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,
    owner_id       UUID NOT NULL REFERENCES users(id),
    template_id    UUID NOT NULL REFERENCES bot_templates(id),
    server_id      UUID REFERENCES servers(id),
    bot_id         BIGINT UNIQUE NOT NULL,
    bot_token      TEXT NOT NULL,
    container_name TEXT UNIQUE NOT NULL,
    container_id   TEXT,
    status         TEXT NOT NULL DEFAULT 'pending',
    expires_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_instances_owner    ON bot_instances(owner_id);
CREATE INDEX IF NOT EXISTS idx_instances_status   ON bot_instances(status);
CREATE INDEX IF NOT EXISTS idx_instances_deleted  ON bot_instances(deleted_at);

-- Invite Links
CREATE TABLE IF NOT EXISTS invite_links (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    token       TEXT UNIQUE NOT NULL,
    bot_type    TEXT NOT NULL,
    label       TEXT,
    use_limit   INT NOT NULL DEFAULT 1,
    used_count  INT NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,
    created_by  UUID REFERENCES users(id)
);
