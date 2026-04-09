-- Phase 2: Custom regex engine hardening
-- Adds validation_status, catastrophic backtracking flag, and lifetime match counter
-- to custom_patterns so operators can see which patterns are safe and how often they fire.

ALTER TABLE custom_patterns
    ADD COLUMN IF NOT EXISTS validation_status  VARCHAR(32) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS backtrack_safe     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS match_count_lifetime BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_matched_at    TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS test_positives     JSONB,   -- [{raw, expected_match}]
    ADD COLUMN IF NOT EXISTS test_negatives     JSONB;   -- [{raw}]

-- validation_status values: pending | valid | invalid | risky
CREATE INDEX IF NOT EXISTS idx_custom_patterns_validation ON custom_patterns(tenant_id, validation_status);
