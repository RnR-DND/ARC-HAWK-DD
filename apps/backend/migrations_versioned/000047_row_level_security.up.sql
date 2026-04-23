-- Enable RLS on the most sensitive tables.
-- Policies enforce tenant isolation using a session variable set per request.
-- IMPORTANT: The database role used by the app must NOT have BYPASSRLS.
-- Run outside migrations (as superuser): ALTER ROLE arc_hawk_app NOBYPASSRLS;
ALTER TABLE findings ENABLE ROW LEVEL SECURITY;
ALTER TABLE scan_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE assets ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_findings ON findings
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_scan_runs ON scan_runs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_assets ON assets
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
