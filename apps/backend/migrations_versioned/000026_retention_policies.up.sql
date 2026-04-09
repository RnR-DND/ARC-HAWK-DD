-- Add retention policy fields to assets table
ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS retention_policy_days    INT          DEFAULT 365,
    ADD COLUMN IF NOT EXISTS retention_policy_name    VARCHAR(256) DEFAULT 'Default',
    ADD COLUMN IF NOT EXISTS retention_policy_basis   VARCHAR(512) DEFAULT 'DPDPA Section 8(7) - Legal Obligation';

-- Helper function: returns when a finding should be deleted given its detection date and policy
CREATE OR REPLACE FUNCTION calculate_deletion_due(detected_at TIMESTAMPTZ, retention_days INT)
RETURNS TIMESTAMPTZ LANGUAGE sql IMMUTABLE AS $$
    SELECT detected_at + (retention_days || ' days')::INTERVAL;
$$;

-- Retention violations view: findings whose retention period has expired
CREATE OR REPLACE VIEW retention_violations AS
SELECT
    f.id                                          AS finding_id,
    a.id                                          AS asset_id,
    a.name                                        AS asset_name,
    f.pattern_name                                AS pii_type,
    f.created_at                                  AS first_detected_at,
    COALESCE(a.retention_policy_days, 365)        AS retention_policy_days,
    calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365)) AS deletion_due_at,
    GREATEST(0, EXTRACT(DAY FROM NOW() - calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365)))::INT) AS days_overdue
FROM findings f
JOIN assets  a ON a.id = f.asset_id
WHERE calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365)) < NOW();

-- Expiring soon view: findings expiring within 30 days
CREATE OR REPLACE VIEW retention_expiring_soon AS
SELECT
    f.id                                          AS finding_id,
    a.id                                          AS asset_id,
    a.name                                        AS asset_name,
    f.pattern_name                                AS pii_type,
    f.created_at                                  AS first_detected_at,
    COALESCE(a.retention_policy_days, 365)        AS retention_policy_days,
    calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365)) AS deletion_due_at
FROM findings f
JOIN assets  a ON a.id = f.asset_id
WHERE calculate_deletion_due(f.created_at, COALESCE(a.retention_policy_days, 365))
      BETWEEN NOW() AND NOW() + INTERVAL '30 days';
