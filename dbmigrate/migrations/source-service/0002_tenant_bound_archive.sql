ALTER TABLE archive_files ADD COLUMN IF NOT EXISTS tenant_id text;
ALTER TABLE bot_file_caches ADD COLUMN IF NOT EXISTS tenant_id text;
CREATE INDEX IF NOT EXISTS idx_archive_files_tenant_id ON archive_files (tenant_id);
CREATE INDEX IF NOT EXISTS idx_bot_file_caches_tenant_id ON bot_file_caches (tenant_id);
-- Existing rows require an operator-selected tenant backfill before NOT NULL can be enforced.
