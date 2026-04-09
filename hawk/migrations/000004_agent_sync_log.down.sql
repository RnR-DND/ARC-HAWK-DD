-- 000004_agent_sync_log.down.sql
-- Reverse agent sync log table.

BEGIN;

DROP TABLE IF EXISTS agent_sync_log CASCADE;

DELETE FROM schema_migrations WHERE version = '000004';

COMMIT;
