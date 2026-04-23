-- Guard against 000039 backfill re-running on legitimately high-confidence rows.
-- The original migration overwrites confidence_score = 1.0 rows unintentionally.
-- This migration adds a guard column so future backfill scripts can skip already-reviewed rows.

ALTER TABLE findings
    ADD COLUMN IF NOT EXISTS manually_reviewed BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE classifications
    ADD COLUMN IF NOT EXISTS manually_reviewed BOOLEAN NOT NULL DEFAULT FALSE;

-- Index for efficient filtering of unreviewed rows
CREATE INDEX IF NOT EXISTS idx_findings_manually_reviewed ON findings(manually_reviewed) WHERE manually_reviewed = FALSE;
CREATE INDEX IF NOT EXISTS idx_classifications_manually_reviewed ON classifications(manually_reviewed) WHERE manually_reviewed = FALSE;

COMMENT ON COLUMN findings.manually_reviewed IS 'Set to TRUE to protect this row from automated confidence backfill scripts.';
COMMENT ON COLUMN classifications.manually_reviewed IS 'Set to TRUE to protect this row from automated confidence backfill scripts.';
