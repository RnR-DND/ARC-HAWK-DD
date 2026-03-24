-- Rollback deduplication support

ALTER TABLE findings DROP COLUMN IF EXISTS occurrence_count;
DROP INDEX IF EXISTS idx_findings_hash;
DROP INDEX IF EXISTS idx_findings_unique;
ALTER TABLE findings DROP COLUMN IF EXISTS normalized_value_hash;
