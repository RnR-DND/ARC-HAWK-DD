-- Rollback policy and consent schema
-- Migration: 000008_add_policy_consent

DROP TRIGGER IF EXISTS update_policies_updated_at ON policies;
DROP TABLE IF EXISTS consent_records;
DROP TABLE IF EXISTS policy_executions;
DROP TABLE IF EXISTS policies;
