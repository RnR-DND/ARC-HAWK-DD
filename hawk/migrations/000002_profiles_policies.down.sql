-- 000002_profiles_policies.down.sql
-- Reverse profiles and policies tables.

BEGIN;

DROP TABLE IF EXISTS profile_policy_assignments CASCADE;
DROP TABLE IF EXISTS policies CASCADE;
DROP TABLE IF EXISTS profiles CASCADE;

DELETE FROM schema_migrations WHERE version = '000002';

COMMIT;
