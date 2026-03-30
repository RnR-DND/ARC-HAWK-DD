-- Add classifier version tracking
-- This allows tracking which version of the classification engine classified each finding

ALTER TABLE classifications
ADD COLUMN IF NOT EXISTS classifier_version VARCHAR(50) DEFAULT 'v2.0-multisignal';

ALTER TABLE classifications
ADD COLUMN IF NOT EXISTS classified_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_classifications_version
ON classifications(classifier_version);
