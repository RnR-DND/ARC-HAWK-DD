-- Migration: 000024_discovery_reports
-- Description: Discovery module — async generated board reports (PDF/CSV/JSON), produced by in-process worker pool.

CREATE TABLE IF NOT EXISTS discovery_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    snapshot_id UUID REFERENCES discovery_snapshots(id) ON DELETE SET NULL,
    requested_by UUID,
    format TEXT NOT NULL CHECK (format IN ('pdf','csv','json','html')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','completed','failed')),
    content BYTEA,
    content_type TEXT,
    error TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    size_bytes INTEGER
);

CREATE INDEX IF NOT EXISTS idx_discovery_reports_tenant_requested ON discovery_reports(tenant_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_discovery_reports_status ON discovery_reports(status);
CREATE INDEX IF NOT EXISTS idx_discovery_reports_tenant_status ON discovery_reports(tenant_id, status);
