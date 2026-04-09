-- Migration 000029: Add chain hash columns to audit_logs for integrity verification.
-- previous_hash chains each entry to the prior entry for the same tenant.
-- entry_hash = SHA256(previous_hash || action || resource_type || resource_id || metadata || created_at ISO8601)

ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS previous_hash TEXT;
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS entry_hash TEXT;

CREATE INDEX IF NOT EXISTS idx_audit_entry_hash ON audit_logs(entry_hash);
