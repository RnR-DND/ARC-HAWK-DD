-- 000001_initial_schema.up.sql
-- Core tables for ARL-Hawk data discovery and governance engine.

BEGIN;

-- Schema migrations tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sources / Connectors
CREATE TABLE sources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  connector_type TEXT NOT NULL,  -- postgres, mysql, s3, kafka, etc.
  connection_config JSONB NOT NULL,  -- encrypted at rest
  attribute_level TEXT NOT NULL DEFAULT 'login',  -- org, person, login
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Scan Jobs
CREATE TABLE scan_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id UUID NOT NULL REFERENCES sources(id),
  status TEXT NOT NULL DEFAULT 'pending',  -- pending, running, completed, cancelled, failed
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  fields_scanned INTEGER DEFAULT 0,
  pii_found INTEGER DEFAULT 0,
  error_message TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_scan_jobs_source ON scan_jobs(source_id);
CREATE INDEX idx_scan_jobs_status ON scan_jobs(status);

-- Assets (discovered data assets)
CREATE TABLE assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id UUID NOT NULL REFERENCES sources(id),
  scan_job_id UUID REFERENCES scan_jobs(id),
  field_name TEXT NOT NULL,
  field_path TEXT NOT NULL,
  source_ref TEXT NOT NULL,  -- e.g., mysql://crm.customers
  pii_category TEXT,
  dpdpa_schedule TEXT,
  confidence NUMERIC(5,4),
  classifier TEXT,  -- rule_based, ml, llm, custom_regex
  custom_regex_id UUID,
  custom_regex_name TEXT,
  sensitivity TEXT,  -- high, medium, low
  tags TEXT[] DEFAULT '{}',
  matched_by TEXT[] DEFAULT '{}',
  ocr_confidence NUMERIC(5,4),
  consent_id UUID,  -- DPDPA Sec 4
  declared_purpose TEXT,  -- DPDPA Sec 5
  retention_policy_days INTEGER,
  is_children_data BOOLEAN DEFAULT false,  -- DPDPA Sec 9
  scanned_at TIMESTAMPTZ DEFAULT NOW(),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_assets_source ON assets(source_id);
CREATE INDEX idx_assets_scan_job ON assets(scan_job_id);
CREATE INDEX idx_assets_pii ON assets(pii_category);
CREATE INDEX idx_assets_sensitivity ON assets(sensitivity);
CREATE INDEX idx_assets_tags ON assets USING GIN(tags);
CREATE INDEX idx_assets_search ON assets USING GIN(
  to_tsvector('english', field_name || ' ' || field_path || ' ' || COALESCE(pii_category, ''))
);

-- Date-partitioned findings (for high-volume scan results)
CREATE TABLE findings (
  id UUID DEFAULT gen_random_uuid(),
  asset_id UUID NOT NULL,
  scan_job_id UUID NOT NULL,
  field_value_hash TEXT,  -- hashed sample for dedup
  classification JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partitions (monthly, 2026 H1)
CREATE TABLE findings_2026_01 PARTITION OF findings FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE findings_2026_02 PARTITION OF findings FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE findings_2026_03 PARTITION OF findings FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE findings_2026_04 PARTITION OF findings FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE findings_2026_05 PARTITION OF findings FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE findings_2026_06 PARTITION OF findings FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');

-- Risk Scores
CREATE TABLE risk_scores (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  asset_id UUID NOT NULL REFERENCES assets(id),
  source_id UUID NOT NULL REFERENCES sources(id),
  pii_density NUMERIC(5,4) NOT NULL,
  sensitivity_weight NUMERIC(5,4) NOT NULL,
  access_exposure NUMERIC(5,4) NOT NULL,
  retention_violation NUMERIC(5,4) NOT NULL,
  total_score NUMERIC(5,2) NOT NULL,
  tier TEXT NOT NULL,  -- critical, high, medium, low
  previous_tier TEXT,
  calculated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_risk_source ON risk_scores(source_id);
CREATE INDEX idx_risk_tier ON risk_scores(tier);

-- Remediation Issues
CREATE TABLE remediation_issues (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  asset_id UUID REFERENCES assets(id),
  source_id UUID REFERENCES sources(id),
  issue_type TEXT NOT NULL,
  severity TEXT NOT NULL,  -- critical, high, medium, low
  status TEXT NOT NULL DEFAULT 'open',  -- open, in_progress, resolved, closed
  sop_id TEXT,
  assigned_to TEXT,
  description TEXT NOT NULL,
  evidence JSONB,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_remediation_status ON remediation_issues(status);

-- Audit Log
CREATE TABLE audit_log (
  id BIGSERIAL PRIMARY KEY,
  user_id TEXT,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT,
  details JSONB,
  ip_address INET,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_audit_created ON audit_log(created_at);

-- Record migration
INSERT INTO schema_migrations (version) VALUES ('000001') ON CONFLICT DO NOTHING;

COMMIT;
