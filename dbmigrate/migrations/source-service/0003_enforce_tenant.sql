-- 0003: enforce tenant isolation after 0002 backfill.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM archive_files WHERE tenant_id IS NULL) THEN
        RAISE EXCEPTION $msg$source-service tenant backfill incomplete: archive_files$msg$;
    END IF;
    IF EXISTS (SELECT 1 FROM bot_file_caches WHERE tenant_id IS NULL) THEN
        RAISE EXCEPTION $msg$source-service tenant backfill incomplete: bot_file_caches$msg$;
    END IF;
END $$;
ALTER TABLE archive_files ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bot_file_caches ALTER COLUMN tenant_id SET NOT NULL;
