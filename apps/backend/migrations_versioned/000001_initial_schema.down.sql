-- Rollback initial schema

DROP TRIGGER IF EXISTS update_review_states_updated_at ON review_states;
DROP TRIGGER IF EXISTS update_classifications_updated_at ON classifications;
DROP TRIGGER IF EXISTS update_patterns_updated_at ON patterns;
DROP TRIGGER IF EXISTS update_findings_updated_at ON findings;
DROP TRIGGER IF EXISTS update_assets_updated_at ON assets;
DROP TRIGGER IF EXISTS update_source_profiles_updated_at ON source_profiles;
DROP TRIGGER IF EXISTS update_scan_runs_updated_at ON scan_runs;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_review_states_status;
DROP INDEX IF EXISTS idx_review_states_finding;
DROP INDEX IF EXISTS idx_relationships_type;
DROP INDEX IF EXISTS idx_relationships_target;
DROP INDEX IF EXISTS idx_relationships_source;
DROP INDEX IF EXISTS idx_classifications_confidence;
DROP INDEX IF EXISTS idx_classifications_type;
DROP INDEX IF EXISTS idx_classifications_finding;
DROP INDEX IF EXISTS idx_findings_created;
DROP INDEX IF EXISTS idx_findings_severity;
DROP INDEX IF EXISTS idx_findings_pattern;
DROP INDEX IF EXISTS idx_findings_asset;
DROP INDEX IF EXISTS idx_findings_scan_run;
DROP INDEX IF EXISTS idx_assets_risk;
DROP INDEX IF EXISTS idx_assets_source;
DROP INDEX IF EXISTS idx_assets_type;
DROP INDEX IF EXISTS idx_assets_stable_id;
DROP INDEX IF EXISTS idx_scan_runs_status;
DROP INDEX IF EXISTS idx_scan_runs_started;
DROP INDEX IF EXISTS idx_scan_runs_profile;

DROP TABLE IF EXISTS review_states;
DROP TABLE IF EXISTS asset_relationships;
DROP TABLE IF EXISTS classifications;
DROP TABLE IF EXISTS findings;
DROP TABLE IF EXISTS patterns;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS source_profiles;
DROP TABLE IF EXISTS scan_runs;

DROP EXTENSION IF EXISTS "uuid-ossp";
