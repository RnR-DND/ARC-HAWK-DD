-- Migration: 000022_discovery_risk_scores
-- Description: Discovery module — per-asset composite risk score history.

CREATE TABLE IF NOT EXISTS discovery_risk_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    asset_id UUID NOT NULL,
    snapshot_id UUID REFERENCES discovery_snapshots(id) ON DELETE SET NULL,
    score NUMERIC(6,2) NOT NULL,
    contributing_factors JSONB NOT NULL DEFAULT '{}'::jsonb,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovery_risk_scores_tenant_asset_computed ON discovery_risk_scores(tenant_id, asset_id, computed_at DESC);
CREATE INDEX IF NOT EXISTS idx_discovery_risk_scores_snapshot ON discovery_risk_scores(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_discovery_risk_scores_tenant_score ON discovery_risk_scores(tenant_id, score DESC);
