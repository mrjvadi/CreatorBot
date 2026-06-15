-- Migration 002: Performance indexes

-- Full-text search برای archive-bot
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Server last_seen index برای heartbeat query
CREATE INDEX IF NOT EXISTS idx_servers_online ON servers(is_online, deleted_at);

-- Instance lookup by bot_id
CREATE INDEX IF NOT EXISTS idx_instances_bot_id ON bot_instances(bot_id);

-- Plan lookup برای wizard
CREATE INDEX IF NOT EXISTS idx_plans_active ON plans(is_active, deleted_at);
CREATE INDEX IF NOT EXISTS idx_plans_template ON plans(template_id) WHERE deleted_at IS NULL;

-- Template type lookup
CREATE INDEX IF NOT EXISTS idx_templates_type ON bot_templates(type, is_active);
