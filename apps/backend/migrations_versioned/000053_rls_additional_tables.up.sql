-- Extend row-level security to additional tables for DPDPA tenant isolation
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE classifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_classifications ON classifications
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE consent_records ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_consent_records ON consent_records
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));

ALTER TABLE masking_audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_masking_audit_log ON masking_audit_log
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));
