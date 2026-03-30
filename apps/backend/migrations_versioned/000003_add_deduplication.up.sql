-- Add deduplication support
-- Enables detecting and preventing duplicate findings across multiple scans

ALTER TABLE findings ADD COLUMN IF NOT EXISTS normalized_value_hash VARCHAR(64);

-- Composite unique index to prevent duplicates
-- Note: This allows same finding across different scan_runs but prevents duplicates within a scan
CREATE UNIQUE INDEX IF NOT EXISTS idx_findings_unique
ON findings(asset_id, pattern_name, normalized_value_hash, scan_run_id);

-- Index for hash-based lookups
CREATE INDEX IF NOT EXISTS idx_findings_hash ON findings(normalized_value_hash);

-- Add occurrence count column for tracking how many times a value appeared
ALTER TABLE findings ADD COLUMN IF NOT EXISTS occurrence_count INTEGER DEFAULT 1;
