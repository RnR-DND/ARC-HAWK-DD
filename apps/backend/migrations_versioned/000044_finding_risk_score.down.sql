DROP INDEX IF EXISTS idx_findings_risk_score;
ALTER TABLE findings DROP COLUMN IF EXISTS risk_score;
