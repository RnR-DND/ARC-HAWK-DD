-- 000003_custom_regex_patterns.up.sql
-- Custom regex patterns and match tracking for classification Layer 1.

BEGIN;

CREATE TABLE custom_regex_patterns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE,
  pattern TEXT NOT NULL,
  description TEXT,
  pii_category TEXT NOT NULL,
  dpdpa_schedule TEXT,
  sensitivity TEXT NOT NULL DEFAULT 'medium',
  tags TEXT[] DEFAULT '{}',
  is_active BOOLEAN DEFAULT true,
  is_auto_deactivated BOOLEAN DEFAULT false,
  false_positive_count INTEGER DEFAULT 0,
  total_match_count INTEGER DEFAULT 0,
  false_positive_rate NUMERIC(5,4) DEFAULT 0,
  test_cases JSONB DEFAULT '[]',
  created_by TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  deleted_at TIMESTAMPTZ  -- soft delete
);
CREATE INDEX idx_custom_regex_active ON custom_regex_patterns(is_active) WHERE deleted_at IS NULL;

CREATE TABLE custom_regex_match_log (
  id BIGSERIAL PRIMARY KEY,
  pattern_id UUID NOT NULL REFERENCES custom_regex_patterns(id),
  scan_job_id UUID,
  match_count INTEGER DEFAULT 0,
  false_positive_overrides INTEGER DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_regex_match_pattern ON custom_regex_match_log(pattern_id);

INSERT INTO schema_migrations (version) VALUES ('000003') ON CONFLICT DO NOTHING;

COMMIT;
