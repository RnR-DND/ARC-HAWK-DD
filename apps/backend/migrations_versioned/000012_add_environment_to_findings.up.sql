ALTER TABLE findings ADD COLUMN IF NOT EXISTS environment VARCHAR(50) NOT NULL DEFAULT 'PROD';
CREATE INDEX IF NOT EXISTS idx_findings_environment ON findings(environment);
