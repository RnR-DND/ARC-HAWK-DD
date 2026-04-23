DROP POLICY IF EXISTS tenant_isolation_audit_logs ON audit_logs;
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_classifications ON classifications;
ALTER TABLE classifications DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_consent_records ON consent_records;
ALTER TABLE consent_records DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_masking_audit_log ON masking_audit_log;
ALTER TABLE masking_audit_log DISABLE ROW LEVEL SECURITY;
