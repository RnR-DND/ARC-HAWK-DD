-- C-2: Persistent JWT token blacklist
-- Tokens stored here remain revoked across restarts and replicas
CREATE TABLE IF NOT EXISTS token_blacklist (
    token_hash TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-cleanup: index on expires_at for efficient TTL purge
CREATE INDEX idx_token_blacklist_expires ON token_blacklist (expires_at);

-- M-16: Add missing indexes on findings table for query performance
CREATE INDEX IF NOT EXISTS idx_findings_scan_run_id ON findings (scan_run_id);
CREATE INDEX IF NOT EXISTS idx_findings_asset_id ON findings (asset_id);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings (severity);
CREATE INDEX IF NOT EXISTS idx_findings_tenant_id ON findings (tenant_id);
