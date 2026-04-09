-- Migration 000030 rollback: remove consent_records indexes and table.
DROP INDEX IF EXISTS idx_consent_subject;
DROP INDEX IF EXISTS idx_consent_asset;
DROP INDEX IF EXISTS idx_consent_tenant;
DROP TABLE IF EXISTS consent_records;
