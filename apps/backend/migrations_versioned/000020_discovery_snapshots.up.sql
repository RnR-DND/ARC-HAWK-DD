-- Migration: 000020_discovery_snapshots
-- Description: Discovery module — point-in-time snapshot headers for trend tracking and board reporting.

CREATE TABLE IF NOT EXISTS discovery_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    taken_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_count INTEGER NOT NULL DEFAULT 0 CHECK (source_count >= 0),
    asset_count INTEGER NOT NULL DEFAULT 0 CHECK (asset_count >= 0),
    finding_count INTEGER NOT NULL DEFAULT 0 CHECK (finding_count >= 0),
    high_risk_count INTEGER NOT NULL DEFAULT 0 CHECK (high_risk_count >= 0),
    composite_risk_score NUMERIC(6,2) NOT NULL DEFAULT 0,
    trigger TEXT NOT NULL CHECK (trigger IN ('manual','cron')),
    triggered_by UUID,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','completed','failed')),
    error TEXT,
    duration_ms BIGINT,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_discovery_snapshots_tenant_taken ON discovery_snapshots(tenant_id, taken_at DESC);
CREATE INDEX IF NOT EXISTS idx_discovery_snapshots_tenant_status ON discovery_snapshots(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_discovery_snapshots_status ON discovery_snapshots(status);
