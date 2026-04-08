-- Migration: 000021_discovery_snapshot_facts
-- Description: Discovery module — per-snapshot detail rows for drift detection and report drilldowns.

CREATE TABLE IF NOT EXISTS discovery_snapshot_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID NOT NULL REFERENCES discovery_snapshots(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    source_id UUID,
    source_name TEXT,
    classification TEXT NOT NULL,
    asset_count INTEGER NOT NULL DEFAULT 0 CHECK (asset_count >= 0),
    finding_count INTEGER NOT NULL DEFAULT 0 CHECK (finding_count >= 0),
    sensitivity_avg NUMERIC(5,2) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_discovery_snapshot_facts_snapshot ON discovery_snapshot_facts(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_discovery_snapshot_facts_tenant ON discovery_snapshot_facts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_discovery_snapshot_facts_tenant_classification ON discovery_snapshot_facts(tenant_id, classification);
