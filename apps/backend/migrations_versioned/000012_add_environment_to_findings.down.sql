DROP INDEX IF EXISTS idx_findings_environment;
ALTER TABLE findings DROP COLUMN IF EXISTS environment;
