DROP POLICY IF EXISTS tenant_isolation_findings ON findings;
DROP POLICY IF EXISTS tenant_isolation_scan_runs ON scan_runs;
DROP POLICY IF EXISTS tenant_isolation_assets ON assets;
DROP POLICY IF EXISTS tenant_isolation_connections ON connections;
ALTER TABLE findings DISABLE ROW LEVEL SECURITY;
ALTER TABLE scan_runs DISABLE ROW LEVEL SECURITY;
ALTER TABLE assets DISABLE ROW LEVEL SECURITY;
ALTER TABLE connections DISABLE ROW LEVEL SECURITY;
