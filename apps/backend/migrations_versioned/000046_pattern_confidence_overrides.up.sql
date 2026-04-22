CREATE TABLE IF NOT EXISTS pattern_confidence_overrides (
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    pattern_code        TEXT NOT NULL,
    min_confidence_score INT NOT NULL DEFAULT 50,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, pattern_code)
);
