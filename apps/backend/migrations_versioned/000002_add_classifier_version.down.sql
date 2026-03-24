-- Rollback classifier version tracking

DROP INDEX IF EXISTS idx_classifications_version;
ALTER TABLE classifications DROP COLUMN IF EXISTS classified_at;
ALTER TABLE classifications DROP COLUMN IF EXISTS classifier_version;
