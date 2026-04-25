-- Extend row-level security to additional tables for DPDPA tenant isolation

-- audit_logs.tenant_id already added in 000014/000015; just enable RLS.
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- classifications has no direct tenant_id; backfill from parent findings row.
ALTER TABLE classifications ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE classifications c
    SET tenant_id = f.tenant_id
    FROM findings f
    WHERE f.id = c.finding_id AND c.tenant_id IS NULL;
ALTER TABLE classifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_classifications ON classifications
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::uuid);

-- consent_records.tenant_id already added in 000030.
ALTER TABLE consent_records ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_consent_records ON consent_records
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- masking_audit_log has no direct tenant_id; backfill from parent assets row.
ALTER TABLE masking_audit_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE masking_audit_log m
    SET tenant_id = a.tenant_id
    FROM assets a
    WHERE a.id = m.asset_id AND m.tenant_id IS NULL;
ALTER TABLE masking_audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_masking_audit_log ON masking_audit_log
    USING (tenant_id = (current_setting('app.current_tenant_id', true))::uuid);
