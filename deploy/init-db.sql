-- CreatorBot — PostgreSQL initialization
-- This script runs once when the postgres container is first created.
-- Creates separate databases for each service sharing the same postgres instance.

CREATE DATABASE uploader_bot  WITH OWNER botuser;
CREATE DATABASE vpn_bot       WITH OWNER botuser;
CREATE DATABASE archive_bot   WITH OWNER botuser;
CREATE DATABASE member_bot    WITH OWNER botuser;
CREATE DATABASE source_svc    WITH OWNER botuser;

-- Enable pg_trgm for archive-bot fuzzy search (runs in botmanager db by default;
-- archive-bot migration also runs this, but having it here ensures it exists.)
\c archive_bot
CREATE EXTENSION IF NOT EXISTS pg_trgm;
\c botmanager
