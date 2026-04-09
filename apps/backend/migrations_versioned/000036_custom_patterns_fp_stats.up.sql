-- Phase 2 extension: false-positive tracking and auto-deactivation support.
-- false_positive_count: number of times an operator has marked a match as a false positive.
-- false_positive_rate: ratio = false_positive_count / GREATEST(match_count_lifetime, 1).
--   Updated by the application whenever a false-positive override is recorded.
--   If this rate exceeds 0.30 the pattern is automatically deactivated by the service.
-- auto_deactivated: TRUE when the service (not the operator) disabled the pattern due to
--   a high false-positive rate.

ALTER TABLE custom_patterns
    ADD COLUMN IF NOT EXISTS false_positive_count  BIGINT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS false_positive_rate   NUMERIC(5,4)   NOT NULL DEFAULT 0.0000,
    ADD COLUMN IF NOT EXISTS auto_deactivated      BOOLEAN        NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_custom_patterns_fp_rate
    ON custom_patterns(tenant_id, false_positive_rate);
