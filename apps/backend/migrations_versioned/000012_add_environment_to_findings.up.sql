ALTER TABLE findings ADD COLUMN environment VARCHAR(50) NOT NULL DEFAULT 'PROD';
CREATE INDEX idx_findings_environment ON findings(environment);
