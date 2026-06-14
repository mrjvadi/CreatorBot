-- Migration 002: Community Service + Ads System

BEGIN;

-- ── Community Service ─────────────────────────────────────

CREATE TABLE IF NOT EXISTS communities (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,
    owner_telegram_id BIGINT NOT NULL,
    chat_id          BIGINT UNIQUE NOT NULL,
    type             TEXT NOT NULL CHECK (type IN ('group', 'channel')),
    name             TEXT NOT NULL,
    username         TEXT,
    invite_link      TEXT UNIQUE,
    status           TEXT NOT NULL DEFAULT 'pending',
    member_count     INTEGER NOT NULL DEFAULT 0,
    quality_score    INTEGER NOT NULL DEFAULT 50,
    INDEX            (owner_telegram_id),
    INDEX            (status)
);

CREATE TABLE IF NOT EXISTS campaign_attributions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    telegram_id  BIGINT NOT NULL,
    community_id UUID NOT NULL REFERENCES communities(id),
    campaign_id  TEXT NOT NULL,
    invite_link  TEXT,
    joined_at    TIMESTAMPTZ NOT NULL,
    left_at      TIMESTAMPTZ,
    valid_at     TIMESTAMPTZ,
    is_valid     BOOLEAN NOT NULL DEFAULT FALSE,
    is_settled   BOOLEAN NOT NULL DEFAULT FALSE,
    revenue      DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_attributions_telegram ON campaign_attributions(telegram_id);
CREATE INDEX IF NOT EXISTS idx_attributions_community ON campaign_attributions(community_id);
CREATE INDEX IF NOT EXISTS idx_attributions_campaign ON campaign_attributions(campaign_id);

CREATE TABLE IF NOT EXISTS community_revenues (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    community_id     UUID NOT NULL REFERENCES communities(id),
    campaign_id      TEXT NOT NULL,
    period           TEXT NOT NULL,
    total_revenue    DOUBLE PRECISION NOT NULL DEFAULT 0,
    owner_share      DOUBLE PRECISION NOT NULL DEFAULT 0,
    members_pool     DOUBLE PRECISION NOT NULL DEFAULT 0,
    platform_share   DOUBLE PRECISION NOT NULL DEFAULT 0,
    valid_joins      INTEGER NOT NULL DEFAULT 0,
    fraud_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1,
    status           TEXT NOT NULL DEFAULT 'pending',
    distributed_at   TIMESTAMPTZ,
    UNIQUE(community_id, campaign_id, period)
);
CREATE INDEX IF NOT EXISTS idx_community_revenues_status ON community_revenues(status);

CREATE TABLE IF NOT EXISTS member_activity_scores (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    community_id    UUID NOT NULL REFERENCES communities(id),
    telegram_id     BIGINT NOT NULL,
    message_count   INTEGER NOT NULL DEFAULT 0,
    reply_count     INTEGER NOT NULL DEFAULT 0,
    reaction_count  INTEGER NOT NULL DEFAULT 0,
    active_days     INTEGER NOT NULL DEFAULT 0,
    membership_days INTEGER NOT NULL DEFAULT 0,
    score           DOUBLE PRECISION NOT NULL DEFAULT 0,
    score_date      TEXT NOT NULL,
    UNIQUE(community_id, telegram_id)
);

CREATE TABLE IF NOT EXISTS member_rewards (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    community_id   UUID NOT NULL REFERENCES communities(id),
    telegram_id    BIGINT NOT NULL,
    period         TEXT NOT NULL,
    activity_score DOUBLE PRECISION NOT NULL,
    pool_total     DOUBLE PRECISION NOT NULL,
    score_total    DOUBLE PRECISION NOT NULL,
    reward         DOUBLE PRECISION NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    paid_at        TIMESTAMPTZ,
    tx_id          TEXT
);
CREATE INDEX IF NOT EXISTS idx_member_rewards_status ON member_rewards(status);

CREATE TABLE IF NOT EXISTS validation_windows (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    community_id UUID UNIQUE NOT NULL REFERENCES communities(id),
    window_hours INTEGER NOT NULL DEFAULT 24
);

-- ── Ads System ────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS ad_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    base_cpj            DOUBLE PRECISION NOT NULL DEFAULT 0.005,
    min_channel_score   INTEGER NOT NULL DEFAULT 30,
    max_fake_percent    DOUBLE PRECISION NOT NULL DEFAULT 30,
    platform_commission DOUBLE PRECISION NOT NULL DEFAULT 20,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS channel_categories (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name           TEXT UNIQUE NOT NULL,
    label          TEXT NOT NULL,
    cpj_multiplier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS publishers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    telegram_id BIGINT UNIQUE NOT NULL,
    username    TEXT,
    first_name  TEXT,
    balance     DOUBLE PRECISION NOT NULL DEFAULT 0,
    is_blocked  BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS ad_channels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    owner_id        UUID NOT NULL REFERENCES publishers(id),
    category_id     UUID REFERENCES channel_categories(id),
    channel_id      BIGINT UNIQUE NOT NULL,
    channel_name    TEXT,
    channel_username TEXT,
    member_count    INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    score           INTEGER NOT NULL DEFAULT 0,
    fake_percent    DOUBLE PRECISION NOT NULL DEFAULT 0,
    real_members    INTEGER NOT NULL DEFAULT 0,
    effective_cpj   DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_ad_channels_status ON ad_channels(status, is_active);

CREATE TABLE IF NOT EXISTS campaigns (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    publisher_id        UUID NOT NULL REFERENCES publishers(id),
    name                TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'draft',
    target_category_id  UUID REFERENCES channel_categories(id),
    media_file_id       TEXT,
    media_type          TEXT,
    caption             TEXT,
    button_text         TEXT,
    button_url          TEXT,
    budget              DOUBLE PRECISION NOT NULL,
    spent               DOUBLE PRECISION NOT NULL DEFAULT 0,
    cpj                 DOUBLE PRECISION NOT NULL,
    total_joins         INTEGER NOT NULL DEFAULT 0,
    real_joins          INTEGER NOT NULL DEFAULT 0,
    target_count        INTEGER NOT NULL DEFAULT 0,
    review_note         TEXT,
    reviewed_at         TIMESTAMPTZ,
    reviewer_id         BIGINT
);
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_publisher ON campaigns(publisher_id);

COMMIT;

SELECT 'Migration 002 completed' AS result;
