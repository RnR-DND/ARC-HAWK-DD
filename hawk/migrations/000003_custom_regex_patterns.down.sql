-- 000003_custom_regex_patterns.down.sql
-- Reverse custom regex patterns and match log.

BEGIN;

DROP TABLE IF EXISTS custom_regex_match_log CASCADE;
DROP TABLE IF EXISTS custom_regex_patterns CASCADE;

DELETE FROM schema_migrations WHERE version = '000003';

COMMIT;
