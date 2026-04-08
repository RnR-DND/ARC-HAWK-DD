-- Migration: 000019_discovery_inventory
-- Description: Discovery module — denormalized inventory table refreshed by snapshot worker.
-- Synthesizes asset + classification + finding state into a single row per (tenant, asset, classification).

CREATE TABLE IF NOT EXISTS discovery_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    asset_name TEXT NOT NULL,
    source_id UUID,
    source_name TEXT,
    classification TEXT NOT NULL,
    sensitivity SMALLINT NOT NULL DEFAULT 0 CHECK (sensitivity BETWEEN 0 AND 100),
    finding_count INTEGER NOT NULL DEFAULT 0 CHECK (finding_count >= 0),
    last_scanned_at TIMESTAMPTZ,
    refreshed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovery_inventory_tenant ON discovery_inventory(tenant_id);
CREATE INDEX IF NOT EXISTS idx_discovery_inventory_tenant_asset ON discovery_inventory(tenant_id, asset_id);
CREATE INDEX IF NOT EXISTS idx_discovery_inventory_tenant_classification ON discovery_inventory(tenant_id, classification);
CREATE INDEX IF NOT EXISTS idx_discovery_inventory_refreshed_at ON discovery_inventory(refreshed_at);

-- Uniqueness: one row per (tenant, asset, classification) — refresh upserts.
CREATE UNIQUE INDEX IF NOT EXISTS uq_discovery_inventory_tenant_asset_classification
    ON discovery_inventory(tenant_id, asset_id, classification);
