-- Migration: 000023_discovery_drift_events
-- Description: Discovery module — drift event log emitted by drift detection service after each snapshot.

CREATE TABLE IF NOT EXISTS discovery_drift_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    snapshot_id UUID NOT NULL REFERENCES discovery_snapshots(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL CHECK (event_type IN (
        'asset_added','asset_removed','classification_changed',
        'risk_increased','risk_decreased','finding_count_spike'
    )),
    asset_id UUID NOT NULL,
    before_state JSONB,
    after_state JSONB,
    severity TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('low','medium','high','critical')),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovery_drift_events_tenant_detected ON discovery_drift_events(tenant_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_discovery_drift_events_snapshot ON discovery_drift_events(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_discovery_drift_events_asset ON discovery_drift_events(asset_id);
CREATE INDEX IF NOT EXISTS idx_discovery_drift_events_tenant_severity ON discovery_drift_events(tenant_id, severity);
