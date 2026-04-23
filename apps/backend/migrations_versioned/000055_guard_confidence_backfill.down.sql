DROP INDEX IF EXISTS idx_findings_manually_reviewed;
DROP INDEX IF EXISTS idx_classifications_manually_reviewed;
ALTER TABLE findings DROP COLUMN IF EXISTS manually_reviewed;
ALTER TABLE classifications DROP COLUMN IF EXISTS manually_reviewed;
