-- Migration 000040: Add remediation_previews table for single-use preview/execute workflow.
-- Stores a serialized preview that is consumed (deleted) on execution or when it expires.

CREATE TABLE IF NOT EXISTS remediation_previews (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    preview_data JSONB      NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 hour',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_remediation_previews_expires ON remediation_previews(expires_at);
