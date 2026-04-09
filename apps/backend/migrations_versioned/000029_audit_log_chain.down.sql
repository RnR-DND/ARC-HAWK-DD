-- Migration 000029 rollback: remove chain hash columns from audit_logs.
DROP INDEX IF EXISTS idx_audit_entry_hash;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS entry_hash;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS previous_hash;
